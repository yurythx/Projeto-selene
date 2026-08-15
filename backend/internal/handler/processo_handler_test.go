package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/handler"
	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
	"projeto-selene/internal/testutil"
)

func setupProcessoRouter(t *testing.T, usuario *models.User) *gin.Engine {
	t.Helper()

	contratoRepo := testutil.NewFakeContratoRepository()
	processoRepo := testutil.NewFakeProcessoPagamentoRepository()
	docRepo := &testutil.FakeDocumentoAnexoRepository{}
	ocorrenciaRepo := testutil.NewFakeOcorrenciaRepository()

	// db=nil é seguro aqui: só exercitamos Listar/Buscar (Listar ->
	// ListByEtapa, Buscar -> FindByID), nenhum dos dois passa por
	// s.db.Transaction — isso só acontece em CriarProcesso/AvancarEtapa,
	// não testados neste arquivo.
	kanbanService := service.NewKanbanService(nil, processoRepo, contratoRepo, docRepo, noopNotifier{})
	fiscalizacaoService := service.NewFiscalizacaoService(docRepo, ocorrenciaRepo)
	h := handler.NewProcessoHandler(kanbanService, fiscalizacaoService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUser, usuario)
		c.Next()
	})
	router.GET("/processos", h.Listar)
	router.GET("/processos/:id", h.Buscar)

	return router
}

// noopNotifier satisfaz service.Notifier sem fazer nada — os testes deste
// arquivo não exercitam nenhum caminho que dispare notificação (só
// AvancarEtapa faz isso), mas o construtor de KanbanService exige um.
type noopNotifier struct{}

func (noopNotifier) EnviarPacoteEmpresa(ctx context.Context, processo *models.ProcessoPagamento, anexos []models.DocumentoAnexo) error {
	return nil
}

func TestProcessoHandler_Listar_ExigeEtapa(t *testing.T) {
	usuario := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", IsFiscal: true}
	router := setupProcessoRouter(t, usuario)

	t.Run("sem query param 'etapa' retorna 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/processos", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("etapa não-numérica retorna 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/processos?etapa=abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("etapa válida retorna 200 com formato paginado", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/processos?etapa=1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}

		var resultado struct {
			Dados []map[string]any `json:"dados"`
			Total int              `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resultado); err != nil {
			t.Fatalf("resposta não é o formato paginado esperado: %v", err)
		}
	})
}

func TestProcessoHandler_Buscar(t *testing.T) {
	usuario := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", IsFiscal: true}
	router := setupProcessoRouter(t, usuario)

	t.Run("id inválido retorna 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/processos/nao-eh-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d", w.Code)
		}
	})

	t.Run("id inexistente retorna 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/processos/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("esperava 404, veio %d: %s", w.Code, w.Body.String())
		}
	})
}
