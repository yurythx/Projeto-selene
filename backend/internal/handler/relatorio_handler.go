package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/service"
)

// RelatorioHandler expõe a geração do Relatório de Pagamento (Coluna 5).
type RelatorioHandler struct {
	relatorioService *service.RelatorioService
}

// NewRelatorioHandler constrói um RelatorioHandler.
func NewRelatorioHandler(relatorioService *service.RelatorioService) *RelatorioHandler {
	return &RelatorioHandler{relatorioService: relatorioService}
}

// Gerar trata GET /api/v1/processos/:id/relatorio — renderiza o Relatório
// de Pagamento em PDF, pronto para impressão e assinatura física.
func (h *RelatorioHandler) Gerar(c *gin.Context) {
	processoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	conteudo, formato, err := h.relatorioService.Gerar(c.Request.Context(), processoID)
	if err != nil {
		respondError(c, err)
		return
	}

	respondDocumentoGerado(c, conteudo, formato, "relatorio-pagamento")
}
