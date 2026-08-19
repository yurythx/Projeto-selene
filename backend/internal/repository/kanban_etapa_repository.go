package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrEtapaNotFound é retornado quando nenhuma etapa do Kanban corresponde
// ao ID informado.
var ErrEtapaNotFound = errors.New("repository: etapa do kanban não encontrada")

// KanbanEtapaRepository abstrai o acesso à tabela `kanban_etapas`, que é
// semeada uma única vez na implantação (ver internal/database.Seed) e
// tratada como somente-leitura pela aplicação.
type KanbanEtapaRepository interface {
	List(ctx context.Context) ([]models.KanbanEtapa, error)
	FindByID(ctx context.Context, id int) (*models.KanbanEtapa, error)
}

type gormKanbanEtapaRepository struct {
	db *gorm.DB
}

// NewKanbanEtapaRepository constrói um KanbanEtapaRepository apoiado em GORM/Postgres.
func NewKanbanEtapaRepository(db *gorm.DB) KanbanEtapaRepository {
	return &gormKanbanEtapaRepository{db: db}
}

func (r *gormKanbanEtapaRepository) List(ctx context.Context) ([]models.KanbanEtapa, error) {
	etapas := []models.KanbanEtapa{}

	if err := r.db.WithContext(ctx).Order("posicao").Find(&etapas).Error; err != nil {
		return nil, fmt.Errorf("repository: listar etapas do kanban: %w", err)
	}

	return etapas, nil
}

func (r *gormKanbanEtapaRepository) FindByID(ctx context.Context, id int) (*models.KanbanEtapa, error) {
	var etapa models.KanbanEtapa

	err := r.db.WithContext(ctx).First(&etapa, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEtapaNotFound
		}
		return nil, fmt.Errorf("repository: buscar etapa do kanban por id: %w", err)
	}

	return &etapa, nil
}
