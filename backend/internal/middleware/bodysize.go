package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// maxJSONBodyBytes limita o corpo de requisições JSON comuns (criar/
// editar contrato, processo, designação etc.) — bem acima do maior
// payload real esperado (nenhum desses corpos tem motivo pra passar de
// poucos KB), mas baixo o bastante pra impedir que uma requisição envie
// um corpo de vários GB só pra forçar alocação de memória no servidor
// (DoS por exaustão de memória).
const maxJSONBodyBytes = 1 << 20 // 1 MiB

// MaxBodySize é um middleware global que envolve Request.Body com
// http.MaxBytesReader, garantindo que NENHUM endpoint JSON (mesmo um
// novo, adicionado no futuro sem essa preocupação em mente) aceite um
// corpo arbitrariamente grande — defesa em profundidade, não depende de
// cada handler lembrar de aplicar seu próprio limite.
//
// Pula explicitamente requisições multipart/form-data: os dois endpoints
// de upload (documento, foto de vistoria) já aplicam seu próprio
// http.MaxBytesReader com um limite maior e mais específico
// (maxUploadBytes = 20MB, ver documento_handler.go/vistoria_handler.go).
// http.MaxBytesReader não é cumulativo — o limite do wrapper MAIS INTERNO
// é quem vale, então se este middleware rodasse antes deles com um teto
// de 1MB, todo upload de arquivo acima de 1MB quebraria mesmo estando
// dentro do limite de 20MB documentado. A checagem por Content-Type evita
// esse conflito sem acoplar este middleware aos detalhes de cada rota.
func MaxBodySize() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBodyBytes)
		c.Next()
	}
}
