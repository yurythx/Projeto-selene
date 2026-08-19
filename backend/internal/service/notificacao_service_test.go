package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/testutil"
)

// fakeNotificacaoRepository é um dublê em memória — replica a
// deduplicação por (usuario_id, chave_alerta) do índice único da
// migration 000014 (ver o comentário lá), já que é exatamente esse
// comportamento que GerarAlertas depende pra não duplicar/reenviar.
type fakeNotificacaoRepository struct {
	mu       sync.Mutex
	porChave map[string]*models.Notificacao // chave: usuario_id + "|" + chave_alerta
}

func newFakeNotificacaoRepository() *fakeNotificacaoRepository {
	return &fakeNotificacaoRepository{porChave: map[string]*models.Notificacao{}}
}

func (f *fakeNotificacaoRepository) Criar(ctx context.Context, n *models.Notificacao) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	chave := n.UsuarioID.String() + "|" + n.ChaveAlerta
	if _, existe := f.porChave[chave]; existe {
		return false, nil
	}
	copia := *n
	f.porChave[chave] = &copia
	return true, nil
}

func (f *fakeNotificacaoRepository) Listar(ctx context.Context, usuarioID uuid.UUID) ([]models.Notificacao, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []models.Notificacao
	for _, n := range f.porChave {
		if n.UsuarioID == usuarioID {
			out = append(out, *n)
		}
	}
	return out, nil
}

func (f *fakeNotificacaoRepository) ContarNaoLidas(ctx context.Context, usuarioID uuid.UUID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var total int64
	for _, n := range f.porChave {
		if n.UsuarioID == usuarioID && !n.Lida {
			total++
		}
	}
	return total, nil
}

func (f *fakeNotificacaoRepository) MarcarLida(ctx context.Context, usuarioID, notificacaoID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, n := range f.porChave {
		if n.ID == notificacaoID && n.UsuarioID == usuarioID {
			n.Lida = true
			return nil
		}
	}
	return repository.ErrNotificacaoNotFound
}

func (f *fakeNotificacaoRepository) MarcarTodasLidas(ctx context.Context, usuarioID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, n := range f.porChave {
		if n.UsuarioID == usuarioID {
			n.Lida = true
		}
	}
	return nil
}

// fakeAlertaNotifier grava as chamadas de EnviarResumoAlertas — só o que
// os testes de GerarAlertas precisam checar (não implementa
// EnviarPacoteEmpresa de verdade, não é exercitado aqui).
type fakeAlertaNotifier struct {
	mu      sync.Mutex
	resumos []resumoEnviado
}

type resumoEnviado struct {
	destinatario string
	itens        []ItemRadar
}

func (f *fakeAlertaNotifier) EnviarPacoteEmpresa(ctx context.Context, processo *models.ProcessoPagamento, anexos []models.DocumentoAnexo) error {
	return nil
}

func (f *fakeAlertaNotifier) EnviarResumoAlertas(ctx context.Context, destinatario, nomeDestinatario string, itens []ItemRadar) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resumos = append(f.resumos, resumoEnviado{destinatario: destinatario, itens: itens})
	return nil
}

func TestNotificacaoService_GerarAlertas(t *testing.T) {
	ctx := context.Background()
	hoje := time.Now().UTC()

	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", Email: "fiscal@example.org", IsFiscal: true}
	admin := &models.User{ID: uuid.New(), Nome: "Admin Teste", Email: "admin@example.org", IsAdmin: true}

	vigenciaCritica := hoje.AddDate(0, 0, 10)
	contrato := &models.Contrato{
		ID: uuid.New(), NumeroContrato: "1/2026", FiscalID: fiscal.ID, Fiscal: fiscal,
		Ativo: true, DataVigenciaFim: &vigenciaCritica,
	}

	contratoRepo := testutil.NewFakeContratoRepository(contrato)
	processoRepo := testutil.NewFakeProcessoPagamentoRepository()
	docRepo := &testutil.FakeDocumentoAnexoRepository{}
	logRepo := &testutil.FakeKanbanLogRepository{}
	radarService := NewRadarService(contratoRepo, processoRepo, docRepo, logRepo)

	userRepo := testutil.NewFakeUserRepository(fiscal, admin)
	notificacaoRepo := newFakeNotificacaoRepository()
	notifier := &fakeAlertaNotifier{}

	svc := NewNotificacaoService(notificacaoRepo, radarService, contratoRepo, userRepo, notifier)

	t.Run("primeira execução: cria notificação pro fiscal E pro admin, manda e-mail pros dois", func(t *testing.T) {
		if err := svc.GerarAlertas(ctx); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		totalFiscal, _ := notificacaoRepo.ContarNaoLidas(ctx, fiscal.ID)
		if totalFiscal != 1 {
			t.Fatalf("esperava 1 notificação não-lida pro fiscal, veio %d", totalFiscal)
		}
		totalAdmin, _ := notificacaoRepo.ContarNaoLidas(ctx, admin.ID)
		if totalAdmin != 1 {
			t.Fatalf("esperava 1 notificação não-lida pro admin, veio %d", totalAdmin)
		}

		notifier.mu.Lock()
		defer notifier.mu.Unlock()
		if len(notifier.resumos) != 2 {
			t.Fatalf("esperava 2 e-mails de resumo (fiscal + admin), veio %d", len(notifier.resumos))
		}
	})

	t.Run("segunda execução com o mesmo alerta: não duplica notificação nem reenvia e-mail", func(t *testing.T) {
		notifier.mu.Lock()
		notifier.resumos = nil
		notifier.mu.Unlock()

		if err := svc.GerarAlertas(ctx); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		totalFiscal, _ := notificacaoRepo.ContarNaoLidas(ctx, fiscal.ID)
		if totalFiscal != 1 {
			t.Fatalf("esperava CONTINUAR com 1 notificação (deduplicada), veio %d", totalFiscal)
		}

		notifier.mu.Lock()
		defer notifier.mu.Unlock()
		if len(notifier.resumos) != 0 {
			t.Fatalf("esperava 0 e-mails na segunda execução (nada novo), veio %d", len(notifier.resumos))
		}
	})
}

