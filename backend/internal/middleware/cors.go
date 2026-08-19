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
//
// gin-contrib/cors entra em PANIC se AllowOrigins vier vazio (a lib trata
// isso como configuração inválida — "conflict settings: all origins
// disabled" — não como "bloqueia tudo"). É exatamente o valor que
// docker-compose.prod.yml usa de propósito (CORS_ALLOWED_ORIGINS="",
// arquitetura BFF: nenhuma chamada cross-site de browser deveria chegar
// aqui) — o backend derrubava em crash loop logo no boot, achado rodando
// o compose de produção de verdade nesta sessão, sem cobertura de teste
// alguma antes disso. Corrigido devolvendo um no-op nesse caso: sem
// nenhum middleware de CORS, o backend nunca manda
// Access-Control-Allow-Origin, e é isso que já faz o navegador bloquear
// toda chamada cross-site por conta própria — mesmo resultado
// fail-closed, sem o crash.
func NewCORS(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}
