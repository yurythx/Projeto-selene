package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/service"
)

// NotificacaoHandler expõe as notificações in-app de prazos/vencimentos
// (Radar) do usuário autenticado — sem restrição de admin/fiscal, é
// sobre a PRÓPRIA conta (mesmo espírito de POST /auth/trocar-senha).
type NotificacaoHandler struct {
	service *service.NotificacaoService
}

func NewNotificacaoHandler(svc *service.NotificacaoService) *NotificacaoHandler {
	return &NotificacaoHandler{service: svc}
}

// notificacaoResposta é o formato de saída — só os campos que a UI
// precisa, sem ChaveAlerta (nunca sai da API, ver o model).
type notificacaoResposta struct {
	ID             uuid.UUID  `json:"id"`
	Tipo           string     `json:"tipo"`
	Nivel          string     `json:"nivel"`
	ContratoID     uuid.UUID  `json:"contrato_id"`
	NumeroContrato string     `json:"numero_contrato,omitempty"`
	ProcessoID     *uuid.UUID `json:"processo_id,omitempty"`
	Mensagem       string     `json:"mensagem"`
	Lida           bool       `json:"lida"`
	CriadaEm       string     `json:"criada_em"`
}

// Listar trata GET /api/v1/notificacoes.
func (h *NotificacaoHandler) Listar(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	notificacoes, err := h.service.Listar(c.Request.Context(), usuario.ID)
	if err != nil {
		respondError(c, err)
		return
	}

	resposta := make([]notificacaoResposta, 0, len(notificacoes))
	for _, n := range notificacoes {
		numeroContrato := ""
		if n.Contrato != nil {
			numeroContrato = n.Contrato.NumeroContrato
		}
		resposta = append(resposta, notificacaoResposta{
			ID:             n.ID,
			Tipo:           n.Tipo,
			Nivel:          n.Nivel,
			ContratoID:     n.ContratoID,
			NumeroContrato: numeroContrato,
			ProcessoID:     n.ProcessoID,
			Mensagem:       n.Mensagem,
			Lida:           n.Lida,
			CriadaEm:       n.CriadaEm.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	c.JSON(http.StatusOK, resposta)
}

// ContarNaoLidas trata GET /api/v1/notificacoes/nao-lidas.
func (h *NotificacaoHandler) ContarNaoLidas(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	total, err := h.service.ContarNaoLidas(c.Request.Context(), usuario.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total})
}

// MarcarLida trata POST /api/v1/notificacoes/:id/marcar-lida.
func (h *NotificacaoHandler) MarcarLida(c *gin.Context) {
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

	if err := h.service.MarcarLida(c.Request.Context(), usuario.ID, id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MarcarTodasLidas trata POST /api/v1/notificacoes/marcar-todas-lidas.
func (h *NotificacaoHandler) MarcarTodasLidas(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	if err := h.service.MarcarTodasLidas(c.Request.Context(), usuario.ID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