func TestNotificacaoService_GerarAlertas_FiscalTambemAdminRecebeUmaSo(t *testing.T) {
	ctx := context.Background()
	hoje := time.Now().UTC()

	fiscalAdmin := &models.User{ID: uuid.New(), Nome: "Fiscal-Admin", Email: "fa@example.org", IsFiscal: true, IsAdmin: true}

	vigenciaCritica := hoje.AddDate(0, 0, 5)
	contrato := &models.Contrato{
		ID: uuid.New(), NumeroContrato: "2/2026", FiscalID: fiscalAdmin.ID, Fiscal: fiscalAdmin,
		Ativo: true, DataVigenciaFim: &vigenciaCritica,
	}

	contratoRepo := testutil.NewFakeContratoRepository(contrato)
	processoRepo := testutil.NewFakeProcessoPagamentoRepository()
	docRepo := &testutil.FakeDocumentoAnexoRepository{}
	logRepo := &testutil.FakeKanbanLogRepository{}
	radarService := NewRadarService(contratoRepo, processoRepo, docRepo, logRepo)

	userRepo := testutil.NewFakeUserRepository(fiscalAdmin)
	notificacaoRepo := newFakeNotificacaoRepository()
	notifier := &fakeAlertaNotifier{}

	svc := NewNotificacaoService(notificacaoRepo, radarService, contratoRepo, userRepo, notifier)

	if err := svc.GerarAlertas(ctx); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	total, _ := notificacaoRepo.ContarNaoLidas(ctx, fiscalAdmin.ID)
	if total != 1 {
		t.Fatalf("esperava exatamente 1 notificação (fiscal==admin, deduplicado), veio %d", total)
	}

	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	if len(notifier.resumos) != 1 {
		t.Fatalf("esperava exatamente 1 e-mail (não 2), veio %d", len(notifier.resumos))
	}
}

func TestNotificacaoService_MarcarLida(t *testing.T) {
	ctx := context.Background()
	usuarioID := uuid.New()
	outroUsuarioID := uuid.New()

	notificacaoRepo := newFakeNotificacaoRepository()
	n := &models.Notificacao{ID: uuid.New(), UsuarioID: usuarioID, ChaveAlerta: "x"}
	if _, err := notificacaoRepo.Criar(ctx, n); err != nil {
		t.Fatalf("setup: %v", err)
	}

	svc := &NotificacaoService{repo: notificacaoRepo}

	t.Run("marcar notificação de outro usuário devolve not found", func(t *testing.T) {
		err := svc.MarcarLida(ctx, outroUsuarioID, n.ID)
		if err == nil {
			t.Fatal("esperava erro ao tentar marcar notificação de outro usuário")
		}
	})

	t.Run("caminho feliz", func(t *testing.T) {
		if err := svc.MarcarLida(ctx, usuarioID, n.ID); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		total, _ := notificacaoRepo.ContarNaoLidas(ctx, usuarioID)
		if total != 0 {
			t.Fatalf("esperava 0 não-lidas depois de marcar, veio %d", total)
		}
	})
}

func TestChaveAlerta(t *testing.T) {
	contratoID := uuid.New()
	processoID := uuid.New()

	semProcesso := ItemRadar{Tipo: TipoAlertaVigenciaContrato, Nivel: NivelAlertaCritico, ContratoID: contratoID}
	comProcesso := ItemRadar{Tipo: TipoAlertaCertidao, Nivel: NivelAlertaAtencao, ContratoID: contratoID, ProcessoID: &processoID}

	if chaveAlerta(semProcesso) == chaveAlerta(comProcesso) {
		t.Fatal("chaves deveriam ser diferentes pra tipos/processo diferentes")
	}

	// Mesmo alerta, nível diferente = chave diferente (escalar deveria
	// gerar uma notificação nova, ver o comentário na migration 000014).
	critico := ItemRadar{Tipo: TipoAlertaCertidao, Nivel: NivelAlertaCritico, ContratoID: contratoID, ProcessoID: &processoID}
	if chaveAlerta(comProcesso) == chaveAlerta(critico) {
		t.Fatal("chaves deveriam ser diferentes quando o nível escala (ATENCAO -> CRITICO)")
	}

	// Determinística: mesma entrada, mesma chave em chamadas separadas.
	primeiraChamada := chaveAlerta(comProcesso)
	segundaChamada := chaveAlerta(comProcesso)
	if primeiraChamada != segundaChamada {
		t.Fatal("chaveAlerta deveria ser determinística")
	}
}
