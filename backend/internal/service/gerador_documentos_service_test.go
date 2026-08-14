package service

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/testutil"
)

func novoContratoTeste() *models.Contrato {
	return &models.Contrato{
		ID:             uuid.New(),
		NumeroContrato: "10/2026",
		ContratadaNome: "Fornecedora Teste Ltda",
		ContratadaCNPJ: "12.345.678/0001-90",
	}
}

func TestGeradorDocumentosService_GerarNotificacao(t *testing.T) {
	ctx := context.Background()
	contrato := novoContratoTeste()
	geradoPorID := uuid.New()

	contratoRepo := testutil.NewFakeContratoRepository(contrato)
	docEmitidoRepo := &testutil.FakeDocumentoEmitidoRepository{}
	svc := NewGeradorDocumentosService(contratoRepo, testutil.NewFakeProcessoPagamentoRepository(), docEmitidoRepo, "")

	t.Run("motivo vazio é rejeitado", func(t *testing.T) {
		_, _, err := svc.GerarNotificacao(ctx, contrato.ID, "   ", geradoPorID)
		if !errors.Is(err, ErrMotivoObrigatorio) {
			t.Fatalf("esperava ErrMotivoObrigatorio, veio %v", err)
		}
	})

	t.Run("contrato inexistente é rejeitado", func(t *testing.T) {
		_, _, err := svc.GerarNotificacao(ctx, uuid.New(), "Atraso na entrega", geradoPorID)
		if !errors.Is(err, repository.ErrContratoNotFound) {
			t.Fatalf("esperava ErrContratoNotFound, veio %v", err)
		}
	})

	t.Run("caminho feliz gera o PDF e registra o histórico", func(t *testing.T) {
		pdf, registro, err := svc.GerarNotificacao(ctx, contrato.ID, "Atraso na entrega dos materiais", geradoPorID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF")) {
			t.Fatal("saída não parece ser um PDF válido (sem cabeçalho %PDF)")
		}
		if registro.Tipo != models.TipoDocumentoEmitidoNotificacao {
			t.Errorf("Tipo = %v, esperado NOTIFICACAO_DESCUMPRIMENTO", registro.Tipo)
		}
		if registro.ContratoID != contrato.ID {
			t.Errorf("ContratoID = %v, esperado %v", registro.ContratoID, contrato.ID)
		}
		if registro.GeradoPorID != geradoPorID {
			t.Errorf("GeradoPorID = %v, esperado %v", registro.GeradoPorID, geradoPorID)
		}
		if registro.CodigoVerificacao == "" {
			t.Error("esperava um CodigoVerificacao não vazio")
		}
		if len(docEmitidoRepo.Documentos) != 1 {
			t.Fatalf("esperava 1 documento emitido persistido, veio %d", len(docEmitidoRepo.Documentos))
		}
	})
}

