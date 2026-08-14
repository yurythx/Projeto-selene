// Package config carrega e valida a configuração da aplicação a partir de
// variáveis de ambiente (opcionalmente lidas de um arquivo .env em
// desenvolvimento).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config agrupa toda a configuração necessária para inicializar o servidor
// Selene: porta HTTP, credenciais do Postgres e URL do JWKS do Keycloak.
type Config struct {
	// AppEnv é "development" ou "production" — controla o modo do Gin
	// (debug/release) e serve de referência geral de ambiente. Nunca
	// assumido implicitamente: "development" é o default seguro para não
	// vazar comportamento de produção (release mode) sem essa variável
	// estar explicitamente setada, mas qualquer deploy real DEVE setar
	// APP_ENV=production.
	AppEnv     string
	ServerPort string

	// CORSAllowedOrigins são as origens autorizadas a chamar a API a
	// partir do navegador (ex: a URL do frontend Next.js). Lista separada
	// por vírgula em CORS_ALLOWED_ORIGINS — vazio por padrão, ou seja,
	// NENHUMA origem cross-site é permitida até isso ser configurado
	// explicitamente (fail-closed, não fail-open).
	CORSAllowedOrigins []string

	// TrustedProxies são os IPs/CIDRs de proxies confiáveis para o Gin
	// resolver corretamente X-Forwarded-For. Vazio por padrão = nenhum
	// proxy confiável (Gin usa o IP da conexão direta), mais seguro que
	// o default do próprio Gin (confiar em todos).
	TrustedProxies []string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// KeycloakJWKSURL é o endpoint de chaves públicas do realm Keycloak da
	// prefeitura (ex: "https://keycloak.prefeitura.gov.br/realms/selene/protocol/openid-connect/certs").
	KeycloakJWKSURL string

	// KeycloakIssuer é o valor exato esperado no claim "iss" do JWT (ex:
	// "https://keycloak.prefeitura.gov.br/realms/selene") — validado em
	// todo request, não só a assinatura.
	KeycloakIssuer string

	// KeycloakAudience é o valor esperado no claim "aud", se definido.
	// Vazio (default) = audience não é validada — nem todo client do
	// Keycloak popula "aud" da mesma forma, então essa checagem é opt-in.
	KeycloakAudience string

	// LogLevel: "debug", "info" (default), "warn" ou "error".
	LogLevel string
	// LogFormat: "json" ou "text". Default depende de AppEnv — "json" em
	// produção (para agregadores de log), "text" em desenvolvimento
	// (legível no terminal).
	LogFormat string

	// RateLimitRPS/RateLimitBurst configuram o limite de requisições por
	// usuário autenticado nas rotas de escrita (token bucket). Ver
	// internal/middleware/ratelimit.go.
	RateLimitRPS   float64
	RateLimitBurst int

	// RedisAddr ("host:porta"), se definido, faz o rate limit usar um
	// backend Redis compartilhado (internal/middleware.RedisRateLimiter)
	// em vez do limiter em memória — necessário pra o limite valer entre
	// múltiplas réplicas do backend. Vazio (default) = limiter em
	// memória, por instância.
	RedisAddr string
	// RedisPassword autentica no Redis quando ele exige senha
	// (--requirepass, recomendado em produção). Vazio (default) = sem
	// autenticação — só aceitável quando o Redis não está exposto fora
	// da rede Docker interna.
	RedisPassword string

	// OTELExporterEndpoint ("host:porta", sem esquema — ex:
	// "jaeger:4318"), se definido, habilita a exportação real de traces
	// via OTLP/HTTP. Vazio (default) = tracing desabilitado (sampler
	// nunca grava, sem exportar nada). Ver internal/tracing.
	OTELExporterEndpoint string
	// OTELServiceName identifica este serviço nos traces exportados.
	OTELServiceName string

	// StorageDir é o diretório raiz onde os documentos anexos (PDFs) são
	// gravados localmente.
	StorageDir string

	// Configuração SMTP para o envio assíncrono do pacote digital à
	// empresa contratada (Etapa 3 do Kanban). Todos opcionais — se
	// SMTPHost estiver vazio, o envio real fica desabilitado e apenas
	// registrado em log (ver internal/service.NewNotifier).
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
}

