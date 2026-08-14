package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/service"
)

// UserHandler expõe as rotas HTTP de administração de contas de usuário
// (Seção 6, "Administração de Fiscais") — restritas a IsAdmin=true, ver
// middleware.RequireAdmin.
type UserHandler struct {
	userService *service.UserService
	authService *service.AuthService
}

// NewUserHandler constrói um UserHandler.
func NewUserHandler(userService *service.UserService, authService *service.AuthService) *UserHandler {
	return &UserHandler{userService: userService, authService: authService}
}

// Listar trata GET /api/v1/admin/users.
func (h *UserHandler) Listar(c *gin.Context) {
	users, err := h.userService.Listar(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, users)
}

// Buscar trata GET /api/v1/admin/users/:id.
func (h *UserHandler) Buscar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	user, err := h.userService.Buscar(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

type atualizarPermissoesRequest struct {
	IsFiscal  *bool   `json:"is_fiscal"`
	IsAdmin   *bool   `json:"is_admin"`
	Matricula *string `json:"matricula"`
}

// AtualizarPermissoes trata PATCH /api/v1/admin/users/:id — só os campos
// presentes no corpo da requisição são alterados.
func (h *UserHandler) AtualizarPermissoes(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req atualizarPermissoesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	user, err := h.userService.AtualizarPermissoes(c.Request.Context(), id, service.AtualizarPermissoesInput{
		IsFiscal:  req.IsFiscal,
		IsAdmin:   req.IsAdmin,
		Matricula: req.Matricula,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

type criarUsuarioLocalRequest struct {
	Nome            string `json:"nome" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
	SenhaTemporaria string `json:"senha_temporaria" binding:"required"`
	IsFiscal        bool   `json:"is_fiscal"`
	IsAdmin         bool   `json:"is_admin"`
}

// CriarLocal trata POST /api/v1/admin/users/local — cria uma conta de
// login tradicional (usuário/senha), sempre com uma senha temporária que
// o próprio usuário troca no primeiro login (ver models.User.
// MustChangePassword). Não há autocadastro público: só um administrador
// cria contas locais, por aqui.
func (h *UserHandler) CriarLocal(c *gin.Context) {
	var req criarUsuarioLocalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	user, err := h.authService.CriarLocal(c.Request.Context(), service.CriarLocalInput{
		Nome:            req.Nome,
		Email:           req.Email,
		SenhaTemporaria: req.SenhaTemporaria,
		IsFiscal:        req.IsFiscal,
		IsAdmin:         req.IsAdmin,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, user)
}
