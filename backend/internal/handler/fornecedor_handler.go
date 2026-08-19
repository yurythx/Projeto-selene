package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/service"
)

// FornecedorHandler expõe o Dossiê do Fornecedor (Fase 4 do roadmap) —
// somente leitura, mesmo nível de acesso de GET /contratos.
type FornecedorHandler struct {
	fornecedorService *service.FornecedorService
}

// NewFornecedorHandler constrói um FornecedorHandler.
func NewFornecedorHandler(fornecedorService *service.FornecedorService) *FornecedorHandler {
	return &FornecedorHandler{fornecedorService: fornecedorService}
}

// Listar trata GET /api/v1/fornecedores.
func (h *FornecedorHandler) Listar(c *gin.Context) {
	fornecedores, err := h.fornecedorService.Listar(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, fornecedores)
}

// Buscar trata GET /api/v1/fornecedores/:cnpj.
func (h *FornecedorHandler) Buscar(c *gin.Context) {
	cnpj := c.Param("cnpj")

	dossie, err := h.fornecedorService.Buscar(c.Request.Context(), cnpj)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dossie)
}
