package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"projeto-selene/internal/database"
)

// HealthHandler expõe o healthcheck da aplicação, verificando a
// dependência principal (Postgres) — um health check que não checa nada
// além do processo estar de pé não serve pra orquestrador nenhum decidir
// se deve reiniciar/tirar de circulação uma instância.
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler constrói um HealthHandler.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Check trata GET /health.
func (h *HealthHandler) Check(c *gin.Context) {
	if err := database.Ping(c.Request.Context(), h.db); err != nil {
		// /health não exige autenticação (convenção de orquestrador/load
		// balancer) — err.Error() de uma falha de Postgres pode incluir
		// host, porta ou detalhes de conexão; isso não vai pra um chamador
		// anônimo, só pro log do servidor.
		slog.ErrorContext(c.Request.Context(), "health check: Postgres indisponível", "erro", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "indisponível"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
