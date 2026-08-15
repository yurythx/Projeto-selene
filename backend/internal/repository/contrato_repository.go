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

// ErrContratoNotFound é retornado quando nenhum contrato corresponde ao ID
// informado.
var ErrContratoNotFound = errors.New("repository: contrato não encontrado")

// ContratoRepository abstrai o acesso à tabela `contratos`.
type ContratoRepository interface {
	Create(ctx context.Context, contrato *models.Contrato) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Contrato, error)
	List(ctx context.Context, pagina Pagina) (ResultadoPaginado[models.Contrato], error)
	Update(ctx context.Context, contrato *models.Contrato) error
	// ListAtivos retorna todos os contratos com Ativo=true, sem paginação
	// — usado pelo RadarService (Fase 1 do roadmap) para varrer vigência
	// de contrato. Não pagina de propósito: o radar precisa ver TODOS os
	// contratos em risco, não uma página por vez; o volume esperado
	// (contratos ativos de uma prefeitura) é pequeno o bastante pra isso
	// ser seguro. Se deixar de ser, revisitar.
	ListAtivos(ctx context.Context) ([]models.Contrato, error)
	// ListTodos retorna TODOS os contratos (ativos e encerrados), sem
	// paginação — usado pelo FornecedorService (Fase 4 do roadmap) para
	// agrupar por CNPJ. Mesma justificativa de volume de ListAtivos: o
	// conjunto de contratos de uma prefeitura é pequeno o bastante pra uma
	// varredura completa em memória ser segura.
	ListTodos(ctx context.Context) ([]models.Contrato, error)
}

type gormContratoRepository struct {
	db *gorm.DB
}

// NewContratoRepository constrói um ContratoRepository apoiado em GORM/Postgres.
func NewContratoRepository(db *gorm.DB) ContratoRepository {
	return &gormContratoRepository{db: db}
}

func (r *gormContratoRepository) Create(ctx context.Context, contrato *models.Contrato) error {
	if err := r.db.WithContext(ctx).Create(contrato).Error; err != nil {
		return fmt.Errorf("repository: criar contrato: %w", err)
	}

	return nil
}

func (r *gormContratoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Contrato, error) {
	var contrato models.Contrato

	err := r.db.WithContext(ctx).Preload("Fiscal").First(&contrato, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContratoNotFound
		}
		return nil, fmt.Errorf("repository: buscar contrato por id: %w", err)
	}

	return &contrato, nil
}

func (r *gormContratoRepository) List(ctx context.Context, pagina Pagina) (ResultadoPaginado[models.Contrato], error) {
	pagina = pagina.Normalizada()

	var total int64
	if err := r.db.WithContext(ctx).Model(&models.Contrato{}).Count(&total).Error; err != nil {
		return ResultadoPaginado[models.Contrato]{}, fmt.Errorf("repository: contar contratos: %w", err)
	}

	contratos := []models.Contrato{}
	err := r.db.WithContext(ctx).
		Preload("Fiscal").
		Order("numero_contrato").
		Offset(pagina.Offset()).
		Limit(pagina.Tamanho).
		Find(&contratos).Error
	if err != nil {
		return ResultadoPaginado[models.Contrato]{}, fmt.Errorf("repository: listar contratos: %w", err)
	}

	return ResultadoPaginado[models.Contrato]{
		Dados:         contratos,
		Total:         total,
		Pagina:        pagina.Numero,
		TamanhoPagina: pagina.Tamanho,
	}, nil
}

func (r *gormContratoRepository) ListAtivos(ctx context.Context) ([]models.Contrato, error) {
	contratos := []models.Contrato{}

	err := r.db.WithContext(ctx).
		Where("ativo = ?", true).
		Order("numero_contrato").
		Find(&contratos).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar contratos ativos: %w", err)
	}

	return contratos, nil
}

func (r *gormContratoRepository) ListTodos(ctx context.Context) ([]models.Contrato, error) {
	contratos := []models.Contrato{}

	err := r.db.WithContext(ctx).
		Preload("Fiscal").
		Order("contratada_cnpj, numero_contrato").
		Find(&contratos).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar todos os contratos: %w", err)
	}

	return contratos, nil
}

// Update salva as colunas próprias de Contrato, sem tocar na tabela
// associada (Fiscal) — ver o comentário equivalente em
// ProcessoPagamentoRepository.Update sobre por que Omit(clause.Associations)
// é necessário sempre que o struct pode chegar com associações
// pré-carregadas.
func (r *gormContratoRepository) Update(ctx context.Context, contrato *models.Contrato) error {
	if err := r.db.WithContext(ctx).Omit(clause.Associations).Save(contrato).Error; err != nil {
		return fmt.Errorf("repository: atualizar contrato: %w", err)
	}

	return nil
}
