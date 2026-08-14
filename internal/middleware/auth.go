// Package middleware contém os middlewares HTTP transversais da API
// (autenticação, CORS, logging). Este arquivo trata exclusivamente da
// autenticação via JWT do Keycloak.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"projeto-selene/internal/models"
)

// ContextKeyUser é a chave usada para armazenar o usuário autenticado no
// gin.Context. Handlers não devem usar essa constante diretamente — use
// UserFromContext.
const ContextKeyUser = "auth_user"

// UserProvisioner resolve o usuário local correspondente a um principal
// autenticado no Keycloak, criando-o (sincronização Just-In-Time) caso o
// KeycloakID ainda não exista na tabela local `users`.
//
// Este middleware não acessa o banco de dados diretamente — depende apenas
// desta interface, que será implementada por uma camada de repository/
// service em um passo futuro e injetada via NewAuthMiddleware. Isso mantém
// a separação de camadas da Clean Architecture mesmo antes de essas
// camadas existirem.
type UserProvisioner interface {
	FindOrCreateByKeycloakID(ctx context.Context, keycloakID, nome, email string) (*models.User, error)
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

// NewAuthMiddleware constrói um middleware Gin que valida tokens JWT
// emitidos pelo Keycloak contra o JWKS (chaves públicas) do realm da
// prefeitura, e sincroniza o usuário local via provisioner (JIT).
//
// O JWKS é buscado uma única vez na construção e mantido em cache/renovado
// automaticamente em segundo plano pela biblioteca keyfunc — por isso esta
// função deve ser chamada UMA VEZ na inicialização do router, nunca por
// requisição. Se o fetch inicial do JWKS falhar, um erro é retornado para
// permitir fail-fast na inicialização do servidor (main.go), em vez de
// falhar silenciosamente na primeira requisição.
func NewAuthMiddleware(ctx context.Context, jwksURL string, provisioner UserProvisioner) (gin.HandlerFunc, error) {
	if provisioner == nil {
		return nil, fmt.Errorf("middleware: UserProvisioner não pode ser nil")
	}

	jwks, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("middleware: falha ao inicializar JWKS a partir de %q: %w", jwksURL, err)
	}

	return func(c *gin.Context) {
		tokenString, err := extractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, jwks.Keyfunc, jwt.WithValidMethods([]string{"RS256"}))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido ou expirado"})
			return
		}

		if !token.Valid || claims.Subject == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "claims do token inválidas"})
			return
		}

		user, err := provisioner.FindOrCreateByKeycloakID(c.Request.Context(), claims.Subject, claims.Name, claims.Email)
		if err != nil {
			// TODO: registrar err em um logger estruturado assim que a
			// camada de logging (internal/middleware/logger.go ou
			// equivalente) existir. Não expor detalhes internos ao cliente.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "falha ao resolver usuário autenticado"})
			return
		}

		c.Set(ContextKeyUser, user)
		c.Next()
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
