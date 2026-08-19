package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/service"
)

// AuthHandler expõe o login tradicional (usuário/senha) — a alternativa
// ao Keycloak (ver internal/service/auth_service.go).
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler constrói um AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type loginRequest struct {
	Email string `json:"email" binding:"required,email"`
	Senha string `json:"senha" binding:"required"`
}

type loginResponse struct {
	AccessToken string      `json:"access_token"`
	Usuario     interface{} `json:"usuario"`
}

// Login trata POST /api/v1/auth/login — rota PÚBLICA (fora do middleware
// de autenticação, ver cmd/api/main.go), sujeita a rate limit por IP
// (proteção contra força bruta).
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	resultado, err := h.authService.Login(c.Request.Context(), req.Email, req.Senha)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		AccessToken: resultado.AccessToken,
		Usuario:     resultado.User,
	})
}

type trocarSenhaRequest struct {
	SenhaAtual string `json:"senha_atual" binding:"required"`
	SenhaNova  string `json:"senha_nova" binding:"required"`
}

// TrocarSenha trata POST /api/v1/auth/trocar-senha — autenticado (qualquer
// usuário logado, local ou Keycloak; contas Keycloak recebem
// ErrContaSemSenhaLocal, já que a senha delas é gerenciada pelo Keycloak).
func (h *AuthHandler) TrocarSenha(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	var req trocarSenhaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	err := h.authService.TrocarSenha(c.Request.Context(), usuario.ID, service.TrocarSenhaInput{
		SenhaAtual: req.SenhaAtual,
		SenhaNova:  req.SenhaNova,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
