package service

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// KanbanService implementa o funil rígido de 6 etapas descrito na Seção 5
// da documentação de domínio: cria processos de pagamento (cards), avança
// etapas somente quando o checklist da etapa de origem está satisfeito, e
// grava toda transição em kanban_logs de forma atômica com a mudança de
// etapa (Seção 7.3).
type KanbanService struct {
	db *gorm.DB // usado apenas para abrir transações — leituras usam os repositories injetados

	processoRepo repository.ProcessoPagamentoRepository
	contratoRepo repository.ContratoRepository
	docRepo      repository.DocumentoAnexoRepository
	notifier     Notifier
}

// NewKanbanService constrói um KanbanService.
func NewKanbanService(
	db *gorm.DB,
	processoRepo repository.ProcessoPagamentoRepository,
	contratoRepo repository.ContratoRepository,
	docRepo repository.DocumentoAnexoRepository,
	notifier Notifier,
) *KanbanService {
	return &KanbanService{
		db:           db,
		processoRepo: processoRepo,
		contratoRepo: contratoRepo,
		docRepo:      docRepo,
		notifier:     notifier,
	}
}

// etapaInicialID é a primeira coluna do funil ("1. Elaborar OF /
// Pré-Empenho") — todo processo novo entra por aqui.
const etapaInicialID = 1

// etapaFinalID é a última coluna do funil ("6. Contabilidade") — a partir
// dela o processo não avança mais, apenas é concluído (ConcluirPagamento).
const etapaFinalID = 6

// etapaEnvioEmpresaID é a coluna cuja confirmação dispara o envio
// assíncrono do pacote digital à empresa contratada (Seção 5, Coluna 3).
const etapaEnvioEmpresaID = 3

// CriarProcesso abre um novo card (processo de pagamento) para o contrato
// informado, sempre na primeira etapa do funil, e grava o log de entrada
// inicial no Kanban.
func (s *KanbanService) CriarProcesso(ctx context.Context, contratoID uuid.UUID, mesReferencia string, usuarioID uuid.UUID) (*models.ProcessoPagamento, error) {
	if _, err := s.contratoRepo.FindByID(ctx, contratoID); err != nil {
		return nil, err
	}

	var processo *models.ProcessoPagamento

	err := s.db.Transaction(func(tx *gorm.DB) error {
		processoRepoTx := repository.NewProcessoPagamentoRepository(tx)
		logRepoTx := repository.NewKanbanLogRepository(tx)

		p := &models.ProcessoPagamento{
			ContratoID:    contratoID,
			MesReferencia: mesReferencia,
			EtapaAtualID:  etapaInicialID,
			Status:        models.StatusProcessoAtivo,
		}
		if err := processoRepoTx.Create(ctx, p); err != nil {
			return err
		}

		destino := etapaInicialID
		logEntrada := &models.KanbanLog{
			ProcessoPagamentoID: p.ID,
			UsuarioID:           usuarioID,
			EtapaOrigemID:       nil, // entrada inicial no funil, sem etapa de origem
			EtapaDestinoID:      destino,
		}
		if err := logRepoTx.Create(ctx, logEntrada); err != nil {
			return err
		}

		processo = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("service: criar processo de pagamento: %w", err)
	}

	return processo, nil
}

// AvancarEtapa move o processo para a próxima etapa do funil, desde que o
// checklist obrigatório da etapa atual esteja completo. A atualização da
// etapa e a gravação do log de auditoria acontecem na mesma transação —
// nunca existe uma transição sem o log correspondente.
func (s *KanbanService) AvancarEtapa(ctx context.Context, processoID uuid.UUID, usuarioID uuid.UUID) (*models.ProcessoPagamento, error) {
	processo, err := s.processoRepo.FindByID(ctx, processoID)
	if err != nil {
		return nil, err
	}

	if processo.EtapaAtualID >= etapaFinalID {
		return nil, ErrEtapaFinal
	}

	etapaOrigem := processo.EtapaAtualID
	etapaDestino := etapaOrigem + 1

	pendentes, err := ChecklistPendente(ctx, s.docRepo, processoID, etapaOrigem, processo.Contrato.TipoObjeto)
	if err != nil {
		return nil, err
	}
	if len(pendentes) > 0 {
		return nil, &ErrChecklistIncompleto{Pendentes: pendentes}
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		processoRepoTx := repository.NewProcessoPagamentoRepository(tx)
		logRepoTx := repository.NewKanbanLogRepository(tx)

		processo.EtapaAtualID = etapaDestino
		if err := processoRepoTx.Update(ctx, processo); err != nil {
			return err
		}

		origem := etapaOrigem
		logTransicao := &models.KanbanLog{
			ProcessoPagamentoID: processo.ID,
			UsuarioID:           usuarioID,
			EtapaOrigemID:       &origem,
			EtapaDestinoID:      etapaDestino,
		}
		return logRepoTx.Create(ctx, logTransicao)
	})
	if err != nil {
		return nil, fmt.Errorf("service: avançar etapa do processo: %w", err)
	}

	// Ação assíncrona (Seção 5, Coluna 3): dispara o envio do pacote
	// digital à empresa contratada em uma goroutine separada, sem
	// bloquear a resposta da API. A transição de etapa já foi confirmada
	// e persistida — uma falha de envio aqui não deve nem pode revertê-la,
	// só é registrada em log para acompanhamento manual.
	if etapaDestino == etapaEnvioEmpresaID {
		//nolint:gosec // G118: intencional — a goroutine sobrevive além do
		// request HTTP que a disparou, então precisa do próprio
		// context.Background() em vez do context (cancelável) da
		// requisição de origem; ver notificarEmpresaAssincrono.
		go s.notificarEmpresaAssincrono(processo.ID)
	}

	return processo, nil
}

