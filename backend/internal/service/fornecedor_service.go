package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// apenasDigitos normaliza um CNPJ pra comparação/roteamento — o CNPJ
// mascarado ("12.345.678/0001-90") contém uma barra, que quebraria o
// roteamento de GET /fornecedores/:cnpj (a barra é separador de segmento
// de URL). Usar só os dígitos como identificador canônico evita esse
// problema na raiz, em vez de depender de codificação de URL (%2F nem
// sempre sobrevive intacto por todos os proxies/roteadores no caminho).
func apenasDigitos(cnpj string) string {
	var b strings.Builder
	for _, r := range cnpj {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// FornecedorService implementa o Módulo 4 do roadmap (Dossiê do
// Fornecedor): consolidação por CNPJ de todos os contratos da mesma
// empresa, histórico de notificações de descumprimento (lido de
// DocumentoEmitido, Fase 2) e um Score de Pontualidade calculado a partir
// do KanbanLog. Majoritariamente leitura/agregação — não há escrita nova
// aqui, só junta dado que já existe.
//
// DECISÃO DE NEGÓCIO (v1, documentada por não haver uma definição de SLA
// no domínio): uma transição de etapa do Kanban conta como "no prazo" se
// aconteceu em até limiarProcessoParadoDias (15 dias, a mesma constante
// que o Radar de Alertas — Fase 1 — usa para marcar um processo como
// "parado há muito tempo"). Reaproveitar esse limiar em vez de inventar
// um novo evita duas definições divergentes de "demorou demais" convivendo
// no mesmo sistema; se um limiar diferente (ou por etapa) for necessário
// no futuro, é uma mudança isolada aqui.
type FornecedorService struct {
	contratoRepo   repository.ContratoRepository
	processoRepo   repository.ProcessoPagamentoRepository
	logRepo        repository.KanbanLogRepository
	docEmitidoRepo repository.DocumentoEmitidoRepository
}

// NewFornecedorService constrói um FornecedorService.
func NewFornecedorService(
	contratoRepo repository.ContratoRepository,
	processoRepo repository.ProcessoPagamentoRepository,
	logRepo repository.KanbanLogRepository,
	docEmitidoRepo repository.DocumentoEmitidoRepository,
) *FornecedorService {
	return &FornecedorService{
		contratoRepo:   contratoRepo,
		processoRepo:   processoRepo,
		logRepo:        logRepo,
		docEmitidoRepo: docEmitidoRepo,
	}
}

// FornecedorResumo é uma linha da listagem consolidada por CNPJ — deixa de
// fora o Score de Pontualidade e as notificações de propósito: calculá-los
// pra TODO fornecedor de uma vez (em vez de só quando o dossiê individual
// é aberto) multiplicaria o custo da lista por N fornecedores sem
// necessidade — ver FornecedorService.Buscar pro dossiê completo.
type FornecedorResumo struct {
	// CNPJ é só dígitos (sem máscara) — identificador canônico, usado
	// pelo frontend pra montar o link GET /fornecedores/{cnpj} (ver o
	// comentário em apenasDigitos pro porquê).
	CNPJ string `json:"cnpj"`
	// CNPJFormatado é o CNPJ como cadastrado no contrato (com
	// pontuação), só pra exibição.
	CNPJFormatado      string `json:"cnpj_formatado"`
	Nome               string `json:"nome"`
	QtdContratos       int    `json:"qtd_contratos"`
	QtdContratosAtivos int    `json:"qtd_contratos_ativos"`
}

// FornecedorDossie é a consolidação completa de um CNPJ.
type FornecedorDossie struct {
	CNPJ          string            `json:"cnpj"`
	CNPJFormatado string            `json:"cnpj_formatado"`
	Nome          string            `json:"nome"`
	Contratos     []models.Contrato `json:"contratos"`
	// Notificacoes são só os DocumentoEmitido do tipo
	// NOTIFICACAO_DESCUMPRIMENTO — o "histórico de penalidades" pedido no
	// roadmap; atestos e minutas de aditivo não são penalidade, ficam de
	// fora.
	Notificacoes []models.DocumentoEmitido `json:"notificacoes"`
	// ScorePontualidade é nil quando não há transições de etapa
	// suficientes pra calcular (nenhum processo nasceu/avançou ainda) —
	// distinto de 0%, que significaria "sempre atrasado". Não inventar um
	// número quando não há dado é mais honesto que uma média vazia.
	ScorePontualidade *float64 `json:"score_pontualidade"`
}

// Listar retorna um resumo por CNPJ de todos os fornecedores com pelo
// menos um contrato cadastrado, ordenado por nome.
func (s *FornecedorService) Listar(ctx context.Context) ([]FornecedorResumo, error) {
	contratos, err := s.contratoRepo.ListTodos(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: listar contratos para consolidar fornecedores: %w", err)
	}

	porCNPJ := make(map[string]*FornecedorResumo)
	for _, c := range contratos {
		chave := apenasDigitos(c.ContratadaCNPJ)
		resumo, ok := porCNPJ[chave]
		if !ok {
			resumo = &FornecedorResumo{CNPJ: chave, CNPJFormatado: c.ContratadaCNPJ, Nome: c.ContratadaNome}
			porCNPJ[chave] = resumo
		}
		resumo.QtdContratos++
		if c.Ativo {
			resumo.QtdContratosAtivos++
		}
	}

	out := make([]FornecedorResumo, 0, len(porCNPJ))
	for _, resumo := range porCNPJ {
		out = append(out, *resumo)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nome < out[j].Nome })

	return out, nil
}

// Buscar monta o dossiê completo de um CNPJ. cnpj pode vir mascarado ou só
// com dígitos — é normalizado antes de comparar (ver apenasDigitos).
func (s *FornecedorService) Buscar(ctx context.Context, cnpj string) (*FornecedorDossie, error) {
	alvo := apenasDigitos(cnpj)

	todos, err := s.contratoRepo.ListTodos(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: listar contratos para o dossiê do fornecedor: %w", err)
	}

	var contratos []models.Contrato
	var contratoIDs []uuid.UUID
	nome, cnpjFormatado := "", ""
	for _, c := range todos {
		if apenasDigitos(c.ContratadaCNPJ) != alvo {
			continue
		}
		contratos = append(contratos, c)
		contratoIDs = append(contratoIDs, c.ID)
		nome = c.ContratadaNome
		cnpjFormatado = c.ContratadaCNPJ
	}
	if len(contratos) == 0 {
		return nil, ErrFornecedorNaoEncontrado
	}
	sort.Slice(contratos, func(i, j int) bool { return contratos[i].NumeroContrato < contratos[j].NumeroContrato })

	notificacoes, err := s.notificacoesDoFornecedor(ctx, contratoIDs)
	if err != nil {
		return nil, err
	}

	score, err := s.scorePontualidade(ctx, contratos)
	if err != nil {
		return nil, err
	}

	return &FornecedorDossie{
		CNPJ:              alvo,
		CNPJFormatado:     cnpjFormatado,
		Nome:              nome,
		Contratos:         contratos,
		Notificacoes:      notificacoes,
		ScorePontualidade: score,
	}, nil
}

func (s *FornecedorService) notificacoesDoFornecedor(ctx context.Context, contratoIDs []uuid.UUID) ([]models.DocumentoEmitido, error) {
	documentos, err := s.docEmitidoRepo.ListByContratoIDs(ctx, contratoIDs)
	if err != nil {
		return nil, fmt.Errorf("service: listar documentos emitidos do fornecedor: %w", err)
	}

	var notificacoes []models.DocumentoEmitido
	for _, d := range documentos {
		if d.Tipo == models.TipoDocumentoEmitidoNotificacao {
			notificacoes = append(notificacoes, d)
		}
	}
	return notificacoes, nil
}

// scorePontualidade calcula o percentual de transições de etapa do Kanban
// que aconteceram dentro do limiar (ver a decisão de negócio documentada
// no comentário do struct FornecedorService), somando os processos de
// TODOS os contratos informados.
func (s *FornecedorService) scorePontualidade(ctx context.Context, contratos []models.Contrato) (*float64, error) {
	var processoIDs []uuid.UUID
	dataAbertura := make(map[uuid.UUID]time.Time)

	for _, c := range contratos {
		processos, err := s.processoRepo.ListByContrato(ctx, c.ID)
		if err != nil {
			return nil, fmt.Errorf("service: listar processos do contrato %s para score de pontualidade: %w", c.NumeroContrato, err)
		}
		for _, p := range processos {
			processoIDs = append(processoIDs, p.ID)
			dataAbertura[p.ID] = p.CreatedAt
		}
	}

	if len(processoIDs) == 0 {
		return nil, nil
	}

	logs, err := s.logRepo.ListByProcessos(ctx, processoIDs)
	if err != nil {
		return nil, fmt.Errorf("service: listar logs do kanban para score de pontualidade: %w", err)
	}

	// Ordena por processo e depois por data — necessário pra "elapsed
	// desde a transição anterior" fazer sentido; não confia na ordem que
	// o repository devolveu (a implementação real do GORM já ordena, mas
	// um fake de teste não precisa garantir isso).
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].ProcessoPagamentoID != logs[j].ProcessoPagamentoID {
			return logs[i].ProcessoPagamentoID.String() < logs[j].ProcessoPagamentoID.String()
		}
		return logs[i].MovidoEm.Before(logs[j].MovidoEm)
	})

	ultimaTransicao := make(map[uuid.UUID]time.Time)
	totalTransicoes, transicoesNoPrazo := 0, 0

	for _, log := range logs {
		referencia, ok := ultimaTransicao[log.ProcessoPagamentoID]
		if !ok {
			referencia = dataAbertura[log.ProcessoPagamentoID]
		}

		dias := diasAte(inicioDoDia(referencia), inicioDoDia(log.MovidoEm))
		totalTransicoes++
		if dias <= limiarProcessoParadoDias {
			transicoesNoPrazo++
		}

		ultimaTransicao[log.ProcessoPagamentoID] = log.MovidoEm
	}

	if totalTransicoes == 0 {
		return nil, nil
	}

	pct := float64(transicoesNoPrazo) / float64(totalTransicoes) * 100
	return &pct, nil
}
