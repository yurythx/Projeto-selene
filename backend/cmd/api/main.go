// Command api inicializa o servidor HTTP do Projeto Selene: carrega a
// configuração, conecta ao Postgres, roda migrations e seed, monta as
// camadas de repository/service/handler, registra as rotas e serve com
// desligamento gracioso.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"projeto-selene/internal/config"
	"projeto-selene/internal/database"
	"projeto-selene/internal/handler"
	"projeto-selene/internal/localauth"
	"projeto-selene/internal/logging"
	"projeto-selene/internal/middleware"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/service"
	"projeto-selene/internal/tracing"
)

// shutdownTimeout é quanto tempo o servidor espera requisições em
// andamento terminarem antes de forçar o encerramento no SIGTERM/SIGINT.
const shutdownTimeout = 15 * time.Second

func main() {
	// config.Load ainda não rodou, então usamos o "log" padrão da stdlib
	// só para este primeiro erro possível — depois daqui em diante, todo
	// log é estruturado via logger (slog).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("falha ao carregar configuração: %v", err)
	}

	logger, err := logging.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		log.Fatalf("falha ao inicializar logger: %v", err)
	}

	fatal := func(msg string, err error) {
		logger.Error(msg, "erro", err)
		os.Exit(1)
	}

	shutdownTracing, err := tracing.New(context.Background(), cfg.OTELServiceName, cfg.OTELExporterEndpoint)
	if err != nil {
		fatal("falha ao inicializar tracing", err)
	}
	defer func() {
		if err := shutdownTracing(context.Background()); err != nil {
			logger.Error("falha ao encerrar tracer provider", "erro", err)
		}
	}()

	// gin.ReleaseMode é obrigatório em produção: modo debug loga rotas e
	// erros com detalhe e roda mais devagar. AppEnv precisa ser
	// explicitamente "production" — o default é "development" (fail-safe:
	// esquecer de setar a variável nunca vaza comportamento de release
	// indevidamente, só o contrário, que é inofensivo).
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg.DSN())
	if err != nil {
		fatal("falha ao conectar ao banco de dados", err)
	}

	if err := database.Migrate(db); err != nil {
		fatal("falha ao executar migrations", err)
	}

	if err := database.Seed(db); err != nil {
		fatal("falha ao semear dados de referência", err)
	}

	// --- Repositories ---
	userRepo := repository.NewUserRepository(db)
	contratoRepo := repository.NewContratoRepository(db)
	etapaRepo := repository.NewKanbanEtapaRepository(db)
	tipoDocRepo := repository.NewTipoDocumentoRepository(db)
	processoRepo := repository.NewProcessoPagamentoRepository(db)
	docRepo := repository.NewDocumentoAnexoRepository(db)
	logRepo := repository.NewKanbanLogRepository(db)
	docEmitidoRepo := repository.NewDocumentoEmitidoRepository(db)
	vistoriaRepo := repository.NewVistoriaRepository(db)
	fotoVistoriaRepo := repository.NewFotoVistoriaRepository(db)

	// SGF-Rondonópolis (Fase 2 do plano) — ver
	// .claude/plans/projeto-selene-rippling-kite.md.
	portariaDesignacaoRepo := repository.NewPortariaDesignacaoRepository(db)
	empenhoRepo := repository.NewEmpenhoRepository(db)
	movimentacaoEmpenhoRepo := repository.NewMovimentacaoEmpenhoRepository(db)
	ocorrenciaRepo := repository.NewOcorrenciaRepository(db)
	modeloDocumentoRepo := repository.NewModeloDocumentoRepository(db)
	modeloDocumentoVersaoRepo := repository.NewModeloDocumentoVersaoRepository(db)
	keycloakConfigRepo := repository.NewKeycloakConfigRepository(db)

	// Chave RSA do login local (usuário/senha) — gerada uma vez por
	// processo, ver a "LIMITAÇÃO CONHECIDA" documentada em
	// internal/localauth.KeyPair sobre sessões locais não sobreviverem a
	// um restart do backend.
	localKeys, err := localauth.NewKeyPair()
	if err != nil {
		fatal("falha ao gerar par de chaves do login local", err)
	}

	// --- Services ---
	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userRepo, localKeys)
	contratoService := service.NewContratoService(contratoRepo, userRepo)
	documentoService := service.NewDocumentoService(docRepo, tipoDocRepo, processoRepo, cfg.StorageDir)

	notifier := service.NewNotifier(service.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})
	kanbanService := service.NewKanbanService(db, processoRepo, contratoRepo, docRepo, notifier)

	modeloDocumentoService := service.NewModeloDocumentoService(db, modeloDocumentoRepo, modeloDocumentoVersaoRepo, cfg.StorageDir)

	relatorioService, err := service.NewRelatorioService(processoRepo, docRepo, ocorrenciaRepo, empenhoRepo, movimentacaoEmpenhoRepo, modeloDocumentoRepo)
	if err != nil {
		fatal("falha ao inicializar serviço de relatório", err)
	}

	radarService := service.NewRadarService(contratoRepo, processoRepo, docRepo, logRepo)

	geradorDocumentosService := service.NewGeradorDocumentosService(contratoRepo, processoRepo, docEmitidoRepo, modeloDocumentoRepo, cfg.PublicURL)

	vistoriaService := service.NewVistoriaService(vistoriaRepo, fotoVistoriaRepo, processoRepo, cfg.StorageDir)

	fornecedorService := service.NewFornecedorService(contratoRepo, processoRepo, logRepo, docEmitidoRepo)

	// SGF-Rondonópolis (Fase 2 do plano).
	designacaoService := service.NewDesignacaoService(portariaDesignacaoRepo, contratoRepo, userRepo)
	empenhoService := service.NewEmpenhoService(empenhoRepo, movimentacaoEmpenhoRepo, contratoRepo)
	ocorrenciaService := service.NewOcorrenciaService(ocorrenciaRepo, processoRepo)
	fiscalizacaoService := service.NewFiscalizacaoService(docRepo, ocorrenciaRepo)

	// --- Middleware de autenticação ---
	// fallbackAuthConfig são as variáveis de ambiente de boot — servem de
	// configuração INICIAL (e continuam sendo o retrato mostrado em
	// Configurações → Keycloak/SSO enquanto nenhum admin salvar nada por
	// lá, ver KeycloakConfigService.Buscar) até serem substituídas por
	// uma linha salva no banco, se existir.
	fallbackAuthConfig := middleware.AuthConfig{
		JWKSURL:  cfg.KeycloakJWKSURL,
		Issuer:   cfg.KeycloakIssuer,
		Audience: cfg.KeycloakAudience,
	}

	authConfigInicial := fallbackAuthConfig
	if configSalva, err := keycloakConfigRepo.Buscar(context.Background()); err == nil {
		audience := ""
		if configSalva.Audience != nil {
			audience = *configSalva.Audience
		}
		authConfigInicial = middleware.AuthConfig{
			JWKSURL:  service.DeriveJWKSURL(configSalva.IssuerURL),
			Issuer:   configSalva.IssuerURL,
			Audience: audience,
		}
		logger.Info("usando configuração de Keycloak salva no banco (Configurações → Keycloak/SSO), não as variáveis de ambiente")
	} else if !errors.Is(err, repository.ErrKeycloakConfigNotFound) {
		fatal("falha ao carregar configuração de keycloak salva", err)
	}

	// O contexto de fundo é usado apenas para o fetch inicial do JWKS na
	// construção do middleware — não está atrelado ao ciclo de vida de
	// nenhuma requisição. authState permite trocar essa configuração em
	// runtime depois (ver KeycloakConfigService.Salvar), sem reiniciar o
	// processo.
	authMiddleware, authState, err := middleware.NewAuthMiddleware(context.Background(), authConfigInicial, userService, localKeys)
	if err != nil {
		fatal("falha ao inicializar middleware de autenticação", err)
	}

	keycloakConfigService := service.NewKeycloakConfigService(keycloakConfigRepo, authState, fallbackAuthConfig)

	// Rate limiter: Redis compartilhado se REDIS_ADDR estiver configurado
	// (vale entre réplicas), senão cai para o limiter em memória (mesmo
	// princípio de graceful degradation do resto da aplicação).
	var rateLimiter middleware.RateLimiter
	if cfg.RedisAddr != "" {
		redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			fatal("falha ao conectar ao Redis (REDIS_ADDR configurado)", err)
		}
		rateLimiter = middleware.NewRedisRateLimiter(redisClient, cfg.RateLimitRPS, cfg.RateLimitBurst)
		logger.Info("rate limiter usando Redis compartilhado", "endereco", cfg.RedisAddr)
	} else {
		rateLimiter = middleware.NewInMemoryRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
		logger.Warn("REDIS_ADDR não configurado — rate limiter em memória, por instância (não vale entre réplicas)")
	}

	// --- Handlers ---
	healthHandler := handler.NewHealthHandler(db)
	contratoHandler := handler.NewContratoHandler(contratoService)
	processoHandler := handler.NewProcessoHandler(kanbanService, fiscalizacaoService)
	documentoHandler := handler.NewDocumentoHandler(documentoService)
	relatorioHandler := handler.NewRelatorioHandler(relatorioService)
	userHandler := handler.NewUserHandler(userService, authService)
	authHandler := handler.NewAuthHandler(authService)
	kanbanRefHandler := handler.NewKanbanRefHandler(etapaRepo, tipoDocRepo)
	radarHandler := handler.NewRadarHandler(radarService)
	geradorDocumentosHandler := handler.NewGeradorDocumentosHandler(geradorDocumentosService)
	vistoriaHandler := handler.NewVistoriaHandler(vistoriaService)
	fornecedorHandler := handler.NewFornecedorHandler(fornecedorService)
	designacaoHandler := handler.NewDesignacaoHandler(designacaoService)
	empenhoHandler := handler.NewEmpenhoHandler(empenhoService)
	ocorrenciaHandler := handler.NewOcorrenciaHandler(ocorrenciaService)
	modeloDocumentoHandler := handler.NewModeloDocumentoHandler(modeloDocumentoService)
	keycloakConfigHandler := handler.NewKeycloakConfigHandler(keycloakConfigService, cfg.InternalAPISecret)

	// gin.New() em vez de gin.Default(): montamos a cadeia de middlewares
	// explicitamente (Recovery, RequestID, log estruturado, métricas,
	// CORS) em vez de aceitar o Logger()/Recovery() padrão do Gin, que
	// loga texto simples em vez de eventos estruturados.
	router := gin.New()
	router.Use(
		gin.Recovery(),
		middleware.RequestID(),
		otelgin.Middleware(cfg.OTELServiceName),
		middleware.StructuredLogger(),
		middleware.Metrics(),
		middleware.NewCORS(cfg.CORSAllowedOrigins),
		// Achado em auditoria de segurança: sem isso, qualquer endpoint
		// JSON aceitava um corpo de tamanho arbitrário (só os dois
		// endpoints de upload multipart tinham limite próprio) — DoS por
		// exaustão de memória com um corpo de vários GB. Ver o comentário
		// em MaxBodySize sobre por que multipart é pulado aqui.
		middleware.MaxBodySize(),
	)

	// TrustedProxies vazio (default) = nenhum proxy é confiável, o Gin
	// resolve o IP do cliente pela conexão direta — mais seguro que o
	// default do próprio Gin (confiar em todos os proxies).
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		fatal("falha ao configurar proxies confiáveis", err)
	}

	router.GET("/health", healthHandler.Check)
	// /metrics é exposto sem autenticação, por convenção do ecossistema
	// Prometheus (o scraper não passa Bearer token) — em produção, o
	// acesso a essa rota deve ser restrito por rede (VPC/firewall), não
	// pela aplicação.
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Verificação de autenticidade de documentos emitidos (Módulo 2 do
	// roadmap) — de propósito FORA do grupo /api/v1 autenticado abaixo:
	// quem escaneia o QR code de um Atesto impresso não tem login no
	// Selene. O backend não é publicamente exposto (só o frontend/BFF, ver
	// DEPLOY.md), então isso não abre a API real ao público — só essa
	// única rota de consulta, sempre chamada server-side pelo BFF.
	router.GET("/api/v1/verificar/:codigo", geradorDocumentosHandler.Verificar)

	// Consultado pelo frontend Next.js pra montar o provider Keycloak do
	// NextAuth em runtime — de propósito FORA do grupo /api/v1
	// autenticado abaixo (não há usuário logado ainda nesse momento) e
	// gated por um segredo compartilhado em vez de JWT, ver o comentário
	// em KeycloakConfigHandler.BuscarInterno.
	router.GET("/internal/keycloak-config", keycloakConfigHandler.BuscarInterno)

	// Login tradicional (usuário/senha) — também PÚBLICO de propósito
	// (ainda não há sessão/token pra exigir), mas sujeito a rate limit por
	// IP (rateLimiter.Middleware() cai pra ClientIP() na ausência de
	// usuário autenticado no contexto — ver o comentário lá) como defesa
	// contra força bruta de senha.
	router.POST("/api/v1/auth/login", rateLimiter.Middleware(), authHandler.Login)

	api := router.Group("/api/v1")
	api.Use(authMiddleware)
	{
		// Rota de verificação: confirma que o token foi validado e o
		// usuário foi resolvido/provisionado corretamente.
		api.GET("/me", func(c *gin.Context) {
			user, ok := middleware.UserFromContext(c)
			if !ok {
				c.JSON(500, gin.H{"error": "usuário autenticado não encontrado no contexto"})
				return
			}

			c.JSON(200, gin.H{
				"id":                   user.ID,
				"nome":                 user.Nome,
				"email":                user.Email,
				"is_fiscal":            user.IsFiscal,
				"is_admin":             user.IsAdmin,
				"must_change_password": user.MustChangePassword,
			})
		})

		// Troca de senha — qualquer usuário autenticado (local ou
		// Keycloak; contas Keycloak recebem um erro claro, a senha delas é
		// gerenciada pelo Keycloak). Sem RequireFiscal/RequireAdmin de
		// propósito: é uma ação sobre a PRÓPRIA conta, não uma permissão
		// de negócio.
		api.POST("/auth/trocar-senha", authHandler.TrocarSenha)

		// Leitura: qualquer usuário autenticado pode consultar.
		api.GET("/kanban/etapas", kanbanRefHandler.ListarEtapas)
		api.GET("/kanban/tipos-documento", kanbanRefHandler.ListarTiposDocumento)
		api.GET("/radar", radarHandler.Listar)
		api.GET("/contratos", contratoHandler.Listar)
		api.GET("/contratos/:id", contratoHandler.Buscar)
		api.GET("/processos", processoHandler.Listar)
		api.GET("/processos/:id", processoHandler.Buscar)
		api.GET("/processos/:id/documentos", documentoHandler.Listar)
		api.GET("/processos/:id/documentos/:docId/download", documentoHandler.Baixar)
		api.GET("/processos/:id/relatorio", relatorioHandler.Gerar)
		api.GET("/processos/:id/vistorias", vistoriaHandler.ListarPorProcesso)
		api.GET("/vistorias/:id/relatorio", vistoriaHandler.GerarRelatorioCampo)
		api.GET("/fornecedores", fornecedorHandler.Listar)
		api.GET("/fornecedores/:cnpj", fornecedorHandler.Buscar)

		// SGF-Rondonópolis (Fase 2 do plano): leitura.
		api.GET("/contratos/:id/designacoes", designacaoHandler.Listar)
		api.GET("/contratos/:id/empenhos", empenhoHandler.Listar)
		api.GET("/empenhos/:id", empenhoHandler.Buscar)
		api.GET("/processos/:id/ocorrencias", ocorrenciaHandler.Listar)
		// Projeção mínima de usuários (ID/Nome/Email), pra popular o
		// seletor de servidor de "Nova designação" — ver o comentário em
		// UserHandler.ListarServidores sobre por que não é admin-only.
		api.GET("/servidores", userHandler.ListarServidores)

		// Escrita/movimentação do Kanban: restrita a fiscais habilitados
		// e sujeita a rate limit (por usuário autenticado).
		fiscal := api.Group("")
		fiscal.Use(middleware.RequireFiscal(), rateLimiter.Middleware())
		{
			fiscal.POST("/contratos", contratoHandler.Criar)
			fiscal.PATCH("/contratos/:id", contratoHandler.Atualizar)
			fiscal.POST("/contratos/:id/encerrar", contratoHandler.Encerrar)
			fiscal.POST("/processos", processoHandler.Criar)
			fiscal.POST("/processos/:id/avancar", processoHandler.Avancar)
			fiscal.POST("/processos/:id/concluir", processoHandler.Concluir)
			fiscal.POST("/processos/:id/documentos", documentoHandler.Upload)
			fiscal.DELETE("/processos/:id/documentos/:docId", documentoHandler.Excluir)

			// Módulo 2 do roadmap: geração dos 3 documentos legais.
			fiscal.POST("/contratos/:id/notificacao", geradorDocumentosHandler.GerarNotificacao)
			fiscal.POST("/processos/:id/atesto", geradorDocumentosHandler.GerarAtesto)
			fiscal.POST("/contratos/:id/minuta-aditivo", geradorDocumentosHandler.GerarMinutaAditivo)

			// Módulo 3 do roadmap: vistorias de campo com registro
			// fotográfico e geolocalização.
			fiscal.POST("/processos/:id/vistorias", vistoriaHandler.Registrar)
			fiscal.POST("/vistorias/:id/fotos", vistoriaHandler.AnexarFoto)

			// SGF-Rondonópolis (Fase 2 do plano): escrita.
			fiscal.POST("/contratos/:id/designacoes", designacaoHandler.Designar)
			fiscal.POST("/contratos/:id/empenhos", empenhoHandler.Criar)
			fiscal.POST("/empenhos/:id/movimentacoes", empenhoHandler.RegistrarMovimentacao)
			fiscal.POST("/processos/:id/ocorrencias", ocorrenciaHandler.Registrar)
			fiscal.POST("/ocorrencias/:id/notificar", ocorrenciaHandler.Notificar)
			fiscal.POST("/ocorrencias/:id/tratar", ocorrenciaHandler.IniciarTratamento)
			fiscal.POST("/ocorrencias/:id/regularizar", ocorrenciaHandler.Regularizar)
		}

		// Administração de contas: restrita a administradores.
		admin := api.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			admin.GET("/users", userHandler.Listar)
			admin.GET("/users/:id", userHandler.Buscar)
			admin.PATCH("/users/:id", userHandler.AtualizarPermissoes)
			// Login tradicional: só um admin cria contas locais (sem
			// autocadastro público) — ver AuthService.CriarLocal.
			admin.POST("/users/local", userHandler.CriarLocal)

			// Configurações — Modelos de Documentos (ver
			// internal/service/modelo_documento_render.go pra como os 4
			// gatilhos consomem o modelo ativo na geração real).
			admin.GET("/modelos-documento", modeloDocumentoHandler.Listar)
			admin.POST("/modelos-documento", modeloDocumentoHandler.Criar)
			admin.GET("/modelos-documento/:id", modeloDocumentoHandler.Buscar)
			admin.PATCH("/modelos-documento/:id", modeloDocumentoHandler.Atualizar)
			admin.POST("/modelos-documento/:id/versoes", modeloDocumentoHandler.NovaVersao)
			admin.GET("/modelos-documento/:id/download", modeloDocumentoHandler.Baixar)
			admin.GET("/modelos-documento/:id/versoes/:versaoId/download", modeloDocumentoHandler.BaixarVersao)

			// Configurações — Keycloak/SSO: pedido explícito do usuário
			// pra poder ver/mudar a configuração ativa sem depender de
			// editar variáveis de ambiente e reiniciar os containers.
			admin.GET("/config/keycloak", keycloakConfigHandler.Buscar)
			admin.PUT("/config/keycloak", keycloakConfigHandler.Atualizar)
		}
	}

	runWithGracefulShutdown(router, cfg.ServerPort, logger)
}

