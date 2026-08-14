package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// DocumentoAnexoRepository abstrai o acesso à tabela `documentos_anexos`.
type DocumentoAnexoRepository interface {
	Create(ctx context.Context, documento *models.DocumentoAnexo) error
	ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.DocumentoAnexo, error)
	FindByProcessoAndHash(ctx context.Context, processoID uuid.UUID, hash string) (*models.DocumentoAnexo, error)
}

// ErrDocumentoNotFound é retornado quando nenhum documento anexo
// corresponde aos critérios de busca informados.
var ErrDocumentoNotFound = errors.New("repository: documento anexo não encontrado")

type gormDocumentoAnexoRepository struct {
	db *gorm.DB
}

// NewDocumentoAnexoRepository constrói um DocumentoAnexoRepository apoiado em GORM/Postgres.
func NewDocumentoAnexoRepository(db *gorm.DB) DocumentoAnexoRepository {
	return &gormDocumentoAnexoRepository{db: db}
}

func (r *gormDocumentoAnexoRepository) Create(ctx context.Context, documento *models.DocumentoAnexo) error {
	if err := r.db.WithContext(ctx).Create(documento).Error; err != nil {
		return fmt.Errorf("repository: criar documento anexo: %w", err)
	}

	return nil
}

func (r *gormDocumentoAnexoRepository) ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.DocumentoAnexo, error) {
	var documentos []models.DocumentoAnexo

	err := r.db.WithContext(ctx).
		Preload("TipoDocumento").
		Preload("EnviadoPor").
		Where("processo_pagamento_id = ?", processoID).
		Order("data_upload").
		Find(&documentos).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar documentos do processo: %w", err)
	}

	return documentos, nil
}

// FindByProcessoAndHash implementa a checagem de duplicidade descrita na
// documentação de domínio: o mesmo arquivo (mesmo SHA-256) já anexado ao
// mesmo processo é reaproveitado em vez de re-enviado.
func (r *gormDocumentoAnexoRepository) FindByProcessoAndHash(ctx context.Context, processoID uuid.UUID, hash string) (*models.DocumentoAnexo, error) {
	var documento models.DocumentoAnexo

	err := r.db.WithContext(ctx).
		Preload("TipoDocumento").
		First(&documento, "processo_pagamento_id = ? AND hash_arquivo = ?", processoID, hash).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocumentoNotFound
		}
		return nil, fmt.Errorf("repository: buscar documento por hash: %w", err)
	}

	return &documento, nil
}
