package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/repository"
)

// NivelAlerta classifica a urgência de um item do radar.
type NivelAlerta string

const (
	NivelAlertaAtencao NivelAlerta = "ATENCAO"
	NivelAlertaCritico NivelAlerta = "CRITICO"
)

// TipoAlerta identifica a natureza de um ItemRadar.
type TipoAlerta string

const (
	TipoAlertaVigenciaContrato TipoAlerta = "vigencia_contrato"
	TipoAlertaCertidao         TipoAlerta = "certidao"
	TipoAlertaProcessoParado   TipoAlerta = "processo_parado"
)

// Limiares (em dias) usados pra classificar os 3 tipos de alerta — Fase 1
// do roadmap. Centralizados aqui pra serem fáceis de ajustar num único
// lugar, sem caçar número mágico espalhado pelo código.
const (
	// Vigência de contrato: CRITICO com <=30 dias restantes (ou já
	// vencido); ATENCAO entre 31 e 90. Sugerido pelo usuário na proposta
	// original do módulo.
	limiarVigenciaCriticoDias = 30
	limiarVigenciaAtencaoDias = 90

	// Certidão: CRITICO se já vencida; ATENCAO se vence em até 30 dias.
	limiarCertidaoAtencaoDias = 30

	// Processo parado na mesma etapa do Kanban: entra no radar a partir
	// de 15 dias sem mover. Um limiar único pra todas as etapas por
	// enquanto — o roadmap já registra que isso pode precisar variar por
	// etapa no futuro (ex: a etapa de tramitação externa tende a demorar
	// mais por natureza, sem que isso seja um problema real).
	limiarProcessoParadoDias = 15
)

// ItemRadar é um alerta individual, consolidado pro painel /radar e pros
// badges de cor no Kanban.
type ItemRadar struct {
	Tipo           TipoAlerta  `json:"tipo"`
	Nivel          NivelAlerta `json:"nivel"`
	ContratoID     uuid.UUID   `json:"contrato_id"`
	NumeroContrato string      `json:"numero_contrato"`
	ProcessoID     *uuid.UUID  `json:"processo_id,omitempty"`
	Mensagem       string      `json:"mensagem"`
	// DiasRestantes é negativo quando o prazo já passou (vigência
	// vencida, certidão vencida, processo parado há N dias) — positivo
	// ou zero quando ainda dentro do prazo, mas perto o bastante pra
	// entrar no radar.
	DiasRestantes int `json:"dias_restantes"`
}

// RadarService consolida os 3 sinais de alerta da Fase 1 do roadmap:
// vigência de contrato perto do fim, certidão vencida/vencendo anexada a
// um processo em andamento, e processo parado na mesma etapa do Kanban
// há muito tempo.
type RadarService struct {
	contratoRepo repository.ContratoRepository
	processoRepo repository.ProcessoPagamentoRepository
	docRepo      repository.DocumentoAnexoRepository
	logRepo      repository.KanbanLogRepository
}

// NewRadarService constrói um RadarService.
func NewRadarService(
	contratoRepo repository.ContratoRepository,
	processoRepo repository.ProcessoPagamentoRepository,
	docRepo repository.DocumentoAnexoRepository,
	logRepo repository.KanbanLogRepository,
) *RadarService {
	return &RadarService{
		contratoRepo: contratoRepo,
		processoRepo: processoRepo,
		docRepo:      docRepo,
		logRepo:      logRepo,
	}
}

