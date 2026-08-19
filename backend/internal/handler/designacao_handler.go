package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
)

// DesignacaoHandler expõe as rotas HTTP de designação de
// fiscal/suplente/gestor/fiscal setorial por contrato (Fase 2 do plano
// SGF-Rondonópolis).
type DesignacaoHandler struct {
	designacaoService *service.DesignacaoService
}

// NewDesignacaoHandler constrói um DesignacaoHandler.
func NewDesignacaoHandler(designacaoService *service.DesignacaoService) *DesignacaoHandler {
	return &DesignacaoHandler{designacaoService: designacaoService}
}

type designarRequest struct {
	ServidorID         uuid.UUID              `json:"servidor_id" binding:"required"`
	Papel              models.PapelDesignacao `json:"papel" binding:"required"`
	NumeroPortaria     string                 `json:"numero_portaria"`
	PublicadoDiorondon string                 `json:"publicado_diorondon"`
	// DataDesignacao é opcional, formato "AAAA-MM-DD" (mesma convenção de
	// NovoContratoRequest.DataAssinatura) — quando ausente, o serviço usa
	// a data atual (designação em tempo real, o caso comum). Parsing
	// acontece no service (DesignacaoService.Designar via parseData), não
	// aqui — mesmo padrão de ContratoHandler.
	DataDesignacao string `json:"data_designacao"`
}

// Designar trata POST /api/v1/contratos/:id/designacoes.
func (h *DesignacaoHandler) Designar(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	contratoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req designarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	designacao, err := h.designacaoService.Designar(c.Request.Context(), service.DesignarInput{
		ContratoID:         contratoID,
		ServidorID:         req.ServidorID,
		Papel:              req.Papel,
		NumeroPortaria:     req.NumeroPortaria,
		PublicadoDiorondon: req.PublicadoDiorondon,
		DataDesignacao:     req.DataDesignacao,
		CriadoPorID:        usuario.ID,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, designacao)
}

// Listar trata GET /api/v1/contratos/:id/designacoes.
func (h *DesignacaoHandler) Listar(c *gin.Context) {
	contratoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	designacoes, err := h.designacaoService.ListarPorContrato(c.Request.Context(), contratoID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, designacoes)
}