// notificarEmpresaAssincrono roda em goroutine própria (ver AvancarEtapa)
// e por isso usa seu próprio context.Background() — o context da
// requisição HTTP original já pode ter sido cancelado quando o handler
// retornar a resposta ao fiscal.
func (s *KanbanService) notificarEmpresaAssincrono(processoID uuid.UUID) {
	ctx := context.Background()

	processo, err := s.processoRepo.FindByID(ctx, processoID)
	if err != nil {
		log.Printf("service: notificação assíncrona: falha ao recarregar processo %s: %v", processoID, err)
		return
	}

	anexos, err := s.docRepo.ListByProcesso(ctx, processoID)
	if err != nil {
		log.Printf("service: notificação assíncrona: falha ao carregar anexos do processo %s: %v", processoID, err)
		return
	}

	if err := s.notifier.EnviarPacoteEmpresa(ctx, processo, anexos); err != nil {
		log.Printf("service: notificação assíncrona: falha ao enviar pacote da empresa para o processo %s: %v", processoID, err)
	}
}

// ConcluirPagamento marca o processo como Concluído. Só é permitido a
// partir da última etapa do funil (Contabilidade/Liquidação) — o Selene
// não controla o pagamento bancário em si, apenas registra a confirmação
// (Seção 7.1: sem controle financeiro de saldo).
func (s *KanbanService) ConcluirPagamento(ctx context.Context, processoID uuid.UUID) (*models.ProcessoPagamento, error) {
	processo, err := s.processoRepo.FindByID(ctx, processoID)
	if err != nil {
		return nil, err
	}

	if processo.EtapaAtualID != etapaFinalID {
		return nil, ErrProcessoNaoElegivelParaConclusao
	}

	processo.Status = models.StatusProcessoConcluido
	if err := s.processoRepo.Update(ctx, processo); err != nil {
		return nil, fmt.Errorf("service: concluir pagamento: %w", err)
	}

	return processo, nil
}

// ListarPorEtapa retorna os processos atualmente na etapa informada —
// alimenta uma coluna do quadro Kanban no frontend.
func (s *KanbanService) ListarPorEtapa(ctx context.Context, etapaID int) ([]models.ProcessoPagamento, error) {
	processos, err := s.processoRepo.ListByEtapa(ctx, etapaID)
	if err != nil {
		return nil, fmt.Errorf("service: listar processos por etapa: %w", err)
	}
	return processos, nil
}

// BuscarProcesso retorna um processo pelo ID.
func (s *KanbanService) BuscarProcesso(ctx context.Context, id uuid.UUID) (*models.ProcessoPagamento, error) {
	return s.processoRepo.FindByID(ctx, id)
}
