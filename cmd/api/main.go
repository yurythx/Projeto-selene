// Command api inicializa o servidor HTTP do Projeto Selene: carrega a
// configuração, conecta ao Postgres, roda migrations e seed, monta as
// camadas de repository/service/handler, registra as rotas e serve com
// desligamento gracioso.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/config"
	"projeto-selene/internal/database"
	"projeto-selene/internal/handler"
	"projeto-selene/internal/middleware"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/service"
)

// shutdownTimeout é quanto tempo o servidor espera requisições em
// andamento terminarem antes de forçar o encerramento no SIGTERM/SIGINT.
const shutdownTimeout = 15 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("falha ao carregar configuração: %v", err)
	}

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
		log.Fatalf("falha ao conectar ao banco de dados: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("falha ao executar migrations: %v", err)
	}

	if err := database.Seed(db); err != nil {
		log.Fatalf("falha ao semear dados de referência: %v", err)
	}

	// --- Repositories ---
	userRepo := repository.NewUserRepository(db)
	contratoRepo := repository.NewContratoRepository(db)
	etapaRepo := repository.NewKanbanEtapaRepository(db)
	tipoDocRepo := repository.NewTipoDocumentoRepository(db)
	processoRepo := repository.NewProcessoPagamentoRepository(db)
	docRepo := repository.NewDocumentoAnexoRepository(db)

	// --- Services ---
	userService := service.NewUserService(userRepo)
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

	relatorioService, err := service.NewRelatorioService(processoRepo, docRepo)
	if err != nil {
		log.Fatalf("falha ao inicializar serviço de relatório: %v", err)
	}

	// --- Middleware de autenticação ---
	// O contexto de fundo é usado apenas para o fetch inicial do JWKS na
	// construção do middleware — não está atrelado ao ciclo de vida de
	// nenhuma requisição.
	authMiddleware, err := middleware.NewAuthMiddleware(context.Background(), cfg.KeycloakJWKSURL, userService)
	if err != nil {
		log.Fatalf("falha ao inicializar middleware de autenticação: %v", err)
	}

	// --- Handlers ---
	healthHandler := handler.NewHealthHandler(db)
	contratoHandler := handler.NewContratoHandler(contratoService)
	processoHandler := handler.NewProcessoHandler(kanbanService)
	documentoHandler := handler.NewDocumentoHandler(documentoService)
	relatorioHandler := handler.NewRelatorioHandler(relatorioService)
	userHandler := handler.NewUserHandler(userService)
	kanbanRefHandler := handler.NewKanbanRefHandler(etapaRepo, tipoDocRepo)

	router := gin.Default()

	// TrustedProxies vazio (default) = nenhum proxy é confiável, o Gin
	// resolve o IP do cliente pela conexão direta — mais seguro que o
	// default do próprio Gin (confiar em todos os proxies).
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("falha ao configurar proxies confiáveis: %v", err)
	}

	router.Use(middleware.NewCORS(cfg.CORSAllowedOrigins))

	router.GET("/health", healthHandler.Check)

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
				"id":        user.ID,
				"nome":      user.Nome,
				"email":     user.Email,
				"is_fiscal": user.IsFiscal,
				"is_admin":  user.IsAdmin,
			})
		})

		// Leitura: qualquer usuário autenticado pode consultar.
		api.GET("/kanban/etapas", kanbanRefHandler.ListarEtapas)
		api.GET("/kanban/tipos-documento", kanbanRefHandler.ListarTiposDocumento)
		api.GET("/contratos", contratoHandler.Listar)
		api.GET("/contratos/:id", contratoHandler.Buscar)
		api.GET("/processos", processoHandler.Listar)
		api.GET("/processos/:id", processoHandler.Buscar)
		api.GET("/processos/:id/documentos", documentoHandler.Listar)
		api.GET("/processos/:id/relatorio", relatorioHandler.Gerar)

		// Escrita/movimentação do Kanban: restrita a fiscais habilitados.
		fiscal := api.Group("")
		fiscal.Use(middleware.RequireFiscal())
		{
			fiscal.POST("/contratos", contratoHandler.Criar)
			fiscal.POST("/processos", processoHandler.Criar)
			fiscal.POST("/processos/:id/avancar", processoHandler.Avancar)
			fiscal.POST("/processos/:id/concluir", processoHandler.Concluir)
			fiscal.POST("/processos/:id/documentos", documentoHandler.Upload)
		}

		// Administração de contas: restrita a administradores.
		admin := api.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			admin.GET("/users", userHandler.Listar)
			admin.GET("/users/:id", userHandler.Buscar)
			admin.PATCH("/users/:id", userHandler.AtualizarPermissoes)
		}
	}

	runWithGracefulShutdown(router, cfg.ServerPort)
}

// runWithGracefulShutdown sobe o servidor HTTP e bloqueia até receber
// SIGINT/SIGTERM, momento em que para de aceitar novas conexões e espera
// até shutdownTimeout para as requisições em andamento terminarem antes de
// encerrar o processo — evita cortar requisições no meio (ex: um upload
// de documento) durante um deploy/restart.
func runWithGracefulShutdown(router *gin.Engine, port string) {
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
		// ReadHeaderTimeout mitiga ataques Slowloris (cliente abre a
		// conexão e manda os headers bytes-a-bytes bem devagar,
		// segurando um worker do servidor indefinidamente).
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("falha ao iniciar servidor HTTP: %v", err)
		}
	}()
	log.Printf("servidor Selene ouvindo em %s", srv.Addr)

	<-ctx.Done()
	stop()
	log.Println("sinal de encerramento recebido, drenando requisições em andamento...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("encerramento forçado do servidor após timeout: %v", err)
	}
	log.Println("servidor encerrado com sucesso")
}
