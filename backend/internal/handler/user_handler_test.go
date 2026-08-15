package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/handler"
	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
	"projeto-selene/internal/testutil"
)

// TestUserHandler_ListarServidores confirma o contrato de segurança de
// ListarServidores: a projeção devolvida traz só ID/Nome/Email, nunca os
// campos administrativos (IsAdmin, IsFiscal, Matricula) que GET
// /admin/users expõe — essencial porque, ao contrário das demais rotas de
// UserHandler, esta não é admin-only (ver o comentário no handler).
func TestUserHandler_ListarServidores(t *testing.T) {
	gin.SetMode(gin.TestMode)

	admin := &models.User{
		ID:        uuid.New(),
		Nome:      "Ana Administradora",
		Email:     "ana@selene.test",
		IsAdmin:   true,
		IsFiscal:  true,
		Matricula: "12345",
	}
	fiscal := &models.User{
		ID:    uuid.New(),
		Nome:  "Bruno Fiscal",
		Email: "bruno@selene.test",
	}

	userRepo := testutil.NewFakeUserRepository(admin, fiscal)
	userService := service.NewUserService(userRepo)
	h := handler.NewUserHandler(userService, nil)

	router := gin.New()
	router.GET("/servidores", h.ListarServidores)

	req := httptest.NewRequest(http.MethodGet, "/servidores", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
	}

	var opcoes []handler.ServidorOpcao
	if err := json.Unmarshal(w.Body.Bytes(), &opcoes); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	if len(opcoes) != 2 {
		t.Fatalf("esperava 2 opções, veio %d", len(opcoes))
	}

	// A resposta bruta não pode conter os campos administrativos —
	// checagem no JSON cru, não só na struct tipada (que já filtraria por
	// definição), pra pegar uma eventual regressão que exponha campos
	// novos sem atualizar ServidorOpcao.
	var bruto []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &bruto); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	for _, campo := range []string{"IsAdmin", "IsFiscal", "Matricula", "PasswordHash", "KeycloakID"} {
		for _, item := range bruto {
			if _, existe := item[campo]; existe {
				t.Fatalf("campo administrativo %q vazou na resposta de /servidores: %v", campo, item)
			}
		}
	}
}
