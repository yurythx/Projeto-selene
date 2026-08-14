package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"projeto-selene/internal/models"
)

// ErrProcessoNotFound é retornado quando nenhum processo de pagamento
// corresponde ao ID informado.
var ErrProcessoNotFound = errors.New("repository: processo de pagamento não encontrado")

// ProcessoPagamentoRepository abstrai o acesso à tabela `processos_pagamento`
// (os "cards" do Kanban).
type ProcessoPagamentoRepository interface {
	Create(ctx context.Context, processo *models.ProcessoPagamento) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.ProcessoPagamento, error)
	ListByEtapa(ctx context.Context, etapaID int, pagina Pagina) (ResultadoPaginado[models.ProcessoPagamento], error)
	ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.ProcessoPagamento, error)
	Update(ctx context.Context, processo *models.ProcessoPagamento) error
	// ListAtivosComContrato retorna todos os processos com
	// Status="Ativo" (não concluídos), com Contrato pré-carregado — usado
	// pelo RadarService (Fase 1 do roadmap) pra varrer certidões
	// vencidas/vencendo e tempo parado na etapa numa única query, sem 1
	// query por contrato. Não filtra por Contrato.Ativo aqui (evitaria um
	// join) — quem chama filtra em memória.
	ListAtivosComContrato(ctx context.Context) ([]models.ProcessoPagamento, error)
}

type gormProcessoPagamentoRepository struct {
	db *gorm.DB
}

// NewProcessoPagamentoRepository constrói um ProcessoPagamentoRepository
// apoiado em GORM/Postgres. Aceita tanto *gorm.DB quanto uma transação
// (*gorm.DB retornado por db.Transaction), permitindo que o service
// componha operações atômicas sem que este pacote precise conhecer o
// conceito de transação.
func NewProcessoPagamentoRepository(db *gorm.DB) ProcessoPagamentoRepository {
	return &gormProcessoPagamentoRepository{db: db}
}

func (r *gormProcessoPagamentoRepository) Create(ctx context.Context, processo *models.ProcessoPagamento) error {
	if err := r.db.WithContext(ctx).Create(processo).Error; err != nil {
		return fmt.Errorf("repository: criar processo de pagamento: %w", err)
	}

	return nil
}

func (r *gormProcessoPagamentoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ProcessoPagamento, error) {
	var processo models.ProcessoPagamento

	err := r.db.WithContext(ctx).
		Preload("Contrato").
		Preload("Contrato.Fiscal").
		Preload("EtapaAtual").
		First(&processo, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProcessoNotFound
		}
		return nil, fmt.Errorf("repository: buscar processo de pagamento por id: %w", err)
	}

	return &processo, nil
}

func (r *gormProcessoPagamentoRepository) ListByEtapa(ctx context.Context, etapaID int, pagina Pagina) (ResultadoPaginado[models.ProcessoPagamento], error) {
	pagina = pagina.Normalizada()

	var total int64
	if err := r.db.WithContext(ctx).Model(&models.ProcessoPagamento{}).Where("etapa_atual_id = ?", etapaID).Count(&total).Error; err != nil {
		return ResultadoPaginado[models.ProcessoPagamento]{}, fmt.Errorf("repository: contar processos por etapa: %w", err)
	}

	var processos []models.ProcessoPagamento
	err := r.db.WithContext(ctx).
		Preload("Contrato").
		Where("etapa_atual_id = ?", etapaID).
		Order("created_at").
		Offset(pagina.Offset()).
		Limit(pagina.Tamanho).
		Find(&processos).Error
	if err != nil {
		return ResultadoPaginado[models.ProcessoPagamento]{}, fmt.Errorf("repository: listar processos por etapa: %w", err)
	}

	return ResultadoPaginado[models.ProcessoPagamento]{
		Dados:         processos,
		Total:         total,
		Pagina:        pagina.Numero,
		TamanhoPagina: pagina.Tamanho,
	}, nil
}

func (r *gormProcessoPagamentoRepository) ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.ProcessoPagamento, error) {
	var processos []models.ProcessoPagamento

	err := r.db.WithContext(ctx).
		Preload("EtapaAtual").
		Where("contrato_id = ?", contratoID).
		Order("mes_referencia").
		Find(&processos).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar processos por contrato: %w", err)
	}

	return processos, nil
}

func (r *gormProcessoPagamentoRepository) ListAtivosComContrato(ctx context.Context) ([]models.ProcessoPagamento, error) {
	var processos []models.ProcessoPagamento

	err := r.db.WithContext(ctx).
		Preload("Contrato").
		Preload("EtapaAtual").
		Where("status = ?", string(models.StatusProcessoAtivo)).
		Find(&processos).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar processos ativos com contrato: %w", err)
	}

	return processos, nil
}

// Update salva as colunas próprias de ProcessoPagamento, sem tocar nas
// tabelas associadas (Contrato, EtapaAtual).
//
// Omit(clause.Associations) é obrigatório aqui: o processo normalmente
// chega com Contrato/EtapaAtual pré-carregados (FindByID faz Preload), e
// sem esse Omit o GORM re-salva essas associações BelongsTo antes do
// registro principal — reescrevendo ContratoID/EtapaAtualID de volta para
// o ID da associação carregada, silenciosamente desfazendo qualquer
// mudança feita direto no campo (ex: EtapaAtualID ao avançar o Kanban).
func (r *gormProcessoPagamentoRepository) Update(ctx context.Context, processo *models.ProcessoPagamento) error {
	if err := r.db.WithContext(ctx).Omit(clause.Associations).Save(processo).Error; err != nil {
		return fmt.Errorf("repository: atualizar processo de pagamento: %w", err)
	}

	return nil
}
