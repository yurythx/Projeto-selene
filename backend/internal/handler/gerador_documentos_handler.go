package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
)

// respondDocumentoGerado devolve o conteúdo gerado (PDF fallback ou
// .docx de um modelo, ver GeradorDocumentosService) com o Content-Type e
// a extensão certos — nomeBase sem extensão, ex: "notificacao-descumprimento".
func respondDocumentoGerado(c *gin.Context, conteudo []byte, formato models.TipoFormatoDocumento, nomeBase string) {
	if formato == models.FormatoDOCX {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.docx"`, nomeBase))
		c.Data(http.StatusOK, docxContentType, conteudo)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s.pdf"`, nomeBase))
	c.Data(http.StatusOK, "application/pdf", conteudo)
}

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

	conteudo, formato, _, err := h.geradorService.GerarNotificacao(c.Request.Context(), contratoID, req.Motivo, usuario.ID, usuario.Nome)
	if err != nil {
		respondError(c, err)
		return
	}

	respondDocumentoGerado(c, conteudo, formato, "notificacao-descumprimento")
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

	conteudo, formato, _, err := h.geradorService.GerarAtesto(c.Request.Context(), processoID, usuario.ID, usuario.Nome)
	if err != nil {
		respondError(c, err)
		return
	}

	respondDocumentoGerado(c, conteudo, formato, "atesto")
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

	conteudo, formato, _, err := h.geradorService.GerarMinutaAditivo(c.Request.Context(), contratoID, service.MinutaAditivoInput{
		TipoAditivo:   req.TipoAditivo,
		Justificativa: req.Justificativa,
		NovoValor:     req.NovoValor,
		NovoPrazo:     req.NovoPrazo,
	}, usuario.ID, usuario.Nome)
	if err != nil {
		respondError(c, err)
		return
	}

	respondDocumentoGerado(c, conteudo, formato, "minuta-aditivo")
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
