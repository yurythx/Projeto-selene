package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/testutil"
)

func TestApenasDigitos(t *testing.T) {
	casos := map[string]string{
		"12.345.678/0001-90": "12345678000190",
		"12345678000190":     "12345678000190",
		"":                   "",
	}
	for entrada, esperado := range casos {
		if got := apenasDigitos(entrada); got != esperado {
			t.Errorf("apenasDigitos(%q) = %q, esperado %q", entrada, got, esperado)
		}
	}
}

func TestFornecedorService_Listar(t *testing.T) {
	ctx := context.Background()
	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste"}

	// Dois contratos da mesma empresa (CNPJ com e sem máscara — precisa
	// agrupar como o MESMO fornecedor), um ativo e um encerrado; mais um
	// contrato de outra empresa.
	c1 := &models.Contrato{ID: uuid.New(), NumeroContrato: "1/2026", ContratadaCNPJ: "12.345.678/0001-90", ContratadaNome: "Fornecedora A", FiscalID: fiscal.ID, Ativo: true}
	c2 := &models.Contrato{ID: uuid.New(), NumeroContrato: "2/2026", ContratadaCNPJ: "12345678000190", ContratadaNome: "Fornecedora A", FiscalID: fiscal.ID, Ativo: false}
	c3 := &models.Contrato{ID: uuid.New(), NumeroContrato: "3/2026", ContratadaCNPJ: "98.765.432/0001-10", ContratadaNome: "Fornecedora B", FiscalID: fiscal.ID, Ativo: true}

	contratoRepo := testutil.NewFakeContratoRepository(c1, c2, c3)
	svc := NewFornecedorService(contratoRepo, testutil.NewFakeProcessoPagamentoRepository(), &testutil.FakeKanbanLogRepository{}, &testutil.FakeDocumentoEmitidoRepository{})

	resumos, err := svc.Listar(ctx)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(resumos) != 2 {
		t.Fatalf("esperava 2 fornecedores distintos, veio %d", len(resumos))
	}

	var fornecedorA *FornecedorResumo
	for i := range resumos {
		if resumos[i].Nome == "Fornecedora A" {
			fornecedorA = &resumos[i]
		}
	}
	if fornecedorA == nil {
		t.Fatal("esperava encontrar 'Fornecedora A' na listagem")
	}
	if fornecedorA.QtdContratos != 2 {
		t.Errorf("QtdContratos = %d, esperado 2 (CNPJ mascarado e não-mascarado deveriam contar como o mesmo fornecedor)", fornecedorA.QtdContratos)
	}
	if fornecedorA.QtdContratosAtivos != 1 {
		t.Errorf("QtdContratosAtivos = %d, esperado 1", fornecedorA.QtdContratosAtivos)
	}
	if fornecedorA.CNPJ != "12345678000190" {
		t.Errorf("CNPJ = %q, esperado só dígitos", fornecedorA.CNPJ)
	}
}