// Load lê a configuração de variáveis de ambiente. Um arquivo .env na raiz
// do projeto é carregado se existir — sua ausência não é um erro, pois em
// produção as variáveis são normalmente injetadas diretamente pelo
// orquestrador (não há um .env no servidor).
//
// Variáveis obrigatórias ausentes causam falha explícita (fail-fast), em
// vez de a aplicação subir com configuração inválida.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("config: falha ao ler arquivo .env: %w", err)
	}

	dbHost, err := requireEnv("DB_HOST")
	if err != nil {
		return nil, err
	}

	dbUser, err := requireEnv("DB_USER")
	if err != nil {
		return nil, err
	}

	dbPassword, err := requireEnv("DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	dbName, err := requireEnv("DB_NAME")
	if err != nil {
		return nil, err
	}

	keycloakJWKSURL, err := requireEnv("KEYCLOAK_JWKS_URL")
	if err != nil {
		return nil, err
	}

	keycloakIssuer, err := requireEnv("KEYCLOAK_ISSUER")
	if err != nil {
		return nil, err
	}

	appEnv := getEnvOrDefault("APP_ENV", "development")

	defaultLogFormat := "text"
	if appEnv == "production" {
		defaultLogFormat = "json"
	}

	rateLimitRPS, err := parseFloatOrDefault("RATE_LIMIT_RPS", 5)
	if err != nil {
		return nil, err
	}
	rateLimitBurst, err := parseIntOrDefault("RATE_LIMIT_BURST", 10)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:     appEnv,
		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),

		CORSAllowedOrigins: splitCommaList(os.Getenv("CORS_ALLOWED_ORIGINS")),
		TrustedProxies:     splitCommaList(os.Getenv("TRUSTED_PROXIES")),

		DBHost:     dbHost,
		DBPort:     getEnvOrDefault("DB_PORT", "5432"),
		DBUser:     dbUser,
		DBPassword: dbPassword,
		DBName:     dbName,
		DBSSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),

		KeycloakJWKSURL:  keycloakJWKSURL,
		KeycloakIssuer:   keycloakIssuer,
		KeycloakAudience: os.Getenv("KEYCLOAK_AUDIENCE"),

		LogLevel:  getEnvOrDefault("LOG_LEVEL", "info"),
		LogFormat: getEnvOrDefault("LOG_FORMAT", defaultLogFormat),

		RateLimitRPS:   rateLimitRPS,
		RateLimitBurst: rateLimitBurst,
		RedisAddr:      os.Getenv("REDIS_ADDR"),
		RedisPassword:  os.Getenv("REDIS_PASSWORD"),

		OTELExporterEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTELServiceName:      getEnvOrDefault("OTEL_SERVICE_NAME", "projeto-selene-backend"),

		StorageDir: getEnvOrDefault("STORAGE_DIR", "./storage"),

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     getEnvOrDefault("SMTP_PORT", "587"),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
	}

	return cfg, nil
}

// DSN monta a connection string do Postgres a partir dos campos discretos
// de Config, em vez de exigir uma URL de conexão já pronta — mantém cada
// credencial explícita e individualmente validável.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// requireEnv lê uma variável de ambiente obrigatória, retornando um erro
// explícito e nomeado se ela estiver ausente ou vazia.
func requireEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("config: variável de ambiente obrigatória %q não definida", key)
	}
	return value, nil
}

// getEnvOrDefault lê uma variável de ambiente opcional, retornando def se
// ela não estiver definida.
func getEnvOrDefault(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}

// parseFloatOrDefault lê uma variável de ambiente opcional como float64,
// retornando def se ausente e um erro explícito se presente mas inválida
// (falha rápido em vez de silenciosamente cair no default por um typo).
func parseFloatOrDefault(key string, def float64) (float64, error) {
	value := os.Getenv(key)
	if value == "" {
		return def, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("config: variável de ambiente %q inválida (esperado número): %w", key, err)
	}
	return parsed, nil
}

// parseIntOrDefault lê uma variável de ambiente opcional como int, com a
// mesma política de erro explícito de parseFloatOrDefault.
func parseIntOrDefault(key string, def int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return def, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("config: variável de ambiente %q inválida (esperado inteiro): %w", key, err)
	}
	return parsed, nil
}

// splitCommaList divide uma lista separada por vírgulas (ex:
// "http://a.com, http://b.com") em um slice, removendo espaços em branco e
// entradas vazias. Uma string vazia resulta em slice vazio (nil), nunca
// em um slice com um elemento "" — evita autorizar uma origem/proxy vazio
// por acidente.
func splitCommaList(value string) []string {
	if value == "" {
		return nil
	}

	var out []string
	for _, item := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
