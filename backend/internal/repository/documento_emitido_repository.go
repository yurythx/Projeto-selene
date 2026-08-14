package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrDocumentoEmitidoNotFound é retornado quando nenhum documento emitido
// corresponde aos critérios de busca informados (ex: código de verificação
// inexistente).
var ErrDocumentoEmitidoNotFound = errors.New("repository: documento emitido não encontrado")

// DocumentoEmitidoRepository abstrai o acesso à tabela
// `documentos_emitidos` (Módulo 2 do roadmap — Gerador Inteligente de
// Documentos Legais).
type DocumentoEmitidoRepository interface {
	Create(ctx context.Context, documento *models.DocumentoEmitido) error
	FindByCodigoVerificacao(ctx context.Context, codigo string) (*models.DocumentoEmitido, error)
	// ListByContrato retorna o histórico de documentos emitidos de um
	// contrato, mais recente primeiro — usado tanto pela tela do contrato
	// quanto pelo Dossiê do Fornecedor (Fase 4).
	ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.DocumentoEmitido, error)
	// ListByContratoIDs retorna o histórico de vários contratos de uma vez
	// (agrupados por ContratoID pelo chamador) — usado pelo Dossiê do
	// Fornecedor (Fase 4) pra evitar 1 query por contrato do CNPJ.
	ListByContratoIDs(ctx context.Context, contratoIDs []uuid.UUID) ([]models.DocumentoEmitido, error)
}

type gormDocumentoEmitidoRepository struct {
	db *gorm.DB
}

// NewDocumentoEmitidoRepository constrói um DocumentoEmitidoRepository
// apoiado em GORM/Postgres.
func NewDocumentoEmitidoRepository(db *gorm.DB) DocumentoEmitidoRepository {
	return &gormDocumentoEmitidoRepository{db: db}
}

func (r *gormDocumentoEmitidoRepository) Create(ctx context.Context, documento *models.DocumentoEmitido) error {
	if err := r.db.WithContext(ctx).Create(documento).Error; err != nil {
		return fmt.Errorf("repository: criar documento emitido: %w", err)
	}

	return nil
}

func (r *gormDocumentoEmitidoRepository) FindByCodigoVerificacao(ctx context.Context, codigo string) (*models.DocumentoEmitido, error) {
	var documento models.DocumentoEmitido

	err := r.db.WithContext(ctx).
		Preload("Contrato").
		Preload("ProcessoPagamento").
		Preload("GeradoPor").
		First(&documento, "codigo_verificacao = ?", codigo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocumentoEmitidoNotFound
		}
		return nil, fmt.Errorf("repository: buscar documento emitido por código de verificação: %w", err)
	}

	return &documento, nil
}

func (r *gormDocumentoEmitidoRepository) ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.DocumentoEmitido, error) {
	var documentos []models.DocumentoEmitido

	err := r.db.WithContext(ctx).
		Preload("GeradoPor").
		Preload("ProcessoPagamento").
		Where("contrato_id = ?", contratoID).
		Order("created_at DESC").
		Find(&documentos).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar documentos emitidos do contrato: %w", err)
	}

	return documentos, nil
}

func (r *gormDocumentoEmitidoRepository) ListByContratoIDs(ctx context.Context, contratoIDs []uuid.UUID) ([]models.DocumentoEmitido, error) {
	if len(contratoIDs) == 0 {
		return nil, nil
	}

	var documentos []models.DocumentoEmitido

	err := r.db.WithContext(ctx).
		Preload("GeradoPor").
		Preload("ProcessoPagamento").
		Where("contrato_id IN ?", contratoIDs).
		Order("created_at DESC").
		Find(&documentos).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar documentos emitidos de vários contratos: %w", err)
	}

	return documentos, nil
}