// runWithGracefulShutdown sobe o servidor HTTP e bloqueia até receber
// SIGINT/SIGTERM, momento em que para de aceitar novas conexões e espera
// até shutdownTimeout para as requisições em andamento terminarem antes de
// encerrar o processo — evita cortar requisições no meio (ex: um upload
// de documento) durante um deploy/restart.
func runWithGracefulShutdown(router *gin.Engine, port string, logger *slog.Logger) {
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
		// ReadHeaderTimeout mitiga ataques Slowloris (cliente abre a
		// conexão e manda os headers bytes-a-bytes bem devagar,
		// segurando um worker do servidor indefinidamente).
		ReadHeaderTimeout: 10 * time.Second,
		// ReadTimeout cobre o corpo inteiro da requisição (não só os
		// headers) — sem isso, um cliente lento enviando o body aos
		// poucos ainda segura a conexão indefinidamente. 30s é folgado o
		// bastante pro maior corpo esperado (upload de documento, até
		// 20MB — ver maxUploadBytes em documento_handler.go) mesmo numa
		// rede ruim.
		ReadTimeout: 30 * time.Second,
		// WriteTimeout cobre da leitura do header até o fim do envio da
		// resposta — segue a mesma folga do ReadTimeout.
		WriteTimeout: 30 * time.Second,
		// IdleTimeout limita quanto tempo uma conexão keep-alive fica
		// aberta sem atividade.
		IdleTimeout: 120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("falha ao iniciar servidor HTTP", "erro", err)
			os.Exit(1)
		}
	}()
	logger.Info("servidor Selene ouvindo", "endereco", srv.Addr)

	<-ctx.Done()
	stop()
	logger.Info("sinal de encerramento recebido, drenando requisições em andamento...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("encerramento forçado do servidor após timeout", "erro", err)
		os.Exit(1)
	}
	logger.Info("servidor encerrado com sucesso")
}
