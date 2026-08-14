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

var _ repository.DocumentoAnexoRepository = (*fakeDocumentoAnexoRepository)(nil)

func comDocumento(processoID uuid.UUID, tipoNome string) models.DocumentoAnexo {
	return models.DocumentoAnexo{
		ProcessoPagamentoID: processoID,
		TipoDocumento:       &models.TipoDocumento{Nome: tipoNome},
	}
}

func TestRequisitosEtapa(t *testing.T) {
	t.Run("etapa 1 exige OF, Pré-Empenho e Ofício, independente do tipo de objeto", func(t *testing.T) {
		got := RequisitosEtapa(1, models.TipoObjetoConsumo)
		want := []string{"Ordem de Fornecimento (OF)", "Pré-Empenho", "Ofício de Solicitação"}
		assertSameElements(t, got, want)
	})

	t.Run("etapa 5 para CONSUMO não inclui os itens condicionais de SERVICO", func(t *testing.T) {
		got := RequisitosEtapa(5, models.TipoObjetoConsumo)
		for _, condicional := range checklistCondicionalServico {
			for _, r := range got {
				if r == condicional {
					t.Fatalf("etapa 5 para CONSUMO não deveria exigir %q", condicional)
				}
			}
		}
	})

	t.Run("etapa 5 para SERVICO inclui Planilha de Medição e Boleto DAM", func(t *testing.T) {
		got := RequisitosEtapa(5, models.TipoObjetoServico)
		assertContains(t, got, "Planilha de Medição de Serviços")
		assertContains(t, got, "Boleto DAM")
	})

	t.Run("etapas 2 e 6 não têm checklist de saída", func(t *testing.T) {
		if got := RequisitosEtapa(2, models.TipoObjetoServico); len(got) != 0 {
			t.Fatalf("etapa 2 deveria ter checklist vazio, veio %v", got)
		}
		if got := RequisitosEtapa(6, models.TipoObjetoServico); len(got) != 0 {
			t.Fatalf("etapa 6 deveria ter checklist vazio, veio %v", got)
		}
	})
}

func TestChecklistPendente(t *testing.T) {
	ctx := context.Background()
	processoID := uuid.New()

	t.Run("sem documentos anexados, tudo pendente", func(t *testing.T) {
		repo := &fakeDocumentoAnexoRepository{}
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 1, models.TipoObjetoConsumo)
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
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 1, models.TipoObjetoConsumo)
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
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 1, models.TipoObjetoConsumo)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		assertContains(t, pendentes, "Ordem de Fornecimento (OF)")
	})

	t.Run("SERVICO com certidões mas sem os condicionais ainda fica pendente", func(t *testing.T) {
		docs := []models.DocumentoAnexo{}
		for _, nome := range checklistBase[5] {
			docs = append(docs, comDocumento(processoID, nome))
		}
		repo := &fakeDocumentoAnexoRepository{documentos: docs}
		pendentes, err := ChecklistPendente(ctx, repo, processoID, 5, models.TipoObjetoServico)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		assertSameElements(t, pendentes, checklistCondicionalServico)
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
