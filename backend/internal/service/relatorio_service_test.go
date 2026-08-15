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

func novoRelatorioServiceDeTeste(processo *models.ProcessoPagamento) (*service.RelatorioService, *testutil.FakeOcorrenciaRepository, *testutil.FakeEmpenhoRepository, *testutil.FakeMovimentacaoEmpenhoRepository) {
	processoRepo := testutil.NewFakeProcessoPagamentoRepository(processo)
	docRepo := &testutil.FakeDocumentoAnexoRepository{}
	ocorrenciaRepo := testutil.NewFakeOcorrenciaRepository()
	empenhoRepo := testutil.NewFakeEmpenhoRepository()
	movimentacaoRepo := &testutil.FakeMovimentacaoEmpenhoRepository{}

	svc, err := service.NewRelatorioService(processoRepo, docRepo, ocorrenciaRepo, empenhoRepo, movimentacaoRepo)
	if err != nil {
		panic(err)
	}
	return svc, ocorrenciaRepo, empenhoRepo, movimentacaoRepo
}

func novoProcessoParaRelatorio() *models.ProcessoPagamento {
	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste"}
	contrato := novoContratoDeTeste(uuid.New())
	contrato.Fiscal = fiscal
	return &models.ProcessoPagamento{
		ID:            uuid.New(),
		ContratoID:    contrato.ID,
		Contrato:      contrato,
		MesReferencia: "01/2026",
		EtapaAtualID:  5,
		Status:        models.StatusProcessoAtivo,
	}
}

// TestRelatorioService_Gerar é um teste de fumaça (smoke test) — PDF
// binário não vale a pena inspecionar byte a byte, mas o código que monta
// as seções novas de Ocorrências/Empenho (Fase 5 do plano) precisa, no
// mínimo, não panicar/errar nesses três cenários.
func TestRelatorioService_Gerar(t *testing.T) {
	ctx := context.Background()

	t.Run("sem ocorrências e sem empenho vinculado", func(t *testing.T) {
		processo := novoProcessoParaRelatorio()
		svc, _, _, _ := novoRelatorioServiceDeTeste(processo)

		pdf, err := svc.Gerar(ctx, processo.ID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if len(pdf) == 0 {
			t.Fatal("esperava PDF não-vazio")
		}
	})

	t.Run("com ocorrências registradas", func(t *testing.T) {
		processo := novoProcessoParaRelatorio()
		svc, ocorrenciaRepo, _, _ := novoRelatorioServiceDeTeste(processo)
		ocorrenciaRepo.Ocorrencias[uuid.New()] = &models.Ocorrencia{
			ID:                  uuid.New(),
			ProcessoPagamentoID: &processo.ID,
			Descricao:           "Atraso na entrega.",
			Estado:              models.OcorrenciaNotificada,
		}

		pdf, err := svc.Gerar(ctx, processo.ID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if len(pdf) == 0 {
			t.Fatal("esperava PDF não-vazio")
		}
	})

	t.Run("com empenho vinculado e saldo reconstruído", func(t *testing.T) {
		processo := novoProcessoParaRelatorio()
		svc, _, empenhoRepo, movimentacaoRepo := novoRelatorioServiceDeTeste(processo)

		empenho := &models.Empenho{
			ID:            uuid.New(),
			ContratoID:    processo.ContratoID,
			NumeroEmpenho: "10/2026",
			DataEmissao:   time.Now(),
			ValorInicial:  100_000,
		}
		empenhoRepo.Empenhos[empenho.ID] = empenho
		movimentacaoRepo.Movimentacoes = append(movimentacaoRepo.Movimentacoes, models.MovimentacaoEmpenho{
			ID:        uuid.New(),
			EmpenhoID: empenho.ID,
			Tipo:      models.MovimentacaoInicial,
			Valor:     100_000,
		})
		processo.EmpenhoID = &empenho.ID

		pdf, err := svc.Gerar(ctx, processo.ID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if len(pdf) == 0 {
			t.Fatal("esperava PDF não-vazio")
		}
	})

	t.Run("empenho vinculado mas inexistente no repositório retorna erro", func(t *testing.T) {
		processo := novoProcessoParaRelatorio()
		svc, _, _, _ := novoRelatorioServiceDeTeste(processo)
		idInexistente := uuid.New()
		processo.EmpenhoID = &idInexistente

		if _, err := svc.Gerar(ctx, processo.ID); err == nil {
			t.Fatal("esperava erro para empenho inexistente")
		}
	})
}
