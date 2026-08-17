package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// EstadoFiscalizacao é um rótulo de LEITURA de Camada 2 (regra do SGF, não
// da norma) computado a partir da etapa Kanban já existente
// (ProcessoPagamento.EtapaAtualID) — nunca persistido em coluna própria.
// Ver a tabela De/Para no plano
// (.claude/plans/projeto-selene-rippling-kite.md) para a fundamentação de
// cada mapeamento etapa→estado.
type EstadoFiscalizacao string

const (
	EstadoAExecutarConferir  EstadoFiscalizacao = "A_EXECUTAR_CONFERIR"
	EstadoEmAnaliseExterna   EstadoFiscalizacao = "EM_ANALISE_EXTERNA"
	EstadoDocumentarAtestar  EstadoFiscalizacao = "DOCUMENTAR_ATESTAR"
	EstadoPendenciaDevolvido EstadoFiscalizacao = "PENDENCIA_DEVOLVIDO"
	EstadoConcluido          EstadoFiscalizacao = "CONCLUIDO"
)

// AcaoOuEspera classifica se a etapa atual do Kanban exige uma ação ativa
// do fiscal ou está em tramitação fora do seu controle direto — pedido
// explícito do briefing SGF ("adaptar os 6 estágios Kanban para Ação do
// Fiscal vs Espera Externa").
type AcaoOuEspera string

const (
	AcaoFiscal    AcaoOuEspera = "ACAO_FISCAL"
	EsperaExterna AcaoOuEspera = "ESPERA_EXTERNA"
)

// mapaEtapaEstado e mapaEtapaAcao replicam, em código, a tabela De/Para do
// plano — chave é ProcessoPagamento.EtapaAtualID (1 a 6, a mesma
// referência seedada em KanbanEtapa, ver kanban_service.go).
var mapaEtapaEstado = map[int]EstadoFiscalizacao{
	1: EstadoAExecutarConferir, // Elaborar OF / Pré-Empenho — IN01 Art.7º-I
	2: EstadoEmAnaliseExterna,  // Tramitar Planejamento/Contabilidade
	3: EstadoAExecutarConferir, // Emitir OS / Envio à Empresa
	4: EstadoDocumentarAtestar, // Execução e Recepção — IN01 Art.5º-IV
	5: EstadoDocumentarAtestar, // Relatório de Pagamento — IN01 Art.9º-12
	6: EstadoEmAnaliseExterna,  // Contabilidade (Liquidação/Pagamento) — IN01 Art.12
}

var mapaEtapaAcao = map[int]AcaoOuEspera{
	1: AcaoFiscal,
	2: EsperaExterna,
	3: AcaoFiscal,
	4: AcaoFiscal,
	5: AcaoFiscal,
	6: EsperaExterna,
}

// Ações do vocabulário fechado exposto em AllowedActions — o frontend usa
// esta lista pra decidir quais botões mostrar no drawer do Kanban (Fase 4
// do plano), substituindo os booleanos hoje hard-coded em
// processo-dialog.tsx (podeAvancar/podeConcluir).
const (
	AcaoAvancarEtapa                 = "AVANCAR_ETAPA"
	AcaoConcluirPagamento            = "CONCLUIR_PAGAMENTO"
	AcaoAnexarDocumento              = "ANEXAR_DOCUMENTO"
	AcaoRegistrarOcorrencia          = "REGISTRAR_OCORRENCIA"
	AcaoRegistrarMovimentacaoEmpenho = "REGISTRAR_MOVIMENTACAO_EMPENHO"
)

// FiscalizacaoService computa, sobre um ProcessoPagamento já carregado, a
// leitura de Camada 2 "FiscalizacaoCompetencia" pedida pelo briefing SGF —
// nunca persiste nada, é uma decoração de leitura em cima do Kanban
// existente (ver o plano, seção De/Para: ProcessoPagamento não é
// renomeado nem substituído).
type FiscalizacaoService struct {
	docRepo        repository.DocumentoAnexoRepository
	ocorrenciaRepo repository.OcorrenciaRepository
}

