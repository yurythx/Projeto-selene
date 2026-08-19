package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
)

// Erros de validação do multipart de lerArquivoMultipart — mensagens
// devolvidas direto ao client (nada sensível nelas, mesmo espírito das
// mensagens já usadas em DocumentoHandler.Upload).
var (
	errCampoArquivoObrigatorio = errors.New("campo 'arquivo' é obrigatório, ou o arquivo excede o limite de 20MB")
	errArquivoExcedeLimite     = errors.New("arquivo excede o limite de 20MB")
	errFalhaAbrirArquivo       = errors.New("falha ao abrir arquivo enviado")
	errFalhaLerArquivo         = errors.New("falha ao ler arquivo enviado")
)

// docxContentType é o Content-Type oficial de um arquivo .docx (Office
// Open XML WordprocessingML) — usado tanto pro upload (não validado
// pelo Content-Type do multipart, só pelo conteúdo, ver
// service.validarDocx) quanto pro download.
const docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

// ModeloDocumentoHandler expõe as rotas HTTP de Configurações — Modelos
// de Documentos (admin-only, ver o grupo /admin em cmd/api/main.go).
type ModeloDocumentoHandler struct {
	modeloService *service.ModeloDocumentoService
}

// NewModeloDocumentoHandler constrói um ModeloDocumentoHandler.
func NewModeloDocumentoHandler(modeloService *service.ModeloDocumentoService) *ModeloDocumentoHandler {
	return &ModeloDocumentoHandler{modeloService: modeloService}
}

// Criar trata POST /api/v1/admin/modelos-documento (multipart/form-data
// com os campos "categoria", "gatilho" opcional e "arquivo").
func (h *ModeloDocumentoHandler) Criar(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	// Mesmo limite/margem de DocumentoHandler.Upload — modelos de
	// documento são .docx institucionais, não deveriam ser maiores que um
	// PDF de checklist.
	const margemMultipart = 64 << 10
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+margemMultipart)

	categoria := c.PostForm("categoria")

	var gatilho *models.TipoGatilhoModelo
	if bruto := c.PostForm("gatilho"); bruto != "" {
		g := models.TipoGatilhoModelo(bruto)
		gatilho = &g
	}

	conteudo, nomeArquivo, err := lerArquivoMultipart(c, "arquivo")
	if err != nil {
		respondArquivoMultipartError(c, err)
		return
	}

	modelo, err := h.modeloService.CriarCategoria(c.Request.Context(), categoria, gatilho, nomeArquivo, conteudo, usuario.ID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, modelo)
}

// NovaVersao trata POST /api/v1/admin/modelos-documento/:id/versoes
// (multipart/form-data com o campo "arquivo") — substitui o arquivo
// ativo da categoria, preservando o histórico.
func (h *ModeloDocumentoHandler) NovaVersao(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	modeloID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	const margemMultipart = 64 << 10
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+margemMultipart)

	conteudo, nomeArquivo, err := lerArquivoMultipart(c, "arquivo")
	if err != nil {
		respondArquivoMultipartError(c, err)
		return
	}

	modelo, err := h.modeloService.NovaVersao(c.Request.Context(), modeloID, nomeArquivo, conteudo, usuario.ID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, modelo)
}

type atualizarModeloDocumentoRequest struct {
	Categoria *string `json:"categoria"`
	// Gatilho, se presente: "" ou "NENHUM" remove a associação atual; um
	// dos 4 valores de models.TipoGatilhoModelo associa a categoria a um
	// fluxo de geração; campo ausente (nil) não mexe na associação atual.
	Gatilho *string `json:"gatilho"`
}

// Atualizar trata PATCH /api/v1/admin/modelos-documento/:id — renomeia a
// categoria e/ou troca o gatilho associado. Não mexe no arquivo (ver
// NovaVersao pra isso).
func (h *ModeloDocumentoHandler) Atualizar(c *gin.Context) {
	modeloID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req atualizarModeloDocumentoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	input := service.AtualizarCategoriaInput{Categoria: req.Categoria}
	if req.Gatilho != nil {
		if *req.Gatilho == "" || *req.Gatilho == "NENHUM" {
			var semGatilho *models.TipoGatilhoModelo
			input.Gatilho = &semGatilho
		} else {
			g := models.TipoGatilhoModelo(*req.Gatilho)
			gp := &g
			input.Gatilho = &gp
		}
	}

	modelo, err := h.modeloService.AtualizarCategoria(c.Request.Context(), modeloID, input)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, modelo)
}

// Listar trata GET /api/v1/admin/modelos-documento.
func (h *ModeloDocumentoHandler) Listar(c *gin.Context) {
	modelos, err := h.modeloService.Listar(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, modelos)
}

// Buscar trata GET /api/v1/admin/modelos-documento/:id.
func (h *ModeloDocumentoHandler) Buscar(c *gin.Context) {
	modeloID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	modelo, err := h.modeloService.Buscar(c.Request.Context(), modeloID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, modelo)
}

// Baixar trata GET /api/v1/admin/modelos-documento/:id/download — sempre
// a versão ATIVA.
func (h *ModeloDocumentoHandler) Baixar(c *gin.Context) {
	modeloID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	conteudo, nomeArquivo, err := h.modeloService.Baixar(c.Request.Context(), modeloID, nil)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+nomeArquivo+`"`)
	c.Data(http.StatusOK, docxContentType, conteudo)
}

// BaixarVersao trata GET /api/v1/admin/modelos-documento/:id/versoes/:versaoId/download
// — uma versão específica do histórico.
func (h *ModeloDocumentoHandler) BaixarVersao(c *gin.Context) {
	modeloID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	versaoID, err := uuid.Parse(c.Param("versaoId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de versão inválido"})
		return
	}

	conteudo, nomeArquivo, err := h.modeloService.Baixar(c.Request.Context(), modeloID, &versaoID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+nomeArquivo+`"`)
	c.Data(http.StatusOK, docxContentType, conteudo)
}

// respondArquivoMultipartError traduz um erro de lerArquivoMultipart pro
// status HTTP certo — 413 especificamente pro limite de tamanho, 400
// pros demais (mesma distinção de DocumentoHandler.Upload).
func respondArquivoMultipartError(c *gin.Context, err error) {
	if errors.Is(err, errArquivoExcedeLimite) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

// lerArquivoMultipart extrai e lê por completo o campo de arquivo
// informado — mesma sequência de validação de DocumentoHandler.Upload
// (FormFile, checagem de tamanho, Open, ReadAll), mas fatorada aqui
// porque este handler tem dois pontos de upload (Criar/NovaVersao) que a
// precisam igualmente.
func lerArquivoMultipart(c *gin.Context, campo string) (conteudo []byte, nomeArquivo string, err error) {
	arquivoHeader, err := c.FormFile(campo)
	if err != nil {
		return nil, "", errCampoArquivoObrigatorio
	}
	if arquivoHeader.Size > maxUploadBytes {
		return nil, "", errArquivoExcedeLimite
	}

	arquivo, err := arquivoHeader.Open()
	if err != nil {
		return nil, "", errFalhaAbrirArquivo
	}
	defer func() {
		if fecharErr := arquivo.Close(); fecharErr != nil {
			slog.WarnContext(c.Request.Context(), "falha ao fechar arquivo de upload", "arquivo", arquivoHeader.Filename, "erro", fecharErr)
		}
	}()

	conteudo, err = io.ReadAll(arquivo)
	if err != nil {
		return nil, "", errFalhaLerArquivo
	}

	return conteudo, arquivoHeader.Filename, nil
}
