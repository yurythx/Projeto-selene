package handler

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/service"
)

// VistoriaHandler expõe as rotas HTTP do Módulo 3 do roadmap (Vistorias e
// Registro Fotográfico com Geolocalização).
type VistoriaHandler struct {
	vistoriaService *service.VistoriaService
}

// NewVistoriaHandler constrói um VistoriaHandler.
func NewVistoriaHandler(vistoriaService *service.VistoriaService) *VistoriaHandler {
	return &VistoriaHandler{vistoriaService: vistoriaService}
}

type registrarVistoriaRequest struct {
	// Latitude/Longitude são ponteiros — omitidos do JSON (ou nulos) quando
	// o browser negou/não suporta geolocalização, ver o comentário no
	// model RegistroVistoria.
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	Observacoes string   `json:"observacoes"`
}

// Registrar trata POST /api/v1/processos/:id/vistorias.
func (h *VistoriaHandler) Registrar(c *gin.Context) {
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

	var req registrarVistoriaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	vistoria, err := h.vistoriaService.Registrar(c.Request.Context(), service.RegistrarInput{
		ProcessoPagamentoID: processoID,
		FiscalID:            usuario.ID,
		Latitude:            req.Latitude,
		Longitude:           req.Longitude,
		Observacoes:         req.Observacoes,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, vistoria)
}

// ListarPorProcesso trata GET /api/v1/processos/:id/vistorias.
func (h *VistoriaHandler) ListarPorProcesso(c *gin.Context) {
	processoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	vistorias, err := h.vistoriaService.ListarPorProcesso(c.Request.Context(), processoID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, vistorias)
}

// AnexarFoto trata POST /api/v1/vistorias/:id/fotos (multipart/form-data,
// campo "foto") — mesmo limite de tamanho e mesma proteção
// MaxBytesReader-antes-do-parse de DocumentoHandler.Upload (ver o
// comentário lá pro porquê).
func (h *VistoriaHandler) AnexarFoto(c *gin.Context) {
	vistoriaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	const margemMultipart = 64 << 10
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+margemMultipart)

	arquivoHeader, err := c.FormFile("foto")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'foto' é obrigatório, ou o arquivo excede o limite de 20MB"})
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
			slog.WarnContext(c.Request.Context(), "falha ao fechar arquivo de foto de vistoria", "arquivo", arquivoHeader.Filename, "erro", err)
		}
	}()

	conteudo, err := io.ReadAll(arquivo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao ler arquivo enviado"})
		return
	}

	foto, err := h.vistoriaService.AnexarFoto(c.Request.Context(), vistoriaID, arquivoHeader.Filename, conteudo)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, foto)
}

// GerarRelatorioCampo trata GET /api/v1/vistorias/:id/relatorio.
func (h *VistoriaHandler) GerarRelatorioCampo(c *gin.Context) {
	vistoriaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	pdf, err := h.vistoriaService.GerarRelatorioCampo(c.Request.Context(), vistoriaID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Disposition", `inline; filename="relatorio-campo.pdf"`)
	c.Data(http.StatusOK, "application/pdf", pdf)
}
