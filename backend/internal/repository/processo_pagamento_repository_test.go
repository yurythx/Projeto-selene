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

// TestProcessoPagamentoRepository_Update_NaoReverteEtapaViaAssociacaoCarregada
// é um teste de regressão para um bug real encontrado em produção manual:
// GORM, por padrão, resalva associações BelongsTo (Contrato, EtapaAtual)
// ANTES de salvar o registro principal — e se o struct foi carregado via
// Preload (como FindByID sempre faz) e depois teve só o campo de FK
// mutado diretamente (ex: EtapaAtualID), o Save() reescrevia a FK de
// volta ao ID da associação antiga, desfazendo a mudança silenciosamente.
// O card do Kanban nunca avançava de etapa por causa disso.
//
// A correção foi usar .Omit(clause.Associations) no repository — este
// teste garante que essa regressão não volte.
func TestProcessoPagamentoRepository_Update_NaoReverteEtapaViaAssociacaoCarregada(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	contratoRepo := repository.NewContratoRepository(db)
	processoRepo := repository.NewProcessoPagamentoRepository(db)

	fiscal := &models.User{KeycloakID: "fiscal-regressao", Nome: "Fiscal Teste", Email: "fiscal@teste.local", IsFiscal: true}
	if err := userRepo.Create(ctx, fiscal); err != nil {
		t.Fatalf("falha ao criar fiscal: %v", err)
	}

	contrato := &models.Contrato{
		NumeroContrato: "REGRESSAO/" + uuid.New().String()[:8],
		DataAssinatura: time.Now(),
		ContratadaNome: "Empresa Regressão",
		ContratadaCNPJ: "00.000.000/0001-00",
		FiscalID:       fiscal.ID,
		TipoObjeto:     models.TipoObjetoConsumo,
		Ativo:          true,
	}
	if err := contratoRepo.Create(ctx, contrato); err != nil {
		t.Fatalf("falha ao criar contrato: %v", err)
	}

	processo := &models.ProcessoPagamento{
		ContratoID:    contrato.ID,
		MesReferencia: "01/2026",
		EtapaAtualID:  1,
		Status:        models.StatusProcessoAtivo,
	}
	if err := processoRepo.Create(ctx, processo); err != nil {
		t.Fatalf("falha ao criar processo: %v", err)
	}

	// Carrega com Preload (Contrato, Contrato.Fiscal, EtapaAtual) — igual
	// ao que KanbanService.AvancarEtapa faz de verdade.
	carregado, err := processoRepo.FindByID(ctx, processo.ID)
	if err != nil {
		t.Fatalf("falha ao carregar processo: %v", err)
	}
	if carregado.EtapaAtual == nil || carregado.EtapaAtual.ID != 1 {
		t.Fatalf("esperava EtapaAtual pré-carregado com ID 1, veio %+v", carregado.EtapaAtual)
	}

	// Muta só o campo de FK, exatamente como o service faz.
	carregado.EtapaAtualID = 2
	if err := processoRepo.Update(ctx, carregado); err != nil {
		t.Fatalf("falha ao atualizar processo: %v", err)
	}

	// Recarrega em uma consulta NOVA (não reaproveita o struct em
	// memória) para confirmar que o valor persistiu de verdade no banco.
	recarregado, err := processoRepo.FindByID(ctx, processo.ID)
	if err != nil {
		t.Fatalf("falha ao recarregar processo: %v", err)
	}
	if recarregado.EtapaAtualID != 2 {
		t.Fatalf("regressão: EtapaAtualID deveria ser 2 após Update, veio %d", recarregado.EtapaAtualID)
	}
}
