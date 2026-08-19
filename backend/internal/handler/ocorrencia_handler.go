package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/service"
)

// OcorrenciaHandler expõe as rotas HTTP de registro e acompanhamento de
// Ocorrencia (Fase 2 do plano SGF-Rondonópolis).
type OcorrenciaHandler struct {
	ocorrenciaService *service.OcorrenciaService
}

// NewOcorrenciaHandler constrói um OcorrenciaHandler.
func NewOcorrenciaHandler(ocorrenciaService *service.OcorrenciaService) *OcorrenciaHandler {
	return &OcorrenciaHandler{ocorrenciaService: ocorrenciaService}
}

type registrarOcorrenciaRequest struct {
	Descricao string `json:"descricao" binding:"required"`
}

// Registrar trata POST /api/v1/processos/:id/ocorrencias.
func (h *OcorrenciaHandler) Registrar(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	processoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req registrarOcorrenciaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	ocorrencia, err := h.ocorrenciaService.Registrar(c.Request.Context(), service.RegistrarOcorrenciaInput{
		ProcessoPagamentoID: processoID,
		Descricao:           req.Descricao,
		RegistradoPorID:     usuario.ID,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ocorrencia)
}

// Listar trata GET /api/v1/processos/:id/ocorrencias.
func (h *OcorrenciaHandler) Listar(c *gin.Context) {
	processoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	ocorrencias, err := h.ocorrenciaService.ListarPorProcesso(c.Request.Context(), processoID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, ocorrencias)
}

// Notificar trata POST /api/v1/ocorrencias/:id/notificar.
func (h *OcorrenciaHandler) Notificar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	ocorrencia, err := h.ocorrenciaService.Notificar(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ocorrencia)
}

// IniciarTratamento trata POST /api/v1/ocorrencias/:id/tratar.
func (h *OcorrenciaHandler) IniciarTratamento(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	ocorrencia, err := h.ocorrenciaService.IniciarTratamento(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ocorrencia)
}

// Regularizar trata POST /api/v1/ocorrencias/:id/regularizar.
func (h *OcorrenciaHandler) Regularizar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	ocorrencia, err := h.ocorrenciaService.Regularizar(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ocorrencia)
}
