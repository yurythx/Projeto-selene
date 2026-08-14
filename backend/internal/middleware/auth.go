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
// O JWKS do Keycloak é buscado uma única vez na construção e mantido em
// cache/renovado automaticamente em segundo plano pela biblioteca keyfunc
// — por isso esta função deve ser chamada UMA VEZ na inicialização do
// router, nunca por requisição. Se o fetch inicial do JWKS falhar, um erro
// é retornado para permitir fail-fast na inicialização do servidor
// (main.go), em vez de falhar silenciosamente na primeira requisição.
func NewAuthMiddleware(ctx context.Context, cfg AuthConfig, provisioner UserProvisioner, localKeys *localauth.KeyPair) (gin.HandlerFunc, error) {
	if provisioner == nil {
		return nil, fmt.Errorf("middleware: UserProvisioner não pode ser nil")
	}
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("middleware: Issuer não pode ser vazio")
	}
	if localKeys == nil {
		return nil, fmt.Errorf("middleware: localauth.KeyPair não pode ser nil")
	}

	jwks, err := keyfunc.NewDefaultCtx(ctx, []string{cfg.JWKSURL})
	if err != nil {
		return nil, fmt.Errorf("middleware: falha ao inicializar JWKS a partir de %q: %w", cfg.JWKSURL, err)
	}

	keycloakParserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(cfg.Issuer),
	}
	if cfg.Audience != "" {
		keycloakParserOptions = append(keycloakParserOptions, jwt.WithAudience(cfg.Audience))
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

		claims := &Claims{}
		token, keycloakErr := jwt.ParseWithClaims(tokenString, claims, jwks.Keyfunc, keycloakParserOptions...)
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
	}, nil
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
