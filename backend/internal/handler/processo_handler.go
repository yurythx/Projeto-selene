package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/service"
)

// ProcessoHandler expõe as rotas HTTP do quadro Kanban: criação de cards,
// consulta por etapa/id, avanço de etapa e conclusão de pagamento.
type ProcessoHandler struct {
	kanbanService *service.KanbanService
}

// NewProcessoHandler constrói um ProcessoHandler.
func NewProcessoHandler(kanbanService *service.KanbanService) *ProcessoHandler {
	return &ProcessoHandler{kanbanService: kanbanService}
}

type criarProcessoRequest struct {
	ContratoID    uuid.UUID `json:"contrato_id" binding:"required"`
	MesReferencia string    `json:"mes_referencia" binding:"required"` // "MM/AAAA"
}

// Criar trata POST /api/v1/processos — abre um novo card na primeira
// etapa do funil.
func (h *ProcessoHandler) Criar(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	var req criarProcessoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	processo, err := h.kanbanService.CriarProcesso(c.Request.Context(), req.ContratoID, req.MesReferencia, usuario.ID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, processo)
}

// Listar trata GET /api/v1/processos?etapa=N&pagina=&tamanho= — alimenta
// uma coluna do quadro Kanban. O parâmetro etapa é obrigatório: listar
// "todos os processos" sem filtro de etapa não existe, já que o board
// sempre navega por coluna.
func (h *ProcessoHandler) Listar(c *gin.Context) {
	etapaParam := c.Query("etapa")
	if etapaParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parâmetro de query 'etapa' é obrigatório"})
		return
	}

	etapaID, err := strconv.Atoi(etapaParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parâmetro 'etapa' precisa ser um número"})
		return
	}

	resultado, err := h.kanbanService.ListarPorEtapa(c.Request.Context(), etapaID, paginaFromQuery(c))
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resultado)
}

// Buscar trata GET /api/v1/processos/:id.
func (h *ProcessoHandler) Buscar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	processo, err := h.kanbanService.BuscarProcesso(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, processo)
}

// Avancar trata POST /api/v1/processos/:id/avancar — move o card para a
// próxima etapa se o checklist da etapa atual estiver completo.
func (h *ProcessoHandler) Avancar(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	processo, err := h.kanbanService.AvancarEtapa(c.Request.Context(), id, usuario.ID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, processo)
}

// Concluir trata POST /api/v1/processos/:id/concluir — marca o processo
// como pago, só permitido a partir da etapa final.
func (h *ProcessoHandler) Concluir(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	processo, err := h.kanbanService.ConcluirPagamento(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, processo)
}
