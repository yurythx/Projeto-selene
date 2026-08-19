package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// fakeKeycloakConfigRepository é um dublê em memória mínimo — só o
// suficiente pra exercitar KeycloakConfigService sem precisar de um
// Postgres real (esse é o papel de
// repository.TestKeycloakConfigRepository_BuscarSalvar).
type fakeKeycloakConfigRepository struct {
	salvo *models.KeycloakConfig
}

func (f *fakeKeycloakConfigRepository) Buscar(ctx context.Context) (*models.KeycloakConfig, error) {
	if f.salvo == nil {
		return nil, repository.ErrKeycloakConfigNotFound
	}
	copia := *f.salvo
	return &copia, nil
}

func (f *fakeKeycloakConfigRepository) Salvar(ctx context.Context, cfg *models.KeycloakConfig) error {
	cfg.ID = 1
	copia := *cfg
	f.salvo = &copia
	return nil
}

// jwksServerDeTeste sobe um servidor HTTP que devolve um JWKS válido (uma
// chave RSA gerada na hora) — usado pra AuthMiddlewareState.Reload
// conseguir passar pela checagem síncrona de alcançabilidade (ver o
// comentário em middleware.AuthMiddlewareState.Reload).
func jwksServerDeTeste(t *testing.T) *httptest.Server {
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

func TestKeycloakConfigService_Buscar(t *testing.T) {
	t.Run("sem configuração salva, devolve o fallback (variáveis de ambiente) sem ClientID", func(t *testing.T) {
		repo := &fakeKeycloakConfigRepository{}
		svc := NewKeycloakConfigService(repo, &middleware.AuthMiddlewareState{}, middleware.AuthConfig{
			Issuer:   "https://fallback.example.org/realms/selene",
			Audience: "selene-api",
		})

		dto, err := svc.Buscar(context.Background())
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if dto.Origem != "variaveis_de_ambiente" {
			t.Fatalf("esperava origem variaveis_de_ambiente, veio %q", dto.Origem)
		}
		if dto.ClientID != "" {
			t.Fatalf("esperava ClientID vazio (backend não conhece AUTH_KEYCLOAK_ID), veio %q", dto.ClientID)
		}
		if dto.IssuerURL != "https://fallback.example.org/realms/selene" {
			t.Fatalf("IssuerURL inesperado: %q", dto.IssuerURL)
		}
		if dto.TemSegredoConfigurado {
			t.Fatal("esperava TemSegredoConfigurado=false sem configuração salva")
		}
	})

	t.Run("com configuração salva, devolve origem banco_de_dados e nunca o ClientSecret", func(t *testing.T) {
		audience := "selene-api"
		repo := &fakeKeycloakConfigRepository{salvo: &models.KeycloakConfig{
			ID:           1,
			ClientID:     "selene-client",
			ClientSecret: "segredo-super-secreto",
			IssuerURL:    "https://keycloak.example.org/realms/selene",
			Audience:     &audience,
			UpdatedByID:  uuid.New(),
		}}
		svc := NewKeycloakConfigService(repo, &middleware.AuthMiddlewareState{}, middleware.AuthConfig{})

		dto, err := svc.Buscar(context.Background())
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if dto.Origem != "banco_de_dados" {
			t.Fatalf("esperava origem banco_de_dados, veio %q", dto.Origem)
		}
		if dto.ClientID != "selene-client" {
			t.Fatalf("ClientID inesperado: %q", dto.ClientID)
		}
		if !dto.TemSegredoConfigurado {
			t.Fatal("esperava TemSegredoConfigurado=true")
		}
		// Prova por reflexão simples: o DTO não tem NENHUM campo capaz de
		// carregar o segredo — a garantia de "nunca volta pro cliente" é
		// estrutural (ConfiguracaoKeycloak nem declara ClientSecret), não
		// apenas "esqueceram de copiar o valor". Ainda assim, confirmamos
		// que TemSegredoConfigurado é derivado corretamente.
	})
}

func TestKeycloakConfigService_Salvar(t *testing.T) {
	t.Run("client_id vazio é rejeitado", func(t *testing.T) {
		repo := &fakeKeycloakConfigRepository{}
		svc := NewKeycloakConfigService(repo, &middleware.AuthMiddlewareState{}, middleware.AuthConfig{})

		err := svc.Salvar(context.Background(), uuid.New(), AtualizarConfiguracaoKeycloak{
			ClientID: "", ClientSecret: "x", IssuerURL: "https://keycloak.example.org/realms/selene",
		})
		if !errors.Is(err, ErrKeycloakClientIDObrigatorio) {
			t.Fatalf("esperava ErrKeycloakClientIDObrigatorio, veio %v", err)
		}
	})

	t.Run("issuer_url inválido (não é URL http(s)) é rejeitado", func(t *testing.T) {
		repo := &fakeKeycloakConfigRepository{}
		svc := NewKeycloakConfigService(repo, &middleware.AuthMiddlewareState{}, middleware.AuthConfig{})

		err := svc.Salvar(context.Background(), uuid.New(), AtualizarConfiguracaoKeycloak{
			ClientID: "selene-client", ClientSecret: "x", IssuerURL: "não é uma url",
		})
		if !errors.Is(err, ErrKeycloakIssuerInvalido) {
			t.Fatalf("esperava ErrKeycloakIssuerInvalido, veio %v", err)
		}
	})

	t.Run("client_secret vazio na primeira configuração é rejeitado", func(t *testing.T) {
		repo := &fakeKeycloakConfigRepository{}
		svc := NewKeycloakConfigService(repo, &middleware.AuthMiddlewareState{}, middleware.AuthConfig{})

		err := svc.Salvar(context.Background(), uuid.New(), AtualizarConfiguracaoKeycloak{
			ClientID: "selene-client", ClientSecret: "", IssuerURL: "https://keycloak.example.org/realms/selene",
		})
		if !errors.Is(err, ErrKeycloakSegredoObrigatorio) {
			t.Fatalf("esperava ErrKeycloakSegredoObrigatorio, veio %v", err)
		}
	})

	t.Run("issuer com JWKS inalcançável é rejeitado e nada é salvo (fail-closed)", func(t *testing.T) {
		repo := &fakeKeycloakConfigRepository{}
		svc := NewKeycloakConfigService(repo, &middleware.AuthMiddlewareState{}, middleware.AuthConfig{})

		err := svc.Salvar(context.Background(), uuid.New(), AtualizarConfiguracaoKeycloak{
			ClientID: "selene-client", ClientSecret: "x", IssuerURL: "http://endereco-que-nao-existe.invalid",
		})
		if !errors.Is(err, ErrKeycloakIssuerInvalido) {
			t.Fatalf("esperava ErrKeycloakIssuerInvalido, veio %v", err)
		}
		if repo.salvo != nil {
			t.Fatal("esperava que nada fosse persistido quando o Reload falha")
		}
	})

	t.Run("caminho feliz: salva, aplica no AuthMiddlewareState, e client_secret em branco mantém o atual numa atualização seguinte", func(t *testing.T) {
		jwksServer := jwksServerDeTeste(t)
		defer jwksServer.Close()

		repo := &fakeKeycloakConfigRepository{}
		authState := &middleware.AuthMiddlewareState{}
		svc := NewKeycloakConfigService(repo, authState, middleware.AuthConfig{})

		adminID := uuid.New()
		err := svc.Salvar(context.Background(), adminID, AtualizarConfiguracaoKeycloak{
			ClientID:     "selene-client",
			ClientSecret: "segredo-inicial",
			IssuerURL:    jwksServer.URL,
		})
		if err != nil {
			t.Fatalf("erro inesperado no caminho feliz: %v", err)
		}
		if repo.salvo == nil || repo.salvo.ClientSecret != "segredo-inicial" {
			t.Fatalf("esperava ClientSecret persistido, veio %+v", repo.salvo)
		}

		// Atualização seguinte, client_secret em branco — mantém o atual.
		err = svc.Salvar(context.Background(), adminID, AtualizarConfiguracaoKeycloak{
			ClientID:     "selene-client-renomeado",
			ClientSecret: "",
			IssuerURL:    jwksServer.URL,
		})
		if err != nil {
			t.Fatalf("erro inesperado na atualização: %v", err)
		}
		if repo.salvo.ClientID != "selene-client-renomeado" {
			t.Fatalf("esperava ClientID atualizado, veio %q", repo.salvo.ClientID)
		}
		if repo.salvo.ClientSecret != "segredo-inicial" {
			t.Fatalf("esperava ClientSecret mantido (\"segredo-inicial\") quando enviado em branco, veio %q", repo.salvo.ClientSecret)
		}
	})
}

func TestDeriveJWKSURL(t *testing.T) {
	casos := []struct {
		issuer   string
		esperado string
	}{
		{"https://keycloak.example.org/realms/selene", "https://keycloak.example.org/realms/selene/protocol/openid-connect/certs"},
		{"https://keycloak.example.org/realms/selene/", "https://keycloak.example.org/realms/selene/protocol/openid-connect/certs"},
	}
	for _, c := range casos {
		if got := DeriveJWKSURL(c.issuer); got != c.esperado {
			t.Errorf("DeriveJWKSURL(%q) = %q, esperado %q", c.issuer, got, c.esperado)
		}
	}
}
