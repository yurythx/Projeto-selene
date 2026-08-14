package handler_test

import (
	"bytes"
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

func init() {
	gin.SetMode(gin.TestMode)
}

// setupContratoRouter monta um handler.ContratoHandler real sobre um
// service.ContratoService real, ambos apoiados em repositories fake em
// memória — testa a pilha HTTP -> handler -> service de verdade, só
// trocando o banco por um dublê.
func setupContratoRouter(t *testing.T, fiscal *models.User) (*gin.Engine, *testutil.FakeContratoRepository) {
	t.Helper()

	userRepo := testutil.NewFakeUserRepository(fiscal)
	contratoRepo := testutil.NewFakeContratoRepository()
	contratoService := service.NewContratoService(contratoRepo, userRepo)
	h := handler.NewContratoHandler(contratoService)

	router := gin.New()
	router.POST("/contratos", h.Criar)
	router.GET("/contratos", h.Listar)
	router.GET("/contratos/:id", h.Buscar)
	router.PATCH("/contratos/:id", h.Atualizar)
	router.POST("/contratos/:id/encerrar", h.Encerrar)

	return router, contratoRepo
}

func TestContratoHandler_Criar(t *testing.T) {
	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", IsFiscal: true}

	t.Run("corpo válido retorna 201", func(t *testing.T) {
		router, _ := setupContratoRouter(t, fiscal)

		corpo := map[string]any{
			"numero_contrato": "125/2026",
			"data_assinatura": "2026-01-15",
			"contratada_nome": "Empresa Teste",
			"contratada_cnpj": "12.345.678/0001-90",
			"fiscal_id":       fiscal.ID.String(),
			"tipo_objeto":     "CONSUMO",
		}
		corpoJSON, _ := json.Marshal(corpo)

		req := httptest.NewRequest(http.MethodPost, "/contratos", bytes.NewReader(corpoJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("esperava 201, veio %d: %s", w.Code, w.Body.String())
		}

		var resposta map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resposta); err != nil {
			t.Fatalf("resposta não é JSON válido: %v", err)
		}
		if resposta["NumeroContrato"] != "125/2026" {
			t.Fatalf("esperava NumeroContrato=125/2026, veio %v", resposta["NumeroContrato"])
		}
	})

	t.Run("JSON malformado retorna 400", func(t *testing.T) {
		router, _ := setupContratoRouter(t, fiscal)

		req := httptest.NewRequest(http.MethodPost, "/contratos", bytes.NewReader([]byte("{não é json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d", w.Code)
		}
	})

	t.Run("campo obrigatório ausente retorna 400", func(t *testing.T) {
		router, _ := setupContratoRouter(t, fiscal)

		// falta numero_contrato
		corpo := map[string]any{
			"data_assinatura": "2026-01-15",
			"contratada_nome": "Empresa Teste",
			"contratada_cnpj": "12.345.678/0001-90",
			"fiscal_id":       fiscal.ID.String(),
			"tipo_objeto":     "CONSUMO",
		}
		corpoJSON, _ := json.Marshal(corpo)

		req := httptest.NewRequest(http.MethodPost, "/contratos", bytes.NewReader(corpoJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("tipo_objeto inválido retorna 400 vindo do service", func(t *testing.T) {
		router, _ := setupContratoRouter(t, fiscal)

		corpo := map[string]any{
			"numero_contrato": "125/2026",
			"data_assinatura": "2026-01-15",
			"contratada_nome": "Empresa Teste",
			"contratada_cnpj": "12.345.678/0001-90",
			"fiscal_id":       fiscal.ID.String(),
			"tipo_objeto":     "TIPO_INEXISTENTE",
		}
		corpoJSON, _ := json.Marshal(corpo)

		req := httptest.NewRequest(http.MethodPost, "/contratos", bytes.NewReader(corpoJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestContratoHandler_Buscar(t *testing.T) {
	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", IsFiscal: true}

	t.Run("id com formato inválido retorna 400", func(t *testing.T) {
		router, _ := setupContratoRouter(t, fiscal)

		req := httptest.NewRequest(http.MethodGet, "/contratos/nao-eh-um-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d", w.Code)
		}
	})

	t.Run("id inexistente retorna 404", func(t *testing.T) {
		router, _ := setupContratoRouter(t, fiscal)

		req := httptest.NewRequest(http.MethodGet, "/contratos/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("esperava 404, veio %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestContratoHandler_Listar_Paginacao(t *testing.T) {
	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", IsFiscal: true}
	router, contratoRepo := setupContratoRouter(t, fiscal)

	for i := 0; i < 3; i++ {
		id := uuid.New()
		contratoRepo.Contratos[id] = &models.Contrato{
			ID:             id,
			NumeroContrato: uuid.NewString(),
			FiscalID:       fiscal.ID,
			TipoObjeto:     models.TipoObjetoConsumo,
			Ativo:          true,
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/contratos?pagina=1&tamanho=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
	}

	var resultado struct {
		Dados         []map[string]any `json:"dados"`
		Total         int              `json:"total"`
		Pagina        int              `json:"pagina"`
		TamanhoPagina int              `json:"tamanho_pagina"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resultado); err != nil {
		t.Fatalf("resposta não é o formato paginado esperado: %v (body: %s)", err, w.Body.String())
	}

	if resultado.Total != 3 {
		t.Fatalf("esperava total=3, veio %d", resultado.Total)
	}
	if len(resultado.Dados) != 2 {
		t.Fatalf("esperava 2 itens na página (tamanho=2), veio %d", len(resultado.Dados))
	}
	if resultado.TamanhoPagina != 2 {
		t.Fatalf("esperava tamanho_pagina=2, veio %d", resultado.TamanhoPagina)
	}
}

func TestContratoHandler_AtualizarEEncerrar(t *testing.T) {
	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", IsFiscal: true}
	router, contratoRepo := setupContratoRouter(t, fiscal)

	id := uuid.New()
	contratoRepo.Contratos[id] = &models.Contrato{
		ID:             id,
		NumeroContrato: "999/2026",
		ContratadaNome: "Nome Original",
		FiscalID:       fiscal.ID,
		TipoObjeto:     models.TipoObjetoConsumo,
		Ativo:          true,
	}

	t.Run("PATCH atualiza só o campo informado", func(t *testing.T) {
		corpo, _ := json.Marshal(map[string]any{"contratada_nome": "Nome Novo"})
		req := httptest.NewRequest(http.MethodPatch, "/contratos/"+id.String(), bytes.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}
		if contratoRepo.Contratos[id].ContratadaNome != "Nome Novo" {
			t.Fatalf("esperava ContratadaNome atualizado, veio %q", contratoRepo.Contratos[id].ContratadaNome)
		}
	})

	t.Run("POST /encerrar marca Ativo=false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/contratos/"+id.String()+"/encerrar", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}
		if contratoRepo.Contratos[id].Ativo {
			t.Fatal("esperava Ativo=false após encerrar")
		}
	})
}
