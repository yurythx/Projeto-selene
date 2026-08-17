package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// ConfiguracaoKeycloak é o DTO de leitura da configuração de Keycloak —
// NUNCA inclui o ClientSecret em texto puro, nem pra admins (só um
// booleano indicando se já existe um configurado). Ver
// KeycloakConfigService.BuscarSegredoCompleto pro único caminho (interno,
// gated por segredo compartilhado) que expõe o valor real.
type ConfiguracaoKeycloak struct {
	ClientID              string
	IssuerURL             string
	Audience              string
	TemSegredoConfigurado bool
	// Origem é "banco_de_dados" (um admin já salvou uma configuração
	// customizada, ativa agora) ou "variaveis_de_ambiente" (ainda
	// rodando no valor de boot — KEYCLOAK_JWKS_URL/ISSUER/AUDIENCE no
	// backend; ClientID não aparece nesse caso porque o backend não
	// conhece AUTH_KEYCLOAK_ID, que é só do frontend).
	Origem            string
	AtualizadoEm      *time.Time
	AtualizadoPorNome string
}

// AtualizarConfiguracaoKeycloak é o payload de escrita. ClientSecret
// vazio numa atualização significa "manter o segredo atual" — mesmo
// padrão de "deixe em branco pra manter a senha" já usado em outras
// telas admin deste app (ex: editar-usuario-dialog.tsx no frontend).
type AtualizarConfiguracaoKeycloak struct {
	ClientID     string
	ClientSecret string
	IssuerURL    string
	Audience     string
}

// KeycloakConfigService gerencia a configuração de Keycloak editável em
// runtime (Configurações → Keycloak/SSO, admin-only) — pedido explícito
// do usuário: "já usamos [Keycloak] hoje mas não temos no front, e se eu
// quiser mudar ou implementar um novo, crie uma opção".
type KeycloakConfigService struct {
	repo      repository.KeycloakConfigRepository
	authState *middleware.AuthMiddlewareState
	// fallback é a configuração de boot (variáveis de ambiente) — usada
	// como retrato do que está ativo enquanto nenhum admin salvou uma
	// configuração customizada ainda.
	fallback middleware.AuthConfig
}

func NewKeycloakConfigService(repo repository.KeycloakConfigRepository, authState *middleware.AuthMiddlewareState, fallback middleware.AuthConfig) *KeycloakConfigService {
	return &KeycloakConfigService{repo: repo, authState: authState, fallback: fallback}
}

// DeriveJWKSURL deriva a URL do JWKS a partir da URL do issuer, seguindo
// a convenção fixa do Keycloak (.../realms/<realm> → .../realms/<realm>/
// protocol/openid-connect/certs) — evita pedir esse campo separado no
// formulário (menos um jeito de o admin errar a configuração).
func DeriveJWKSURL(issuerURL string) string {
	return strings.TrimRight(issuerURL, "/") + "/protocol/openid-connect/certs"
}

// Buscar devolve a configuração de Keycloak ATIVA agora — do banco, se
// algum admin já salvou uma, senão um retrato das variáveis de ambiente
// de boot (ClientID fica vazio nesse caso, ver o comentário no struct).
func (s *KeycloakConfigService) Buscar(ctx context.Context) (*ConfiguracaoKeycloak, error) {
	cfg, err := s.repo.Buscar(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrKeycloakConfigNotFound) {
			return &ConfiguracaoKeycloak{
				IssuerURL: s.fallback.Issuer,
				Audience:  s.fallback.Audience,
				Origem:    "variaveis_de_ambiente",
			}, nil
		}
		return nil, fmt.Errorf("service: buscar configuração de keycloak: %w", err)
	}

	audience := ""
	if cfg.Audience != nil {
		audience = *cfg.Audience
	}

	dto := &ConfiguracaoKeycloak{
		ClientID:              cfg.ClientID,
		IssuerURL:             cfg.IssuerURL,
		Audience:              audience,
		TemSegredoConfigurado: cfg.ClientSecret != "",
		Origem:                "banco_de_dados",
		AtualizadoEm:          &cfg.UpdatedAt,
	}
	if cfg.UpdatedBy != nil {
		dto.AtualizadoPorNome = cfg.UpdatedBy.Nome
	}
	return dto, nil
}