func TestFornecedorService_Buscar(t *testing.T) {
	ctx := context.Background()
	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste"}
	geradoPorID := uuid.New()

	contrato := &models.Contrato{
		ID: uuid.New(), NumeroContrato: "1/2026", ContratadaCNPJ: "12.345.678/0001-90",
		ContratadaNome: "Fornecedora A", FiscalID: fiscal.ID, Ativo: true,
	}
	contratoRepo := testutil.NewFakeContratoRepository(contrato)

	t.Run("CNPJ inexistente é rejeitado", func(t *testing.T) {
		svc := NewFornecedorService(contratoRepo, testutil.NewFakeProcessoPagamentoRepository(), &testutil.FakeKanbanLogRepository{}, &testutil.FakeDocumentoEmitidoRepository{})
		_, err := svc.Buscar(ctx, "00.000.000/0000-00")
		if !errors.Is(err, ErrFornecedorNaoEncontrado) {
			t.Fatalf("esperava ErrFornecedorNaoEncontrado, veio %v", err)
		}
	})

	t.Run("aceita CNPJ mascarado ou só dígitos", func(t *testing.T) {
		svc := NewFornecedorService(contratoRepo, testutil.NewFakeProcessoPagamentoRepository(), &testutil.FakeKanbanLogRepository{}, &testutil.FakeDocumentoEmitidoRepository{})

		porMascara, err := svc.Buscar(ctx, "12.345.678/0001-90")
		if err != nil {
			t.Fatalf("erro inesperado (mascarado): %v", err)
		}
		porDigitos, err := svc.Buscar(ctx, "12345678000190")
		if err != nil {
			t.Fatalf("erro inesperado (só dígitos): %v", err)
		}
		if porMascara.Nome != porDigitos.Nome {
			t.Error("os dois formatos deveriam resolver pro mesmo fornecedor")
		}
	})

	t.Run("inclui só notificações de descumprimento no histórico", func(t *testing.T) {
		docEmitidoRepo := &testutil.FakeDocumentoEmitidoRepository{
			Documentos: []models.DocumentoEmitido{
				{ID: uuid.New(), ContratoID: contrato.ID, Tipo: models.TipoDocumentoEmitidoNotificacao, Motivo: "Atraso", GeradoPorID: geradoPorID, CodigoVerificacao: "SEL-1"},
				{ID: uuid.New(), ContratoID: contrato.ID, Tipo: models.TipoDocumentoEmitidoAtesto, GeradoPorID: geradoPorID, CodigoVerificacao: "SEL-2"},
			},
		}
		svc := NewFornecedorService(contratoRepo, testutil.NewFakeProcessoPagamentoRepository(), &testutil.FakeKanbanLogRepository{}, docEmitidoRepo)

		dossie, err := svc.Buscar(ctx, contrato.ContratadaCNPJ)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if len(dossie.Notificacoes) != 1 {
			t.Fatalf("esperava 1 notificação (atesto não conta como penalidade), veio %d", len(dossie.Notificacoes))
		}
	})

	t.Run("sem processos, score é nil", func(t *testing.T) {
		svc := NewFornecedorService(contratoRepo, testutil.NewFakeProcessoPagamentoRepository(), &testutil.FakeKanbanLogRepository{}, &testutil.FakeDocumentoEmitidoRepository{})
		dossie, err := svc.Buscar(ctx, contrato.ContratadaCNPJ)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if dossie.ScorePontualidade != nil {
			t.Errorf("esperava ScorePontualidade nil sem processos/transições, veio %v", *dossie.ScorePontualidade)
		}
		// Diferente de ScorePontualidade (que É nil de propósito, um
		// *float64 opcional), Notificacoes é um slice que precisa vir
		// vazio, não nil — nil serializaria como `null` em JSON, não
		// `[]` (encoding/json). Ver o comentário em
		// notificacoesDoFornecedor sobre o bug real encontrado rodando
		// docker-compose.prod.yml.
		if dossie.Notificacoes == nil {
			t.Error("Notificacoes veio nil sem nenhuma notificação — deveria ser []models.DocumentoEmitido{}")
		}
	})

	t.Run("score de pontualidade calcula a proporção de transições no prazo", func(t *testing.T) {
		processo := &models.ProcessoPagamento{ID: uuid.New(), ContratoID: contrato.ID, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		processoRepo := testutil.NewFakeProcessoPagamentoRepository(processo)

		logRepo := &testutil.FakeKanbanLogRepository{
			Logs: []models.KanbanLog{
				// 1ª transição: 5 dias após a abertura — no prazo (limiar 15 dias).
				{ID: uuid.New(), ProcessoPagamentoID: processo.ID, UsuarioID: fiscal.ID, EtapaDestinoID: 2, MovidoEm: time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC)},
				// 2ª transição: 20 dias depois da 1ª — atrasada.
				{ID: uuid.New(), ProcessoPagamentoID: processo.ID, UsuarioID: fiscal.ID, EtapaDestinoID: 3, MovidoEm: time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC)},
			},
		}

		svc := NewFornecedorService(contratoRepo, processoRepo, logRepo, &testutil.FakeDocumentoEmitidoRepository{})
		dossie, err := svc.Buscar(ctx, contrato.ContratadaCNPJ)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if dossie.ScorePontualidade == nil {
			t.Fatal("esperava um ScorePontualidade calculado")
		}
		if *dossie.ScorePontualidade != 50 {
			t.Errorf("ScorePontualidade = %.2f, esperado 50 (1 de 2 transições no prazo)", *dossie.ScorePontualidade)
		}
	})
}
