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

func TestCanTransition(t *testing.T) {
	processoAtivo := func(etapa int) *models.ProcessoPagamento {
		return &models.ProcessoPagamento{ID: uuid.New(), EtapaAtualID: etapa, Status: models.StatusProcessoAtivo}
	}
	ocorrenciaAberta := []models.Ocorrencia{{Estado: models.OcorrenciaRegistrada}}

	t.Run("fiscal, sem pendências, etapa intermediária: pode avançar", func(t *testing.T) {
		permitido, acoes, motivo := service.CanTransition(processoAtivo(3), nil, nil, true)
		if !permitido {
			t.Fatalf("esperava permitido=true, motivo: %q", motivo)
		}
		if !contains(acoes, service.AcaoAvancarEtapa) {
			t.Fatalf("esperava AVANCAR_ETAPA em %v", acoes)
		}
		if contains(acoes, service.AcaoConcluirPagamento) {
			t.Fatalf("não esperava CONCLUIR_PAGAMENTO fora da etapa final: %v", acoes)
		}
	})

	t.Run("não-fiscal: nunca pode avançar, sem ações de escrita", func(t *testing.T) {
		permitido, acoes, motivo := service.CanTransition(processoAtivo(1), nil, nil, false)
		if permitido {
			t.Fatal("esperava permitido=false para usuário sem papel de fiscal")
		}
		if motivo == "" {
			t.Fatal("esperava um motivo de bloqueio")
		}
		if len(acoes) != 0 {
			t.Fatalf("esperava nenhuma ação permitida, veio %v", acoes)
		}
		// nil serializaria como `null` em JSON (allowed_actions), não `[]`
		// — ver o comentário em CanTransition sobre o bug real encontrado
		// rodando docker-compose.prod.yml.
		if acoes == nil {
			t.Fatal("acoesPermitidas veio nil — deveria ser []string{}")
		}
	})

	t.Run("checklist pendente bloqueia o avanço mas não as demais ações", func(t *testing.T) {
		permitido, acoes, motivo := service.CanTransition(processoAtivo(1), nil, []string{"Nota de Empenho"}, true)
		if permitido {
			t.Fatal("esperava permitido=false com checklist pendente")
		}
		if motivo == "" {
			t.Fatal("esperava motivo de bloqueio")
		}
		if !contains(acoes, service.AcaoRegistrarOcorrencia) {
			t.Fatalf("esperava que REGISTRAR_OCORRENCIA continuasse disponível mesmo com checklist pendente: %v", acoes)
		}
	})

	t.Run("ocorrência aberta bloqueia o avanço (regra de Camada 2 confirmada com o usuário)", func(t *testing.T) {
		permitido, _, motivo := service.CanTransition(processoAtivo(4), ocorrenciaAberta, nil, true)
		if permitido {
			t.Fatal("esperava permitido=false com ocorrência aberta")
		}
		if motivo == "" {
			t.Fatal("esperava motivo de bloqueio mencionando a ocorrência")
		}
	})

	t.Run("etapa final: pode concluir, não pode avançar", func(t *testing.T) {
		permitido, acoes, _ := service.CanTransition(processoAtivo(6), nil, nil, true)
		if permitido {
			t.Fatal("esperava permitido=false na etapa final (não há próxima etapa)")
		}
		if !contains(acoes, service.AcaoConcluirPagamento) {
			t.Fatalf("esperava CONCLUIR_PAGAMENTO na etapa final: %v", acoes)
		}
		if contains(acoes, service.AcaoAvancarEtapa) {
			t.Fatalf("não esperava AVANCAR_ETAPA na etapa final: %v", acoes)
		}
	})

	t.Run("processo concluído: nenhuma ação de escrita disponível", func(t *testing.T) {
		concluido := &models.ProcessoPagamento{ID: uuid.New(), EtapaAtualID: 6, Status: models.StatusProcessoConcluido}
		permitido, acoes, motivo := service.CanTransition(concluido, nil, nil, true)
		if permitido {
			t.Fatal("esperava permitido=false para processo já concluído")
		}
		if motivo == "" {
			t.Fatal("esperava motivo de bloqueio")
		}
		if len(acoes) != 0 {
			t.Fatalf("esperava nenhuma ação disponível para processo concluído, veio %v", acoes)
		}
		if acoes == nil {
			t.Fatal("acoesPermitidas veio nil — deveria ser []string{}")
		}
	})
}

