package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// fakeDocumentoAnexoRepository é um dublê de teste em memória para
// repository.DocumentoAnexoRepository — permite testar a lógica de
// checklist sem precisar de um banco real.
type fakeDocumentoAnexoRepository struct {
	documentos []models.DocumentoAnexo
}

func (f *fakeDocumentoAnexoRepository) Create(ctx context.Context, documento *models.DocumentoAnexo) error {
	f.documentos = append(f.documentos, *documento)
	return nil
}

func (f *fakeDocumentoAnexoRepository) ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.DocumentoAnexo, error) {
	var out []models.DocumentoAnexo
	for _, d := range f.documentos {
		if d.ProcessoPagamentoID == processoID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeDocumentoAnexoRepository) FindByProcessoAndHash(ctx context.Context, processoID uuid.UUID, hash string) (*models.DocumentoAnexo, error) {
	for _, d := range f.documentos {
		if d.ProcessoPagamentoID == processoID && d.HashArquivo == hash {
			return &d, nil
		}
	}
	return nil, repository.ErrDocumentoNotFound
}

func (f *fakeDocumentoAnexoRepository) FindByProcessoAndTipo(ctx context.Context, processoID uuid.UUID, tipoDocumentoID int) (*models.DocumentoAnexo, error) {
	for _, d := range f.documentos {
		if d.ProcessoPagamentoID == processoID && d.TipoDocumentoID == tipoDocumentoID {
			return &d, nil
		}
	}
	return nil, repository.ErrDocumentoNotFound
}

func (f *fakeDocumentoAnexoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.DocumentoAnexo, error) {
	for _, d := range f.documentos {
		if d.ID == id {
			return &d, nil
		}
	}
	return nil, repository.ErrDocumentoNotFound
}

func (f *fakeDocumentoAnexoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	for i, d := range f.documentos {
		if d.ID == id {
			f.documentos = append(f.documentos[:i], f.documentos[i+1:]...)
			return nil
		}
	}
	return repository.ErrDocumentoNotFound
}

var _ repository.DocumentoAnexoRepository = (*fakeDocumentoAnexoRepository)(nil)

func comDocumento(processoID uuid.UUID, tipoNome string) models.DocumentoAnexo {
	return models.DocumentoAnexo{
		ProcessoPagamentoID: processoID,
		TipoDocumento:       &models.TipoDocumento{Nome: tipoNome},
	}
}

func TestRequisitosEtapa(t *testing.T) {
	t.Run("etapa 1 exige OF, Pré-Empenho e Ofício, independente do tipo de objeto", func(t *testing.T) {
		got := RequisitosEtapa(1, models.TipoObjetoConsumo, false)
		want := []string{"Ordem de Fornecimento (OF)", "Pré-Empenho", "Ofício de Solicitação"}
		assertSameElements(t, got, want)
	})

	t.Run("etapa 5 para CONSUMO não inclui os itens condicionais de SERVICO", func(t *testing.T) {
		got := RequisitosEtapa(5, models.TipoObjetoConsumo, false)
		for _, condicional := range checklistCondicionalServico {
			for _, r := range got {
				if r == condicional {
					t.Fatalf("etapa 5 para CONSUMO não deveria exigir %q", condicional)
				}
			}
		}
	})

	t.Run("etapa 5 para SERVICO inclui Planilha de Medição e Boleto DAM", func(t *testing.T) {
		got := RequisitosEtapa(5, models.TipoObjetoServico, false)
		assertContains(t, got, "Planilha de Medição de Serviços")
		assertContains(t, got, "Boleto DAM")
	})

	t.Run("etapas 2 e 6 não têm checklist de saída", func(t *testing.T) {
		if got := RequisitosEtapa(2, models.TipoObjetoServico, false); len(got) != 0 {
			t.Fatalf("etapa 2 deveria ter checklist vazio, veio %v", got)
		}
		if got := RequisitosEtapa(6, models.TipoObjetoServico, false); len(got) != 0 {
			t.Fatalf("etapa 6 deveria ter checklist vazio, veio %v", got)
		}
	})

	t.Run("etapa 5 sem exigeFiscalizacaoTerceirizacao não inclui os itens de mão de obra", func(t *testing.T) {
		got := RequisitosEtapa(5, models.TipoObjetoServico, false)
		for _, condicional := range checklistCondicionalTerceirizacao {
			for _, r := range got {
				if r == condicional {
					t.Fatalf("sem exigeFiscalizacaoTerceirizacao não deveria exigir %q", condicional)
				}
			}
		}
	})

	t.Run("etapa 5 com exigeFiscalizacaoTerceirizacao inclui os documentos do Art.9º-XXXII", func(t *testing.T) {
		got := RequisitosEtapa(5, models.TipoObjetoServico, true)
		for _, condicional := range checklistCondicionalTerceirizacao {
			assertContains(t, got, condicional)
		}
		// Cumulativo com os condicionais de SERVICO, não excludente.
		assertContains(t, got, "Planilha de Medição de Serviços")
	})
}

func TestChecklistPendente(t *testing.T) {
	ctx := context.Background()
	processoID := uuid.New()

	t.Run("sem documentos anexados, tudo pendente", func(t *testing.T) {
		repo := &fakeDocumentoAnexoRepository{}
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 1, models.TipoObjetoConsumo, false)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		assertSameElements(t, pendentes, []string{"Ordem de Fornecimento (OF)", "Pré-Empenho", "Ofício de Solicitação"})
	})

	t.Run("com todos os documentos anexados, checklist satisfeito", func(t *testing.T) {
		repo := &fakeDocumentoAnexoRepository{documentos: []models.DocumentoAnexo{
			comDocumento(processoID, "Ordem de Fornecimento (OF)"),
			comDocumento(processoID, "Pré-Empenho"),
			comDocumento(processoID, "Ofício de Solicitação"),
		}}
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 1, models.TipoObjetoConsumo, false)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if len(pendentes) != 0 {
			t.Fatalf("esperava checklist satisfeito, faltaram: %v", pendentes)
		}
	})

	t.Run("documentos de OUTRO processo não contam para este processo", func(t *testing.T) {
		outroProcesso := uuid.New()
		repo := &fakeDocumentoAnexoRepository{documentos: []models.DocumentoAnexo{
			comDocumento(outroProcesso, "Ordem de Fornecimento (OF)"),
		}}
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 1, models.TipoObjetoConsumo, false)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		assertContains(t, pendentes, "Ordem de Fornecimento (OF)")
	})

	t.Run("SERVICO com certidões mas sem os condicionais ainda fica pendente", func(t *testing.T) {
		// Cumulativo agora (RequisitosAcumulados): pra isolar só a
		// pendência dos condicionais de SERVICO, os documentos base das
		// etapas 1/3/4 (percorridas antes de chegar na 5) também
		// precisam estar anexados — senão eles apareceriam como
		// pendentes também, o que é o comportamento novo correto, mas
		// não o que este teste quer isolar.
		docs := documentosBaseAteEtapa(processoID, 5)
		repo := &fakeDocumentoAnexoRepository{documentos: docs}
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 5, models.TipoObjetoServico, false)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		assertSameElements(t, pendentes, checklistCondicionalServico)
	})

	t.Run("exigeFiscalizacaoTerceirizacao acrescenta os documentos do Art.9º-XXXII à pendência", func(t *testing.T) {
		docs := documentosBaseAteEtapa(processoID, 5)
		for _, nome := range checklistCondicionalServico {
			docs = append(docs, comDocumento(processoID, nome))
		}
		repo := &fakeDocumentoAnexoRepository{documentos: docs}
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 5, models.TipoObjetoServico, true)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		assertSameElements(t, pendentes, checklistCondicionalTerceirizacao)
	})

	t.Run("cumulativo: documento obrigatório de uma etapa anterior excluído bloqueia o avanço da etapa atual", func(t *testing.T) {
		// Pedido explícito do usuário: um documento apagado depois de ter
		// satisfeito o checklist de uma etapa já concluída precisa voltar
		// a bloquear o avanço, mesmo que o processo já esteja numa etapa
		// mais adiante — não só a etapa atual isolada.
		docs := documentosBaseAteEtapa(processoID, 4)
		// Remove "Pré-Empenho" (obrigatório na etapa 1) da lista, simulando exclusão.
		filtrados := docs[:0]
		for _, d := range docs {
			if d.TipoDocumento.Nome != "Pré-Empenho" {
				filtrados = append(filtrados, d)
			}
		}
		repo := &fakeDocumentoAnexoRepository{documentos: filtrados}
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 4, models.TipoObjetoConsumo, false)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		assertSameElements(t, pendentes, []string{"Pré-Empenho"})
	})
}

