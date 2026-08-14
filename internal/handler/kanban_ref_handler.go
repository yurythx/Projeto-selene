package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/repository"
)

// KanbanRefHandler expõe as tabelas de referência do Kanban (etapas fixas
// e tipos de documento), somente leitura — usadas pelo frontend para
// montar as colunas do quadro e os selects de upload.
type KanbanRefHandler struct {
	etapaRepo   repository.KanbanEtapaRepository
	tipoDocRepo repository.TipoDocumentoRepository
}

// NewKanbanRefHandler constrói um KanbanRefHandler.
func NewKanbanRefHandler(etapaRepo repository.KanbanEtapaRepository, tipoDocRepo repository.TipoDocumentoRepository) *KanbanRefHandler {
	return &KanbanRefHandler{etapaRepo: etapaRepo, tipoDocRepo: tipoDocRepo}
}

// ListarEtapas trata GET /api/v1/kanban/etapas.
func (h *KanbanRefHandler) ListarEtapas(c *gin.Context) {
	etapas, err := h.etapaRepo.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, etapas)
}

// ListarTiposDocumento trata GET /api/v1/kanban/tipos-documento.
func (h *KanbanRefHandler) ListarTiposDocumento(c *gin.Context) {
	tipos, err := h.tipoDocRepo.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, tipos)
}
