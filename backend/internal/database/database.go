// Package database cuida da conexão com o Postgres e da execução das
// migrations versionadas (golang-migrate) do schema do Selene.
package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connect abre a conexão com o Postgres usando a DSN fornecida (ver
// config.Config.DSN), instrumentada com OpenTelemetry (spans de query
// aparecem como filhos do span do handler/service que a chamou, na mesma
// árvore de trace). Não realiza migrations — chame Migrate separadamente
// após validar a conexão.
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("database: falha ao conectar ao Postgres: %w", err)
	}

	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		return nil, fmt.Errorf("database: falha ao instrumentar GORM com OpenTelemetry: %w", err)
	}

	// Achado em auditoria de segurança: sem limites explícitos, o
	// database/sql do Go abre conexões ilimitadas com o Postgres sob
	// carga/pico de requisições — cada uma consome um slot de
	// max_connections do Postgres (compartilhado entre TODAS as réplicas
	// do backend); esgotar esse limite derruba o banco pra aplicação
	// inteira, não só pra quem gerou a carga (DoS). maxOpenConns/
	// maxIdleConns são folgados o bastante pro volume esperado (uso
	// interno de uma prefeitura, não tráfego público de internet) sem
	// deixar o teto literalmente aberto.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: obter conexão sql subjacente para configurar o pool: %w", err)
	}
	const (
		maxOpenConns    = 25
		maxIdleConns    = 5
		connMaxLifetime = 30 * time.Minute
		connMaxIdleTime = 5 * time.Minute
	)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	return db, nil
}

// Migrate aplica todas as migrations versionadas pendentes, embutidas no
// binário a partir de internal/database/migrations/ (ver 000001_initial_
// schema.up.sql). Este é o único caminho de evolução de schema em
// produção: o AutoMigrate do GORM foi deliberadamente descartado depois do
// protótipo inicial porque não versiona mudanças nem sabe reverter — SQL
// explícito e numerado é auditável, revisável em code review e reversível
// (cada *.up.sql tem um *.down.sql correspondente).
//
// As tags GORM nos structs de internal/models continuam servindo para
// leitura/escrita via ORM (nomes de coluna, tipos em Go), mas deixam de
// ser a fonte de verdade do schema — essa fonte agora são os arquivos
// desta pasta.
func Migrate(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: obter conexão sql subjacente para migrations: %w", err)
	}

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{})
	if err != nil {
		return fmt.Errorf("database: inicializar driver de migrations: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("database: carregar arquivos de migration embutidos: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("database: inicializar migrate: %w", err)
	}
	// Deliberadamente NÃO chamamos m.Close() aqui: o driver postgres do
	// golang-migrate fecha o *sql.DB subjacente em Close() mesmo quando
	// ele veio de WithInstance (não é dono da conexão, mas fecha do mesmo
	// jeito — comportamento documentado da lib) — chamar Close()
	// derrubaria a conexão que o GORM continua usando depois. O advisory
	// lock do Postgres usado internamente para serializar migrations é
	// adquirido e liberado dentro do próprio m.Up(), então isso não vaza
	// mesmo sem Close().

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("database: falha ao executar migrations: %w", err)
	}

	return nil
}

// Ping verifica se a conexão com o Postgres está de fato respondendo —
// usado pelo endpoint /health, que precisa refletir a saúde real da
// dependência principal da aplicação, não só "o processo Go está de pé".
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: obter conexão sql subjacente: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database: ping ao Postgres falhou: %w", err)
	}

	return nil
}
