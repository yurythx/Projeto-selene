package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// ============================================================================
// DECISÃO DE ESCOPO (confirmada com o usuário antes de implementar): a API
// REAL do Diário Oficial da cidade ainda não está definida/documentada.
// Este service é a estrutura GENÉRICA — configuração (URL base + chave de
// API), teste de conexão, e uma busca por nome/CPF/data com um contrato
// de request/response RAZOÁVEL, mas assumido, não confirmado contra uma
// API real:
//
//   - Busca:    GET {BaseURL}/contratos?nome=&cpf=&data=
//   - Auth:     header "Authorization: Bearer {APIKey}"
//   - Resposta: qualquer JSON válido (array ou objeto) — repassado ao
//     cliente sem validar campos específicos, já que não há um schema
//     real pra validar contra ainda (ver BuscarContratos).
//
// Quando a API real existir, os pontos a ajustar são só
// buscarContratosURL (o caminho/query string) e o cabeçalho de auth em
// requisicaoAutenticada — o resto (config, teste de conexão, handler,
// frontend) não deveria precisar mudar.
// ============================================================================

const (
	diarioOficialHTTPTimeout = 15 * time.Second
	// diarioOficialMaxRespostaBytes limita quanto da resposta é lido —
	// tanto no teste de conexão (só precisamos confirmar que respondeu)
	// quanto na busca (uma API mal-comportada não deveria conseguir
	// prender memória do processo com uma resposta gigante).
	diarioOficialMaxRespostaBytes = 5 << 20 // 5 MiB
)

// ConfiguracaoDiarioOficial é o DTO de leitura — NUNCA inclui a chave de
// API em texto puro (mesmo padrão de ConfiguracaoKeycloak).
type ConfiguracaoDiarioOficial struct {
	BaseURL             string
	TemChaveConfigurada bool
	AtualizadoEm        *time.Time
	AtualizadoPorNome   string
}

// AtualizarConfiguracaoDiarioOficial é o payload de escrita. APIKey vazia
// numa atualização significa "manter a chave atual" — mesmo padrão de
// AtualizarConfiguracaoKeycloak.
type AtualizarConfiguracaoDiarioOficial struct {
	BaseURL string
	APIKey  string
}

// ResultadoTesteConexao é a resposta de TestarConexao — reporta o que
// aconteceu de forma honesta: um StatusHTTP preenchido (mesmo que seja
// 404 ou 401) já significa que o servidor respondeu, o que é informação
// útil mesmo sem saber o schema real da API. Erro só quando a conexão em
// si falhou (DNS, timeout, recusada).
type ResultadoTesteConexao struct {
	Sucesso     bool
	StatusHTTP  int
	LatenciaMS  int64
	Erro        string
	TrechoCorpo string
}

// FiltroBuscaContratos são os 3 critérios pedidos explicitamente pelo
// usuário: nome, CPF e data. Todos opcionais — repassados como veio pro
// query string da API externa, sem validação de formato aqui (não
// sabemos o formato exato que a API real vai exigir).
type FiltroBuscaContratos struct {
	Nome string
	CPF  string
	Data string
}

// DiarioOficialService gerencia a configuração da integração com o
// Diário Oficial e faz a ponte (proxy autenticado) com a API externa —
// pedido explícito do usuário: "uma sessão nas configurações onde vamos
// pesquisar novos contratos [...] direto do diário oficial da cidade".
type DiarioOficialService struct {
	repo   repository.DiarioOficialConfigRepository
	client *http.Client
}

func NewDiarioOficialService(repo repository.DiarioOficialConfigRepository) *DiarioOficialService {
	return &DiarioOficialService{
		repo:   repo,
		client: &http.Client{Timeout: diarioOficialHTTPTimeout},
	}
}

// Buscar devolve a configuração ativa (sem a chave de API).
func (s *DiarioOficialService) Buscar(ctx context.Context) (*ConfiguracaoDiarioOficial, error) {
	cfg, err := s.repo.Buscar(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrDiarioOficialConfigNotFound) {
			return &ConfiguracaoDiarioOficial{}, nil
		}
		return nil, fmt.Errorf("service: buscar configuração de diário oficial: %w", err)
	}

	dto := &ConfiguracaoDiarioOficial{
		BaseURL:             cfg.BaseURL,
		TemChaveConfigurada: cfg.APIKey != "",
		AtualizadoEm:        &cfg.UpdatedAt,
	}
	if cfg.UpdatedBy != nil {
		dto.AtualizadoPorNome = cfg.UpdatedBy.Nome
	}
	return dto, nil
}

