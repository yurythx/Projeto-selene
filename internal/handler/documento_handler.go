package handler

import (
	"io"
	"log"
	"net/http"
	"strconv"

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

	tipoDocumentoID, err := strconv.Atoi(c.PostForm("tipo_documento_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'tipo_documento_id' é obrigatório e precisa ser um número"})
		return
	}

	arquivoHeader, err := c.FormFile("arquivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'arquivo' é obrigatório"})
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
			log.Printf("handler: falha ao fechar arquivo de upload %q: %v", arquivoHeader.Filename, err)
		}
	}()

	conteudo, err := io.ReadAll(arquivo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao ler arquivo enviado"})
		return
	}

	documento, err := h.documentoService.Upload(c.Request.Context(), processoID, tipoDocumentoID, arquivoHeader.Filename, conteudo, usuario.ID)
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