// NewFiscalizacaoService constrói um FiscalizacaoService.
func NewFiscalizacaoService(docRepo repository.DocumentoAnexoRepository, ocorrenciaRepo repository.OcorrenciaRepository) *FiscalizacaoService {
	return &FiscalizacaoService{docRepo: docRepo, ocorrenciaRepo: ocorrenciaRepo}
}

// ProcessoComFiscalizacao é a resposta enriquecida de GET
// /processos/:id — embute o *models.ProcessoPagamento existente (todos os
// campos hoje já expostos continuam no mesmo lugar, achatados no mesmo
// nível JSON via embedding) e acrescenta os 3 campos computados de Camada
// 2. Nenhum consumidor existente da API quebra: são só chaves adicionais.
type ProcessoComFiscalizacao struct {
	*models.ProcessoPagamento
	EstadoFiscalizacao EstadoFiscalizacao `json:"estado_fiscalizacao"`
	AcaoOuEspera       AcaoOuEspera       `json:"acao_ou_espera"`
	AllowedActions     []string           `json:"allowed_actions"`
	// DocumentosRequeridos é a lista COMPLETA de nomes de TipoDocumento
	// exigidos pra sair da etapa atual (RequisitosEtapa) — diferente do
	// 422 de POST .../avancar, que só devolve os que faltam. O frontend
	// cruza esta lista com os documentos já anexados (GET
	// .../documentos, por TipoDocumento.Nome) pra desenhar o checklist
	// visual (✓ anexado / x pendente) na página do processo — sem isso, a
	// única forma de saber o checklist completo era tentar avançar e ler
	// o erro. Vazio nas etapas sem checklist (2 e 6) ou quando o
	// processo não tem Contrato carregado.
	DocumentosRequeridos []string `json:"documentos_requeridos"`
}

// Decorar computa a leitura enriquecida de um processo já carregado.
// isFiscal indica se o usuário autenticado tem o papel de Fiscal — única
// granularidade de papel disponível em models.User hoje; papéis
// Gestor/Financeiro (Responsabilidade) ficam para uma fase futura de
// modelagem, ver o plano, seção "Lacunas conhecidas".
func (s *FiscalizacaoService) Decorar(ctx context.Context, processo *models.ProcessoPagamento, isFiscal bool) (*ProcessoComFiscalizacao, error) {
	ocorrenciasAbertas, err := s.ocorrenciaRepo.ListAbertasPorProcesso(ctx, processo.ID)
	if err != nil {
		return nil, fmt.Errorf("service: carregar ocorrências abertas do processo: %w", err)
	}

	var checklistPendente []string
	if processo.Status == models.StatusProcessoAtivo && processo.EtapaAtualID < etapaFinalID && processo.Contrato != nil {
		checklistPendente, err = ChecklistPendente(ctx, s.docRepo, processo.ID, processo.EtapaAtualID, processo.Contrato.TipoObjeto, processo.Contrato.ExigeFiscalizacaoTerceirizacao)
		if err != nil {
			return nil, err
		}
	}

	// Lista completa (não só os pendentes) e CUMULATIVA — todas as etapas
	// já percorridas até a atual, não só a atual isolada (ver o
	// comentário de RequisitosAcumulados) — computada independente do
	// Status/etapa final, pra a página do processo poder mostrar o
	// checklist já satisfeito mesmo depois de concluído.
	documentosRequeridos := []string{}
	if processo.Contrato != nil {
		if req := RequisitosAcumulados(processo.EtapaAtualID, processo.Contrato.TipoObjeto, processo.Contrato.ExigeFiscalizacaoTerceirizacao); req != nil {
			documentosRequeridos = req
		}
	}

	_, acoes, _ := CanTransition(processo, ocorrenciasAbertas, checklistPendente, isFiscal)

	return &ProcessoComFiscalizacao{
		ProcessoPagamento:    processo,
		EstadoFiscalizacao:   estadoFiscalizacao(processo, ocorrenciasAbertas),
		AcaoOuEspera:         mapaEtapaAcao[processo.EtapaAtualID],
		AllowedActions:       acoes,
		DocumentosRequeridos: documentosRequeridos,
	}, nil
}

