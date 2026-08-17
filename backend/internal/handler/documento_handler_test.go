package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/handler"
	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
	"projeto-selene/internal/testutil"
)

// setupDocumentoRouter monta um handler.DocumentoHandler real sobre um
// service.DocumentoService real (repositories fake) e injeta um usuário
// autenticado fake no contexto — Upload usa middleware.UserFromContext,
// então precisa de algo lá mesmo fora do middleware de auth de verdade.
// Retorna também o ID do processo já cadastrado no repository fake — os
// testes precisam usar ESSE id na URL, não um uuid.New() qualquer, senão
// o FindByID do fake não acha o processo (404 antes de chegar na lógica
// que o teste quer exercitar).
func setupDocumentoRouter(t *testing.T, storageDir string, usuario *models.User) (*gin.Engine, *testutil.FakeDocumentoAnexoRepository, uuid.UUID) {
	t.Helper()

	processoID := uuid.New()
	docRepo := &testutil.FakeDocumentoAnexoRepository{}
	tipoDocRepo := testutil.NewFakeTipoDocumentoRepository(&models.TipoDocumento{ID: 1, Nome: "Nota Fiscal / Fatura"})
	processoRepo := testutil.NewFakeProcessoPagamentoRepository(&models.ProcessoPagamento{ID: processoID})

	documentoService := service.NewDocumentoService(docRepo, tipoDocRepo, processoRepo, storageDir)
	h := handler.NewDocumentoHandler(documentoService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUser, usuario)
		c.Next()
	})
	router.POST("/processos/:id/documentos", h.Upload)
	router.GET("/processos/:id/documentos", h.Listar)
	router.GET("/processos/:id/documentos/:docId/download", h.Baixar)

	return router, docRepo, processoID
}

func montarMultipart(t *testing.T, tipoDocumentoID, filename string, conteudo []byte) (*bytes.Buffer, string) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("tipo_documento_id", tipoDocumentoID); err != nil {
		t.Fatalf("falha ao escrever campo tipo_documento_id: %v", err)
	}

	parte, err := writer.CreateFormFile("arquivo", filename)
	if err != nil {
		t.Fatalf("falha ao criar form file: %v", err)
	}
	if _, err := parte.Write(conteudo); err != nil {
		t.Fatalf("falha ao escrever conteúdo do arquivo: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("falha ao fechar multipart writer: %v", err)
	}

	return &buf, writer.FormDataContentType()
}

