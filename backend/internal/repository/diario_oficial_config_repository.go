package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"projeto-selene/internal/models"
)

// ErrDiarioOficialConfigNotFound é devolvido quando nenhum admin salvou
// ainda uma configuração de integração com o Diário Oficial — a tela de
// busca simplesmente fica indisponível até alguém configurar (ver
// DiarioOficialService).
var ErrDiarioOficialConfigNotFound = errors.New("repository: configuração de diário oficial não encontrada")

// DiarioOficialConfigRepository abstrai a persistência da configuração
// da API do Diário Oficial (linha única, ver a migration 000013).
type DiarioOficialConfigRepository interface {
	Buscar(ctx context.Context) (*models.DiarioOficialConfig, error)
	Salvar(ctx context.Context, cfg *models.DiarioOficialConfig) error
}

type gormDiarioOficialConfigRepository struct {
	db *gorm.DB
}

func NewDiarioOficialConfigRepository(db *gorm.DB) DiarioOficialConfigRepository {
	return &gormDiarioOficialConfigRepository{db: db}
}

func (r *gormDiarioOficialConfigRepository) Buscar(ctx context.Context) (*models.DiarioOficialConfig, error) {
	var cfg models.DiarioOficialConfig
	err := r.db.WithContext(ctx).Preload("UpdatedBy").First(&cfg, "id = 1").Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDiarioOficialConfigNotFound
		}
		return nil, fmt.Errorf("repository: buscar configuração de diário oficial: %w", err)
	}
	return &cfg, nil
}

// Salvar faz upsert da linha única (id=1) — mesmo padrão de
// KeycloakConfigRepository.Salvar (ver o comentário lá).
func (r *gormDiarioOficialConfigRepository) Salvar(ctx context.Context, cfg *models.DiarioOficialConfig) error {
	cfg.ID = 1
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"base_url", "api_key", "updated_by_id", "updated_at"}),
	}).Create(cfg).Error
	if err != nil {
		return fmt.Errorf("repository: salvar configuração de diário oficial: %w", err)
	}
	return nil
}
