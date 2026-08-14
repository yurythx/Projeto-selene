package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/service"
)

// GeradorDocumentosHandler expõe as rotas HTTP do Módulo 2 do roadmap
// ("Gerador Inteligente de Documentos Legais"): geração dos 3 PDFs
// oficiais e a verificação pública de autenticidade via QR code.
type GeradorDocumentosHandler struct {
	geradorService *service.GeradorDocumentosService
}

// NewGeradorDocumentosHandler constrói um GeradorDocumentosHandler.
func NewGeradorDocumentosHandler(geradorService *service.GeradorDocumentosService) *GeradorDocumentosHandler {
	return &GeradorDocumentosHandler{geradorService: geradorService}
}

type gerarNotificacaoRequest struct {
	Motivo string `json:"motivo" binding:"required"`
}

// GerarNotificacao trata POST /api/v1/contratos/:id/notificacao.
func (h *GeradorDocumentosHandler) GerarNotificacao(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	contratoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req gerarNotificacaoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	pdf, _, err := h.geradorService.GerarNotificacao(c.Request.Context(), contratoID, req.Motivo, usuario.ID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Disposition", `inline; filename="notificacao-descumprimento.pdf"`)
	c.Data(http.StatusOK, "application/pdf", pdf)
}

// GerarAtesto trata POST /api/v1/processos/:id/atesto.
func (h *GeradorDocumentosHandler) GerarAtesto(c *gin.Context) {
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

	pdf, _, err := h.geradorService.GerarAtesto(c.Request.Context(), processoID, usuario.ID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Disposition", `inline; filename="atesto.pdf"`)
	c.Data(http.StatusOK, "application/pdf", pdf)
}

type gerarMinutaAditivoRequest struct {
	TipoAditivo   string `json:"tipo_aditivo" binding:"required,oneof=VALOR PRAZO VALOR_E_PRAZO"`
	Justificativa string `json:"justificativa" binding:"required"`
	NovoValor     string `json:"novo_valor"`
	NovoPrazo     string `json:"novo_prazo"`
}

// GerarMinutaAditivo trata POST /api/v1/contratos/:id/minuta-aditivo.
func (h *GeradorDocumentosHandler) GerarMinutaAditivo(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	contratoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req gerarMinutaAditivoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	pdf, _, err := h.geradorService.GerarMinutaAditivo(c.Request.Context(), contratoID, service.MinutaAditivoInput{
		TipoAditivo:   req.TipoAditivo,
		Justificativa: req.Justificativa,
		NovoValor:     req.NovoValor,
		NovoPrazo:     req.NovoPrazo,
	}, usuario.ID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Disposition", `inline; filename="minuta-aditivo.pdf"`)
	c.Data(http.StatusOK, "application/pdf", pdf)
}

type verificarDocumentoResponse struct {
	Valido            bool    `json:"valido"`
	Tipo              string  `json:"tipo,omitempty"`
	NumeroContrato    string  `json:"numero_contrato,omitempty"`
	ContratadaNome    string  `json:"contratada_nome,omitempty"`
	MesReferencia     string  `json:"mes_referencia,omitempty"`
	GeradoPorNome     string  `json:"gerado_por_nome,omitempty"`
	CriadoEm          *string `json:"criado_em,omitempty"`
	CodigoVerificacao string  `json:"codigo_verificacao,omitempty"`
}

// Verificar trata GET /api/v1/verificar/:codigo — rota PÚBLICA (fora do
// middleware de autenticação, ver cmd/api/main.go): quem escaneia o QR
// code de um Atesto impresso não tem login no Selene. Nunca retorna 404
// pra código inexistente — responde 200 com valido=false, pra não vazar
// informação de existência via status code (e pra a página de verificação
// no frontend poder tratar "inválido" e "erro de rede" de formas
// diferentes).
func (h *GeradorDocumentosHandler) Verificar(c *gin.Context) {
	codigo := c.Param("codigo")

	documento, err := h.geradorService.Verificar(c.Request.Context(), codigo)
	if err != nil {
		c.JSON(http.StatusOK, verificarDocumentoResponse{Valido: false})
		return
	}

	resp := verificarDocumentoResponse{
		Valido:            true,
		Tipo:              string(documento.Tipo),
		CodigoVerificacao: documento.CodigoVerificacao,
	}
	if documento.Contrato != nil {
		resp.NumeroContrato = documento.Contrato.NumeroContrato
		resp.ContratadaNome = documento.Contrato.ContratadaNome
	}
	if documento.ProcessoPagamento != nil {
		resp.MesReferencia = documento.ProcessoPagamento.MesReferencia
	}
	if documento.GeradoPor != nil {
		resp.GeradoPorNome = documento.GeradoPor.Nome
	}
	criadoEm := documento.CreatedAt.Format("02/01/2006 15:04")
	resp.CriadoEm = &criadoEm

	c.JSON(http.StatusOK, resp)
}
