package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// OcorrenciaService implementa o registro e o ciclo de vida de
// Ocorrencia (IN01 Art.3º-III/Art.5º-IV,IX; IN04 Art.3º-VIII/
// Art.5º-VIII,XVI-XVII) — ver o comentário em models.Ocorrencia sobre a
// simplificação de Camada 2 do fluxo linear
// REGISTRADA→NOTIFICADA→EM_TRATAMENTO→REGULARIZADA.
type OcorrenciaService struct {
	ocorrenciaRepo repository.OcorrenciaRepository
	processoRepo   repository.ProcessoPagamentoRepository
}

// NewOcorrenciaService constrói um OcorrenciaService.
func NewOcorrenciaService(ocorrenciaRepo repository.OcorrenciaRepository, processoRepo repository.ProcessoPagamentoRepository) *OcorrenciaService {
	return &OcorrenciaService{ocorrenciaRepo: ocorrenciaRepo, processoRepo: processoRepo}
}

// RegistrarInput agrupa os dados de abertura de uma nova ocorrência —
// sempre vinculada a um processo de pagamento específico (o ContratoID é
// resolvido a partir dele), mesmo padrão de VistoriaService.Registrar.
type RegistrarOcorrenciaInput struct {
	ProcessoPagamentoID uuid.UUID
	Descricao           string
	RegistradoPorID     uuid.UUID
}

// Registrar abre uma nova ocorrência no estado REGISTRADA.
func (s *OcorrenciaService) Registrar(ctx context.Context, input RegistrarOcorrenciaInput) (*models.Ocorrencia, error) {
	processo, err := s.processoRepo.FindByID(ctx, input.ProcessoPagamentoID)
	if err != nil {
		return nil, err
	}

	ocorrencia := &models.Ocorrencia{
		ContratoID:          processo.ContratoID,
		ProcessoPagamentoID: &input.ProcessoPagamentoID,
		Descricao:           input.Descricao,
		Estado:              models.OcorrenciaRegistrada,
		RegistradoPorID:     input.RegistradoPorID,
	}
	if err := s.ocorrenciaRepo.Create(ctx, ocorrencia); err != nil {
		return nil, fmt.Errorf("service: registrar ocorrência: %w", err)
	}

	return ocorrencia, nil
}

// transicoesPermitidas descreve o fluxo linear de Camada 2 adotado nesta
// fase — ver o comentário em models.EstadoOcorrencia sobre o ramo de
// escalonamento da IN04 Art.5º-XVII não estar modelado ainda.
var transicoesPermitidas = map[models.EstadoOcorrencia]models.EstadoOcorrencia{
	models.OcorrenciaRegistrada:   models.OcorrenciaNotificada,
	models.OcorrenciaNotificada:   models.OcorrenciaEmTratamento,
	models.OcorrenciaEmTratamento: models.OcorrenciaRegularizada,
}

// transicionar é o motor comum de Notificar/IniciarTratamento/Regularizar
// abaixo: busca a ocorrência, confere se destino é o próximo passo válido
// de transicoesPermitidas (rejeita com ErrTransicaoOcorrenciaInvalida se
// não for — ex: pular direto de REGISTRADA pra REGULARIZADA), aplica o
// novo Estado e, se aplicarData não for nil, usa-o para preencher o campo
// de data específico da transição (DataNotificacaoGestor,
// DataRegularizacao) antes de persistir.
func (s *OcorrenciaService) transicionar(ctx context.Context, id uuid.UUID, destino models.EstadoOcorrencia, aplicarData func(*models.Ocorrencia, time.Time)) (*models.Ocorrencia, error) {
	ocorrencia, err := s.ocorrenciaRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if transicoesPermitidas[ocorrencia.Estado] != destino {
		return nil, ErrTransicaoOcorrenciaInvalida
	}

	ocorrencia.Estado = destino
	if aplicarData != nil {
		aplicarData(ocorrencia, time.Now())
	}

	if err := s.ocorrenciaRepo.Update(ctx, ocorrencia); err != nil {
		return nil, fmt.Errorf("service: atualizar estado da ocorrência: %w", err)
	}

	return ocorrencia, nil
}

// Notificar transiciona REGISTRADA → NOTIFICADA (comunicação formal ao
// Gestor — IN04 Art.5º-XVI).
func (s *OcorrenciaService) Notificar(ctx context.Context, id uuid.UUID) (*models.Ocorrencia, error) {
	return s.transicionar(ctx, id, models.OcorrenciaNotificada, func(o *models.Ocorrencia, agora time.Time) {
		o.DataNotificacaoGestor = &agora
	})
}

// IniciarTratamento transiciona NOTIFICADA → EM_TRATAMENTO (Gestor já
// definiu as medidas a tomar — IN04 Art.5º-XVII).
func (s *OcorrenciaService) IniciarTratamento(ctx context.Context, id uuid.UUID) (*models.Ocorrencia, error) {
	return s.transicionar(ctx, id, models.OcorrenciaEmTratamento, nil)
}

// Regularizar transiciona EM_TRATAMENTO → REGULARIZADA — a partir daqui,
// esta ocorrência para de bloquear o avanço de etapa do Kanban (ver
// CanTransition em fiscalizacao_service.go).
func (s *OcorrenciaService) Regularizar(ctx context.Context, id uuid.UUID) (*models.Ocorrencia, error) {
	return s.transicionar(ctx, id, models.OcorrenciaRegularizada, func(o *models.Ocorrencia, agora time.Time) {
		o.DataRegularizacao = &agora
	})
}

// ListarPorProcesso retorna todas as ocorrências de um processo, mais
// recente primeiro.
func (s *OcorrenciaService) ListarPorProcesso(ctx context.Context, processoID uuid.UUID) ([]models.Ocorrencia, error) {
	ocorrencias, err := s.ocorrenciaRepo.ListByProcesso(ctx, processoID)
	if err != nil {
		return nil, fmt.Errorf("service: listar ocorrências do processo: %w", err)
	}
	return ocorrencias, nil
}
