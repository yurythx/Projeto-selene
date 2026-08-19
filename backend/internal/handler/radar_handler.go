package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/service"
)

// RadarHandler expõe o Radar de Alertas e Prazos Legais (Fase 1 do
// roadmap) — somente leitura, qualquer usuário autenticado pode consultar
// (mesmo nível de acesso de GET /contratos e GET /processos).
type RadarHandler struct {
	radarService *service.RadarService
}

// NewRadarHandler constrói um RadarHandler.
func NewRadarHandler(radarService *service.RadarService) *RadarHandler {
	return &RadarHandler{radarService: radarService}
}

// Listar trata GET /api/v1/radar.
func (h *RadarHandler) Listar(c *gin.Context) {
	itens, err := h.radarService.Listar(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, itens)
}
