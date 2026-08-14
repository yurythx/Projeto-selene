package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewCORS constrói o middleware de CORS. allowedOrigins vazio significa
// "nenhuma origem cross-site autorizada" (fail-closed) — só quando
// CORS_ALLOWED_ORIGINS é configurado explicitamente é que o frontend
// consegue chamar a API a partir do navegador.
func NewCORS(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}
