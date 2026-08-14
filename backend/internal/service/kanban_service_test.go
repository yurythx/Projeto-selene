package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/service"
	"projeto-selene/internal/testutil"
)

// keycloakIDPtr converte um KeycloakID literal pro *string que o model
// User agora exige (nulo pra contas de login local, ver models.User).
func keycloakIDPtr(id string) *string {
	return &id
}

// recordingNotifier é um dublê de Notifier que registra as chamadas
// recebidas de forma thread-safe (AvancarEtapa dispara a notificação numa
// goroutine separada) e expõe um jeito de esperar a chamada acontecer sem
// sleeps arbitrários no teste.
type recordingNotifier struct {
	mu       sync.Mutex
	chamadas []uuid.UUID
	sinal    chan struct{}
}

func newRecordingNotifier() *recordingNotifier {
	return &recordingNotifier{sinal: make(chan struct{}, 8)}
}

func (n *recordingNotifier) EnviarPacoteEmpresa(ctx context.Context, processo *models.ProcessoPagamento, anexos []models.DocumentoAnexo) error {
	n.mu.Lock()
	n.chamadas = append(n.chamadas, processo.ID)
	n.mu.Unlock()
	n.sinal <- struct{}{}
	return nil
}

func (n *recordingNotifier) esperarChamada(t *testing.T) {
	t.Helper()
	select {
	case <-n.sinal:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout esperando a goroutine de notificação assíncrona chamar o Notifier")
	}
}

var _ service.Notifier = (*recordingNotifier)(nil)

// hashUnico gera um hash hexadecimal de 64 caracteres (mesmo formato de
// um SHA-256 real), único a cada chamada — suficiente para os testes, que
// não precisam do conteúdo real do arquivo, só de um valor que respeite a
// coluna char(64) e não colida entre documentos diferentes.
func hashUnico() string {
	soma := sha256.Sum256([]byte(uuid.NewString()))
	return hex.EncodeToString(soma[:])
}

func anexar(t *testing.T, ctx context.Context, docRepo repository.DocumentoAnexoRepository, tipoDocRepo repository.TipoDocumentoRepository, processoID, enviadoPorID uuid.UUID, nomeTipo string) {
	t.Helper()
	tipo, err := tipoDocRepo.FindByNome(ctx, nomeTipo)
	if err != nil {
		t.Fatalf("tipo de documento seed %q não encontrado: %v", nomeTipo, err)
	}
	doc := &models.DocumentoAnexo{
		ProcessoPagamentoID: processoID,
		TipoDocumentoID:     tipo.ID,
		NomeArquivo:         nomeTipo + ".pdf",
		CaminhoStorage:      "/tmp/" + nomeTipo,
		HashArquivo:         hashUnico(), // 64 chars hex, único por chamada — coluna é char(64)
		EnviadoPorID:        enviadoPorID,
	}
	if err := docRepo.Create(ctx, doc); err != nil {
		t.Fatalf("falha ao anexar documento %q: %v", nomeTipo, err)
	}
}

