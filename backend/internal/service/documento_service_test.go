package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// fakeProcessoPagamentoRepositoryOK só implementa o suficiente pra
// DocumentoService.Upload achar o processo (FindByID sempre "existe") —
// os demais métodos da interface não são exercitados neste teste.
type fakeProcessoPagamentoRepositoryOK struct{}

func (f *fakeProcessoPagamentoRepositoryOK) Create(ctx context.Context, p *models.ProcessoPagamento) error {
	return nil
}
func (f *fakeProcessoPagamentoRepositoryOK) FindByID(ctx context.Context, id uuid.UUID) (*models.ProcessoPagamento, error) {
	return &models.ProcessoPagamento{ID: id}, nil
}
func (f *fakeProcessoPagamentoRepositoryOK) ListByEtapa(ctx context.Context, etapaID int, pagina repository.Pagina) (repository.ResultadoPaginado[models.ProcessoPagamento], error) {
	return repository.ResultadoPaginado[models.ProcessoPagamento]{}, nil
}
func (f *fakeProcessoPagamentoRepositoryOK) ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.ProcessoPagamento, error) {
	return nil, nil
}
func (f *fakeProcessoPagamentoRepositoryOK) Update(ctx context.Context, p *models.ProcessoPagamento) error {
	return nil
}
func (f *fakeProcessoPagamentoRepositoryOK) ListAtivosComContrato(ctx context.Context) ([]models.ProcessoPagamento, error) {
	return nil, nil
}

// fakeTipoDocumentoRepositoryOK: idem, só FindByID precisa "achar" o tipo.
type fakeTipoDocumentoRepositoryOK struct{}

func (f *fakeTipoDocumentoRepositoryOK) List(ctx context.Context) ([]models.TipoDocumento, error) {
	return nil, nil
}
func (f *fakeTipoDocumentoRepositoryOK) FindByID(ctx context.Context, id int) (*models.TipoDocumento, error) {
	return &models.TipoDocumento{ID: id, Nome: "Tipo Teste"}, nil
}
func (f *fakeTipoDocumentoRepositoryOK) FindByNome(ctx context.Context, nome string) (*models.TipoDocumento, error) {
	return &models.TipoDocumento{Nome: nome}, nil
}

func TestSanitizarNomeArquivo(t *testing.T) {
	casos := []struct {
		nome   string
		motivo string
	}{
		{"../../../../etc/passwd", "traversal Unix simples"},
		{"../../../../../../etc/shadow", "traversal Unix profundo"},
		{"..", "apenas dois pontos"},
		{".", "apenas um ponto"},
		{"", "vazio"},
	}

	for _, c := range casos {
		t.Run(c.motivo, func(t *testing.T) {
			resultado := sanitizarNomeArquivo(c.nome)
			if strings.ContainsAny(resultado, `/\`) {
				t.Fatalf("resultado ainda contém separador de caminho: %q (entrada: %q)", resultado, c.nome)
			}
			if resultado == ".." || resultado == "." || resultado == "" {
				t.Fatalf("resultado ainda é um componente de navegação/vazio: %q (entrada: %q)", resultado, c.nome)
			}
		})
	}

	t.Run("remove caracteres de controle (CRLF)", func(t *testing.T) {
		resultado := sanitizarNomeArquivo("boleto.pdf\r\nContent-Type: text/html")
		if strings.ContainsAny(resultado, "\r\n") {
			t.Fatalf("resultado ainda contém CR/LF: %q", resultado)
		}
	})

	t.Run("nome legítimo passa praticamente inalterado", func(t *testing.T) {
		resultado := sanitizarNomeArquivo("Nota_Fiscal-123.pdf")
		if resultado != "Nota_Fiscal-123.pdf" {
			t.Fatalf("esperava nome legítimo preservado, veio %q", resultado)
		}
	})
}

// TestDocumentoServiceUpload_PathTraversal é o teste de regressão para o
// achado de segurança: um filename malicioso não pode fazer o arquivo
// sair do diretório do processo (destDir) dentro de storageDir. Antes da
// correção, "../../../../evil.sh" escapava de destDir via filepath.Join
// (o prefixo hash_ só anula UM nível de "../").
func TestDocumentoServiceUpload_PathTraversal(t *testing.T) {
	storageDir := t.TempDir()

	docRepo := &fakeDocumentoAnexoRepository{}
	svc := NewDocumentoService(docRepo, &fakeTipoDocumentoRepositoryOK{}, &fakeProcessoPagamentoRepositoryOK{}, storageDir)

	processoID := uuid.New()
	nomesMaliciosos := []string{
		"../../../../evil.sh",
		"../../../../../../../../etc/passwd",
		"..\\..\\..\\evil.exe",
	}

	for _, nomeMalicioso := range nomesMaliciosos {
		t.Run(nomeMalicioso, func(t *testing.T) {
			doc, err := svc.Upload(context.Background(), processoID, 1, nomeMalicioso, []byte("conteúdo de teste"), uuid.New(), nil)
			if err != nil {
				t.Fatalf("upload falhou inesperadamente: %v", err)
			}

			// O arquivo gravado precisa estar DENTRO de storageDir/<processoID>/,
			// nunca fora dele.
			destDir := filepath.Join(storageDir, processoID.String())
			caminhoAbsoluto, err := filepath.Abs(doc.CaminhoStorage)
			if err != nil {
				t.Fatalf("falha ao resolver caminho absoluto: %v", err)
			}
			destDirAbsoluto, err := filepath.Abs(destDir)
			if err != nil {
				t.Fatalf("falha ao resolver destDir absoluto: %v", err)
			}

			rel, err := filepath.Rel(destDirAbsoluto, caminhoAbsoluto)
			if err != nil || strings.HasPrefix(rel, "..") {
				t.Fatalf("regressão de path traversal: arquivo gravado em %q, fora de %q (rel=%q)", caminhoAbsoluto, destDirAbsoluto, rel)
			}
		})
	}

	// Confirma que nenhum arquivo vazou para fora de storageDir (ex: como
	// "evil.sh" direto na raiz, fora da subpasta do processo).
	entradas, err := os.ReadDir(storageDir)
	if err != nil {
		t.Fatalf("falha ao ler storageDir: %v", err)
	}
	for _, e := range entradas {
		if e.Name() != processoID.String() {
			t.Fatalf("regressão de path traversal: entrada inesperada em storageDir: %q", e.Name())
		}
	}
}