// BuscarSegredoCompleto devolve a configuração de Keycloak com o
// ClientSecret em texto puro — usado SÓ pelo endpoint interno (gated por
// segredo compartilhado, nunca por JWT de usuário comum) que o frontend
// consulta pra montar o provider Keycloak do NextAuth em runtime, sem
// reiniciar o container quando a configuração muda. Nunca exponha o
// resultado disto por uma rota alcançável com um token de usuário.
func (s *KeycloakConfigService) BuscarSegredoCompleto(ctx context.Context) (*models.KeycloakConfig, error) {
	return s.repo.Buscar(ctx)
}

// Salvar valida e persiste uma nova configuração de Keycloak, e aplica
// ela IMEDIATAMENTE à validação de token deste backend (via
// AuthMiddlewareState.Reload) — sem reiniciar o processo. A validação do
// novo issuer/JWKS acontece ANTES de gravar no banco: se o Keycloak
// informado não responder, a configuração antiga continua valendo e
// nada é salvo (fail-closed — um admin não consegue travar a
// autenticação de todo mundo com uma URL errada).
//
// Efeito colateral esperado, documentado pro admin na UI: sessões já
// abertas via SSO Keycloak podem precisar logar de novo se o Client
// ID/Secret mudar (o client antigo pode não conseguir mais renovar o
// token). Contas de login local não são afetadas.
func (s *KeycloakConfigService) Salvar(ctx context.Context, atualizadoPorID uuid.UUID, entrada AtualizarConfiguracaoKeycloak) error {
	clientID := strings.TrimSpace(entrada.ClientID)
	if clientID == "" {
		return ErrKeycloakClientIDObrigatorio
	}

	issuerURL := strings.TrimSpace(entrada.IssuerURL)
	if !issuerURLValida(issuerURL) {
		return ErrKeycloakIssuerInvalido
	}

	existente, err := s.repo.Buscar(ctx)
	if err != nil && !errors.Is(err, repository.ErrKeycloakConfigNotFound) {
		return fmt.Errorf("service: carregar configuração de keycloak existente: %w", err)
	}

	segredo := strings.TrimSpace(entrada.ClientSecret)
	if segredo == "" {
		if existente == nil || existente.ClientSecret == "" {
			return ErrKeycloakSegredoObrigatorio
		}
		segredo = existente.ClientSecret
	}

	audience := strings.TrimSpace(entrada.Audience)
	if err := s.authState.Reload(ctx, middleware.AuthConfig{
		JWKSURL:  DeriveJWKSURL(issuerURL),
		Issuer:   issuerURL,
		Audience: audience,
	}); err != nil {
		return fmt.Errorf("%w: não foi possível validar o novo issuer/JWKS (%v) — configuração NÃO foi salva", ErrKeycloakIssuerInvalido, err)
	}

	var audiencePtr *string
	if audience != "" {
		audiencePtr = &audience
	}
	novo := &models.KeycloakConfig{
		ClientID:     clientID,
		ClientSecret: segredo,
		IssuerURL:    issuerURL,
		Audience:     audiencePtr,
		UpdatedByID:  atualizadoPorID,
	}
	if err := s.repo.Salvar(ctx, novo); err != nil {
		return fmt.Errorf("service: salvar configuração de keycloak: %w", err)
	}

	return nil
}

// issuerURLValida confere só a forma (esquema http/https + host) — não
// faz uma requisição de rede aqui; isso já acontece em Reload (via
// keyfunc.NewDefaultCtx), que é quem realmente confirma que o Keycloak
// informado responde.
func issuerURLValida(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