func TestKanbanService_FluxoCompleto(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	contratoRepo := repository.NewContratoRepository(db)
	processoRepo := repository.NewProcessoPagamentoRepository(db)
	docRepo := repository.NewDocumentoAnexoRepository(db)
	tipoDocRepo := repository.NewTipoDocumentoRepository(db)
	logRepo := repository.NewKanbanLogRepository(db)

	notifier := newRecordingNotifier()
	kanban := service.NewKanbanService(db, processoRepo, contratoRepo, docRepo, notifier)

	fiscal := &models.User{KeycloakID: keycloakIDPtr("fiscal-" + uuid.NewString()), Nome: "Fiscal Kanban", Email: "fiscal@teste.local", IsFiscal: true}
	if err := userRepo.Create(ctx, fiscal); err != nil {
		t.Fatalf("falha ao criar fiscal: %v", err)
	}

	contrato := &models.Contrato{
		NumeroContrato:  "KANBAN/" + uuid.NewString()[:8],
		DataAssinatura:  time.Now(),
		ContratadaNome:  "Empresa Kanban",
		ContratadaCNPJ:  "11.111.111/0001-11",
		ContratadaEmail: "empresa@teste.local",
		FiscalID:        fiscal.ID,
		TipoObjeto:      models.TipoObjetoServico,
		Ativo:           true,
	}
	if err := contratoRepo.Create(ctx, contrato); err != nil {
		t.Fatalf("falha ao criar contrato: %v", err)
	}

	processo, err := kanban.CriarProcesso(ctx, contrato.ID, "01/2026", fiscal.ID)
	if err != nil {
		t.Fatalf("falha ao criar processo: %v", err)
	}
	if processo.EtapaAtualID != 1 {
		t.Fatalf("processo novo deveria começar na etapa 1, veio %d", processo.EtapaAtualID)
	}

	// Etapa 1 -> 2: sem documentos, deve falhar com checklist incompleto.
	_, err = kanban.AvancarEtapa(ctx, processo.ID, fiscal.ID)
	var checklistErr *service.ErrChecklistIncompleto
	if !errors.As(err, &checklistErr) {
		t.Fatalf("esperava ErrChecklistIncompleto, veio %v", err)
	}
	if len(checklistErr.Pendentes) != 3 {
		t.Fatalf("esperava 3 documentos pendentes na etapa 1, veio %v", checklistErr.Pendentes)
	}

	for _, nome := range []string{"Ordem de Fornecimento (OF)", "Pré-Empenho", "Ofício de Solicitação"} {
		anexar(t, ctx, docRepo, tipoDocRepo, processo.ID, fiscal.ID, nome)
	}

	processo, err = kanban.AvancarEtapa(ctx, processo.ID, fiscal.ID)
	if err != nil {
		t.Fatalf("etapa 1 -> 2 deveria funcionar com checklist completo: %v", err)
	}
	if processo.EtapaAtualID != 2 {
		t.Fatalf("esperava etapa 2, veio %d", processo.EtapaAtualID)
	}

	// Etapa 2 -> 3: sem checklist, avança direto. Dispara notificação assíncrona.
	processo, err = kanban.AvancarEtapa(ctx, processo.ID, fiscal.ID)
	if err != nil {
		t.Fatalf("etapa 2 -> 3 deveria funcionar sem checklist: %v", err)
	}
	if processo.EtapaAtualID != 3 {
		t.Fatalf("esperava etapa 3, veio %d", processo.EtapaAtualID)
	}
	notifier.esperarChamada(t)

	anexar(t, ctx, docRepo, tipoDocRepo, processo.ID, fiscal.ID, "Nota de Empenho")
	processo, err = kanban.AvancarEtapa(ctx, processo.ID, fiscal.ID)
	if err != nil {
		t.Fatalf("etapa 3 -> 4: %v", err)
	}

	for _, nome := range []string{"Nota Fiscal / Fatura", "Ordem de Recepção"} {
		anexar(t, ctx, docRepo, tipoDocRepo, processo.ID, fiscal.ID, nome)
	}
	processo, err = kanban.AvancarEtapa(ctx, processo.ID, fiscal.ID)
	if err != nil {
		t.Fatalf("etapa 4 -> 5: %v", err)
	}
	if processo.EtapaAtualID != 5 {
		t.Fatalf("esperava etapa 5, veio %d", processo.EtapaAtualID)
	}

	// Etapa 5 é a mais rica: certidões + condicionais de SERVICO.
	for _, nome := range []string{
		"Extrato do Empenho", "Declaração do Simples Nacional", "CND Trabalhista",
		"CND FGTS", "CND Municipal", "CND Estadual", "CND Federal", "CND INSS",
		"Relatório de Pagamento Assinado", "Planilha de Medição de Serviços", "Boleto DAM",
	} {
		anexar(t, ctx, docRepo, tipoDocRepo, processo.ID, fiscal.ID, nome)
	}
	processo, err = kanban.AvancarEtapa(ctx, processo.ID, fiscal.ID)
	if err != nil {
		t.Fatalf("etapa 5 -> 6: %v", err)
	}
	if processo.EtapaAtualID != 6 {
		t.Fatalf("esperava etapa 6 (final), veio %d", processo.EtapaAtualID)
	}

	// Etapa 6 é final: não avança mais.
	_, err = kanban.AvancarEtapa(ctx, processo.ID, fiscal.ID)
	if !errors.Is(err, service.ErrEtapaFinal) {
		t.Fatalf("esperava ErrEtapaFinal, veio %v", err)
	}

	// Concluir só funciona a partir da etapa final.
	processo, err = kanban.ConcluirPagamento(ctx, processo.ID)
	if err != nil {
		t.Fatalf("concluir pagamento: %v", err)
	}
	if processo.Status != models.StatusProcessoConcluido {
		t.Fatalf("esperava status Concluido, veio %q", processo.Status)
	}

	// Auditoria: 1 entrada inicial + 5 transições = 6 logs.
	logs, err := logRepo.ListByProcesso(ctx, processo.ID)
	if err != nil {
		t.Fatalf("falha ao listar logs: %v", err)
	}
	if len(logs) != 6 {
		t.Fatalf("esperava 6 entradas em kanban_logs (1 inicial + 5 transições), veio %d", len(logs))
	}
	if logs[0].EtapaOrigemID != nil {
		t.Fatalf("primeiro log deveria ter EtapaOrigemID nulo (entrada inicial), veio %v", *logs[0].EtapaOrigemID)
	}
}

