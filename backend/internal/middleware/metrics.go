package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/metrics"
)

// Metrics registra HTTPRequestsTotal/HTTPRequestDuration para toda
// requisição. Usa c.FullPath() (o PADRÃO da rota, ex: "/processos/:id"),
// não a URL crua — evita explodir a cardinalidade das métricas com um
// label por UUID já visto.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		inicio := time.Now()

		c.Next()

		rota := c.FullPath()
		if rota == "" {
			rota = "não_encontrada" // requisição não casou com nenhuma rota registrada (404)
		}

		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, rota, strconv.Itoa(c.Writer.Status())).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, rota).Observe(time.Since(inicio).Seconds())
	}
}
