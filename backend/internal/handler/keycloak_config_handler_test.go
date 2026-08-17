package handler_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
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

// fakeKeycloakConfigRepositoryHandler é o mesmo dublê mínimo usado em
// internal/service/keycloak_config_service_test.go, reimplementado aqui
// (pacote handler_test não pode importar um tipo não exportado de
// service) — só o suficiente pra exercitar o handler.
type fakeKeycloakConfigRepositoryHandler struct {
	salvo *models.KeycloakConfig
}

func (f *fakeKeycloakConfigRepositoryHandler) Buscar(ctx context.Context) (*models.KeycloakConfig, error) {
	if f.salvo == nil {
		return nil, repository.ErrKeycloakConfigNotFound
	}
	copia := *f.salvo
	return &copia, nil
}

func (f *fakeKeycloakConfigRepositoryHandler) Salvar(ctx context.Context, cfg *models.KeycloakConfig) error {
	cfg.ID = 1
	copia := *cfg
	f.salvo = &copia
	return nil
}

func jwksServerDeTesteHandler(t *testing.T) *httptest.Server {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("falha ao gerar chave de teste: %v", err)
	}
	n := base64.RawURLEncoding.EncodeToString(priv.N.Bytes())
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, _ := json.Marshal(map[string]any{
			"keys": []map[string]string{
				{"kty": "RSA", "use": "sig", "kid": "test-kid", "alg": "RS256", "n": n, "e": "AQAB"},
			},
		})
		_, _ = w.Write(body)
	}))
}

func setupKeycloakConfigRouter(t *testing.T, internalSecret string) (*gin.Engine, *fakeKeycloakConfigRepositoryHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := &fakeKeycloakConfigRepositoryHandler{}
	authState := &middleware.AuthMiddlewareState{}
	svc := service.NewKeycloakConfigService(repo, authState, middleware.AuthConfig{})
	h := handler.NewKeycloakConfigHandler(svc, internalSecret)

	usuario := &models.User{ID: uuid.New(), Nome: "Admin Teste", IsAdmin: true}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUser, usuario)
		c.Next()
	})
	router.GET("/admin/config/keycloak", h.Buscar)
	router.PUT("/admin/config/keycloak", h.Atualizar)
	router.GET("/internal/keycloak-config", h.BuscarInterno)

	return router, repo
}

func TestKeycloakConfigHandler_BuscarAtualizar(t *testing.T) {
	router, _ := setupKeycloakConfigRouter(t, "segredo-interno-de-teste")

	t.Run("GET sem configuração salva devolve origem variaveis_de_ambiente", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/config/keycloak", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}
		var resposta map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resposta); err != nil {
			t.Fatalf("falha ao decodificar resposta: %v", err)
		}
		if resposta["Origem"] != "variaveis_de_ambiente" {
			t.Fatalf("esperava Origem=variaveis_de_ambiente, veio %v", resposta["Origem"])
		}
		if _, temSegredo := resposta["ClientSecret"]; temSegredo {
			t.Fatal("regressão: ClientSecret não deveria aparecer na resposta JSON")
		}
	})

	t.Run("PUT com issuer inalcançável é rejeitado com 400, nada é salvo", func(t *testing.T) {
		corpo, _ := json.Marshal(map[string]string{
			"client_id": "selene-client", "client_secret": "x", "issuer_url": "http://endereco-que-nao-existe.invalid",
		})
		req := httptest.NewRequest(http.MethodPut, "/admin/config/keycloak", bytes.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("PUT válido salva e reflete na leitura seguinte", func(t *testing.T) {
		jwksServer := jwksServerDeTesteHandler(t)
		defer jwksServer.Close()

		corpo, _ := json.Marshal(map[string]string{
			"client_id": "selene-client", "client_secret": "segredo", "issuer_url": jwksServer.URL,
		})
		req := httptest.NewRequest(http.MethodPut, "/admin/config/keycloak", bytes.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}

		getReq := httptest.NewRequest(http.MethodGet, "/admin/config/keycloak", nil)
		getW := httptest.NewRecorder()
		router.ServeHTTP(getW, getReq)

		var resposta map[string]any
		if err := json.Unmarshal(getW.Body.Bytes(), &resposta); err != nil {
			t.Fatalf("falha ao decodificar resposta: %v", err)
		}
		if resposta["ClientID"] != "selene-client" {
			t.Fatalf("esperava ClientID persistido, veio %v", resposta["ClientID"])
		}
		if resposta["Origem"] != "banco_de_dados" {
			t.Fatalf("esperava Origem=banco_de_dados depois de salvar, veio %v", resposta["Origem"])
		}
	})
}

// TestKeycloakConfigHandler_BuscarInterno cobre o modelo de confiança do
// endpoint interno: sem o segredo compartilhado correto, 401 mesmo pra
// quem sabe a URL; com o segredo certo, devolve inclusive o
// ClientSecret em texto puro (só alcançável nesse caminho, nunca pelo
// admin GET normal).
func TestKeycloakConfigHandler_BuscarInterno(t *testing.T) {
	const segredoInterno = "segredo-interno-de-teste"
	router, repo := setupKeycloakConfigRouter(t, segredoInterno)

	t.Run("sem X-Internal-Secret é rejeitado com 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/keycloak-config", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("esperava 401, veio %d", w.Code)
		}
	})

	t.Run("com X-Internal-Secret errado é rejeitado com 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/keycloak-config", nil)
		req.Header.Set("X-Internal-Secret", "segredo-errado")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("esperava 401, veio %d", w.Code)
		}
	})

	t.Run("sem configuração salva devolve 404 (frontend cai pro próprio fallback)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/keycloak-config", nil)
		req.Header.Set("X-Internal-Secret", segredoInterno)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("esperava 404, veio %d", w.Code)
		}
	})

	t.Run("com segredo correto e configuração salva, devolve o client_secret em texto puro", func(t *testing.T) {
		audience := "selene-api"
		repo.salvo = &models.KeycloakConfig{
			ID: 1, ClientID: "selene-client", ClientSecret: "segredo-real", IssuerURL: "https://keycloak.example.org/realms/selene", Audience: &audience,
		}

		req := httptest.NewRequest(http.MethodGet, "/internal/keycloak-config", nil)
		req.Header.Set("X-Internal-Secret", segredoInterno)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}
		var resposta map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resposta); err != nil {
			t.Fatalf("falha ao decodificar resposta: %v", err)
		}
		if resposta["client_secret"] != "segredo-real" {
			t.Fatalf("esperava client_secret real no endpoint interno, veio %q", resposta["client_secret"])
		}
	})
}
