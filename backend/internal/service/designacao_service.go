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

// DesignacaoService implementa a Fase 2 do plano SGF-Rondonópolis:
// designação formal de fiscal/suplente/gestor/fiscal setorial por
// contrato (IN01 Art.4º-I/Art.6º; IN04 Art.4º-I/Art.10), com histórico
// auditável em PortariaDesignacao.
type DesignacaoService struct {
	portariaRepo repository.PortariaDesignacaoRepository
	contratoRepo repository.ContratoRepository
}

// NewDesignacaoService constrói um DesignacaoService.
func NewDesignacaoService(portariaRepo repository.PortariaDesignacaoRepository, contratoRepo repository.ContratoRepository) *DesignacaoService {
	return &DesignacaoService{portariaRepo: portariaRepo, contratoRepo: contratoRepo}
}

// DesignarInput agrupa os dados de uma nova designação. DataDesignacao é
// string "AAAA-MM-DD" opcional (mesma convenção de
// ContratoService.CriarContratoInput.DataAssinatura — parseData em
// util.go, chamado aqui no service, não no handler) — vazio usa a data
// atual. A coluna portarias_designacao.data_designacao é `date`, não
// `timestamptz` (migration 000007), por isso data-only e não RFC3339.
type DesignarInput struct {
	ContratoID         uuid.UUID
	ServidorID         uuid.UUID
	Papel              models.PapelDesignacao
	NumeroPortaria     string
	PublicadoDiorondon string
	DataDesignacao     string
	CriadoPorID        uuid.UUID
}

// Designar cria um novo registro de designação e, se já existir uma
// designação ATIVA do mesmo papel para o mesmo contrato, a revoga
// (DataRevogacao = agora) — nunca existem duas designações ativas do
// mesmo papel simultaneamente para o mesmo contrato.
//
// Quando Papel == FISCAL, também sincroniza Contrato.FiscalID com o novo
// servidor — esse campo continua existindo como cache de leitura rápida
// do fiscal atual (ver o plano, seção De/Para); PortariaDesignacao passa a
// ser a fonte de verdade auditável.
func (s *DesignacaoService) Designar(ctx context.Context, input DesignarInput) (*models.PortariaDesignacao, error) {
	if _, err := s.contratoRepo.FindByID(ctx, input.ContratoID); err != nil {
		return nil, err
	}

	ativa, err := s.portariaRepo.FindAtivaPorContratoEPapel(ctx, input.ContratoID, input.Papel)
	if err == nil {
		agora := time.Now()
		ativa.DataRevogacao = &agora
		if err := s.portariaRepo.Update(ctx, ativa); err != nil {
			return nil, fmt.Errorf("service: revogar designação anterior: %w", err)
		}
	} else if !errors.Is(err, repository.ErrPortariaDesignacaoNotFound) {
		return nil, fmt.Errorf("service: verificar designação ativa: %w", err)
	}

	dataDesignacao := time.Now()
	if input.DataDesignacao != "" {
		parsed, err := parseData(input.DataDesignacao)
		if err != nil {
			return nil, fmt.Errorf("service: data de designação inválida: %w: %w", ErrDataInvalida, err)
		}
		dataDesignacao = parsed
	}

	designacao := &models.PortariaDesignacao{
		ContratoID:         input.ContratoID,
		ServidorID:         input.ServidorID,
		Papel:              input.Papel,
		NumeroPortaria:     input.NumeroPortaria,
		PublicadoDiorondon: input.PublicadoDiorondon,
		DataDesignacao:     dataDesignacao,
		CriadoPorID:        input.CriadoPorID,
	}
	if err := s.portariaRepo.Create(ctx, designacao); err != nil {
		return nil, fmt.Errorf("service: criar designação: %w", err)
	}

	if input.Papel == models.PapelFiscal {
		contrato, err := s.contratoRepo.FindByID(ctx, input.ContratoID)
		if err != nil {
			return nil, err
		}
		contrato.FiscalID = input.ServidorID
		if err := s.contratoRepo.Update(ctx, contrato); err != nil {
			return nil, fmt.Errorf("service: sincronizar fiscal do contrato: %w", err)
		}
	}

	return designacao, nil
}

// ListarPorContrato retorna o histórico completo de designações de um
// contrato, mais recente primeiro.
func (s *DesignacaoService) ListarPorContrato(ctx context.Context, contratoID uuid.UUID) ([]models.PortariaDesignacao, error) {
	designacoes, err := s.portariaRepo.ListByContrato(ctx, contratoID)
	if err != nil {
		return nil, fmt.Errorf("service: listar designações do contrato: %w", err)
	}
	return designacoes, nil
}
