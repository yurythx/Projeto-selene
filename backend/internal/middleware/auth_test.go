package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"projeto-selene/internal/localauth"
	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
)

// spyProvisioner registra qual dos dois métodos de UserProvisioner foi
// chamado, e com quais argumentos — o que este teste quer verificar é o
// ROTEAMENTO do middleware (token do Keycloak vai por um caminho, token
// local vai por outro), não a lógica de provisionamento em si (isso já é
// testado em internal/service).
type spyProvisioner struct {
	keycloakChamadoCom  string
	localChamadoCom     uuid.UUID
	usuarioParaRetornar *models.User
}

func (s *spyProvisioner) FindOrCreateByKeycloakID(ctx context.Context, keycloakID, nome, email string) (*models.User, error) {
	s.keycloakChamadoCom = keycloakID
	return s.usuarioParaRetornar, nil
}

func (s *spyProvisioner) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	s.localChamadoCom = id
	return s.usuarioParaRetornar, nil
}

// jwksJSON monta o corpo (RFC 7517) que keyfunc.NewDefaultCtx espera obter
// da URL do JWKS — o suficiente pra descrever uma chave pública RSA usada
// só nestes testes (não é o formato completo que o Keycloak de verdade
// devolve, só o mínimo que a biblioteca de verificação exige).
func jwksJSON(t *testing.T, pub *rsa.PublicKey, kid string) []byte {
	t.Helper()
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eBytes := big3(pub.E)
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	body := map[string]any{
		"keys": []map[string]string{
			{"kty": "RSA", "use": "sig", "kid": kid, "alg": "RS256", "n": n, "e": e},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("falha ao serializar JWKS de teste: %v", err)
	}
	return raw
}

// big3 codifica um expoente RSA (int, normalmente 65537) nos bytes
// big-endian mínimos exigidos pelo campo "e" de um JWK.
func big3(e int) []byte {
	if e == 0 {
		return []byte{0}
	}
	var b []byte
	for e > 0 {
		b = append([]byte{byte(e & 0xff)}, b...)
		e >>= 8
	}
	return b
}

func assinarTokenKeycloak(t *testing.T, priv *rsa.PrivateKey, kid, issuer, sub string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   sub,
		"name":  "Fiscal Teste",
		"email": "fiscal@teste.local",
		"iss":   issuer,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("falha ao assinar token de teste do Keycloak: %v", err)
	}
	return signed
}

func TestNewAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Chave RSA "do Keycloak" — separada da chave do login local, pra
	// provar que o middleware não confunde uma com a outra.
	keycloakPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("falha ao gerar chave de teste do Keycloak: %v", err)
	}
	const kid = "test-kid-1"
	const issuer = "http://keycloak-de-teste/realms/selene"

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksJSON(t, &keycloakPriv.PublicKey, kid))
	}))
	defer jwksServer.Close()

	localKeys, err := localauth.NewKeyPair()
	if err != nil {
		t.Fatalf("falha ao gerar par de chaves local: %v", err)
	}

	usuarioEsperado := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", Email: "fiscal@teste.local"}

	novoMiddleware := func(provisioner *spyProvisioner) gin.HandlerFunc {
		mw, err := middleware.NewAuthMiddleware(context.Background(), middleware.AuthConfig{
			JWKSURL: jwksServer.URL,
			Issuer:  issuer,
		}, provisioner, localKeys)
		if err != nil {
			t.Fatalf("falha ao construir middleware: %v", err)
		}
		return mw
	}

	rodar := func(mw gin.HandlerFunc, token string) *httptest.ResponseRecorder {
		router := gin.New()
		router.Use(mw)
		router.GET("/protegido", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("token do Keycloak válido resolve via FindOrCreateByKeycloakID", func(t *testing.T) {
		spy := &spyProvisioner{usuarioParaRetornar: usuarioEsperado}
		token := assinarTokenKeycloak(t, keycloakPriv, kid, issuer, "keycloak-sub-123")

		rec := rodar(novoMiddleware(spy), token)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, esperado 200 (corpo: %s)", rec.Code, rec.Body.String())
		}
		if spy.keycloakChamadoCom != "keycloak-sub-123" {
			t.Errorf("FindOrCreateByKeycloakID chamado com %q, esperado \"keycloak-sub-123\"", spy.keycloakChamadoCom)
		}
		if spy.localChamadoCom != uuid.Nil {
			t.Error("FindByID não deveria ter sido chamado pra um token do Keycloak")
		}
	})

	t.Run("token de login local válido resolve via FindByID, não FindOrCreateByKeycloakID", func(t *testing.T) {
		spy := &spyProvisioner{usuarioParaRetornar: usuarioEsperado}
		token, err := localKeys.Emitir(usuarioEsperado.ID, usuarioEsperado.Nome, usuarioEsperado.Email)
		if err != nil {
			t.Fatalf("falha ao emitir token local de teste: %v", err)
		}

		rec := rodar(novoMiddleware(spy), token)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, esperado 200 (corpo: %s)", rec.Code, rec.Body.String())
		}
		if spy.localChamadoCom != usuarioEsperado.ID {
			t.Errorf("FindByID chamado com %v, esperado %v", spy.localChamadoCom, usuarioEsperado.ID)
		}
		if spy.keycloakChamadoCom != "" {
			t.Error("FindOrCreateByKeycloakID não deveria ter sido chamado pra um token local")
		}
	})

	t.Run("token com assinatura de nenhum dos dois emissores é rejeitado", func(t *testing.T) {
		outraChave, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("falha ao gerar chave de teste: %v", err)
		}
		tokenForjado := assinarTokenKeycloak(t, outraChave, kid, issuer, "sub-forjado")

		spy := &spyProvisioner{usuarioParaRetornar: usuarioEsperado}
		rec := rodar(novoMiddleware(spy), tokenForjado)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, esperado 401", rec.Code)
		}
		if spy.keycloakChamadoCom != "" || spy.localChamadoCom != uuid.Nil {
			t.Error("nenhum provisionamento deveria ter sido tentado pra um token com assinatura inválida")
		}
	})

	t.Run("sem cabeçalho Authorization é rejeitado", func(t *testing.T) {
		spy := &spyProvisioner{usuarioParaRetornar: usuarioEsperado}
		rec := rodar(novoMiddleware(spy), "")

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, esperado 401", rec.Code)
		}
	})

	t.Run("token de login local emitido por outra chave (assinatura errada) é rejeitado", func(t *testing.T) {
		outrasChaves, err := localauth.NewKeyPair()
		if err != nil {
			t.Fatalf("falha ao gerar outro par de chaves local: %v", err)
		}
		tokenDeOutraChave, err := outrasChaves.Emitir(usuarioEsperado.ID, usuarioEsperado.Nome, usuarioEsperado.Email)
		if err != nil {
			t.Fatalf("falha ao emitir token de teste: %v", err)
		}

		spy := &spyProvisioner{usuarioParaRetornar: usuarioEsperado}
		rec := rodar(novoMiddleware(spy), tokenDeOutraChave)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, esperado 401", rec.Code)
		}
	})
}