func TestDocumentoHandler_Upload(t *testing.T) {
	usuario := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", IsFiscal: true}

	t.Run("upload válido retorna 201", func(t *testing.T) {
		storageDir := t.TempDir()
		router, _, processoID := setupDocumentoRouter(t, storageDir, usuario)

		body, contentType := montarMultipart(t, "1", "nota_fiscal.pdf", []byte("conteúdo de teste"))

		req := httptest.NewRequest(http.MethodPost, "/processos/"+processoID.String()+"/documentos", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("esperava 201, veio %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("campo 'arquivo' ausente retorna 400", func(t *testing.T) {
		storageDir := t.TempDir()
		router, _, processoID := setupDocumentoRouter(t, storageDir, usuario)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		if err := writer.WriteField("tipo_documento_id", "1"); err != nil {
			t.Fatalf("falha ao escrever campo: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("falha ao fechar multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/processos/"+processoID.String()+"/documentos", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d", w.Code)
		}
	})

	t.Run("tipo_documento_id ausente/inválido retorna 400", func(t *testing.T) {
		storageDir := t.TempDir()
		router, _, processoID := setupDocumentoRouter(t, storageDir, usuario)

		body, contentType := montarMultipart(t, "não-é-número", "nota.pdf", []byte("x"))

		req := httptest.NewRequest(http.MethodPost, "/processos/"+processoID.String()+"/documentos", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, veio %d", w.Code)
		}
	})

	// Teste de regressão de segurança fim-a-fim: um filename malicioso
	// enviado via multipart de verdade (não só chamando o service direto)
	// precisa sair sanitizado da ponta a ponta da pilha HTTP.
	t.Run("filename com path traversal é sanitizado (regressão de segurança)", func(t *testing.T) {
		storageDir := t.TempDir()
		router, docRepo, processoID := setupDocumentoRouter(t, storageDir, usuario)

		body, contentType := montarMultipart(t, "1", "../../../../evil.sh", []byte("conteúdo malicioso"))

		req := httptest.NewRequest(http.MethodPost, "/processos/"+processoID.String()+"/documentos", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("esperava 201, veio %d: %s", w.Code, w.Body.String())
		}

		if len(docRepo.Documentos) != 1 {
			t.Fatalf("esperava 1 documento registrado, veio %d", len(docRepo.Documentos))
		}
		registrado := docRepo.Documentos[0]
		if registrado.NomeArquivo == "../../../../evil.sh" || registrado.NomeArquivo != "evil.sh" {
			t.Fatalf("regressão: NomeArquivo não foi sanitizado, veio %q", registrado.NomeArquivo)
		}

		var resposta map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resposta); err == nil {
			if _, temCaminho := resposta["CaminhoStorage"]; temCaminho {
				t.Fatal("regressão: CaminhoStorage não deveria aparecer na resposta JSON (json:\"-\")")
			}
		}
	})
}

// TestDocumentoHandler_Baixar cobre a otimização pedida pelo usuário ("a
// visualização de documento é tão lenta pra aparecer"): cache HTTP
// agressivo baseado no hash do conteúdo (ETag) + "immutable", e que uma
// segunda requisição com If-None-Match batendo o ETag recebe 304 sem
// corpo — a evidência de que o servidor não releu/retransmitiu o arquivo
// à toa.
func TestDocumentoHandler_Baixar(t *testing.T) {
	usuario := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", IsFiscal: true}
	storageDir := t.TempDir()
	router, _, processoID := setupDocumentoRouter(t, storageDir, usuario)

	body, contentType := montarMultipart(t, "1", "nota_fiscal.pdf", []byte("%PDF-1.4 conteúdo de teste"))
	uploadReq := httptest.NewRequest(http.MethodPost, "/processos/"+processoID.String()+"/documentos", body)
	uploadReq.Header.Set("Content-Type", contentType)
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)
	if uploadW.Code != http.StatusCreated {
		t.Fatalf("falha ao preparar documento de teste: %d: %s", uploadW.Code, uploadW.Body.String())
	}
	var documento struct {
		ID string
	}
	if err := json.Unmarshal(uploadW.Body.Bytes(), &documento); err != nil {
		t.Fatalf("falha ao decodificar resposta do upload: %v", err)
	}

	downloadURL := "/processos/" + processoID.String() + "/documentos/" + documento.ID + "/download"

	t.Run("primeira requisição devolve 200 com ETag, Cache-Control imutável e o conteúdo certo", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, downloadURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
		}
		if w.Body.String() != "%PDF-1.4 conteúdo de teste" {
			t.Fatalf("conteúdo inesperado: %q", w.Body.String())
		}
		if etag := w.Header().Get("ETag"); etag == "" {
			t.Fatal("esperava header ETag presente")
		}
		cacheControl := w.Header().Get("Cache-Control")
		if !strings.Contains(cacheControl, "immutable") || !strings.Contains(cacheControl, "private") {
			t.Fatalf("Cache-Control inesperado: %q", cacheControl)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
			t.Fatalf("esperava Content-Type sniffado como application/pdf, veio %q", ct)
		}
	})

	t.Run("segunda requisição com If-None-Match batendo o ETag devolve 304 sem corpo", func(t *testing.T) {
		primeira := httptest.NewRequest(http.MethodGet, downloadURL, nil)
		primeiraW := httptest.NewRecorder()
		router.ServeHTTP(primeiraW, primeira)
		etag := primeiraW.Header().Get("ETag")

		req := httptest.NewRequest(http.MethodGet, downloadURL, nil)
		req.Header.Set("If-None-Match", etag)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotModified {
			t.Fatalf("esperava 304, veio %d", w.Code)
		}
		if w.Body.Len() != 0 {
			t.Fatalf("esperava corpo vazio no 304, veio %d bytes", w.Body.Len())
		}
	})
}
