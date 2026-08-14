package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter é satisfeita tanto por InMemoryRateLimiter (default, sempre
// disponível) quanto por RedisRateLimiter (ratelimit_redis.go, usado
// quando REDIS_ADDR está configurado) — main.go escolhe qual construir,
// o resto do wiring (fiscal.Use(rateLimiter.Middleware())) não muda.
type RateLimiter interface {
	Middleware() gin.HandlerFunc
}

// tempoOciosoLimite é quanto tempo um limiter de usuário fica sem uso
// antes de ser descartado — evita crescimento ilimitado do mapa de
// limiters ao longo do tempo (um usuário que não volta mais não deveria
// ocupar memória para sempre).
const tempoOciosoLimite = 3 * time.Minute

// InMemoryRateLimiter aplica um limite de requisições por usuário
// autenticado (token bucket, via golang.org/x/time/rate) nas rotas de
// escrita.
//
// LIMITAÇÃO CONHECIDA: o estado é em memória, por instância do processo.
// Com múltiplas réplicas da API atrás de um load balancer, cada réplica
// aplica seu próprio limite independente (um usuário poderia, na prática,
// ter o limite multiplicado pelo número de réplicas). Para um limite
// verdadeiramente global entre réplicas, configure REDIS_ADDR — main.go
// troca automaticamente para RedisRateLimiter (ver ratelimit_redis.go).
type InMemoryRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitanteLimiter

	rps   rate.Limit
	burst int
}

type visitanteLimiter struct {
	limiter   *rate.Limiter
	ultimoUso time.Time
}

// NewInMemoryRateLimiter constrói um InMemoryRateLimiter e inicia sua
// goroutine de limpeza periódica — vive pelo tempo de vida do processo
// (construído uma única vez em main.go), não por requisição nem por
// teste.
func NewInMemoryRateLimiter(rps float64, burst int) *InMemoryRateLimiter {
	rl := &InMemoryRateLimiter{
		visitors: make(map[string]*visitanteLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
	go rl.limparPeriodicamente()
	return rl
}

func (rl *InMemoryRateLimiter) limparPeriodicamente() {
	for {
		time.Sleep(time.Minute)

		rl.mu.Lock()
		for chave, v := range rl.visitors {
			if time.Since(v.ultimoUso) > tempoOciosoLimite {
				delete(rl.visitors, chave)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *InMemoryRateLimiter) obterLimiter(chave string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, existe := rl.visitors[chave]
	if !existe {
		v = &visitanteLimiter{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.visitors[chave] = v
	}
	v.ultimoUso = time.Now()

	return v.limiter
}

// Middleware limita requisições por usuário autenticado — identificado
// pelo ID do usuário (UserFromContext), não pelo IP, já que vários
// fiscais de uma mesma prefeitura podem estar atrás do mesmo NAT/proxy.
// Cai de volta para o IP só se, por algum motivo, rodar antes do
// middleware de autenticação (não deveria acontecer no wiring normal).
// Precisa rodar DEPOIS do middleware de autenticação na cadeia.
func (rl *InMemoryRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		chave := c.ClientIP()
		if user, ok := UserFromContext(c); ok {
			chave = user.ID.String()
		}

		if !rl.obterLimiter(chave).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "muitas requisições, tente novamente em instantes"})
			return
		}

		c.Next()
	}
}

var _ RateLimiter = (*InMemoryRateLimiter)(nil)