func TestFiscalizacaoService_Decorar(t *testing.T) {
	ctx := context.Background()

	t.Run("estado deriva da etapa quando não há ocorrência aberta", func(t *testing.T) {
		processo := &models.ProcessoPagamento{
			ID:           uuid.New(),
			EtapaAtualID: 2, // Tramitar Planejamento/Contabilidade
			Status:       models.StatusProcessoAtivo,
			Contrato:     &models.Contrato{TipoObjeto: models.TipoObjetoConsumo},
		}
		docRepo := &testutil.FakeDocumentoAnexoRepository{}
		ocorrenciaRepo := testutil.NewFakeOcorrenciaRepository()
		svc := service.NewFiscalizacaoService(docRepo, ocorrenciaRepo)

		decorado, err := svc.Decorar(ctx, processo, true)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if decorado.EstadoFiscalizacao != service.EstadoEmAnaliseExterna {
			t.Fatalf("esperava EM_ANALISE_EXTERNA (etapa 2), veio %s", decorado.EstadoFiscalizacao)
		}
		if decorado.AcaoOuEspera != service.EsperaExterna {
			t.Fatalf("esperava ESPERA_EXTERNA (etapa 2), veio %s", decorado.AcaoOuEspera)
		}
		// Campos originais de ProcessoPagamento continuam acessíveis
		// (embedding) — confirma que nada foi perdido na decoração.
		if decorado.ID != processo.ID {
			t.Fatalf("esperava ID do processo preservado via embedding")
		}
	})

	t.Run("ocorrência aberta força PENDENCIA_DEVOLVIDO independente da etapa", func(t *testing.T) {
		processo := &models.ProcessoPagamento{
			ID:           uuid.New(),
			EtapaAtualID: 4,
			Status:       models.StatusProcessoAtivo,
			Contrato:     &models.Contrato{TipoObjeto: models.TipoObjetoConsumo},
		}
		docRepo := &testutil.FakeDocumentoAnexoRepository{}
		ocorrenciaRepo := testutil.NewFakeOcorrenciaRepository(&models.Ocorrencia{
			ID:                  uuid.New(),
			ProcessoPagamentoID: &processo.ID,
			Estado:              models.OcorrenciaRegistrada,
		})
		svc := service.NewFiscalizacaoService(docRepo, ocorrenciaRepo)

		decorado, err := svc.Decorar(ctx, processo, true)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if decorado.EstadoFiscalizacao != service.EstadoPendenciaDevolvido {
			t.Fatalf("esperava PENDENCIA_DEVOLVIDO, veio %s", decorado.EstadoFiscalizacao)
		}
		if contains(decorado.AllowedActions, service.AcaoAvancarEtapa) {
			t.Fatalf("não esperava AVANCAR_ETAPA permitido com ocorrência aberta: %v", decorado.AllowedActions)
		}
	})

	t.Run("processo concluído retorna estado CONCLUIDO", func(t *testing.T) {
		processo := &models.ProcessoPagamento{
			ID:           uuid.New(),
			EtapaAtualID: 6,
			Status:       models.StatusProcessoConcluido,
			Contrato:     &models.Contrato{TipoObjeto: models.TipoObjetoConsumo},
		}
		docRepo := &testutil.FakeDocumentoAnexoRepository{}
		ocorrenciaRepo := testutil.NewFakeOcorrenciaRepository()
		svc := service.NewFiscalizacaoService(docRepo, ocorrenciaRepo)

		decorado, err := svc.Decorar(ctx, processo, true)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if decorado.EstadoFiscalizacao != service.EstadoConcluido {
			t.Fatalf("esperava CONCLUIDO, veio %s", decorado.EstadoFiscalizacao)
		}
	})
}

func TestFiscalizacaoService_VerificarAvancoPermitido(t *testing.T) {
	ctx := context.Background()
	docRepo := &testutil.FakeDocumentoAnexoRepository{}

	t.Run("sem ocorrência aberta: permitido", func(t *testing.T) {
		svc := service.NewFiscalizacaoService(docRepo, testutil.NewFakeOcorrenciaRepository())
		if err := svc.VerificarAvancoPermitido(ctx, uuid.New()); err != nil {
			t.Fatalf("esperava nil, veio %v", err)
		}
	})

	t.Run("ocorrência aberta: bloqueado", func(t *testing.T) {
		processoID := uuid.New()
		ocorrenciaRepo := testutil.NewFakeOcorrenciaRepository(&models.Ocorrencia{
			ID:                  uuid.New(),
			ProcessoPagamentoID: &processoID,
			Estado:              models.OcorrenciaNotificada,
		})
		svc := service.NewFiscalizacaoService(docRepo, ocorrenciaRepo)

		err := svc.VerificarAvancoPermitido(ctx, processoID)
		if !errors.Is(err, service.ErrOcorrenciaAbertaBloqueiaAvanco) {
			t.Fatalf("esperava ErrOcorrenciaAbertaBloqueiaAvanco, veio %v", err)
		}
	})

	t.Run("ocorrência já regularizada: permitido", func(t *testing.T) {
		processoID := uuid.New()
		ocorrenciaRepo := testutil.NewFakeOcorrenciaRepository(&models.Ocorrencia{
			ID:                  uuid.New(),
			ProcessoPagamentoID: &processoID,
			Estado:              models.OcorrenciaRegularizada,
		})
		svc := service.NewFiscalizacaoService(docRepo, ocorrenciaRepo)

		if err := svc.VerificarAvancoPermitido(ctx, processoID); err != nil {
			t.Fatalf("esperava nil (ocorrência já regularizada não bloqueia), veio %v", err)
		}
	})
}

func contains(itens []string, alvo string) bool {
	for _, item := range itens {
		if item == alvo {
			return true
		}
	}
	return false
}
