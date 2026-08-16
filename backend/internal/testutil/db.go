// Package testutil fornece infraestrutura compartilhada para os testes de
// integração de repository/service — não é usado pelo binário da
// aplicação, só por arquivos _test.go de outros pacotes.
package testutil

import (
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"projeto-selene/internal/database"
)

// defaultTestDSN é usada quando a variável de ambiente TEST_DATABASE_URL
// não está definida — aponta para o Postgres descartável de
// desenvolvimento (ver README, seção de testes).
const defaultTestDSN = "host=localhost port=55432 user=selene password=selene dbname=projeto_selene sslmode=disable"

// tabelasTransacionais são truncadas antes de cada teste de integração
// para garantir estado limpo. As tabelas de referência (kanban_etapas,
// tipos_documento) NÃO são truncadas — são semeadas uma vez e reutilizadas
// entre testes, como na aplicação real.
var tabelasTransacionais = []string{"kanban_logs", "documentos_anexos", "modelo_documento_versoes", "modelos_documento", "processos_pagamento", "contratos", "users"}

// OpenTestDB conecta a um Postgres real, roda migrations + seed, e
// devolve um *gorm.DB pronto para uso em testes de integração de
// repository/service.
//
// Se o banco não estiver acessível, o teste é PULADO (t.Skip), não
// falhado — testes de integração exigem infraestrutura externa (Postgres)
// que nem todo ambiente local tem disponível; o CI sempre sobe um serviço
// Postgres (ver .github/workflows/ci.yml), então lá eles sempre rodam de
// verdade.
func OpenTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	if testing.Short() {
		t.Skip("testutil: -short ativo — pulando teste de integração (requer Postgres)")
	}

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("testutil: Postgres de teste indisponível (%v) — pulando teste de integração. Defina TEST_DATABASE_URL ou suba o Postgres descrito no README para rodá-lo.", err)
	}

	sqlDB, err := db.DB()
	if err != nil || sqlDB.Ping() != nil {
		t.Skip("testutil: Postgres de teste inacessível — pulando teste de integração.")
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("testutil: falha ao migrar banco de teste: %v", err)
	}
	if err := database.Seed(db); err != nil {
		t.Fatalf("testutil: falha ao semear banco de teste: %v", err)
	}

	truncarTabelasTransacionais(t, db)
	t.Cleanup(func() {
		truncarTabelasTransacionais(t, db)
		// Fecha a pool de conexões deste teste explicitamente — sem isso,
		// cada teste de integração abre uma pool nova e nenhuma é fechada
		// até o fim do processo, acumulando conexões ociosas no Postgres
		// ao longo da suíte inteira.
		if err := sqlDB.Close(); err != nil {
			t.Logf("testutil: aviso — falha ao fechar conexão de teste: %v", err)
		}
	})

	return db
}

// truncarTabelasTransacionais limpa as tabelas transacionais entre testes.
// Deadlocks do Postgres (SQLSTATE 40P01) são transitórios por definição —
// o próprio Postgres já abortou uma das transações em conflito para
// quebrar o ciclo, então tentar de novo tende a funcionar (é a
// recomendação oficial do Postgres para esse erro, não um workaround
// nosso). Sob testes de integração concorrentes/paralelos isso pode
// acontecer ocasionalmente; sem retry o teste falharia de forma
// instável (flaky) por um motivo que não é um bug de verdade.
func truncarTabelasTransacionais(t *testing.T, db *gorm.DB) {
	t.Helper()

	const maxTentativas = 3

	for _, tabela := range tabelasTransacionais {
		var err error
		for tentativa := 1; tentativa <= maxTentativas; tentativa++ {
			err = db.Exec("TRUNCATE TABLE " + tabela + " RESTART IDENTITY CASCADE").Error
			if err == nil {
				break
			}
			if !strings.Contains(err.Error(), "40P01") {
				break // erro não é deadlock — não adianta tentar de novo
			}
			time.Sleep(time.Duration(tentativa) * 50 * time.Millisecond)
		}
		if err != nil {
			t.Fatalf("testutil: falha ao truncar tabela %q após %d tentativa(s): %v", tabela, maxTentativas, err)
		}
	}
}
