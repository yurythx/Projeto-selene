package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/service"
	"projeto-selene/internal/testutil"
)

func TestOcorrenciaService_CicloDeVida(t *testing.T) {
	ctx := context.Background()

	contrato := novoContratoDeTeste(uuid.New())
	processo := &models.ProcessoPagamento{
		ID:            uuid.New(),
		ContratoID:    contrato.ID,
		MesReferencia: "01/2026",
		EtapaAtualID:  1,
		Status:        models.StatusProcessoAtivo,
	}

	processoRepo := testutil.NewFakeProcessoPagamentoRepository(processo)
	ocorrenciaRepo := testutil.NewFakeOcorrenciaRepository()
	svc := service.NewOcorrenciaService(ocorrenciaRepo, processoRepo)

	registradoPor := uuid.New()
	ocorrencia, err := svc.Registrar(ctx, service.RegistrarOcorrenciaInput{
		ProcessoPagamentoID: processo.ID,
		Descricao:           "Atraso na entrega dos documentos.",
		RegistradoPorID:     registradoPor,
	})
	if err != nil {
		t.Fatalf("erro ao registrar ocorrência: %v", err)
	}
	if ocorrencia.Estado != models.OcorrenciaRegistrada {
		t.Fatalf("esperava estado REGISTRADA, veio %s", ocorrencia.Estado)
	}
	if ocorrencia.ContratoID != contrato.ID {
		t.Fatalf("esperava ContratoID resolvido a partir do processo (%s), veio %s", contrato.ID, ocorrencia.ContratoID)
	}

	t.Run("pular etapa (REGISTRADA -> EM_TRATAMENTO) é rejeitado", func(t *testing.T) {
		_, err := svc.IniciarTratamento(ctx, ocorrencia.ID)
		if !errors.Is(err, service.ErrTransicaoOcorrenciaInvalida) {
			t.Fatalf("esperava ErrTransicaoOcorrenciaInvalida, veio %v", err)
		}
	})

	t.Run("fluxo linear completo", func(t *testing.T) {
		notificada, err := svc.Notificar(ctx, ocorrencia.ID)
		if err != nil {
			t.Fatalf("erro ao notificar: %v", err)
		}
		if notificada.Estado != models.OcorrenciaNotificada || notificada.DataNotificacaoGestor == nil {
			t.Fatalf("estado/data de notificação incorretos: %+v", notificada)
		}

		emTratamento, err := svc.IniciarTratamento(ctx, ocorrencia.ID)
		if err != nil {
			t.Fatalf("erro ao iniciar tratamento: %v", err)
		}
		if emTratamento.Estado != models.OcorrenciaEmTratamento {
			t.Fatalf("esperava EM_TRATAMENTO, veio %s", emTratamento.Estado)
		}

		regularizada, err := svc.Regularizar(ctx, ocorrencia.ID)
		if err != nil {
			t.Fatalf("erro ao regularizar: %v", err)
		}
		if regularizada.Estado != models.OcorrenciaRegularizada || regularizada.DataRegularizacao == nil {
			t.Fatalf("estado/data de regularização incorretos: %+v", regularizada)
		}

		// Regularizada é terminal: não permite mais nenhuma transição.
		if _, err := svc.Notificar(ctx, ocorrencia.ID); !errors.Is(err, service.ErrTransicaoOcorrenciaInvalida) {
			t.Fatalf("esperava ErrTransicaoOcorrenciaInvalida após regularização, veio %v", err)
		}
	})
}

func TestOcorrenciaService_Registrar_ProcessoInexistente(t *testing.T) {
	ctx := context.Background()
	svc := service.NewOcorrenciaService(testutil.NewFakeOcorrenciaRepository(), testutil.NewFakeProcessoPagamentoRepository())

	_, err := svc.Registrar(ctx, service.RegistrarOcorrenciaInput{
		ProcessoPagamentoID: uuid.New(),
		Descricao:           "x",
	})
	if err == nil {
		t.Fatal("esperava erro para processo inexistente")
	}
}
