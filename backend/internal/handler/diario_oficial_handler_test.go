package handler_test

import (
	"bytes"
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
	"projeto-selene/internal/repository"
	"projeto-selene/internal/service"
)

// fakeDiarioOficialConfigRepositoryHandler é o mesmo dublê mínimo usado
// em internal/service/diario_oficial_service_test.go, reimplementado
// aqui (pacote handler_test não pode importar um tipo não exportado de
// service) — mesmo padrão de fakeKeycloakConfigRepositoryHandler.
type fakeDiarioOficialConfigRepositoryHandler struct {
	salvo *models.DiarioOficialConfig
}

func (f *fakeDiarioOficialConfigRepositoryHandler) Buscar(ctx context.Context) (*models.DiarioOficialConfig, error) {
	if f.salvo == nil {
		return nil, repository.ErrDiarioOficialConfigNotFound
	}
	copia := *f.salvo
	return &copia, nil
}

func (f *fakeDiarioOficialConfigRepositoryHandler) Salvar(ctx context.Context, cfg *models.DiarioOficialConfig) error {
	cfg.ID = 1
	copia := *cfg
	f.salvo = &copia
	return nil
}

func setupDiarioOficialRouter(t *testing.T) (*gin.Engine, *fakeDiarioOficialConfigRepositoryHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := &fakeDiarioOficialConfigRepositoryHandler{}
	svc := service.NewDiarioOficialService(repo)
	h := handler.NewDiarioOficialHandler(svc)

	usuario := &models.User{ID: uuid.New(), Nome: "Admin Teste", IsAdmin: true}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUser, usuario)
		c.Next()
	})
	router.GET("/admin/config/diario-oficial", h.Buscar)
	router.PUT("/admin/config/diario-oficial", h.Atualizar)
	router.POST("/admin/config/diario-oficial/testar", h.TestarConexao)
	router.GET("/admin/diario-oficial/contratos", h.BuscarContratos)

	return router, repo
}

func TestDiarioOficialHandler_BuscarAtualizar(t *testing.T) {
	router, _ := setupDiarioOficialRouter(t)

	t.Run("GET sem configuração salva devolve TemChaveConfigurada=false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/config/diario-oficial", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}
		var resposta map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resposta); err != nil {
			t.Fatalf("falha ao decodificar resposta: %v", err)
		}
		if resposta["TemChaveConfigurada"] != false {
			t.Fatalf("esperava TemChaveConfigurada=false, veio %v", resposta["TemChaveConfigurada"])
		}
		if _, temChave := resposta["APIKey"]; temChave {
			t.Fatal("regressão: APIKey não deveria aparecer na resposta JSON")
		}
	})

	t.Run("PUT com base_url inválida é rejeitado com 400", func(t *testing.T) {
		corpo, _ := json.Marshal(map[string]string{"base_url": "não é uma url", "api_key": "x"})
		req := httptest.NewRequest(http.MethodPut, "/admin/config/diario-oficial", bytes.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("PUT caminho feliz salva e devolve 200", func(t *testing.T) {
		corpo, _ := json.Marshal(map[string]string{
			"base_url": "https://diario.example.gov.br/api", "api_key": "chave-teste",
		})
		req := httptest.NewRequest(http.MethodPut, "/admin/config/diario-oficial", bytes.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDiarioOficialHandler_TestarConexao(t *testing.T) {
	router, _ := setupDiarioOficialRouter(t)

	t.Run("sem configuração salva devolve 412", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/config/diario-oficial/testar", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusPreconditionFailed {
			t.Fatalf("esperava 412, veio %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDiarioOficialHandler_BuscarContratos(t *testing.T) {
	router, _ := setupDiarioOficialRouter(t)

	t.Run("sem configuração salva devolve 412", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/diario-oficial/contratos?nome=Fulano", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusPreconditionFailed {
			t.Fatalf("esperava 412, veio %d: %s", w.Code, w.Body.String())
		}
	})
}
