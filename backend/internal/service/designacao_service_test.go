package service_test

import (
	"context"
	"errors"
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

	// Servidores usados pelos subtestes abaixo — precisam existir no
	// userRepo agora que Designar valida ServidorID (ver ErrServidorInvalido).
	// novoFiscal, primeiro e segundo são designados pra papéis que existem
	// de verdade no fluxo (FISCAL, GESTOR); só novoFiscal precisa
	// IsFiscal=true (papeisQueExigemFiscal).
	novoFiscal := &models.User{ID: uuid.New(), Nome: "Novo Fiscal", IsFiscal: true}
	naoFiscal := &models.User{ID: uuid.New(), Nome: "Servidor Sem IsFiscal", IsFiscal: false}
	primeiroGestor := &models.User{ID: uuid.New(), Nome: "Primeiro Gestor"}
	segundoGestor := &models.User{ID: uuid.New(), Nome: "Segundo Gestor"}
	fiscalSetorial := &models.User{ID: uuid.New(), Nome: "Fiscal Setorial"}

	contratoRepo := testutil.NewFakeContratoRepository(contrato)
	portariaRepo := testutil.NewFakePortariaDesignacaoRepository()
	userRepo := testutil.NewFakeUserRepository(novoFiscal, naoFiscal, primeiroGestor, segundoGestor, fiscalSetorial)
	svc := service.NewDesignacaoService(portariaRepo, contratoRepo, userRepo)

	criadoPor := uuid.New()

	t.Run("primeira designação de FISCAL sincroniza Contrato.FiscalID", func(t *testing.T) {
		designacao, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:     contrato.ID,
			ServidorID:     novoFiscal.ID,
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
		if contratoAtualizado.FiscalID != novoFiscal.ID {
			t.Fatalf("esperava Contrato.FiscalID sincronizado com %s, veio %s", novoFiscal.ID, contratoAtualizado.FiscalID)
		}
	})

	t.Run("segunda designação do mesmo papel revoga a anterior", func(t *testing.T) {
		primeira, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  contrato.ID,
			ServidorID:  primeiroGestor.ID,
			Papel:       models.PapelGestor,
			CriadoPorID: criadoPor,
		})
		if err != nil {
			t.Fatalf("erro inesperado na primeira designação: %v", err)
		}

		if _, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  contrato.ID,
			ServidorID:  segundoGestor.ID,
			Papel:       models.PapelGestor,
			CriadoPorID: criadoPor,
		}); err != nil {
			t.Fatalf("erro inesperado na segunda designação: %v", err)
		}

		ativa, err := portariaRepo.FindAtivaPorContratoEPapel(ctx, contrato.ID, models.PapelGestor)
		if err != nil {
			t.Fatalf("erro ao buscar designação ativa: %v", err)
		}
		if ativa.ServidorID != segundoGestor.ID {
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
			ServidorID:  fiscalSetorial.ID,
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
			ServidorID:  novoFiscal.ID,
			Papel:       models.PapelFiscal,
			CriadoPorID: criadoPor,
		})
		if err == nil {
			t.Fatal("esperava erro para contrato inexistente")
		}
	})

	t.Run("servidor inexistente é rejeitado", func(t *testing.T) {
		_, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  contrato.ID,
			ServidorID:  uuid.New(), // não cadastrado no userRepo
			Papel:       models.PapelGestor,
			CriadoPorID: criadoPor,
		})
		if !errors.Is(err, service.ErrServidorInvalido) {
			t.Fatalf("esperava ErrServidorInvalido, veio %v", err)
		}
	})

	t.Run("designar FISCAL com servidor sem IsFiscal é rejeitado", func(t *testing.T) {
		_, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  contrato.ID,
			ServidorID:  naoFiscal.ID,
			Papel:       models.PapelFiscal,
			CriadoPorID: criadoPor,
		})
		if !errors.Is(err, service.ErrFiscalInvalido) {
			t.Fatalf("esperava ErrFiscalInvalido, veio %v", err)
		}
	})

	t.Run("designar FISCAL_SUPLENTE com servidor sem IsFiscal é rejeitado", func(t *testing.T) {
		_, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  contrato.ID,
			ServidorID:  naoFiscal.ID,
			Papel:       models.PapelFiscalSuplente,
			CriadoPorID: criadoPor,
		})
		if !errors.Is(err, service.ErrFiscalInvalido) {
			t.Fatalf("esperava ErrFiscalInvalido, veio %v", err)
		}
	})

	t.Run("designar GESTOR com servidor sem IsFiscal é aceito (papel não exige)", func(t *testing.T) {
		if _, err := svc.Designar(ctx, service.DesignarInput{
			ContratoID:  contrato.ID,
			ServidorID:  naoFiscal.ID,
			Papel:       models.PapelGestor,
			CriadoPorID: criadoPor,
		}); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
	})
}
