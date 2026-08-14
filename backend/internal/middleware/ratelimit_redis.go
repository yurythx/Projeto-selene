package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	redisrate "github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter aplica o mesmo limite de InMemoryRateLimiter, mas com
// estado compartilhado num Redis — o limite por usuário vale de verdade
// entre múltiplas réplicas do backend, não por instância. Usa GCRA (leaky
// bucket), atômico via Lua no lado do Redis — mesma família de algoritmo
// do token bucket usado pelo limiter em memória, só que coordenado
// externamente.
type RedisRateLimiter struct {
	limiter *redisrate.Limiter
	limite  redisrate.Limit
}

// NewRedisRateLimiter constrói um RedisRateLimiter sobre um cliente Redis
// já conectado. rps é truncado para inteiro — o algoritmo GCRA do
// go-redis_rate trabalha com taxas inteiras por período, ao contrário do
// golang.org/x/time/rate (usado pelo InMemoryRateLimiter), que aceita
// frações.
func NewRedisRateLimiter(client *redis.Client, rps float64, burst int) *RedisRateLimiter {
	return &RedisRateLimiter{
		limiter: redisrate.NewLimiter(client),
		limite: redisrate.Limit{
			Rate:   int(rps),
			Period: time.Second,
			Burst:  burst,
		},
	}
}

// Middleware tem a mesma semântica de InMemoryRateLimiter.Middleware:
// limita por usuário autenticado (UserFromContext), cai para IP se rodar
// antes do middleware de autenticação.
func (rl *RedisRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		chave := c.ClientIP()
		if user, ok := UserFromContext(c); ok {
			chave = user.ID.String()
		}

		resultado, err := rl.limiter.Allow(c.Request.Context(), "ratelimit:"+chave, rl.limite)
		if err != nil {
			// Redis fora do ar não deveria derrubar a API inteira — loga e
			// deixa passar (fail-open) em vez de bloquear todo mundo por
			// causa de uma dependência secundária indisponível.
			slog.ErrorContext(c.Request.Context(), "rate limiter: falha ao consultar Redis, permitindo a requisição (fail-open)", "erro", err)
			c.Next()
			return
		}

		if resultado.Allowed == 0 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "muitas requisições, tente novamente em instantes"})
			return
		}

		c.Next()
	}
}

var _ RateLimiter = (*RedisRateLimiter)(nil)
