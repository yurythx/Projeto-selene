package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"projeto-selene/internal/models"
)

// ErrContratoNotFound é retornado quando nenhum contrato corresponde ao ID
// informado.
var ErrContratoNotFound = errors.New("repository: contrato não encontrado")

// ErrNumeroContratoDuplicado é retornado ao tentar criar um contrato cujo
// NumeroContrato já existe (violação do índice único
// idx_contratos_numero_contrato, migration 000001). Sem este mapeamento, o
// erro do Postgres caía no caso default de respondError e virava um 500
// "erro interno" opaco — um cenário real (reenvio duplo do formulário,
// número digitado igual a um contrato já cadastrado) que merece uma
// mensagem clara em vez de um erro de servidor.
var ErrNumeroContratoDuplicado = errors.New("repository: já existe um contrato com este número")

// pgUniqueViolationCode é o SQLSTATE do Postgres para violação de
// restrição UNIQUE (23505 — unique_violation).
const pgUniqueViolationCode = "23505"

// isUniqueViolation reconhece uma violação de UNIQUE/índice único do
// Postgres através da cadeia de erros retornada pelo driver pgx (usado
// pelo gorm.io/driver/postgres).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode
}

// FiltroContrato define os critérios opcionais de busca/filtro de
// GET /contratos — todo campo vazio/zero significa "sem filtro nesse
// critério" (nunca falha por filtro malformado, mesmo espírito de
// Pagina.Normalizada em pagination.go).
type FiltroContrato struct {
	// Busca casa (ILIKE, sem diferenciar maiúsc./minúsc. nem acento do lado
	// do usuário — Postgres ILIKE já ignora caixa; acento não é
	// normalizado, mesma limitação aceita em qualquer busca por texto livre
	// deste projeto) contra NumeroContrato, ContratadaNome ou
	// ContratadaCNPJ (com ou sem máscara, já que a coluna guarda o valor
	// como foi cadastrado).
	Busca string
	// TipoObjeto, quando um dos 3 valores válidos, restringe a esse tipo.
	// Qualquer outro valor (incluindo "") é tratado como "sem filtro".
	TipoObjeto models.TipoObjeto
	// Situacao: "ativo" filtra Ativo=true, "encerrado" filtra Ativo=false,
	// qualquer outro valor (incluindo "") não filtra.
	Situacao string
}

// aplicarFiltroContrato encadeia as cláusulas WHERE de FiltroContrato numa
// query GORM já iniciada — compartilhado entre a contagem (Count) e a
// busca da página (Find) em List, pra nunca divergir entre "quantos tem"
// e "quais são".
func aplicarFiltroContrato(q *gorm.DB, filtro FiltroContrato) *gorm.DB {
	if busca := strings.TrimSpace(filtro.Busca); busca != "" {
		padrao := "%" + busca + "%"
		q = q.Where("numero_contrato ILIKE ? OR contratada_nome ILIKE ? OR contratada_cnpj ILIKE ?", padrao, padrao, padrao)
	}

	switch filtro.TipoObjeto {
	case models.TipoObjetoConsumo, models.TipoObjetoPermanente, models.TipoObjetoServico:
		q = q.Where("tipo_objeto = ?", filtro.TipoObjeto)
	}

	switch filtro.Situacao {
	case "ativo":
		q = q.Where("ativo = ?", true)
	case "encerrado":
		q = q.Where("ativo = ?", false)
	}

	return q
}

// ContratoRepository abstrai o acesso à tabela `contratos`.
type ContratoRepository interface {
	Create(ctx context.Context, contrato *models.Contrato) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Contrato, error)
	List(ctx context.Context, pagina Pagina, filtro FiltroContrato) (ResultadoPaginado[models.Contrato], error)
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
		if isUniqueViolation(err) {
			return ErrNumeroContratoDuplicado
		}
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

func (r *gormContratoRepository) List(ctx context.Context, pagina Pagina, filtro FiltroContrato) (ResultadoPaginado[models.Contrato], error) {
	pagina = pagina.Normalizada()

	var total int64
	if err := aplicarFiltroContrato(r.db.WithContext(ctx).Model(&models.Contrato{}), filtro).Count(&total).Error; err != nil {
		return ResultadoPaginado[models.Contrato]{}, fmt.Errorf("repository: contar contratos: %w", err)
	}

	contratos := []models.Contrato{}
	err := aplicarFiltroContrato(r.db.WithContext(ctx).Model(&models.Contrato{}), filtro).
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
