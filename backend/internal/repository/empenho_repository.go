package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrEmpenhoNotFound é retornado quando nenhum Empenho corresponde ao ID
// informado.
var ErrEmpenhoNotFound = errors.New("repository: empenho não encontrado")

// EmpenhoRepository abstrai o acesso à tabela `empenhos` — ver
// models.Empenho para o porquê deste ser um registro PARALELO/informativo,
// não a fonte de verdade orçamentária.
type EmpenhoRepository interface {
	Create(ctx context.Context, empenho *models.Empenho) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Empenho, error)
	ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.Empenho, error)
}

type gormEmpenhoRepository struct {
	db *gorm.DB
}

// NewEmpenhoRepository constrói um EmpenhoRepository apoiado em
// GORM/Postgres.
func NewEmpenhoRepository(db *gorm.DB) EmpenhoRepository {
	return &gormEmpenhoRepository{db: db}
}

func (r *gormEmpenhoRepository) Create(ctx context.Context, empenho *models.Empenho) error {
	if err := r.db.WithContext(ctx).Create(empenho).Error; err != nil {
		return fmt.Errorf("repository: criar empenho: %w", err)
	}
	return nil
}

func (r *gormEmpenhoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Empenho, error) {
	var empenho models.Empenho

	err := r.db.WithContext(ctx).First(&empenho, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmpenhoNotFound
		}
		return nil, fmt.Errorf("repository: buscar empenho por id: %w", err)
	}

	return &empenho, nil
}

func (r *gormEmpenhoRepository) ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.Empenho, error) {
	empenhos := []models.Empenho{}

	err := r.db.WithContext(ctx).
		Where("contrato_id = ?", contratoID).
		Order("data_emissao DESC").
		Find(&empenhos).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar empenhos do contrato: %w", err)
	}

	return empenhos, nil
}

// --- MovimentacaoEmpenhoRepository ---

// MovimentacaoEmpenhoRepository abstrai o acesso à tabela
// `movimentacoes_empenho` — lançamentos imutáveis (nunca há Update: uma
// movimentação registrada é fato histórico, corrigi-la significa lançar
// uma nova movimentação compensatória, não editar a existente).
type MovimentacaoEmpenhoRepository interface {
	Create(ctx context.Context, movimentacao *models.MovimentacaoEmpenho) error
	ListByEmpenho(ctx context.Context, empenhoID uuid.UUID) ([]models.MovimentacaoEmpenho, error)
}

type gormMovimentacaoEmpenhoRepository struct {
	db *gorm.DB
}

// NewMovimentacaoEmpenhoRepository constrói um
// MovimentacaoEmpenhoRepository apoiado em GORM/Postgres.
func NewMovimentacaoEmpenhoRepository(db *gorm.DB) MovimentacaoEmpenhoRepository {
	return &gormMovimentacaoEmpenhoRepository{db: db}
}

func (r *gormMovimentacaoEmpenhoRepository) Create(ctx context.Context, movimentacao *models.MovimentacaoEmpenho) error {
	if err := r.db.WithContext(ctx).Create(movimentacao).Error; err != nil {
		return fmt.Errorf("repository: criar movimentação de empenho: %w", err)
	}
	return nil
}

func (r *gormMovimentacaoEmpenhoRepository) ListByEmpenho(ctx context.Context, empenhoID uuid.UUID) ([]models.MovimentacaoEmpenho, error) {
	movimentacoes := []models.MovimentacaoEmpenho{}

	err := r.db.WithContext(ctx).
		Where("empenho_id = ?", empenhoID).
		Order("created_at ASC").
		Find(&movimentacoes).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar movimentações do empenho: %w", err)
	}

	return movimentacoes, nil
}
