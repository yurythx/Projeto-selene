package handler

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/service"
)

// KeycloakConfigHandler expõe a configuração de Keycloak editável em
// runtime: leitura/escrita admin-only (Configurações → Keycloak/SSO) e
// um endpoint interno consumido pelo frontend pra montar o provider
// Keycloak do NextAuth sem reiniciar o container quando a configuração
// muda — ver o comentário em BuscarInterno sobre o modelo de confiança.
type KeycloakConfigHandler struct {
	service        *service.KeycloakConfigService
	internalSecret string
}

func NewKeycloakConfigHandler(svc *service.KeycloakConfigService, internalSecret string) *KeycloakConfigHandler {
	return &KeycloakConfigHandler{service: svc, internalSecret: internalSecret}
}

// Buscar trata GET /api/v1/admin/config/keycloak.
func (h *KeycloakConfigHandler) Buscar(c *gin.Context) {
	cfg, err := h.service.Buscar(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

type atualizarKeycloakConfigRequest struct {
	ClientID string `json:"client_id" binding:"required"`
	// ClientSecret sem "required" de propósito — vazio numa atualização
	// significa "manter o segredo atual" (ver o comentário no service).
	ClientSecret string `json:"client_secret"`
	IssuerURL    string `json:"issuer_url" binding:"required"`
	Audience     string `json:"audience"`
}

// Atualizar trata PUT /api/v1/admin/config/keycloak.
func (h *KeycloakConfigHandler) Atualizar(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	var req atualizarKeycloakConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	err := h.service.Salvar(c.Request.Context(), usuario.ID, service.AtualizarConfiguracaoKeycloak{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		IssuerURL:    req.IssuerURL,
		Audience:     req.Audience,
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

// BuscarInterno trata GET /internal/keycloak-config — deliberadamente
// FORA do grupo /api/v1 (não passa pelo middleware de JWT): é chamado
// pelo frontend Next.js ANTES de qualquer usuário existir, pra descobrir
// Client ID/Secret/Issuer atuais e montar o provider Keycloak do NextAuth
// — não há token de usuário nesse momento pra exigir. Gated por um
// segredo compartilhado fixo (X-Internal-Secret, INTERNAL_API_SECRET em
// ambos os containers) comparado em tempo constante. O backend não é
// publicamente exposto (só o frontend/BFF, ver DEPLOY.md) — este header
// é defesa em profundidade caso essa topologia de rede mude no futuro,
// não a única barreira.
//
// 404 (não 401) quando nenhum admin salvou uma configuração customizada
// ainda — sinaliza pro frontend cair de volta nas suas próprias
// variáveis de ambiente (AUTH_KEYCLOAK_ID/SECRET/ISSUER), preservando o
// comportamento de hoje pra quem nunca tocou a nova tela.
func (h *KeycloakConfigHandler) BuscarInterno(c *gin.Context) {
	recebido := c.GetHeader("X-Internal-Secret")
	if subtle.ConstantTimeCompare([]byte(recebido), []byte(h.internalSecret)) != 1 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "segredo interno inválido"})
		return
	}

	cfg, err := h.service.BuscarSegredoCompleto(c.Request.Context())
	if err != nil {
		if errors.Is(err, repository.ErrKeycloakConfigNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "nenhuma configuração customizada salva"})
			return
		}
		respondError(c, err)
		return
	}

	audience := ""
	if cfg.Audience != nil {
		audience = *cfg.Audience
	}
	c.JSON(http.StatusOK, gin.H{
		"client_id":     cfg.ClientID,
		"client_secret": cfg.ClientSecret,
		"issuer_url":    cfg.IssuerURL,
		"audience":      audience,
	})
}
