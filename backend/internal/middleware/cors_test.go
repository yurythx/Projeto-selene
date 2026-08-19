package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestNewCORS_OrigensVaziasNaoEntraEmPanic reproduz o crash de boot
// encontrado ao rodar docker-compose.prod.yml de verdade:
// CORS_ALLOWED_ORIGINS="" (fail-closed intencional, ver o comentário em
// NewCORS) fazia gin-contrib/cors entrar em panic ("conflict settings:
// all origins disabled") em vez de simplesmente bloquear tudo.
func TestNewCORS_OrigensVaziasNaoEntraEmPanic(t *testing.T) {
	router := gin.New()
	router.Use(middleware.NewCORS(nil))
	router.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://qualquer-origem.example")
	w := httptest.NewRecorder()

	// A asserção principal é implícita: se NewCORS(nil) entrar em panic,
	// este teste falha com o panic propagando, não com um erro comum.
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("com origens vazias, não deveria haver ACAO header, veio %q", got)
	}
}

func TestNewCORS_OrigemConfiguradaLiberaAcesso(t *testing.T) {
	router := gin.New()
	router.Use(middleware.NewCORS([]string{"https://selene.papermoon.cloud"}))
	router.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://selene.papermoon.cloud")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://selene.papermoon.cloud" {
		t.Fatalf("esperava ACAO=https://selene.papermoon.cloud, veio %q", got)
	}
}