// Listar varre contratos e processos ativos e retorna todo item que
// entra em algum dos 3 sinais de alerta. Sem paginação de propósito — o
// radar precisa ver tudo em risco de uma vez, não uma página por vez (ver
// o comentário em ContratoRepository.ListAtivos sobre o volume esperado).
//
// N+1 consciente: pra cada processo ativo, uma chamada a
// ListByProcesso (documentos) e outra a ListByProcesso (logs). Aceitável
// pro volume esperado de processos em andamento numa prefeitura — não é
// um endpoint de alta frequência. Se isso deixar de ser verdade,
// revisitar com queries em lote.
func (s *RadarService) Listar(ctx context.Context) ([]ItemRadar, error) {
	hoje := inicioDoDia(time.Now())

	contratos, err := s.contratoRepo.ListAtivos(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: listar contratos ativos para o radar: %w", err)
	}

	var itens []ItemRadar

	contratoAtivoPorID := make(map[uuid.UUID]bool, len(contratos))
	for _, c := range contratos {
		contratoAtivoPorID[c.ID] = true

		if c.DataVigenciaFim == nil {
			continue
		}
		dias := diasAte(hoje, *c.DataVigenciaFim)
		nivel, alerta := nivelVigencia(dias)
		if !alerta {
			continue
		}
		itens = append(itens, ItemRadar{
			Tipo:           TipoAlertaVigenciaContrato,
			Nivel:          nivel,
			ContratoID:     c.ID,
			NumeroContrato: c.NumeroContrato,
			Mensagem:       mensagemVigencia(dias),
			DiasRestantes:  dias,
		})
	}

	processos, err := s.processoRepo.ListAtivosComContrato(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: listar processos ativos para o radar: %w", err)
	}

	for _, p := range processos {
		// Só processos de contratos ainda ativos — ListAtivosComContrato
		// não filtra isso (evita um join), filtramos em memória aqui.
		if p.Contrato == nil || !contratoAtivoPorID[p.ContratoID] {
			continue
		}
		processoID := p.ID

		documentos, err := s.docRepo.ListByProcesso(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("service: listar documentos do processo %s para o radar: %w", p.ID, err)
		}
		for _, doc := range documentos {
			if doc.DataValidade == nil {
				continue
			}
			dias := diasAte(hoje, *doc.DataValidade)
			nivel, alerta := nivelCertidao(dias)
			if !alerta {
				continue
			}
			nomeTipo := "Documento"
			if doc.TipoDocumento != nil {
				nomeTipo = doc.TipoDocumento.Nome
			}
			itens = append(itens, ItemRadar{
				Tipo:           TipoAlertaCertidao,
				Nivel:          nivel,
				ContratoID:     p.ContratoID,
				NumeroContrato: p.Contrato.NumeroContrato,
				ProcessoID:     &processoID,
				Mensagem:       fmt.Sprintf("%s %s", nomeTipo, mensagemCertidao(dias)),
				DiasRestantes:  dias,
			})
		}

		logs, err := s.logRepo.ListByProcesso(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("service: listar logs do processo %s para o radar: %w", p.ID, err)
		}
		if len(logs) == 0 {
			continue
		}
		// ListByProcesso ordena por movido_em ascendente — o último
		// elemento é a transição mais recente, ou seja, quando o
		// processo entrou na etapa atual.
		ultimaMovimentacao := logs[len(logs)-1].MovidoEm
		diasParado := diasAte(inicioDoDia(ultimaMovimentacao), hoje)
		if diasParado < limiarProcessoParadoDias {
			continue
		}
		etapaNome := "etapa atual"
		if p.EtapaAtual != nil {
			etapaNome = p.EtapaAtual.Nome
		}
		itens = append(itens, ItemRadar{
			Tipo:           TipoAlertaProcessoParado,
			Nivel:          NivelAlertaAtencao,
			ContratoID:     p.ContratoID,
			NumeroContrato: p.Contrato.NumeroContrato,
			ProcessoID:     &processoID,
			Mensagem:       fmt.Sprintf("Parado em %q há %d dias", etapaNome, diasParado),
			DiasRestantes:  -diasParado,
		})
	}

	return itens, nil
}

func inicioDoDia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// diasAte retorna quantos dias faltam de "de" até "para" — negativo se
// "para" já passou.
func diasAte(de, para time.Time) int {
	return int(inicioDoDia(para).Sub(inicioDoDia(de)).Hours() / 24)
}

func nivelVigencia(diasRestantes int) (NivelAlerta, bool) {
	switch {
	case diasRestantes <= limiarVigenciaCriticoDias:
		return NivelAlertaCritico, true
	case diasRestantes <= limiarVigenciaAtencaoDias:
		return NivelAlertaAtencao, true
	default:
		return "", false
	}
}

func nivelCertidao(diasRestantes int) (NivelAlerta, bool) {
	switch {
	case diasRestantes < 0:
		return NivelAlertaCritico, true
	case diasRestantes <= limiarCertidaoAtencaoDias:
		return NivelAlertaAtencao, true
	default:
		return "", false
	}
}

func mensagemVigencia(diasRestantes int) string {
	if diasRestantes < 0 {
		return fmt.Sprintf("Vigência do contrato venceu há %d dias", -diasRestantes)
	}
	return fmt.Sprintf("Faltam %d dias para o fim da vigência do contrato", diasRestantes)
}

func mensagemCertidao(diasRestantes int) string {
	if diasRestantes < 0 {
		return fmt.Sprintf("vencida há %d dias", -diasRestantes)
	}
	return fmt.Sprintf("vence em %d dias", diasRestantes)
}
