package handler

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/service"
)

// DocumentoHandler expõe as rotas HTTP de upload/consulta de documentos
// anexos (PDFs) de um processo de pagamento.
type DocumentoHandler struct {
	documentoService *service.DocumentoService
}

// NewDocumentoHandler constrói um DocumentoHandler.
func NewDocumentoHandler(documentoService *service.DocumentoService) *DocumentoHandler {
	return &DocumentoHandler{documentoService: documentoService}
}

// maxUploadBytes limita o tamanho de um único arquivo anexado (20 MiB) —
// evita que um upload malformado ou abusivo esgote memória/disco.
const maxUploadBytes = 20 << 20

// Upload trata POST /api/v1/processos/:id/documentos (multipart/form-data
// com os campos "tipo_documento_id" e "arquivo").
func (h *DocumentoHandler) Upload(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	processoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	// Envolve o corpo da requisição ANTES de qualquer parse de
	// multipart/form-data (c.PostForm/c.FormFile disparam esse parse na
	// primeira chamada). Sem isso, o Gin bufferiza/grava em disco o corpo
	// inteiro antes da checagem de tamanho abaixo rodar — um upload de
	// vários GB já teria consumido memória/disco antes de ser rejeitado.
	// Com MaxBytesReader, o próprio parser aborta assim que o limite é
	// excedido, com uma margem pequena pro overhead do multipart
	// (boundary, headers de cada parte, o campo tipo_documento_id).
	const margemMultipart = 64 << 10 // 64KiB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+margemMultipart)

	tipoDocumentoID, err := strconv.Atoi(c.PostForm("tipo_documento_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'tipo_documento_id' é obrigatório e precisa ser um número, ou o arquivo excede o limite de 20MB"})
		return
	}

	arquivoHeader, err := c.FormFile("arquivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'arquivo' é obrigatório, ou o arquivo excede o limite de 20MB"})
		return
	}
	if arquivoHeader.Size > maxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "arquivo excede o limite de 20MB"})
		return
	}

	arquivo, err := arquivoHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao abrir arquivo enviado"})
		return
	}
	defer func() {
		if err := arquivo.Close(); err != nil {
			slog.WarnContext(c.Request.Context(), "falha ao fechar arquivo de upload", "arquivo", arquivoHeader.Filename, "erro", err)
		}
	}()

	conteudo, err := io.ReadAll(arquivo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao ler arquivo enviado"})
		return
	}

	// "data_validade" é opcional — só faz sentido pra certidões (ver
	// TipoDocumento.ExigeValidade). Formato "AAAA-MM-DD", igual
	// Contrato.DataAssinatura no resto da API. Se vier ausente ou
	// malformado, o documento é anexado do mesmo jeito, só sem entrar no
	// radar de certidões (ver o comentário em DocumentoService.Upload).
	var dataValidade *time.Time
	if bruto := c.PostForm("data_validade"); bruto != "" {
		if parsed, err := time.Parse("2006-01-02", bruto); err == nil {
			dataValidade = &parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'data_validade' precisa estar no formato AAAA-MM-DD"})
			return
		}
	}

	documento, err := h.documentoService.Upload(c.Request.Context(), processoID, tipoDocumentoID, arquivoHeader.Filename, conteudo, usuario.ID, dataValidade)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, documento)
}

// Listar trata GET /api/v1/processos/:id/documentos.
func (h *DocumentoHandler) Listar(c *gin.Context) {
	processoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	documentos, err := h.documentoService.Listar(c.Request.Context(), processoID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, documentos)
}
