package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/testutil"
)

// TestContratoRepository_List_Filtro é o teste de integração (Postgres
// real) do FiltroContrato — garante que o ILIKE de Busca e as cláusulas
// de TipoObjeto/Situacao casam com o que a query real do Postgres produz,
// algo que um dublê em memória não pega (ver fakes.go, que espelha a
// mesma semântica mas não exercita SQL de verdade).
func TestContratoRepository_List_Filtro(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	contratoRepo := repository.NewContratoRepository(db)

	keycloakID := "fiscal-filtro-contrato"
	fiscal := &models.User{KeycloakID: &keycloakID, Nome: "Fiscal Filtro", Email: "fiscal.filtro@teste.local", IsFiscal: true}
	if err := userRepo.Create(ctx, fiscal); err != nil {
		t.Fatalf("falha ao criar fiscal: %v", err)
	}

	sufixo := uuid.New().String()[:8]
	criar := func(numero, contratadaNome, cnpj string, tipoObjeto models.TipoObjeto, ativo bool) {
		t.Helper()
		contrato := &models.Contrato{
			NumeroContrato: numero + "/" + sufixo,
			DataAssinatura: time.Now(),
			ContratadaNome: contratadaNome,
			ContratadaCNPJ: cnpj,
			FiscalID:       fiscal.ID,
			TipoObjeto:     tipoObjeto,
			Ativo:          ativo,
		}
		if err := contratoRepo.Create(ctx, contrato); err != nil {
			t.Fatalf("falha ao criar contrato %q: %v", numero, err)
		}
		if !ativo {
			// Create sempre grava Ativo=true por causa do default do GORM
			// (ver o comentário ARMADILHA em models.Contrato) — um Update
			// explícito é a forma documentada de encerrar.
			contrato.Ativo = false
			if err := contratoRepo.Update(ctx, contrato); err != nil {
				t.Fatalf("falha ao encerrar contrato %q: %v", numero, err)
			}
		}
	}

	criar("100", "Alfa Construções Ltda", "11.111.111/0001-11", models.TipoObjetoServico, true)
	criar("200", "Beta Suprimentos SA", "22.222.222/0001-22", models.TipoObjetoConsumo, true)
	criar("300", "Gama Equipamentos ME", "33.333.333/0001-33", models.TipoObjetoPermanente, false)

	t.Run("busca casa número do contrato", func(t *testing.T) {
		resultado, err := contratoRepo.List(ctx, repository.Pagina{}, repository.FiltroContrato{Busca: "100/" + sufixo})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.Total != 1 || resultado.Dados[0].ContratadaNome != "Alfa Construções Ltda" {
			t.Fatalf("esperava só o contrato 100, veio %+v", resultado.Dados)
		}
	})

	t.Run("busca casa nome da contratada, case-insensitive", func(t *testing.T) {
		resultado, err := contratoRepo.List(ctx, repository.Pagina{}, repository.FiltroContrato{Busca: "beta suprimentos"})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.Total != 1 || resultado.Dados[0].NumeroContrato != "200/"+sufixo {
			t.Fatalf("esperava só o contrato 200, veio %+v", resultado.Dados)
		}
	})

	t.Run("busca casa CNPJ", func(t *testing.T) {
		resultado, err := contratoRepo.List(ctx, repository.Pagina{}, repository.FiltroContrato{Busca: "33.333.333"})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.Total != 1 || resultado.Dados[0].NumeroContrato != "300/"+sufixo {
			t.Fatalf("esperava só o contrato 300, veio %+v", resultado.Dados)
		}
	})

	t.Run("filtro por tipo_objeto", func(t *testing.T) {
		resultado, err := contratoRepo.List(ctx, repository.Pagina{}, repository.FiltroContrato{Busca: sufixo, TipoObjeto: models.TipoObjetoServico})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.Total != 1 || resultado.Dados[0].NumeroContrato != "100/"+sufixo {
			t.Fatalf("esperava só o contrato SERVICO (100), veio %+v", resultado.Dados)
		}
	})

	t.Run("tipo_objeto inválido é ignorado (sem filtro)", func(t *testing.T) {
		resultado, err := contratoRepo.List(ctx, repository.Pagina{}, repository.FiltroContrato{Busca: sufixo, TipoObjeto: "LIXO"})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.Total != 3 {
			t.Fatalf("esperava os 3 contratos (filtro inválido ignorado), veio %d", resultado.Total)
		}
	})

	t.Run("filtro por situacao=encerrado", func(t *testing.T) {
		resultado, err := contratoRepo.List(ctx, repository.Pagina{}, repository.FiltroContrato{Busca: sufixo, Situacao: "encerrado"})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.Total != 1 || resultado.Dados[0].NumeroContrato != "300/"+sufixo {
			t.Fatalf("esperava só o contrato encerrado (300), veio %+v", resultado.Dados)
		}
	})

	t.Run("filtro por situacao=ativo", func(t *testing.T) {
		resultado, err := contratoRepo.List(ctx, repository.Pagina{}, repository.FiltroContrato{Busca: sufixo, Situacao: "ativo"})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.Total != 2 {
			t.Fatalf("esperava 2 contratos ativos, veio %d", resultado.Total)
		}
	})

	t.Run("busca e filtro combinados sem resultado", func(t *testing.T) {
		resultado, err := contratoRepo.List(ctx, repository.Pagina{}, repository.FiltroContrato{Busca: "Alfa", TipoObjeto: models.TipoObjetoConsumo})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.Total != 0 {
			t.Fatalf("Alfa é SERVICO, não deveria casar com filtro CONSUMO — veio %d", resultado.Total)
		}
	})

	t.Run("sem filtro nenhum, contagem inclui os 3 pelo sufixo isolado no nome do teste", func(t *testing.T) {
		resultado, err := contratoRepo.List(ctx, repository.Pagina{}, repository.FiltroContrato{Busca: sufixo})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.Total != 3 {
			t.Fatalf("esperava os 3 contratos deste teste, veio %d", resultado.Total)
		}
	})
}