func TestGeradorDocumentosService_GerarAtesto(t *testing.T) {
	ctx := context.Background()
	contrato := novoContratoTeste()
	processo := &models.ProcessoPagamento{
		ID: uuid.New(), ContratoID: contrato.ID, Contrato: contrato,
		MesReferencia: "01/2026", EtapaAtualID: 1,
	}
	geradoPorID := uuid.New()

	contratoRepo := testutil.NewFakeContratoRepository(contrato)
	processoRepo := testutil.NewFakeProcessoPagamentoRepository(processo)
	docEmitidoRepo := &testutil.FakeDocumentoEmitidoRepository{}

	t.Run("processo inexistente é rejeitado", func(t *testing.T) {
		svc := NewGeradorDocumentosService(contratoRepo, processoRepo, docEmitidoRepo, "")
		_, _, err := svc.GerarAtesto(ctx, uuid.New(), geradoPorID)
		if !errors.Is(err, repository.ErrProcessoNotFound) {
			t.Fatalf("esperava ErrProcessoNotFound, veio %v", err)
		}
	})

	t.Run("caminho feliz gera o PDF com QR code e registra o histórico", func(t *testing.T) {
		svc := NewGeradorDocumentosService(contratoRepo, processoRepo, docEmitidoRepo, "https://selene.papermoon.cloud")
		pdf, registro, err := svc.GerarAtesto(ctx, processo.ID, geradoPorID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF")) {
			t.Fatal("saída não parece ser um PDF válido (sem cabeçalho %PDF)")
		}
		if registro.Tipo != models.TipoDocumentoEmitidoAtesto {
			t.Errorf("Tipo = %v, esperado ATESTO", registro.Tipo)
		}
		if registro.ProcessoPagamentoID == nil || *registro.ProcessoPagamentoID != processo.ID {
			t.Error("esperava ProcessoPagamentoID preenchido com o ID do processo")
		}
		if registro.ContratoID != contrato.ID {
			t.Errorf("ContratoID = %v, esperado %v (herdado do processo)", registro.ContratoID, contrato.ID)
		}

		// A verificação embutida no QR precisa apontar pro código gerado —
		// checado indiretamente via Verificar, já que o conteúdo do PNG do
		// QR em si não é prático de decodificar num teste unitário.
		encontrado, err := svc.Verificar(ctx, registro.CodigoVerificacao)
		if err != nil {
			t.Fatalf("Verificar() erro inesperado: %v", err)
		}
		if encontrado.ID != registro.ID {
			t.Errorf("Verificar() retornou registro diferente do gerado")
		}
	})
}

func TestGeradorDocumentosService_GerarMinutaAditivo(t *testing.T) {
	ctx := context.Background()
	contrato := novoContratoTeste()
	geradoPorID := uuid.New()

	contratoRepo := testutil.NewFakeContratoRepository(contrato)
	docEmitidoRepo := &testutil.FakeDocumentoEmitidoRepository{}
	svc := NewGeradorDocumentosService(contratoRepo, testutil.NewFakeProcessoPagamentoRepository(), docEmitidoRepo, "")

	t.Run("justificativa vazia é rejeitada", func(t *testing.T) {
		_, _, err := svc.GerarMinutaAditivo(ctx, contrato.ID, MinutaAditivoInput{TipoAditivo: "VALOR"}, geradoPorID)
		if !errors.Is(err, ErrMotivoObrigatorio) {
			t.Fatalf("esperava ErrMotivoObrigatorio, veio %v", err)
		}
	})

	t.Run("caminho feliz gera o PDF e serializa os dados extras", func(t *testing.T) {
		input := MinutaAditivoInput{
			TipoAditivo:   "VALOR_E_PRAZO",
			Justificativa: "Necessidade de ampliação do escopo por determinação da Secretaria.",
			NovoValor:     "R$ 250.000,00",
			NovoPrazo:     "31/12/2026",
		}
		pdf, registro, err := svc.GerarMinutaAditivo(ctx, contrato.ID, input, geradoPorID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF")) {
			t.Fatal("saída não parece ser um PDF válido (sem cabeçalho %PDF)")
		}
		if registro.Tipo != models.TipoDocumentoEmitidoAditivo {
			t.Errorf("Tipo = %v, esperado MINUTA_ADITIVO", registro.Tipo)
		}
		if registro.Motivo != input.Justificativa {
			t.Errorf("Motivo = %q, esperado %q", registro.Motivo, input.Justificativa)
		}
		if !strings.Contains(registro.DadosExtras, "R$ 250.000,00") {
			t.Errorf("DadosExtras não contém o novo valor informado: %q", registro.DadosExtras)
		}
	})
}

func TestGeradorDocumentosService_Verificar(t *testing.T) {
	ctx := context.Background()
	docEmitidoRepo := &testutil.FakeDocumentoEmitidoRepository{}
	svc := NewGeradorDocumentosService(testutil.NewFakeContratoRepository(), testutil.NewFakeProcessoPagamentoRepository(), docEmitidoRepo, "")

	_, err := svc.Verificar(ctx, "SEL-CODIGO-INEXISTENTE")
	if !errors.Is(err, repository.ErrDocumentoEmitidoNotFound) {
		t.Fatalf("esperava ErrDocumentoEmitidoNotFound pra código inexistente, veio %v", err)
	}
}
