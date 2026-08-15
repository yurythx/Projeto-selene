package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrOcorrenciaNotFound é retornado quando nenhuma Ocorrencia corresponde
// ao ID informado.
var ErrOcorrenciaNotFound = errors.New("repository: ocorrência não encontrada")

// OcorrenciaRepository abstrai o acesso à tabela `ocorrencias` (ver
// models.Ocorrencia).
type OcorrenciaRepository interface {
	Create(ctx context.Context, ocorrencia *models.Ocorrencia) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Ocorrencia, error)
	Update(ctx context.Context, ocorrencia *models.Ocorrencia) error
	ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.Ocorrencia, error)
	// ListAbertasPorProcesso retorna as ocorrências do processo cujo Estado
	// ainda não é REGULARIZADA — usado por CanTransition (ver
	// fiscalizacao_service.go) para bloquear o avanço de etapa do Kanban
	// enquanto houver pendência sem tratativa (regra de Camada 2).
	ListAbertasPorProcesso(ctx context.Context, processoID uuid.UUID) ([]models.Ocorrencia, error)
}

type gormOcorrenciaRepository struct {
	db *gorm.DB
}

// NewOcorrenciaRepository constrói um OcorrenciaRepository apoiado em
// GORM/Postgres.
func NewOcorrenciaRepository(db *gorm.DB) OcorrenciaRepository {
	return &gormOcorrenciaRepository{db: db}
}

func (r *gormOcorrenciaRepository) Create(ctx context.Context, ocorrencia *models.Ocorrencia) error {
	if err := r.db.WithContext(ctx).Create(ocorrencia).Error; err != nil {
		return fmt.Errorf("repository: criar ocorrência: %w", err)
	}
	return nil
}

func (r *gormOcorrenciaRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Ocorrencia, error) {
	var ocorrencia models.Ocorrencia

	err := r.db.WithContext(ctx).
		Preload("RegistradoPor").
		First(&ocorrencia, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOcorrenciaNotFound
		}
		return nil, fmt.Errorf("repository: buscar ocorrência por id: %w", err)
	}

	return &ocorrencia, nil
}

func (r *gormOcorrenciaRepository) Update(ctx context.Context, ocorrencia *models.Ocorrencia) error {
	// Omit(clause.Associations): mesma cautela documentada em
	// contrato_repository.go/processo_pagamento_repository.go — evita que o
	// GORM resalve Contrato/ProcessoPagamento/RegistradoPor pré-carregados e
	// reverta silenciosamente uma mudança feita só no campo de FK/Estado.
	if err := r.db.WithContext(ctx).Omit("Contrato", "ProcessoPagamento", "RegistradoPor").Save(ocorrencia).Error; err != nil {
		return fmt.Errorf("repository: atualizar ocorrência: %w", err)
	}
	return nil
}

func (r *gormOcorrenciaRepository) ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.Ocorrencia, error) {
	ocorrencias := []models.Ocorrencia{}

	err := r.db.WithContext(ctx).
		Preload("RegistradoPor").
		Where("processo_pagamento_id = ?", processoID).
		Order("created_at DESC").
		Find(&ocorrencias).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar ocorrências do processo: %w", err)
	}

	return ocorrencias, nil
}

func (r *gormOcorrenciaRepository) ListAbertasPorProcesso(ctx context.Context, processoID uuid.UUID) ([]models.Ocorrencia, error) {
	ocorrencias := []models.Ocorrencia{}

	err := r.db.WithContext(ctx).
		Where("processo_pagamento_id = ? AND estado <> ?", processoID, models.OcorrenciaRegularizada).
		Find(&ocorrencias).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar ocorrências abertas do processo: %w", err)
	}

	return ocorrencias, nil
}
