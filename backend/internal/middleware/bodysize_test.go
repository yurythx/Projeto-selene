package middleware_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestMaxBodySize_JSON confirma o achado da auditoria de segurança: sem
// este middleware, um endpoint JSON aceitava um corpo de tamanho
// arbitrário (DoS por exaustão de memória).
func TestMaxBodySize_JSON(t *testing.T) {
	router := gin.New()
	router.Use(middleware.MaxBodySize())
	router.POST("/echo", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusRequestEntityTooLarge, "corpo excede o limite")
			return
		}
		c.String(http.StatusOK, "%d bytes", len(body))
	})

	t.Run("corpo pequeno passa normalmente", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"a":1}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("corpo acima de 1MiB é rejeitado, não lido inteiro na memória", func(t *testing.T) {
		corpoGrande := bytes.Repeat([]byte("a"), (1<<20)+1) // 1 byte acima do limite
		req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(corpoGrande))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("esperava 413, veio %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestMaxBodySize_PulaMultipart confirma que requisições multipart/
// form-data NÃO são afetadas por este middleware — os handlers de upload
// (documento, foto de vistoria) aplicam seu próprio limite maior
// (20MB), e http.MaxBytesReader não é cumulativo: se este middleware
// também envolvesse o body com um teto de 1MiB, todo upload de arquivo
// acima de 1MiB quebraria mesmo estando dentro do limite de 20MB
// documentado nesses handlers.
func TestMaxBodySize_PulaMultipart(t *testing.T) {
	router := gin.New()
	router.Use(middleware.MaxBodySize())
	router.POST("/echo", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusRequestEntityTooLarge, "corpo excede o limite")
			return
		}
		c.String(http.StatusOK, "%d bytes", len(body))
	})

	// Corpo "multipart" fake de 2MiB, acima do teto de 1MiB que este
	// middleware aplicaria a JSON — precisa passar batido.
	corpoGrande := bytes.Repeat([]byte("a"), 2<<20)
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(corpoGrande))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200 (multipart não deveria ser limitado por este middleware), veio %d: %s", w.Code, w.Body.String())
	}
}