func TestKanbanService_ConcluirAntesDaEtapaFinalFalha(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	contratoRepo := repository.NewContratoRepository(db)
	processoRepo := repository.NewProcessoPagamentoRepository(db)
	docRepo := repository.NewDocumentoAnexoRepository(db)

	kanban := service.NewKanbanService(db, processoRepo, contratoRepo, docRepo, newRecordingNotifier())

	fiscal := &models.User{KeycloakID: keycloakIDPtr("fiscal-" + uuid.NewString()), Nome: "Fiscal", Email: "f@teste.local", IsFiscal: true}
	if err := userRepo.Create(ctx, fiscal); err != nil {
		t.Fatalf("falha ao criar fiscal: %v", err)
	}
	contrato := &models.Contrato{
		NumeroContrato: "EARLY/" + uuid.NewString()[:8],
		DataAssinatura: time.Now(),
		ContratadaNome: "Empresa",
		ContratadaCNPJ: "22.222.222/0001-22",
		FiscalID:       fiscal.ID,
		TipoObjeto:     models.TipoObjetoConsumo,
		Ativo:          true,
	}
	if err := contratoRepo.Create(ctx, contrato); err != nil {
		t.Fatalf("falha ao criar contrato: %v", err)
	}

	processo, err := kanban.CriarProcesso(ctx, contrato.ID, "02/2026", fiscal.ID)
	if err != nil {
		t.Fatalf("falha ao criar processo: %v", err)
	}

	_, err = kanban.ConcluirPagamento(ctx, processo.ID)
	if !errors.Is(err, service.ErrProcessoNaoElegivelParaConclusao) {
		t.Fatalf("esperava ErrProcessoNaoElegivelParaConclusao, veio %v", err)
	}
}

func TestKanbanService_ContratoEncerradoNaoAceitaNovoProcesso(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	contratoRepo := repository.NewContratoRepository(db)
	processoRepo := repository.NewProcessoPagamentoRepository(db)
	docRepo := repository.NewDocumentoAnexoRepository(db)

	kanban := service.NewKanbanService(db, processoRepo, contratoRepo, docRepo, newRecordingNotifier())

	fiscal := &models.User{KeycloakID: keycloakIDPtr("fiscal-" + uuid.NewString()), Nome: "Fiscal", Email: "f@teste.local", IsFiscal: true}
	if err := userRepo.Create(ctx, fiscal); err != nil {
		t.Fatalf("falha ao criar fiscal: %v", err)
	}
	// Cria ativo (fluxo normal) e só depois encerra via Update — replica o
	// caminho real (ContratoService.Encerrar). Criar já com Ativo=false
	// direto não funcionaria aqui: a tag `gorm:"default:true"` do campo
	// faz o GORM OMITIR a coluna no INSERT quando o valor Go é o
	// zero-value (false), deixando o Postgres aplicar o DEFAULT — mesma
	// categoria de armadilha do bug de associação já documentado em
	// ProcessoPagamentoRepository.Update. Update (não Create) não tem
	// esse problema: envia o valor false explicitamente.
	contrato := &models.Contrato{
		NumeroContrato: "ENCERRADO/" + uuid.NewString()[:8],
		DataAssinatura: time.Now(),
		ContratadaNome: "Empresa",
		ContratadaCNPJ: "33.333.333/0001-33",
		FiscalID:       fiscal.ID,
		TipoObjeto:     models.TipoObjetoConsumo,
		Ativo:          true,
	}
	if err := contratoRepo.Create(ctx, contrato); err != nil {
		t.Fatalf("falha ao criar contrato: %v", err)
	}

	contrato.Ativo = false
	if err := contratoRepo.Update(ctx, contrato); err != nil {
		t.Fatalf("falha ao encerrar contrato: %v", err)
	}

	_, err := kanban.CriarProcesso(ctx, contrato.ID, "03/2026", fiscal.ID)
	if !errors.Is(err, service.ErrContratoEncerrado) {
		t.Fatalf("esperava ErrContratoEncerrado, veio %v", err)
	}
}
