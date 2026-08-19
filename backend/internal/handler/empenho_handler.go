package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
)

// EmpenhoHandler expõe as rotas HTTP do acompanhamento PARALELO/
// informativo de saldo de empenho (Fase 2 do plano SGF-Rondonópolis) —
// ver o comentário em models.Empenho sobre este não ser a fonte de
// verdade orçamentária.
type EmpenhoHandler struct {
	empenhoService *service.EmpenhoService
}

// NewEmpenhoHandler constrói um EmpenhoHandler.
func NewEmpenhoHandler(empenhoService *service.EmpenhoService) *EmpenhoHandler {
	return &EmpenhoHandler{empenhoService: empenhoService}
}

type criarEmpenhoRequest struct {
	NumeroEmpenho string `json:"numero_empenho" binding:"required"`
	DataEmissao   string `json:"data_emissao" binding:"required"`  // "AAAA-MM-DD"
	ValorInicial  int64  `json:"valor_inicial" binding:"required"` // centavos
}

// Criar trata POST /api/v1/contratos/:id/empenhos.
func (h *EmpenhoHandler) Criar(c *gin.Context) {
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

	var req criarEmpenhoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	empenho, err := h.empenhoService.CriarEmpenho(c.Request.Context(), service.CriarEmpenhoInput{
		ContratoID:      contratoID,
		NumeroEmpenho:   req.NumeroEmpenho,
		DataEmissao:     req.DataEmissao,
		ValorInicial:    req.ValorInicial,
		RegistradoPorID: usuario.ID,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, empenho)
}

// Listar trata GET /api/v1/contratos/:id/empenhos.
func (h *EmpenhoHandler) Listar(c *gin.Context) {
	contratoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	empenhos, err := h.empenhoService.ListarPorContrato(c.Request.Context(), contratoID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, empenhos)
}

type empenhoComSaldoResponse struct {
	*models.Empenho
	Saldo int64 `json:"saldo"` // centavos, reconstruído do histórico — ver EmpenhoService.CalcularSaldo
}

// Buscar trata GET /api/v1/empenhos/:id — inclui o saldo reconstruído do
// histórico de movimentações.
func (h *EmpenhoHandler) Buscar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	empenho, err := h.empenhoService.BuscarEmpenho(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	saldo, err := h.empenhoService.CalcularSaldo(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, empenhoComSaldoResponse{Empenho: empenho, Saldo: saldo})
}

type registrarMovimentacaoRequest struct {
	Tipo                models.TipoMovimentacaoEmpenho `json:"tipo" binding:"required"`
	Valor               int64                          `json:"valor" binding:"required"` // centavos
	ProcessoPagamentoID *uuid.UUID                     `json:"processo_pagamento_id"`
	Observacao          string                         `json:"observacao"`
}

// RegistrarMovimentacao trata POST /api/v1/empenhos/:id/movimentacoes.
func (h *EmpenhoHandler) RegistrarMovimentacao(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	empenhoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req registrarMovimentacaoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	movimentacao, err := h.empenhoService.RegistrarMovimentacao(c.Request.Context(), service.RegistrarMovimentacaoInput{
		EmpenhoID:           empenhoID,
		Tipo:                req.Tipo,
		Valor:               req.Valor,
		ProcessoPagamentoID: req.ProcessoPagamentoID,
		Observacao:          req.Observacao,
		RegistradoPorID:     usuario.ID,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, movimentacao)
}
