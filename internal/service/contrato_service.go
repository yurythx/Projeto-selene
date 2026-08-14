package service

import (
	"context"
	"errors"
	"fmt"

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

	contrato := &models.Contrato{
		NumeroContrato:   input.NumeroContrato,
		PortariaNomeacao: input.PortariaNomeacao,
		DataAssinatura:   dataAssinatura,
		ContratadaNome:   input.ContratadaNome,
		ContratadaCNPJ:   input.ContratadaCNPJ,
		ContratadaEmail:  input.ContratadaEmail,
		FiscalID:         input.FiscalID,
		TipoObjeto:       input.TipoObjeto,
	}

	if err := s.contratoRepo.Create(ctx, contrato); err != nil {
		return nil, fmt.Errorf("service: criar contrato: %w", err)
	}

	return contrato, nil
}

// Listar retorna todos os contratos cadastrados.
func (s *ContratoService) Listar(ctx context.Context) ([]models.Contrato, error) {
	contratos, err := s.contratoRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: listar contratos: %w", err)
	}
	return contratos, nil
}

// Buscar retorna um contrato pelo ID.
func (s *ContratoService) Buscar(ctx context.Context, id uuid.UUID) (*models.Contrato, error) {
	contrato, err := s.contratoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return contrato, nil
}
