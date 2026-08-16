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

// ErrModeloDocumentoNotFound é retornado quando nenhuma categoria de
// modelo de documento corresponde ao ID/critério informado.
var ErrModeloDocumentoNotFound = errors.New("repository: modelo de documento não encontrado")

// ErrCategoriaModeloDuplicada é retornado ao criar/renomear uma
// categoria pra um nome que já existe (índice único case-insensitive,
// migration 000009) — mesmo mapeamento de ErrNumeroContratoDuplicado em
// contrato_repository.go, via isUniqueViolation.
var ErrCategoriaModeloDuplicada = errors.New("repository: já existe uma categoria de modelo com este nome")

// ErrGatilhoModeloJaAssociado é retornado ao associar um gatilho a uma
// categoria quando outra categoria já o possui (índice único parcial,
// migration 000009) — só uma categoria pode "ser dona" de cada um dos 4
// fluxos de geração, senão a geração real não saberia qual modelo usar.
var ErrGatilhoModeloJaAssociado = errors.New("repository: este gatilho já está associado a outra categoria")

// ModeloDocumentoRepository abstrai o acesso à tabela `modelos_documento`
// (Configurações — Modelos de Documentos).
type ModeloDocumentoRepository interface {
	Create(ctx context.Context, modelo *models.ModeloDocumento) error
	Update(ctx context.Context, modelo *models.ModeloDocumento) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.ModeloDocumento, error)
	// FindByCategoria busca por nome de categoria, case-insensitive —
	// usado pra checar duplicidade antes de tentar Create (mensagem de
	// erro melhor que só a violação de índice único crua).
	FindByCategoria(ctx context.Context, categoria string) (*models.ModeloDocumento, error)
	List(ctx context.Context) ([]models.ModeloDocumento, error)
	// FindAtivoByGatilho é o método consumido pelos 4 fluxos de geração
	// (ver internal/service/modelo_documento_render.go) — retorna
	// ErrModeloDocumentoNotFound quando nenhuma categoria reivindica
	// este gatilho, o caminho NORMAL (fallback fpdf), não um erro real.
	FindAtivoByGatilho(ctx context.Context, gatilho models.TipoGatilhoModelo) (*models.ModeloDocumento, error)
}

type gormModeloDocumentoRepository struct {
	db *gorm.DB
}

// NewModeloDocumentoRepository constrói um ModeloDocumentoRepository
// apoiado em GORM/Postgres.
func NewModeloDocumentoRepository(db *gorm.DB) ModeloDocumentoRepository {
	return &gormModeloDocumentoRepository{db: db}
}

func (r *gormModeloDocumentoRepository) Create(ctx context.Context, modelo *models.ModeloDocumento) error {
	if err := r.db.WithContext(ctx).Create(modelo).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrCategoriaModeloDuplicada
		}
		return fmt.Errorf("repository: criar modelo de documento: %w", err)
	}

	return nil
}

// Update salva as colunas próprias de ModeloDocumento, sem tocar nas
// associações pré-carregadas (VersaoAtiva/Versoes) — mesmo padrão de
// ContratoRepository.Update.
func (r *gormModeloDocumentoRepository) Update(ctx context.Context, modelo *models.ModeloDocumento) error {
	if err := r.db.WithContext(ctx).Omit(clause.Associations).Save(modelo).Error; err != nil {
		if isUniqueViolation(err) {
			// O índice violado pode ser o de categoria OU o de gatilho —
			// distinguir exigiria inspecionar o nome da constraint no
			// erro do driver; como as duas mensagens já orientam o admin
			// a corrigir o mesmo tipo de problema (nome/gatilho em uso),
			// ErrGatilhoModeloJaAssociado cobre o caso mais provável de
			// um Update (criar já passa pelo FindByCategoria antes).
			return ErrGatilhoModeloJaAssociado
		}
		return fmt.Errorf("repository: atualizar modelo de documento: %w", err)
	}

	return nil
}

func (r *gormModeloDocumentoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ModeloDocumento, error) {
	var modelo models.ModeloDocumento

	err := r.db.WithContext(ctx).
		Preload("VersaoAtiva.EnviadoPor").
		Preload("Versoes", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Preload("Versoes.EnviadoPor").
		First(&modelo, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModeloDocumentoNotFound
		}
		return nil, fmt.Errorf("repository: buscar modelo de documento por id: %w", err)
	}

	return &modelo, nil
}

func (r *gormModeloDocumentoRepository) FindByCategoria(ctx context.Context, categoria string) (*models.ModeloDocumento, error) {
	var modelo models.ModeloDocumento

	err := r.db.WithContext(ctx).
		Preload("VersaoAtiva").
		First(&modelo, "lower(categoria) = lower(?)", categoria).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModeloDocumentoNotFound
		}
		return nil, fmt.Errorf("repository: buscar modelo de documento por categoria: %w", err)
	}

	return &modelo, nil
}

func (r *gormModeloDocumentoRepository) List(ctx context.Context) ([]models.ModeloDocumento, error) {
	modelos := []models.ModeloDocumento{}

	err := r.db.WithContext(ctx).
		Preload("VersaoAtiva").
		Order("categoria").
		Find(&modelos).Error
	if err != nil {
		return nil, fmt.Errorf("repository: listar modelos de documento: %w", err)
	}

	return modelos, nil
}

func (r *gormModeloDocumentoRepository) FindAtivoByGatilho(ctx context.Context, gatilho models.TipoGatilhoModelo) (*models.ModeloDocumento, error) {
	var modelo models.ModeloDocumento

	err := r.db.WithContext(ctx).
		Preload("VersaoAtiva").
		First(&modelo, "gatilho = ?", gatilho).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModeloDocumentoNotFound
		}
		return nil, fmt.Errorf("repository: buscar modelo de documento por gatilho: %w", err)
	}

	return &modelo, nil
}
