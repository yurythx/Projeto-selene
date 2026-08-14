package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrFotoVistoriaNotFound é retornado quando nenhuma foto corresponde aos
// critérios de busca informados.
var ErrFotoVistoriaNotFound = errors.New("repository: foto de vistoria não encontrada")

// FotoVistoriaRepository abstrai o acesso à tabela `fotos_vistoria`
// (Módulo 3 do roadmap).
type FotoVistoriaRepository interface {
	Create(ctx context.Context, foto *models.FotoVistoria) error
	FindByVistoriaAndHash(ctx context.Context, vistoriaID uuid.UUID, hash string) (*models.FotoVistoria, error)
}

type gormFotoVistoriaRepository struct {
	db *gorm.DB
}

// NewFotoVistoriaRepository constrói um FotoVistoriaRepository apoiado em GORM/Postgres.
func NewFotoVistoriaRepository(db *gorm.DB) FotoVistoriaRepository {
	return &gormFotoVistoriaRepository{db: db}
}

func (r *gormFotoVistoriaRepository) Create(ctx context.Context, foto *models.FotoVistoria) error {
	if err := r.db.WithContext(ctx).Create(foto).Error; err != nil {
		return fmt.Errorf("repository: criar foto de vistoria: %w", err)
	}

	return nil
}

// FindByVistoriaAndHash implementa a mesma deduplicação por SHA-256 já
// usada em DocumentoAnexo — reenviar a mesma foto pra mesma vistoria
// reaproveita o registro existente em vez de duplicar.
func (r *gormFotoVistoriaRepository) FindByVistoriaAndHash(ctx context.Context, vistoriaID uuid.UUID, hash string) (*models.FotoVistoria, error) {
	var foto models.FotoVistoria

	err := r.db.WithContext(ctx).
		First(&foto, "vistoria_id = ? AND hash_arquivo = ?", vistoriaID, hash).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFotoVistoriaNotFound
		}
		return nil, fmt.Errorf("repository: buscar foto de vistoria por hash: %w", err)
	}

	return &foto, nil
}
