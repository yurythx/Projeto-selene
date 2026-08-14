package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/logging"
)

// HeaderRequestID é o cabeçalho HTTP usado para propagar/expor o request
// ID — tanto na entrada (se o chamador já mandar um, ex: um proxy que
// gera correlation IDs) quanto na resposta (para o cliente conseguir citar
// esse ID ao reportar um problema).
const HeaderRequestID = "X-Request-ID"

// RequestID garante que toda requisição tenha um identificador único,
// usado para correlacionar as linhas de log de uma mesma requisição
// (especialmente relevante para a notificação assíncrona da Etapa 3, que
// termina de rodar bem depois da resposta HTTP já ter sido enviada).
//
// Se o cliente/proxy já mandou um X-Request-ID, ele é reaproveitado (não
// sobrescrito) — permite correlacionar com um load balancer/API gateway
// na frente da API. Caso contrário, um novo UUID é gerado.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Writer.Header().Set(HeaderRequestID, requestID)
		ctx := logging.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
