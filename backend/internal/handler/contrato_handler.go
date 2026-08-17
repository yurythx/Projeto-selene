package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/service"
)

// ContratoHandler expõe as rotas HTTP de cadastro/consulta de contratos.
type ContratoHandler struct {
	contratoService *service.ContratoService
}

// NewContratoHandler constrói um ContratoHandler.
func NewContratoHandler(contratoService *service.ContratoService) *ContratoHandler {
	return &ContratoHandler{contratoService: contratoService}
}

type criarContratoRequest struct {
	NumeroContrato   string            `json:"numero_contrato" binding:"required"`
	PortariaNomeacao string            `json:"portaria_nomeacao"`
	DataAssinatura   string            `json:"data_assinatura" binding:"required"` // "AAAA-MM-DD"
	ContratadaNome   string            `json:"contratada_nome" binding:"required"`
	ContratadaCNPJ   string            `json:"contratada_cnpj" binding:"required"`
	ContratadaEmail  string            `json:"contratada_email"`
	FiscalID         uuid.UUID         `json:"fiscal_id" binding:"required"`
	TipoObjeto       models.TipoObjeto `json:"tipo_objeto" binding:"required"`
	// DataVigenciaFim ("AAAA-MM-DD") é opcional — alimenta o Radar de
	// Alertas (Fase 1 do roadmap); sem ela, o contrato não aparece no
	// radar de vigência.
	DataVigenciaFim string `json:"data_vigencia_fim"`
	// ExigeFiscalizacaoTerceirizacao é opcional (default false) — marca o
	// contrato como sujeito à IN SCL Nº 04/2021 (mão de obra
	// terceirizada), ver o comentário do campo homônimo em models.Contrato.
	ExigeFiscalizacaoTerceirizacao bool `json:"exige_fiscalizacao_terceirizacao"`
}

// Criar trata POST /api/v1/contratos.
func (h *ContratoHandler) Criar(c *gin.Context) {
	var req criarContratoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	contrato, err := h.contratoService.Criar(c.Request.Context(), service.NovoContratoInput{
		NumeroContrato:                 req.NumeroContrato,
		PortariaNomeacao:               req.PortariaNomeacao,
		DataAssinatura:                 req.DataAssinatura,
		ContratadaNome:                 req.ContratadaNome,
		ContratadaCNPJ:                 req.ContratadaCNPJ,
		ContratadaEmail:                req.ContratadaEmail,
		FiscalID:                       req.FiscalID,
		TipoObjeto:                     req.TipoObjeto,
		DataVigenciaFim:                req.DataVigenciaFim,
		ExigeFiscalizacaoTerceirizacao: req.ExigeFiscalizacaoTerceirizacao,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, contrato)
}

// Listar trata GET
// /api/v1/contratos?pagina=&tamanho=&busca=&tipo_objeto=&situacao=. Os
// três últimos são opcionais — ver repository.FiltroContrato pro
// comportamento de cada um (inclusive valores inválidos, que são
// ignorados em vez de gerar erro).
func (h *ContratoHandler) Listar(c *gin.Context) {
	filtro := repository.FiltroContrato{
		Busca:      c.Query("busca"),
		TipoObjeto: models.TipoObjeto(c.Query("tipo_objeto")),
		Situacao:   c.Query("situacao"),
	}

	resultado, err := h.contratoService.Listar(c.Request.Context(), paginaFromQuery(c), filtro)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resultado)
}

// Buscar trata GET /api/v1/contratos/:id.
func (h *ContratoHandler) Buscar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	contrato, err := h.contratoService.Buscar(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contrato)
}

type atualizarContratoRequest struct {
	PortariaNomeacao               *string `json:"portaria_nomeacao"`
	ContratadaNome                 *string `json:"contratada_nome"`
	ContratadaCNPJ                 *string `json:"contratada_cnpj"`
	ContratadaEmail                *string `json:"contratada_email"`
	DataVigenciaFim                *string `json:"data_vigencia_fim"`
	ExigeFiscalizacaoTerceirizacao *bool   `json:"exige_fiscalizacao_terceirizacao"`
}

// Atualizar trata PATCH /api/v1/contratos/:id — só os campos presentes no
// corpo da requisição são alterados.
func (h *ContratoHandler) Atualizar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req atualizarContratoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	contrato, err := h.contratoService.Atualizar(c.Request.Context(), id, service.AtualizarContratoInput{
		PortariaNomeacao:               req.PortariaNomeacao,
		ContratadaNome:                 req.ContratadaNome,
		ContratadaCNPJ:                 req.ContratadaCNPJ,
		ContratadaEmail:                req.ContratadaEmail,
		DataVigenciaFim:                req.DataVigenciaFim,
		ExigeFiscalizacaoTerceirizacao: req.ExigeFiscalizacaoTerceirizacao,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contrato)
}

// Encerrar trata POST /api/v1/contratos/:id/encerrar.
func (h *ContratoHandler) Encerrar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	contrato, err := h.contratoService.Encerrar(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contrato)
}
