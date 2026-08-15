package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
	"projeto-selene/internal/testutil"
)

func novoContratoDeTeste(fiscalID uuid.UUID) *models.Contrato {
	return &models.Contrato{
		ID:             uuid.New(),
		NumeroContrato: "001/2026",
		DataAssinatura: time.Now(),
		ContratadaNome: "Empresa Teste",
		ContratadaCNPJ: "00.000.000/0001-00",
		FiscalID:       fiscalID,
		TipoObjeto:     models.TipoObjetoServico,
		Ativo:          true,
	}
}

func TestDesignacaoService_Designar(t *testing.T) {
	ctx := context.Background()
	fiscalOriginal := uuid.New()
	contrato := novoContratoDeTeste(fiscalOriginal)

	contratoRepo := testutil.NewFakeContratoRepository(contrato)
	portariaRepo := testutil.NewFakePortariaDesignacaoRepository()
	svc := service.NewDesignacaoService(portariaRepo, contratoRepo)

	criadoPor := uuid.New()

	t.Run("primeira designação de FISCAL sincroniza Contrato.FiscalID", func(t *testing.T) {
		novoFiscal := uuid.New()

		designacao, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:     contrato.ID,
			ServidorID:     novoFiscal,
			Papel:          models.PapelFiscal,
			NumeroPortaria: "123/2026",
			CriadoPorID:    criadoPor,
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if designacao.DataRevogacao != nil {
			t.Fatalf("designação recém-criada não deveria ter DataRevogacao")
		}

		contratoAtualizado, err := contratoRepo.FindByID(ctx, contrato.ID)
		if err != nil {
			t.Fatalf("erro ao buscar contrato: %v", err)
		}
		if contratoAtualizado.FiscalID != novoFiscal {
			t.Fatalf("esperava Contrato.FiscalID sincronizado com %s, veio %s", novoFiscal, contratoAtualizado.FiscalID)
		}
	})

	t.Run("segunda designação do mesmo papel revoga a anterior", func(t *testing.T) {
		primeiro := uuid.New()
		segundo := uuid.New()

		primeira, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  contrato.ID,
			ServidorID:  primeiro,
			Papel:       models.PapelGestor,
			CriadoPorID: criadoPor,
		})
		if err != nil {
			t.Fatalf("erro inesperado na primeira designação: %v", err)
		}

		if _, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  contrato.ID,
			ServidorID:  segundo,
			Papel:       models.PapelGestor,
			CriadoPorID: criadoPor,
		}); err != nil {
			t.Fatalf("erro inesperado na segunda designação: %v", err)
		}

		ativa, err := portariaRepo.FindAtivaPorContratoEPapel(ctx, contrato.ID, models.PapelGestor)
		if err != nil {
			t.Fatalf("erro ao buscar designação ativa: %v", err)
		}
		if ativa.ServidorID != segundo {
			t.Fatalf("esperava que a designação ativa fosse do segundo servidor, veio %s", ativa.ServidorID)
		}

		primeiraRecarregada := portariaRepo.Designacoes[primeira.ID]
		if primeiraRecarregada.DataRevogacao == nil {
			t.Fatal("esperava que a primeira designação tivesse sido revogada")
		}
	})

	t.Run("designação de FISCAL_SETORIAL não mexe em Contrato.FiscalID", func(t *testing.T) {
		fiscalAntes, err := contratoRepo.FindByID(ctx, contrato.ID)
		if err != nil {
			t.Fatalf("erro ao buscar contrato: %v", err)
		}
		fiscalIDAntes := fiscalAntes.FiscalID

		if _, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  contrato.ID,
			ServidorID:  uuid.New(),
			Papel:       models.PapelFiscalSetorial,
			CriadoPorID: criadoPor,
		}); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		fiscalDepois, err := contratoRepo.FindByID(ctx, contrato.ID)
		if err != nil {
			t.Fatalf("erro ao buscar contrato: %v", err)
		}
		if fiscalDepois.FiscalID != fiscalIDAntes {
			t.Fatalf("Contrato.FiscalID não deveria mudar ao designar um fiscal setorial")
		}
	})

	t.Run("contrato inexistente retorna erro", func(t *testing.T) {
		_, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  uuid.New(),
			ServidorID:  uuid.New(),
			Papel:       models.PapelFiscal,
			CriadoPorID: criadoPor,
		})
		if err == nil {
			t.Fatal("esperava erro para contrato inexistente")
		}
	})
}
