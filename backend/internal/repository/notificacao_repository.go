package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"projeto-selene/internal/models"
)

// ErrNotificacaoNotFound é retornado por MarcarLida quando o ID não
// existe OU não pertence ao usuário que está tentando marcar — as duas
// situações são indistinguíveis de propósito (não vazamos se um ID de
// outro usuário existe).
var ErrNotificacaoNotFound = errors.New("repository: notificação não encontrada")

// NotificacaoRepository abstrai a persistência de notificações in-app
// (ver a migration 000014 pro esquema completo).
type NotificacaoRepository interface {
	// Criar insere uma notificação nova — ON CONFLICT DO NOTHING no par
	// (usuario_id, chave_alerta), então `criada=false` significa "já
	// existia" (não é um erro, é o caminho normal de deduplicação, ver
	// NotificacaoService.GerarAlertas).
	Criar(ctx context.Context, n *models.Notificacao) (criada bool, err error)
	Listar(ctx context.Context, usuarioID uuid.UUID) ([]models.Notificacao, error)
	ContarNaoLidas(ctx context.Context, usuarioID uuid.UUID) (int64, error)
	MarcarLida(ctx context.Context, usuarioID, notificacaoID uuid.UUID) error
	MarcarTodasLidas(ctx context.Context, usuarioID uuid.UUID) error
}

type gormNotificacaoRepository struct {
	db *gorm.DB
}

func NewNotificacaoRepository(db *gorm.DB) NotificacaoRepository {
	return &gormNotificacaoRepository{db: db}
}

func (r *gormNotificacaoRepository) Criar(ctx context.Context, n *models.Notificacao) (bool, error) {
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "usuario_id"}, {Name: "chave_alerta"}},
		DoNothing: true,
	}).Create(n)
	if result.Error != nil {
		return false, fmt.Errorf("repository: criar notificação: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// Listar devolve as notificações de um usuário, não-lidas primeiro, mais
// recentes primeiro dentro de cada grupo — sem paginação de propósito
// (mesmo raciocínio de RadarService.Listar: o volume esperado é baixo o
// bastante pra não precisar).
func (r *gormNotificacaoRepository) Listar(ctx context.Context, usuarioID uuid.UUID) ([]models.Notificacao, error) {
	notificacoes := []models.Notificacao{}
	err := r.db.WithContext(ctx).
		Preload("Contrato").
		Where("usuario_id = ?", usuarioID).
		Order("lida ASC, criada_em DESC").
		Find(&notificacoes).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar notificações: %w", err)
	}
	return notificacoes, nil
}

func (r *gormNotificacaoRepository) ContarNaoLidas(ctx context.Context, usuarioID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&models.Notificacao{}).
		Where("usuario_id = ? AND lida = false", usuarioID).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("repository: contar notificações não lidas: %w", err)
	}
	return total, nil
}

func (r *gormNotificacaoRepository) MarcarLida(ctx context.Context, usuarioID, notificacaoID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&models.Notificacao{}).
		Where("id = ? AND usuario_id = ?", notificacaoID, usuarioID).
		Updates(map[string]any{"lida": true, "lida_em": gorm.Expr("now()")})
	if result.Error != nil {
		return fmt.Errorf("repository: marcar notificação como lida: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotificacaoNotFound
	}
	return nil
}

func (r *gormNotificacaoRepository) MarcarTodasLidas(ctx context.Context, usuarioID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&models.Notificacao{}).
		Where("usuario_id = ? AND lida = false", usuarioID).
		Updates(map[string]any{"lida": true, "lida_em": gorm.Expr("now()")}).Error
	if err != nil {
		return fmt.Errorf("repository: marcar todas as notificações como lidas: %w", err)
	}
	return nil
}
