package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
)

// ContratoHandler expõe as rotas HTTP de cadastro/consulta de contratos.
type ContratoHandler struct {
	contratoService *service.ContratoService
}

// NewContratoHandler constrói um ContratoHandler.
func NewContratoHandler(contratoService *service.ContratoService) *ContratoHandler {
	return &ContratoHandler{contratoService: contratoService}
}

type criarContratoRequest struct {
	NumeroContrato   string            `json:"numero_contrato" binding:"required"`
	PortariaNomeacao string            `json:"portaria_nomeacao"`
	DataAssinatura   string            `json:"data_assinatura" binding:"required"` // "AAAA-MM-DD"
	ContratadaNome   string            `json:"contratada_nome" binding:"required"`
	ContratadaCNPJ   string            `json:"contratada_cnpj" binding:"required"`
	ContratadaEmail  string            `json:"contratada_email"`
	FiscalID         uuid.UUID         `json:"fiscal_id" binding:"required"`
	TipoObjeto       models.TipoObjeto `json:"tipo_objeto" binding:"required"`
}

// Criar trata POST /api/v1/contratos.
func (h *ContratoHandler) Criar(c *gin.Context) {
	var req criarContratoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	contrato, err := h.contratoService.Criar(c.Request.Context(), service.NovoContratoInput{
		NumeroContrato:   req.NumeroContrato,
		PortariaNomeacao: req.PortariaNomeacao,
		DataAssinatura:   req.DataAssinatura,
		ContratadaNome:   req.ContratadaNome,
		ContratadaCNPJ:   req.ContratadaCNPJ,
		ContratadaEmail:  req.ContratadaEmail,
		FiscalID:         req.FiscalID,
		TipoObjeto:       req.TipoObjeto,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, contrato)
}

// Listar trata GET /api/v1/contratos.
func (h *ContratoHandler) Listar(c *gin.Context) {
	contratos, err := h.contratoService.Listar(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contratos)
}

// Buscar trata GET /api/v1/contratos/:id.
func (h *ContratoHandler) Buscar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	contrato, err := h.contratoService.Buscar(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contrato)
}
