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

func novoEmpenhoServiceDeTeste(contratos ...*models.Contrato) (*service.EmpenhoService, *testutil.FakeEmpenhoRepository, *testutil.FakeMovimentacaoEmpenhoRepository) {
	empenhoRepo := testutil.NewFakeEmpenhoRepository()
	movimentacaoRepo := &testutil.FakeMovimentacaoEmpenhoRepository{}
	contratoRepo := testutil.NewFakeContratoRepository(contratos...)
	return service.NewEmpenhoService(empenhoRepo, movimentacaoRepo, contratoRepo), empenhoRepo, movimentacaoRepo
}

func TestEmpenhoService_CriarEmpenho(t *testing.T) {
	ctx := context.Background()
	contrato := novoContratoDeTeste(uuid.New())
	svc, _, movimentacaoRepo := novoEmpenhoServiceDeTeste(contrato)

	t.Run("valor inicial inválido é rejeitado", func(t *testing.T) {
		_, err := svc.CriarEmpenho(ctx, service.CriarEmpenhoInput{
			ContratoID:   contrato.ID,
			ValorInicial: 0,
		})
		if !errors.Is(err, service.ErrValorInvalido) {
			t.Fatalf("esperava ErrValorInvalido, veio %v", err)
		}
	})

	t.Run("caminho feliz registra a movimentação INICIAL", func(t *testing.T) {
		registradoPor := uuid.New()
		empenho, err := svc.CriarEmpenho(ctx, service.CriarEmpenhoInput{
			ContratoID:      contrato.ID,
			NumeroEmpenho:   "500/2026",
			DataEmissao:     "2026-01-05",
			ValorInicial:    100_000, // R$ 1.000,00 em centavos
			RegistradoPorID: registradoPor,
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		movimentacoes, err := movimentacaoRepo.ListByEmpenho(ctx, empenho.ID)
		if err != nil {
			t.Fatalf("erro ao listar movimentações: %v", err)
		}
		if len(movimentacoes) != 1 {
			t.Fatalf("esperava 1 movimentação inicial, veio %d", len(movimentacoes))
		}
		if movimentacoes[0].Tipo != models.MovimentacaoInicial || movimentacoes[0].Valor != 100_000 {
			t.Fatalf("movimentação inicial incorreta: %+v", movimentacoes[0])
		}
	})
}

func TestEmpenhoService_CalcularSaldo(t *testing.T) {
	ctx := context.Background()
	contrato := novoContratoDeTeste(uuid.New())
	svc, _, _ := novoEmpenhoServiceDeTeste(contrato)

	empenho, err := svc.CriarEmpenho(ctx, service.CriarEmpenhoInput{
		ContratoID:   contrato.ID,
		DataEmissao:  "2026-01-05",
		ValorInicial: 1_000_00, // R$ 1.000,00
	})
	if err != nil {
		t.Fatalf("erro ao criar empenho: %v", err)
	}

	// Saldo reconstruído: 1000 (inicial) + 500 (reforço) - 200 (anulação) -
	// 300 (fatura apropriada) = 1000.
	for _, mov := range []struct {
		tipo  models.TipoMovimentacaoEmpenho
		valor int64
	}{
		{models.MovimentacaoReforco, 500_00},
		{models.MovimentacaoAnulacao, 200_00},
		{models.MovimentacaoFaturaApropriada, 300_00},
	} {
		if _, err := svc.RegistrarMovimentacao(ctx, service.RegistrarMovimentacaoInput{
			EmpenhoID: empenho.ID,
			Tipo:      mov.tipo,
			Valor:     mov.valor,
		}); err != nil {
			t.Fatalf("erro ao registrar movimentação %s: %v", mov.tipo, err)
		}
	}

	saldo, err := svc.CalcularSaldo(ctx, empenho.ID)
	if err != nil {
		t.Fatalf("erro ao calcular saldo: %v", err)
	}
	if saldo != 1_000_00 {
		t.Fatalf("esperava saldo 100000, veio %d", saldo)
	}
}

func TestEmpenhoService_RegistrarMovimentacao_RejeitaSaldoInsuficiente(t *testing.T) {
	ctx := context.Background()
	contrato := novoContratoDeTeste(uuid.New())
	svc, _, _ := novoEmpenhoServiceDeTeste(contrato)

	empenho, err := svc.CriarEmpenho(ctx, service.CriarEmpenhoInput{
		ContratoID:   contrato.ID,
		DataEmissao:  "2026-01-05",
		ValorInicial: 100_00, // R$ 100,00
	})
	if err != nil {
		t.Fatalf("erro ao criar empenho: %v", err)
	}

	t.Run("anulação maior que o saldo é rejeitada", func(t *testing.T) {
		_, err := svc.RegistrarMovimentacao(ctx, service.RegistrarMovimentacaoInput{
			EmpenhoID: empenho.ID,
			Tipo:      models.MovimentacaoAnulacao,
			Valor:     200_00, // maior que os R$ 100,00 disponíveis
		})
		if !errors.Is(err, service.ErrSaldoInsuficiente) {
			t.Fatalf("esperava ErrSaldoInsuficiente, veio %v", err)
		}
	})

	t.Run("fatura apropriada maior que o saldo é rejeitada", func(t *testing.T) {
		_, err := svc.RegistrarMovimentacao(ctx, service.RegistrarMovimentacaoInput{
			EmpenhoID: empenho.ID,
			Tipo:      models.MovimentacaoFaturaApropriada,
			Valor:     200_00,
		})
		if !errors.Is(err, service.ErrSaldoInsuficiente) {
			t.Fatalf("esperava ErrSaldoInsuficiente, veio %v", err)
		}
	})

	t.Run("anulação igual ao saldo disponível é aceita (zera o saldo)", func(t *testing.T) {
		_, err := svc.RegistrarMovimentacao(ctx, service.RegistrarMovimentacaoInput{
			EmpenhoID: empenho.ID,
			Tipo:      models.MovimentacaoAnulacao,
			Valor:     100_00,
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		saldo, err := svc.CalcularSaldo(ctx, empenho.ID)
		if err != nil {
			t.Fatalf("erro ao calcular saldo: %v", err)
		}
		if saldo != 0 {
			t.Fatalf("esperava saldo 0, veio %d", saldo)
		}
	})
}

func TestEmpenhoService_RegistrarMovimentacao_RejeitaTipoInicial(t *testing.T) {
	ctx := context.Background()
	contrato := novoContratoDeTeste(uuid.New())
	svc, _, _ := novoEmpenhoServiceDeTeste(contrato)

	empenho, err := svc.CriarEmpenho(ctx, service.CriarEmpenhoInput{ContratoID: contrato.ID, DataEmissao: "2026-01-05", ValorInicial: 100})
	if err != nil {
		t.Fatalf("erro ao criar empenho: %v", err)
	}

	_, err = svc.RegistrarMovimentacao(ctx, service.RegistrarMovimentacaoInput{
		EmpenhoID: empenho.ID,
		Tipo:      models.MovimentacaoInicial,
		Valor:     100,
	})
	if !errors.Is(err, service.ErrTipoMovimentacaoInvalido) {
		t.Fatalf("esperava ErrTipoMovimentacaoInvalido, veio %v", err)
	}
}