// Salvar valida e persiste a configuração — ao contrário de
// KeycloakConfigService.Salvar, não testa a conexão antes de gravar: uma
// API externa de terceiro (fora do nosso controle) pode estar fora do ar
// temporariamente sem que isso invalide a URL/chave configuradas. Testar
// é uma ação separada e explícita (TestarConexao), disparada pelo botão
// "Testar conexão" da tela.
func (s *DiarioOficialService) Salvar(ctx context.Context, atualizadoPorID uuid.UUID, entrada AtualizarConfiguracaoDiarioOficial) error {
	baseURL := strings.TrimRight(strings.TrimSpace(entrada.BaseURL), "/")
	if !urlValida(baseURL) {
		return ErrDiarioOficialURLInvalida
	}

	existente, err := s.repo.Buscar(ctx)
	if err != nil && !errors.Is(err, repository.ErrDiarioOficialConfigNotFound) {
		return fmt.Errorf("service: carregar configuração de diário oficial existente: %w", err)
	}

	apiKey := strings.TrimSpace(entrada.APIKey)
	if apiKey == "" {
		if existente == nil || existente.APIKey == "" {
			return ErrDiarioOficialChaveObrigatoria
		}
		apiKey = existente.APIKey
	}

	novo := &models.DiarioOficialConfig{
		BaseURL:     baseURL,
		APIKey:      apiKey,
		UpdatedByID: atualizadoPorID,
	}
	if err := s.repo.Salvar(ctx, novo); err != nil {
		return fmt.Errorf("service: salvar configuração de diário oficial: %w", err)
	}

	return nil
}

// TestarConexao faz uma requisição real contra a BaseURL configurada e
// reporta o que aconteceu — não temos um endpoint de "health check"
// garantido (não é uma API nossa), então qualquer resposta HTTP (mesmo
// 404/401) já conta como "conseguiu conectar"; só erro de rede conta
// como falha de verdade.
func (s *DiarioOficialService) TestarConexao(ctx context.Context) (*ResultadoTesteConexao, error) {
	cfg, err := s.repo.Buscar(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrDiarioOficialConfigNotFound) {
			return nil, ErrDiarioOficialNaoConfigurado
		}
		return nil, fmt.Errorf("service: carregar configuração de diário oficial: %w", err)
	}

	inicio := time.Now()
	resp, err := s.requisicaoAutenticada(ctx, cfg, cfg.BaseURL)
	latencia := time.Since(inicio).Milliseconds()

	if err != nil {
		return &ResultadoTesteConexao{Sucesso: false, LatenciaMS: latencia, Erro: err.Error()}, nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "diário oficial: falha ao fechar corpo da resposta de teste", "erro", err)
		}
	}()

	corpo, _ := io.ReadAll(io.LimitReader(resp.Body, 512))

	return &ResultadoTesteConexao{
		Sucesso:     true,
		StatusHTTP:  resp.StatusCode,
		LatenciaMS:  latencia,
		TrechoCorpo: string(corpo),
	}, nil
}

// BuscarContratos proxeia a busca pra API externa configurada — ver o
// comentário de escopo no topo do arquivo pro contrato assumido de
// request/response. Devolve o JSON decodificado como `any` (não um
// struct fixo): sem um schema real pra validar contra, forçar um
// formato aqui só quebraria na primeira resposta real que viesse
// diferente do que assumimos.
func (s *DiarioOficialService) BuscarContratos(ctx context.Context, filtro FiltroBuscaContratos) (any, error) {
	cfg, err := s.repo.Buscar(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrDiarioOficialConfigNotFound) {
			return nil, ErrDiarioOficialNaoConfigurado
		}
		return nil, fmt.Errorf("service: carregar configuração de diário oficial: %w", err)
	}

	query := url.Values{}
	if filtro.Nome != "" {
		query.Set("nome", filtro.Nome)
	}
	if filtro.CPF != "" {
		query.Set("cpf", filtro.CPF)
	}
	if filtro.Data != "" {
		query.Set("data", filtro.Data)
	}

	buscaURL := cfg.BaseURL + "/contratos"
	if encoded := query.Encode(); encoded != "" {
		buscaURL += "?" + encoded
	}

	resp, err := s.requisicaoAutenticada(ctx, cfg, buscaURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiarioOficialFalhaNaBusca, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "diário oficial: falha ao fechar corpo da resposta de busca", "erro", err)
		}
	}()

	corpo, err := io.ReadAll(io.LimitReader(resp.Body, diarioOficialMaxRespostaBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: ler resposta: %v", ErrDiarioOficialFalhaNaBusca, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: API respondeu %d", ErrDiarioOficialFalhaNaBusca, resp.StatusCode)
	}

	var decodificado any
	if err := json.Unmarshal(corpo, &decodificado); err != nil {
		return nil, fmt.Errorf("%w: resposta não é JSON válido", ErrDiarioOficialFalhaNaBusca)
	}

	return decodificado, nil
}

// requisicaoAutenticada monta e executa um GET com o cabeçalho de auth
// assumido (Bearer) — ver o comentário de escopo no topo do arquivo.
func (s *DiarioOficialService) requisicaoAutenticada(ctx context.Context, cfg *models.DiarioOficialConfig, alvo string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, alvo, nil)
	if err != nil {
		return nil, fmt.Errorf("montar requisição: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Accept", "application/json")

	return s.client.Do(req)
}

func urlValida(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