// documentosBaseAteEtapa monta a lista de DocumentoAnexo já anexados,
// cobrindo os requisitos base (sem condicionais) de todas as etapas de 1
// até ateEtapaID (inclusive) — usado nos testes cumulativos pra isolar
// só o que cada teste quer exercitar.
func documentosBaseAteEtapa(processoID uuid.UUID, ateEtapaID int) []models.DocumentoAnexo {
	var docs []models.DocumentoAnexo
	for etapa := 1; etapa <= ateEtapaID; etapa++ {
		for _, nome := range checklistBase[etapa] {
			docs = append(docs, comDocumento(processoID, nome))
		}
	}
	return docs
}

func TestTipoDocumentoAplicavel(t *testing.T) {
	servico := models.TipoObjetoServico

	t.Run("documento sem restrição se aplica a qualquer tipo de contrato", func(t *testing.T) {
		tipo := models.TipoDocumento{Nome: "Nota Fiscal / Fatura"}
		for _, obj := range []models.TipoObjeto{models.TipoObjetoConsumo, models.TipoObjetoPermanente, models.TipoObjetoServico} {
			if !TipoDocumentoAplicavel(tipo, obj, false) {
				t.Fatalf("documento sem restrição deveria se aplicar a %v", obj)
			}
		}
	})

	t.Run("documento restrito a SERVICO não se aplica a CONSUMO/PERMANENTE", func(t *testing.T) {
		tipo := models.TipoDocumento{Nome: "Boleto DAM", RestritoTipoObjeto: &servico}
		if TipoDocumentoAplicavel(tipo, models.TipoObjetoConsumo, false) {
			t.Fatal("não deveria se aplicar a CONSUMO")
		}
		if TipoDocumentoAplicavel(tipo, models.TipoObjetoPermanente, false) {
			t.Fatal("não deveria se aplicar a PERMANENTE")
		}
		if !TipoDocumentoAplicavel(tipo, models.TipoObjetoServico, false) {
			t.Fatal("deveria se aplicar a SERVICO")
		}
	})

	t.Run("documento restrito a terceirização só se aplica quando a flag do contrato é true", func(t *testing.T) {
		tipo := models.TipoDocumento{Nome: "Protocolo GFIP", RestritoTerceirizacao: true}
		if TipoDocumentoAplicavel(tipo, models.TipoObjetoServico, false) {
			t.Fatal("não deveria se aplicar sem ExigeFiscalizacaoTerceirizacao")
		}
		if !TipoDocumentoAplicavel(tipo, models.TipoObjetoServico, true) {
			t.Fatal("deveria se aplicar com ExigeFiscalizacaoTerceirizacao=true")
		}
	})
}

func assertContains(t *testing.T, got []string, want string) {
	t.Helper()
	for _, g := range got {
		if g == want {
			return
		}
	}
	t.Fatalf("esperava encontrar %q em %v", want, got)
}

func assertSameElements(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tamanhos diferentes: got=%v want=%v", got, want)
	}
	set := make(map[string]bool, len(want))
	for _, w := range want {
		set[w] = true
	}
	for _, g := range got {
		if !set[g] {
			t.Fatalf("elemento inesperado %q: got=%v want=%v", g, got, want)
		}
	}
}
