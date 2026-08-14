package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// tiposObjetoValidos são os únicos valores aceitos para Contrato.TipoObjeto
// (Seção 4.2 da documentação de domínio).
var tiposObjetoValidos = map[models.TipoObjeto]bool{
	models.TipoObjetoConsumo:    true,
	models.TipoObjetoPermanente: true,
	models.TipoObjetoServico:    true,
}

// NovoContratoInput agrupa os dados necessários para cadastrar um
// contrato — separado do model para não expor campos gerenciados pelo
// sistema (ID, CreatedAt/UpdatedAt) na camada de entrada.
type NovoContratoInput struct {
	NumeroContrato   string
	PortariaNomeacao string
	DataAssinatura   string // formato "2006-01-02"
	ContratadaNome   string
	ContratadaCNPJ   string
	ContratadaEmail  string
	FiscalID         uuid.UUID
	TipoObjeto       models.TipoObjeto
	// DataVigenciaFim (formato "2006-01-02") alimenta o Radar de Alertas
	// (Fase 1 do roadmap) — opcional: string vazia = não informado, o
	// contrato simplesmente não aparece no radar de vigência (ver o
	// comentário no campo do model).
	DataVigenciaFim string
}

// ContratoService contém os casos de uso de cadastro/consulta de
// contratos.
type ContratoService struct {
	contratoRepo repository.ContratoRepository
	userRepo     repository.UserRepository
}

// NewContratoService constrói um ContratoService.
func NewContratoService(contratoRepo repository.ContratoRepository, userRepo repository.UserRepository) *ContratoService {
	return &ContratoService{contratoRepo: contratoRepo, userRepo: userRepo}
}

// Criar valida e persiste um novo contrato. O FiscalID precisa
// corresponder a um usuário com IsFiscal=true — um contrato não pode ser
// atribuído a alguém sem permissão de fiscalização.
func (s *ContratoService) Criar(ctx context.Context, input NovoContratoInput) (*models.Contrato, error) {
	if !tiposObjetoValidos[input.TipoObjeto] {
		return nil, ErrTipoObjetoInvalido
	}

	fiscal, err := s.userRepo.FindByID(ctx, input.FiscalID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrFiscalInvalido
		}
		return nil, fmt.Errorf("service: buscar fiscal do contrato: %w", err)
	}
	if !fiscal.IsFiscal {
		return nil, ErrFiscalInvalido
	}

	dataAssinatura, err := parseData(input.DataAssinatura)
	if err != nil {
		return nil, fmt.Errorf("service: data de assinatura inválida: %w", err)
	}

	var dataVigenciaFim *time.Time
	if input.DataVigenciaFim != "" {
		parsed, err := parseData(input.DataVigenciaFim)
		if err != nil {
			return nil, fmt.Errorf("service: data de vigência inválida: %w", err)
		}
		dataVigenciaFim = &parsed
	}

	contrato := &models.Contrato{
		NumeroContrato:   input.NumeroContrato,
		PortariaNomeacao: input.PortariaNomeacao,
		DataAssinatura:   dataAssinatura,
		ContratadaNome:   input.ContratadaNome,
		ContratadaCNPJ:   input.ContratadaCNPJ,
		ContratadaEmail:  input.ContratadaEmail,
		FiscalID:         input.FiscalID,
		TipoObjeto:       input.TipoObjeto,
		DataVigenciaFim:  dataVigenciaFim,
		// Explícito, não implícito: o zero-value de bool em Go é false, e
		// GORM.Create envia esse false para o banco em vez de deixar o
		// DEFAULT true da coluna se aplicar (não há como o GORM
		// distinguir "não defini" de "defini como false" num bool puro).
		// Sem essa linha, todo contrato novo nasceria encerrado.
		Ativo: true,
	}

	if err := s.contratoRepo.Create(ctx, contrato); err != nil {
		return nil, fmt.Errorf("service: criar contrato: %w", err)
	}

	return contrato, nil
}

// Listar retorna uma página de contratos cadastrados.
func (s *ContratoService) Listar(ctx context.Context, pagina repository.Pagina) (repository.ResultadoPaginado[models.Contrato], error) {
	resultado, err := s.contratoRepo.List(ctx, pagina)
	if err != nil {
		return repository.ResultadoPaginado[models.Contrato]{}, fmt.Errorf("service: listar contratos: %w", err)
	}
	return resultado, nil
}

// Buscar retorna um contrato pelo ID.
func (s *ContratoService) Buscar(ctx context.Context, id uuid.UUID) (*models.Contrato, error) {
	contrato, err := s.contratoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return contrato, nil
}

// AtualizarContratoInput agrupa os campos editáveis de um contrato
// existente. Ponteiros distinguem "não informado" de "definido como
// vazio" — só o que foi explicitamente enviado é alterado. NumeroContrato,
// FiscalID e TipoObjeto não são editáveis aqui: são a identidade e a
// classificação do contrato, mudá-los depois de criado tem implicações
// (checklist já aplicado, cards já abertos) que exigem um fluxo próprio,
// não uma edição genérica.
type AtualizarContratoInput struct {
	PortariaNomeacao *string
	ContratadaNome   *string
	ContratadaCNPJ   *string
	ContratadaEmail  *string
	// DataVigenciaFim (formato "2006-01-02"), se não-nil, atualiza a
	// vigência usada pelo Radar de Alertas. Ponteiro-pra-string (não
	// ponteiro-pra-time.Time): distingue "não enviado" (nil) de "enviado
	// vazio" (string vazia, que limpa a vigência cadastrada).
	DataVigenciaFim *string
}

// Atualizar aplica as alterações de AtualizarContratoInput a um contrato
// existente.
func (s *ContratoService) Atualizar(ctx context.Context, id uuid.UUID, input AtualizarContratoInput) (*models.Contrato, error) {
	contrato, err := s.contratoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.PortariaNomeacao != nil {
		contrato.PortariaNomeacao = *input.PortariaNomeacao
	}
	if input.ContratadaNome != nil {
		contrato.ContratadaNome = *input.ContratadaNome
	}
	if input.ContratadaCNPJ != nil {
		contrato.ContratadaCNPJ = *input.ContratadaCNPJ
	}
	if input.ContratadaEmail != nil {
		contrato.ContratadaEmail = *input.ContratadaEmail
	}
	if input.DataVigenciaFim != nil {
		if *input.DataVigenciaFim == "" {
			contrato.DataVigenciaFim = nil
		} else {
			parsed, err := parseData(*input.DataVigenciaFim)
			if err != nil {
				return nil, fmt.Errorf("service: data de vigência inválida: %w", err)
			}
			contrato.DataVigenciaFim = &parsed
		}
	}

	if err := s.contratoRepo.Update(ctx, contrato); err != nil {
		return nil, fmt.Errorf("service: atualizar contrato: %w", err)
	}

	return contrato, nil
}

// Encerrar marca o contrato como inativo (soft-close): não pode mais
// receber novos processos de pagamento, mas o histórico continua
// consultável. Não há reversão automática — reativar exigiria uma
// decisão administrativa deliberada, não implementada aqui de propósito.
func (s *ContratoService) Encerrar(ctx context.Context, id uuid.UUID) (*models.Contrato, error) {
	contrato, err := s.contratoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	contrato.Ativo = false
	if err := s.contratoRepo.Update(ctx, contrato); err != nil {
		return nil, fmt.Errorf("service: encerrar contrato: %w", err)
	}

	return contrato, nil
}
