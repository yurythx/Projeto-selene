package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/logging"
)

// StructuredLogger substitui o gin.Logger() padrão (texto, não
// estruturado) por um que loga via slog — cada requisição vira um evento
// estruturado com method/path/status/duração/request_id, prontos para um
// agregador de log (ex: Loki, CloudWatch, Datadog) filtrar/correlacionar.
//
// Precisa rodar DEPOIS de RequestID() na cadeia de middlewares, para que
// logging.FromContext já encontre o request_id no contexto da requisição.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		inicio := time.Now()
		caminho := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			caminho = caminho + "?" + raw
		}

		c.Next()

		logger := logging.FromContext(c.Request.Context())
		logger.Info("requisição HTTP",
			"method", c.Request.Method,
			"path", caminho,
			"status", c.Writer.Status(),
			"duracao_ms", time.Since(inicio).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
