package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// KanbanLogRepository abstrai o acesso à tabela `kanban_logs`, o registro
// de auditoria de toda transição de etapa exigido pela documentação de
// domínio (Seção 7.3).
type KanbanLogRepository interface {
	Create(ctx context.Context, log *models.KanbanLog) error
	ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.KanbanLog, error)
	// ListByProcessos é a versão bulk de ListByProcesso — usada pelo
	// FornecedorService (Fase 4 do roadmap) para calcular o Score de
	// Pontualidade de todos os processos de um CNPJ numa única query, em
	// vez de uma por processo.
	ListByProcessos(ctx context.Context, processoIDs []uuid.UUID) ([]models.KanbanLog, error)
}

type gormKanbanLogRepository struct {
	db *gorm.DB
}

// NewKanbanLogRepository constrói um KanbanLogRepository apoiado em
// GORM/Postgres. Assim como ProcessoPagamentoRepository, aceita uma
// transação para permitir que o service grave o log atomicamente junto
// com a atualização da etapa.
func NewKanbanLogRepository(db *gorm.DB) KanbanLogRepository {
	return &gormKanbanLogRepository{db: db}
}

func (r *gormKanbanLogRepository) Create(ctx context.Context, log *models.KanbanLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("repository: criar log de transição do kanban: %w", err)
	}

	return nil
}

func (r *gormKanbanLogRepository) ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.KanbanLog, error) {
	logs := []models.KanbanLog{}

	err := r.db.WithContext(ctx).
		Preload("Usuario").
		Preload("EtapaOrigem").
		Preload("EtapaDestino").
		Where("processo_pagamento_id = ?", processoID).
		Order("movido_em").
		Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar logs do kanban por processo: %w", err)
	}

	return logs, nil
}

func (r *gormKanbanLogRepository) ListByProcessos(ctx context.Context, processoIDs []uuid.UUID) ([]models.KanbanLog, error) {
	if len(processoIDs) == 0 {
		return nil, nil
	}

	logs := []models.KanbanLog{}

	err := r.db.WithContext(ctx).
		Where("processo_pagamento_id IN ?", processoIDs).
		Order("processo_pagamento_id, movido_em").
		Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar logs do kanban por vários processos: %w", err)
	}

	return logs, nil
}
