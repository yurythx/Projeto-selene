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
}

// NewUserHandler constrói um UserHandler.
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
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
