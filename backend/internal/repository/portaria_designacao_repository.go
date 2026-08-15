package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrPortariaDesignacaoNotFound é retornado quando nenhum registro de
// designação corresponde ao ID informado.
var ErrPortariaDesignacaoNotFound = errors.New("repository: registro de designação não encontrado")

// PortariaDesignacaoRepository abstrai o acesso à tabela
// `portarias_designacao` — histórico auditável de designação de
// fiscal/suplente/gestor/fiscal setorial por contrato (ver
// models.PortariaDesignacao).
type PortariaDesignacaoRepository interface {
	Create(ctx context.Context, designacao *models.PortariaDesignacao) error
	Update(ctx context.Context, designacao *models.PortariaDesignacao) error
	ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.PortariaDesignacao, error)
	// FindAtivaPorContratoEPapel retorna o registro ativo (DataRevogacao
	// nula) mais recente de um papel específico para um contrato — usado
	// pra saber quem é o fiscal/gestor/fiscal setorial atual e pra fechar
	// (revogar) esse registro quando uma nova designação do mesmo papel é
	// criada.
	FindAtivaPorContratoEPapel(ctx context.Context, contratoID uuid.UUID, papel models.PapelDesignacao) (*models.PortariaDesignacao, error)
}

type gormPortariaDesignacaoRepository struct {
	db *gorm.DB
}

// NewPortariaDesignacaoRepository constrói um PortariaDesignacaoRepository
// apoiado em GORM/Postgres.
func NewPortariaDesignacaoRepository(db *gorm.DB) PortariaDesignacaoRepository {
	return &gormPortariaDesignacaoRepository{db: db}
}

func (r *gormPortariaDesignacaoRepository) Create(ctx context.Context, designacao *models.PortariaDesignacao) error {
	if err := r.db.WithContext(ctx).Create(designacao).Error; err != nil {
		return fmt.Errorf("repository: criar designação: %w", err)
	}
	return nil
}

func (r *gormPortariaDesignacaoRepository) Update(ctx context.Context, designacao *models.PortariaDesignacao) error {
	// Omit(clause.Associations): mesma cautela documentada em
	// contrato_repository.go/processo_pagamento_repository.go — este
	// Update só é usado pra fechar um registro (setar DataRevogacao), nunca
	// pra alterar ContratoID/ServidorID.
	if err := r.db.WithContext(ctx).Omit("Contrato", "Servidor").Save(designacao).Error; err != nil {
		return fmt.Errorf("repository: atualizar designação: %w", err)
	}
	return nil
}

func (r *gormPortariaDesignacaoRepository) ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.PortariaDesignacao, error) {
	var designacoes []models.PortariaDesignacao

	err := r.db.WithContext(ctx).
		Preload("Servidor").
		Where("contrato_id = ?", contratoID).
		Order("data_designacao DESC").
		Find(&designacoes).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar designações do contrato: %w", err)
	}

	return designacoes, nil
}

func (r *gormPortariaDesignacaoRepository) FindAtivaPorContratoEPapel(ctx context.Context, contratoID uuid.UUID, papel models.PapelDesignacao) (*models.PortariaDesignacao, error) {
	var designacao models.PortariaDesignacao

	err := r.db.WithContext(ctx).
		Where("contrato_id = ? AND papel = ? AND data_revogacao IS NULL", contratoID, papel).
		Order("data_designacao DESC").
		First(&designacao).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPortariaDesignacaoNotFound
		}
		return nil, fmt.Errorf("repository: buscar designação ativa: %w", err)
	}

	return &designacao, nil
}
