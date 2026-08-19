// Package middleware contém os middlewares HTTP transversais da API
// (autenticação, CORS, logging). Este arquivo trata exclusivamente da
// autenticação via JWT do Keycloak.
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"projeto-selene/internal/localauth"
	"projeto-selene/internal/models"
)

// ContextKeyUser é a chave usada para armazenar o usuário autenticado no
// gin.Context. Handlers não devem usar essa constante diretamente — use
// UserFromContext.
const ContextKeyUser = "auth_user"

// UserProvisioner resolve o usuário local correspondente a um principal
// autenticado. Dois caminhos, um por tipo de token aceito (ver
// NewAuthMiddleware):
//   - FindOrCreateByKeycloakID: token do Keycloak — cria o usuário
//     (sincronização Just-In-Time) caso o KeycloakID ainda não exista.
//   - FindByID: token de login local — a conta já existe (criada por um
//     admin), nunca é provisionada aqui.
//
// Este middleware não acessa o banco de dados diretamente — depende apenas
// desta interface, implementada por internal/service e injetada via
// NewAuthMiddleware. Isso mantém a separação de camadas da Clean
// Architecture mesmo antes de essas camadas existirem.
type UserProvisioner interface {
	FindOrCreateByKeycloakID(ctx context.Context, keycloakID, nome, email string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// Claims é o subconjunto de claims do JWT emitido pelo Keycloak que esta
// API utiliza. RegisteredClaims fornece Subject ("sub"), ExpiresAt, etc.
type Claims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// UserFromContext recupera o usuário autenticado/provisionado (JIT) que o
// middleware de autenticação armazenou no contexto da requisição. Handlers
// devem usar esta função em vez de acessar ContextKeyUser diretamente.
func UserFromContext(c *gin.Context) (*models.User, bool) {
	value, exists := c.Get(ContextKeyUser)
	if !exists {
		return nil, false
	}

	user, ok := value.(*models.User)
	if !ok {
		return nil, false
	}

	return user, true
}

// AuthConfig agrupa os parâmetros necessários para validar tokens do
// Keycloak. Issuer é sempre validado (o claim "iss" precisa bater
// exatamente); Audience só é validado se não estiver vazio — nem todo
// client do Keycloak popula "aud" da mesma forma, então essa checagem é
// opt-in em vez de assumida.
type AuthConfig struct {
	JWKSURL  string
	Issuer   string
	Audience string
}

// keycloakValidator agrupa o JWKS ativo e as opções de parser derivadas
// de um AuthConfig — o que AuthMiddlewareState troca atomicamente a cada
// Reload, sem precisar recriar o middleware nem reiniciar o processo.
type keycloakValidator struct {
	jwks    keyfunc.Keyfunc
	options []jwt.ParserOption
	cancel  context.CancelFunc
}

// AuthMiddlewareState guarda a validação Keycloak ATUAL (JWKS + issuer +
// audience) atrás de um RWMutex, permitindo trocar a configuração em
// runtime — pedido do usuário: "já usamos [Keycloak] hoje mas não temos
// no front, e se eu quiser mudar ou implementar um novo, crie uma
// opção". Ver KeycloakConfigService.Salvar, que chama Reload depois de
// persistir uma configuração nova, e cmd/api/main.go, que popula o
// estado inicial a partir do banco (se um admin já salvou algo) ou das
// variáveis de ambiente (fallback pro primeiro boot).
type AuthMiddlewareState struct {
	mu        sync.RWMutex
	validator *keycloakValidator
}

// jwksProbeTimeout limita quanto tempo Reload espera pela checagem
// síncrona de alcançabilidade do novo JWKS antes de desistir.
const jwksProbeTimeout = 5 * time.Second

// Reload substitui a validação Keycloak ativa por uma nova, construída a
// partir de cfg — falha (sem tocar no estado atual, fail-closed) se o
// JWKS do novo issuer não puder ser buscado, pra um admin não conseguir
// travar a autenticação de todo mundo com uma URL errada.
func (s *AuthMiddlewareState) Reload(ctx context.Context, cfg AuthConfig) error {
	if cfg.Issuer == "" {
		return fmt.Errorf("middleware: Issuer não pode ser vazio")
	}

	// ACHADO: keyfunc.NewDefaultCtx NÃO falha sincronamente quando a URL
	// é inalcançável — ela só loga o erro e deixa a goroutine de refresh
	// em segundo plano tentar de novo depois, então por si só não serve
	// como validação "fail-closed". Uma checagem HTTP síncrona própria,
	// ANTES de trocar o estado ativo, é o que garante que Reload rejeita
	// um JWKS ruim em vez de aceitar silenciosamente uma configuração
	// que nunca vai autenticar ninguém.
	probeCtx, cancelProbe := context.WithTimeout(ctx, jwksProbeTimeout)
	defer cancelProbe()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, cfg.JWKSURL, nil)
	if err != nil {
		return fmt.Errorf("middleware: URL de JWKS inválida (%q): %w", cfg.JWKSURL, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("middleware: JWKS em %q não respondeu: %w", cfg.JWKSURL, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("middleware: JWKS em %q respondeu com status %d (esperado 200)", cfg.JWKSURL, resp.StatusCode)
	}

	// Contexto próprio (não o da requisição HTTP que disparou o reload,
	// que morre quando a resposta é enviada) — precisa sobreviver por
	// todo o tempo em que este JWKS ficar ativo, já que é ele que
	// controla a goroutine de refresh em segundo plano da keyfunc.
	jwksCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	jwks, err := keyfunc.NewDefaultCtx(jwksCtx, []string{cfg.JWKSURL})
	if err != nil {
		cancel()
		return fmt.Errorf("middleware: falha ao inicializar JWKS a partir de %q: %w", cfg.JWKSURL, err)
	}

	options := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(cfg.Issuer),
	}
	if cfg.Audience != "" {
		options = append(options, jwt.WithAudience(cfg.Audience))
	}

	novo := &keycloakValidator{jwks: jwks, options: options, cancel: cancel}

	s.mu.Lock()
	anterior := s.validator
	s.validator = novo
	s.mu.Unlock()

	// Libera a goroutine de refresh do JWKS anterior — sem isso, cada
	// Reload vazaria uma goroutine (e as requisições HTTP mantidas em
	// background pela biblioteca) pro JWKS antigo, que não é mais usado
	// por ninguém a partir daqui.
	if anterior != nil {
		anterior.cancel()
	}

	return nil
}

func (s *AuthMiddlewareState) snapshot() *keycloakValidator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.validator
}

// NewAuthMiddleware constrói um middleware Gin que aceita DOIS tipos de
// token de acesso, nesta ordem de tentativa:
//
//  1. Keycloak — validado contra o JWKS (chaves públicas) do realm da
//     prefeitura: assinatura, expiração, issuer e (opcionalmente)
//     audience. O usuário local é sincronizado via provisioner (JIT).
//  2. Login local (usuário/senha) — validado contra a chave pública do
//     próprio backend (localKeys), emitida por AuthService no login. O
//     usuário JÁ existe (criado por um admin) — resolvido por ID direto,
//     nunca provisionado aqui.
//
// Os dois usam o mesmo formato de claims (Claims/localauth.Claims são
// estruturalmente idênticas) e o mesmo algoritmo (RS256) — o que
// distingue um do outro é o issuer, verificado explicitamente em cada
// tentativa de parse.
//
// O JWKS do Keycloak é buscado uma única vez na construção (via
// AuthMiddlewareState.Reload) e mantido em cache/renovado automaticamente
// em segundo plano pela biblioteca keyfunc — mas, diferente da versão
// anterior desta função, PODE ser trocado depois em runtime (ver
// AuthMiddlewareState), sem recriar o middleware. Se o fetch inicial do
// JWKS falhar, um erro é retornado para permitir fail-fast na
// inicialização do servidor (main.go), em vez de falhar silenciosamente
// na primeira requisição.
func NewAuthMiddleware(ctx context.Context, cfg AuthConfig, provisioner UserProvisioner, localKeys *localauth.KeyPair) (gin.HandlerFunc, *AuthMiddlewareState, error) {
	if provisioner == nil {
		return nil, nil, fmt.Errorf("middleware: UserProvisioner não pode ser nil")
	}
	if localKeys == nil {
		return nil, nil, fmt.Errorf("middleware: localauth.KeyPair não pode ser nil")
	}

	state := &AuthMiddlewareState{}
	if err := state.Reload(ctx, cfg); err != nil {
		return nil, nil, err
	}

	localKeyfunc := func(token *jwt.Token) (interface{}, error) { return localKeys.PublicKey(), nil }
	localParserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(localauth.Issuer),
	}

	return func(c *gin.Context) {
		tokenString, err := extractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		validator := state.snapshot()

		claims := &Claims{}
		token, keycloakErr := jwt.ParseWithClaims(tokenString, claims, validator.jwks.Keyfunc, validator.options...)
		if keycloakErr == nil && token.Valid {
			if claims.Subject == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "claims do token inválidas"})
				return
			}

			user, err := provisioner.FindOrCreateByKeycloakID(c.Request.Context(), claims.Subject, claims.Name, claims.Email)
			if err != nil {
				slog.ErrorContext(c.Request.Context(), "falha ao resolver/provisionar usuário autenticado", "erro", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "falha ao resolver usuário autenticado"})
				return
			}

			c.Set(ContextKeyUser, user)
			c.Next()
			return
		}

		localClaims := &Claims{}
		localToken, localErr := jwt.ParseWithClaims(tokenString, localClaims, localKeyfunc, localParserOptions...)
		if localErr == nil && localToken.Valid && localClaims.Subject != "" {
			userID, parseErr := uuid.Parse(localClaims.Subject)
			if parseErr == nil {
				user, err := provisioner.FindByID(c.Request.Context(), userID)
				if err == nil {
					c.Set(ContextKeyUser, user)
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido ou expirado"})
	}, state, nil
}

// extractBearerToken extrai o token do cabeçalho "Authorization: Bearer <token>".
func extractBearerToken(header string) (string, error) {
	const prefix = "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return "", fmt.Errorf("cabeçalho Authorization ausente ou mal formatado")
	}

	token := strings.TrimPrefix(header, prefix)
	if token == "" {
		return "", fmt.Errorf("cabeçalho Authorization ausente ou mal formatado")
	}

	return token, nil
}
