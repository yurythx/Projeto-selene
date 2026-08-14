package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrVistoriaNotFound é retornado quando nenhum registro de vistoria
// corresponde ao ID informado.
var ErrVistoriaNotFound = errors.New("repository: registro de vistoria não encontrado")

// VistoriaRepository abstrai o acesso à tabela `registros_vistoria`
// (Módulo 3 do roadmap).
type VistoriaRepository interface {
	Create(ctx context.Context, vistoria *models.RegistroVistoria) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.RegistroVistoria, error)
	ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.RegistroVistoria, error)
}

type gormVistoriaRepository struct {
	db *gorm.DB
}

// NewVistoriaRepository constrói um VistoriaRepository apoiado em GORM/Postgres.
func NewVistoriaRepository(db *gorm.DB) VistoriaRepository {
	return &gormVistoriaRepository{db: db}
}

func (r *gormVistoriaRepository) Create(ctx context.Context, vistoria *models.RegistroVistoria) error {
	if err := r.db.WithContext(ctx).Create(vistoria).Error; err != nil {
		return fmt.Errorf("repository: criar registro de vistoria: %w", err)
	}

	return nil
}

func (r *gormVistoriaRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.RegistroVistoria, error) {
	var vistoria models.RegistroVistoria

	err := r.db.WithContext(ctx).
		Preload("Fotos").
		Preload("Fiscal").
		Preload("ProcessoPagamento").
		Preload("ProcessoPagamento.Contrato").
		First(&vistoria, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVistoriaNotFound
		}
		return nil, fmt.Errorf("repository: buscar registro de vistoria por id: %w", err)
	}

	return &vistoria, nil
}

func (r *gormVistoriaRepository) ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.RegistroVistoria, error) {
	var vistorias []models.RegistroVistoria

	err := r.db.WithContext(ctx).
		Preload("Fotos").
		Preload("Fiscal").
		Where("processo_pagamento_id = ?", processoID).
		Order("data_hora DESC").
		Find(&vistorias).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar vistorias do processo: %w", err)
	}

	return vistorias, nil
}