// VerificarAvancoPermitido é a trava de ESCRITA de Camada 2 confirmada
// com o usuário: uma Ocorrencia aberta (Estado != REGULARIZADA) vinculada
// ao processo bloqueia de verdade o avanço de etapa, não só a leitura de
// AllowedActions. Chamada pelo handler ANTES de
// KanbanService.AvancarEtapa (ver processo_handler.go) — de propósito
// fora de kanban_service.go, que continua com sua própria lógica de
// checklist intocada e testada (ver o plano, seção De/Para).
func (s *FiscalizacaoService) VerificarAvancoPermitido(ctx context.Context, processoID uuid.UUID) error {
	ocorrenciasAbertas, err := s.ocorrenciaRepo.ListAbertasPorProcesso(ctx, processoID)
	if err != nil {
		return fmt.Errorf("service: carregar ocorrências abertas do processo: %w", err)
	}
	if len(ocorrenciasAbertas) > 0 {
		return ErrOcorrenciaAbertaBloqueiaAvanco
	}
	return nil
}

// estadoFiscalizacao aplica a regra: Concluído tem prioridade máxima,
// depois ocorrência aberta (bloqueio de Camada 2 confirmado no plano),
// depois o mapeamento direto da etapa.
func estadoFiscalizacao(processo *models.ProcessoPagamento, ocorrenciasAbertas []models.Ocorrencia) EstadoFiscalizacao {
	if processo.Status == models.StatusProcessoConcluido {
		return EstadoConcluido
	}
	if len(ocorrenciasAbertas) > 0 {
		return EstadoPendenciaDevolvido
	}
	return mapaEtapaEstado[processo.EtapaAtualID]
}

// CanTransition decide se o processo pode avançar de etapa AGORA (leitura
// — não substitui a validação que KanbanService.AvancarEtapa faz de novo
// no momento da escrita, ver o plano, seção De/Para) e monta a lista de
// ações permitidas pro usuário atual. Função pura: não acessa
// repositório/banco, só os dados já carregados pelo chamador.
func CanTransition(
	processo *models.ProcessoPagamento,
	ocorrenciasAbertas []models.Ocorrencia,
	checklistPendente []string,
	isFiscal bool,
) (permitido bool, acoesPermitidas []string, motivoBloqueio string) {
	ocorrenciaBloqueando := len(ocorrenciasAbertas) > 0
	ativo := processo.Status == models.StatusProcessoAtivo

	podeAvancar := isFiscal && ativo && processo.EtapaAtualID < etapaFinalID &&
		len(checklistPendente) == 0 && !ocorrenciaBloqueando

	podeConcluir := isFiscal && ativo && processo.EtapaAtualID == etapaFinalID && !ocorrenciaBloqueando

	// acoesPermitidas vai pro campo JSON allowed_actions
	// (ProcessoComFiscalizacao) — vazio, não nil, pelo mesmo motivo
	// documentado em RadarService.Listar: um slice nil vira `null` no
	// JSON, não `[]`. O frontend hoje já se defende disso
	// (`?? []` em processo-dialog.tsx), mas a garantia pertence à origem
	// do dado, não a cada consumidor ter que lembrar de tratar.
	acoes := []string{}
	if podeAvancar {
		acoes = append(acoes, AcaoAvancarEtapa)
	}
	if podeConcluir {
		acoes = append(acoes, AcaoConcluirPagamento)
	}
	if isFiscal && ativo {
		acoes = append(acoes, AcaoAnexarDocumento, AcaoRegistrarOcorrencia, AcaoRegistrarMovimentacaoEmpenho)
	}

	var motivo string
	switch {
	case !ativo:
		motivo = "processo já concluído"
	case !isFiscal:
		motivo = "usuário não tem papel de fiscal"
	case ocorrenciaBloqueando:
		motivo = "existe ocorrência aberta vinculada a este processo — regularize antes de avançar (regra do SGF)"
	case len(checklistPendente) > 0:
		motivo = "checklist da etapa atual está incompleto"
		// processo.EtapaAtualID >= etapaFinalID sem nenhuma das condições
		// acima: não é bloqueio, é ausência de próxima etapa — motivo
		// intencionalmente vazio.
	}

	return podeAvancar, acoes, motivo
}
