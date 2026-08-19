package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrTipoDocumentoNotFound é retornado quando nenhum tipo de documento
// corresponde ao identificador/nome informado.
var ErrTipoDocumentoNotFound = errors.New("repository: tipo de documento não encontrado")

// TipoDocumentoRepository abstrai o acesso à tabela `tipos_documento`, que
// é semeada uma única vez na implantação (ver internal/database.Seed) e
// tratada como somente-leitura pela aplicação.
type TipoDocumentoRepository interface {
	List(ctx context.Context) ([]models.TipoDocumento, error)
	FindByID(ctx context.Context, id int) (*models.TipoDocumento, error)
	FindByNome(ctx context.Context, nome string) (*models.TipoDocumento, error)
}

type gormTipoDocumentoRepository struct {
	db *gorm.DB
}

// NewTipoDocumentoRepository constrói um TipoDocumentoRepository apoiado em GORM/Postgres.
func NewTipoDocumentoRepository(db *gorm.DB) TipoDocumentoRepository {
	return &gormTipoDocumentoRepository{db: db}
}

func (r *gormTipoDocumentoRepository) List(ctx context.Context) ([]models.TipoDocumento, error) {
	tipos := []models.TipoDocumento{}

	if err := r.db.WithContext(ctx).Order("nome").Find(&tipos).Error; err != nil {
		return nil, fmt.Errorf("repository: listar tipos de documento: %w", err)
	}

	return tipos, nil
}

func (r *gormTipoDocumentoRepository) FindByID(ctx context.Context, id int) (*models.TipoDocumento, error) {
	var tipo models.TipoDocumento

	err := r.db.WithContext(ctx).First(&tipo, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTipoDocumentoNotFound
		}
		return nil, fmt.Errorf("repository: buscar tipo de documento por id: %w", err)
	}

	return &tipo, nil
}

func (r *gormTipoDocumentoRepository) FindByNome(ctx context.Context, nome string) (*models.TipoDocumento, error) {
	var tipo models.TipoDocumento

	err := r.db.WithContext(ctx).First(&tipo, "nome = ?", nome).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTipoDocumentoNotFound
		}
		return nil, fmt.Errorf("repository: buscar tipo de documento por nome: %w", err)
	}

	return &tipo, nil
}
