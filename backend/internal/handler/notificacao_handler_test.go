package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/handler"
	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/service"
)

// fakeNotificacaoRepositoryHandler é um dublê mínimo — mesmo espírito de
// fakeNotificacaoRepository em internal/service/notificacao_service_test.go
// (reimplementado aqui, pacote handler_test não importa tipos não
// exportados de service).
type fakeNotificacaoRepositoryHandler struct {
	mu    sync.Mutex
	itens []*models.Notificacao
}

func (f *fakeNotificacaoRepositoryHandler) Criar(ctx context.Context, n *models.Notificacao) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.itens = append(f.itens, n)
	return true, nil
}

func (f *fakeNotificacaoRepositoryHandler) Listar(ctx context.Context, usuarioID uuid.UUID) ([]models.Notificacao, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []models.Notificacao
	for _, n := range f.itens {
		if n.UsuarioID == usuarioID {
			out = append(out, *n)
		}
	}
	return out, nil
}

func (f *fakeNotificacaoRepositoryHandler) ContarNaoLidas(ctx context.Context, usuarioID uuid.UUID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var total int64
	for _, n := range f.itens {
		if n.UsuarioID == usuarioID && !n.Lida {
			total++
		}
	}
	return total, nil
}

func (f *fakeNotificacaoRepositoryHandler) MarcarLida(ctx context.Context, usuarioID, notificacaoID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, n := range f.itens {
		if n.ID == notificacaoID && n.UsuarioID == usuarioID {
			n.Lida = true
			return nil
		}
	}
	return repository.ErrNotificacaoNotFound
}

func (f *fakeNotificacaoRepositoryHandler) MarcarTodasLidas(ctx context.Context, usuarioID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, n := range f.itens {
		if n.UsuarioID == usuarioID {
			n.Lida = true
		}
	}
	return nil
}

func setupNotificacaoRouter(t *testing.T, usuario *models.User) (*gin.Engine, *fakeNotificacaoRepositoryHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := &fakeNotificacaoRepositoryHandler{}
	// NotificacaoService.GerarAlertas não é exercitado aqui (precisa de
	// RadarService/ContratoRepository/UserRepository reais) — os testes
	// deste arquivo cobrem só Listar/ContarNaoLidas/MarcarLida/
	// MarcarTodasLidas, que não chamam GerarAlertas.
	svc := service.NewNotificacaoService(repo, nil, nil, nil, nil)
	h := handler.NewNotificacaoHandler(svc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUser, usuario)
		c.Next()
	})
	router.GET("/notificacoes", h.Listar)
	router.GET("/notificacoes/nao-lidas", h.ContarNaoLidas)
	router.POST("/notificacoes/:id/marcar-lida", h.MarcarLida)
	router.POST("/notificacoes/marcar-todas-lidas", h.MarcarTodasLidas)

	return router, repo
}

func TestNotificacaoHandler_Listar(t *testing.T) {
	usuario := &models.User{ID: uuid.New(), Nome: "Fiscal Teste"}
	router, repo := setupNotificacaoRouter(t, usuario)

	_, _ = repo.Criar(context.Background(), &models.Notificacao{
		ID: uuid.New(), UsuarioID: usuario.ID, Tipo: "vigencia_contrato", Nivel: "CRITICO",
		ContratoID: uuid.New(), Mensagem: "teste",
	})

	req := httptest.NewRequest(http.MethodGet, "/notificacoes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
	}
	var resposta []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("falha ao decodificar resposta: %v", err)
	}
	if len(resposta) != 1 {
		t.Fatalf("esperava 1 notificação, veio %d", len(resposta))
	}
	if _, temChaveAlerta := resposta[0]["chave_alerta"]; temChaveAlerta {
		t.Fatal("regressão: chave_alerta não deveria aparecer na resposta JSON")
	}
}

func TestNotificacaoHandler_MarcarLida(t *testing.T) {
	usuario := &models.User{ID: uuid.New(), Nome: "Fiscal Teste"}
	outroUsuario := &models.User{ID: uuid.New(), Nome: "Outro"}
	router, repo := setupNotificacaoRouter(t, usuario)

	n := &models.Notificacao{ID: uuid.New(), UsuarioID: outroUsuario.ID, Tipo: "vigencia_contrato", Nivel: "CRITICO", ContratoID: uuid.New()}
	_, _ = repo.Criar(context.Background(), n)

	t.Run("marcar notificação de outro usuário devolve 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/notificacoes/"+n.ID.String()+"/marcar-lida", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("esperava 404, veio %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("id inválido devolve 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/notificacoes/nao-e-um-uuid/marcar-lida", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestNotificacaoHandler_ContarNaoLidas(t *testing.T) {
	usuario := &models.User{ID: uuid.New(), Nome: "Fiscal Teste"}
	router, repo := setupNotificacaoRouter(t, usuario)

	_, _ = repo.Criar(context.Background(), &models.Notificacao{ID: uuid.New(), UsuarioID: usuario.ID, Tipo: "certidao", Nivel: "ATENCAO", ContratoID: uuid.New()})
	_, _ = repo.Criar(context.Background(), &models.Notificacao{ID: uuid.New(), UsuarioID: usuario.ID, Tipo: "certidao", Nivel: "ATENCAO", ContratoID: uuid.New()})

	req := httptest.NewRequest(http.MethodGet, "/notificacoes/nao-lidas", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
	}
	var resposta map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("falha ao decodificar resposta: %v", err)
	}
	if resposta["total"] != float64(2) {
		t.Fatalf("esperava total=2, veio %v", resposta["total"])
	}
}
