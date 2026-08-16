package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrModeloDocumentoVersaoNotFound é retornado quando nenhuma versão
// corresponde ao ID informado.
var ErrModeloDocumentoVersaoNotFound = errors.New("repository: versão de modelo de documento não encontrada")

// ModeloDocumentoVersaoRepository abstrai o acesso à tabela
// `modelo_documento_versoes` — histórico de arquivos publicados por
// categoria (ver o comentário em models.ModeloDocumentoVersao sobre não
// deduplicar por hash, diferente de DocumentoAnexo/FotoVistoria).
type ModeloDocumentoVersaoRepository interface {
	Create(ctx context.Context, versao *models.ModeloDocumentoVersao) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.ModeloDocumentoVersao, error)
}

type gormModeloDocumentoVersaoRepository struct {
	db *gorm.DB
}

// NewModeloDocumentoVersaoRepository constrói um
// ModeloDocumentoVersaoRepository apoiado em GORM/Postgres.
func NewModeloDocumentoVersaoRepository(db *gorm.DB) ModeloDocumentoVersaoRepository {
	return &gormModeloDocumentoVersaoRepository{db: db}
}

func (r *gormModeloDocumentoVersaoRepository) Create(ctx context.Context, versao *models.ModeloDocumentoVersao) error {
	if err := r.db.WithContext(ctx).Create(versao).Error; err != nil {
		return fmt.Errorf("repository: criar versão de modelo de documento: %w", err)
	}

	return nil
}

func (r *gormModeloDocumentoVersaoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ModeloDocumentoVersao, error) {
	var versao models.ModeloDocumentoVersao

	err := r.db.WithContext(ctx).First(&versao, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModeloDocumentoVersaoNotFound
		}
		return nil, fmt.Errorf("repository: buscar versão de modelo de documento por id: %w", err)
	}

	return &versao, nil
}
