package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/service"
)

// DiarioOficialHandler expõe a configuração da integração com o Diário
// Oficial da cidade (leitura/escrita/teste de conexão, admin-only) e a
// busca de novos contratos publicados — Configurações → Diário Oficial.
type DiarioOficialHandler struct {
	service *service.DiarioOficialService
}

func NewDiarioOficialHandler(svc *service.DiarioOficialService) *DiarioOficialHandler {
	return &DiarioOficialHandler{service: svc}
}

// Buscar trata GET /api/v1/admin/config/diario-oficial.
func (h *DiarioOficialHandler) Buscar(c *gin.Context) {
	cfg, err := h.service.Buscar(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

type atualizarDiarioOficialConfigRequest struct {
	BaseURL string `json:"base_url" binding:"required"`
	// APIKey sem "required" de propósito — vazio numa atualização
	// significa "manter a chave atual" (ver o comentário no service).
	APIKey string `json:"api_key"`
}

// Atualizar trata PUT /api/v1/admin/config/diario-oficial.
func (h *DiarioOficialHandler) Atualizar(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	var req atualizarDiarioOficialConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	err := h.service.Salvar(c.Request.Context(), usuario.ID, service.AtualizarConfiguracaoDiarioOficial{
		BaseURL: req.BaseURL,
		APIKey:  req.APIKey,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	cfg, err := h.service.Buscar(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// TestarConexao trata POST /api/v1/admin/config/diario-oficial/testar.
func (h *DiarioOficialHandler) TestarConexao(c *gin.Context) {
	resultado, err := h.service.TestarConexao(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resultado)
}

// BuscarContratos trata GET /api/v1/admin/diario-oficial/contratos?nome=&cpf=&data=.
func (h *DiarioOficialHandler) BuscarContratos(c *gin.Context) {
	resultado, err := h.service.BuscarContratos(c.Request.Context(), service.FiltroBuscaContratos{
		Nome: c.Query("nome"),
		CPF:  c.Query("cpf"),
		Data: c.Query("data"),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"resultado": resultado})
}
