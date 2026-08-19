package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/testutil"
)

// criarDocxDeTeste monta um .docx mínimo, mas real (Office Open XML
// válido o bastante pra github.com/lukasjarosch/go-docx abrir), com um
// parágrafo por item de `paragrafosXML` — cada string já é o XML interno
// do parágrafo (permite montar tanto um placeholder inteiro num só
// <w:r> quanto um FRAGMENTADO em múltiplos <w:r>, o cenário real que
// motivou a escolha desta biblioteca, ver o comentário em
// modelo_documento_render.go).
func criarDocxDeTeste(t *testing.T, paragrafosXML ...string) []byte {
	t.Helper()

	var corpo bytes.Buffer
	corpo.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	corpo.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, p := range paragrafosXML {
		corpo.WriteString(p)
	}
	corpo.WriteString(`</w:body></w:document>`)

	arquivos := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": corpo.String(),
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for nome, conteudo := range arquivos {
		w, err := zw.Create(nome)
		if err != nil {
			t.Fatalf("criar %s no zip de teste: %v", nome, err)
		}
		if _, err := w.Write([]byte(conteudo)); err != nil {
			t.Fatalf("escrever %s no zip de teste: %v", nome, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("fechar zip de teste: %v", err)
	}

	return buf.Bytes()
}

// extrairTextoDocumentXML abre o .docx resultante e devolve o
// word/document.xml cru, só pra asserção textual simples nos testes
// abaixo (não precisamos de um parser XML completo pra confirmar que um
// valor foi substituído).
func extrairTextoDocumentXML(t *testing.T, conteudoDocx []byte) string {
	t.Helper()

	zr, err := zip.NewReader(bytes.NewReader(conteudoDocx), int64(len(conteudoDocx)))
	if err != nil {
		t.Fatalf("abrir .docx resultante como zip: %v", err)
	}
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("abrir word/document.xml do resultado: %v", err)
		}
		defer func() { _ = rc.Close() }()
		var out bytes.Buffer
		if _, err := out.ReadFrom(rc); err != nil {
			t.Fatalf("ler word/document.xml do resultado: %v", err)
		}
		return out.String()
	}
	t.Fatal("word/document.xml não encontrado no .docx resultante")
	return ""
}

func salvarComoVersaoAtiva(t *testing.T, dir string, conteudo []byte) *models.ModeloDocumento {
	t.Helper()

	caminho := filepath.Join(dir, "modelo.docx")
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
		t.Fatalf("gravar .docx de teste em disco: %v", err)
	}

	versao := &models.ModeloDocumentoVersao{ID: uuid.New(), CaminhoStorage: caminho}
	gatilho := models.GatilhoAtesto
	return &models.ModeloDocumento{
		ID:          uuid.New(),
		Categoria:   "Atesto",
		Gatilho:     &gatilho,
		VersaoAtiva: versao,
	}
}

func TestRenderizarComModelo(t *testing.T) {
	ctx := context.Background()

	t.Run("sem modelo cadastrado pro gatilho: usado=false, sem erro (fallback fpdf)", func(t *testing.T) {
		modeloRepo := testutil.NewFakeModeloDocumentoRepository()

		conteudo, usado, err := renderizarComModelo(ctx, modeloRepo, models.GatilhoAtesto, CamposMerge{})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if usado {
			t.Fatal("esperava usado=false sem modelo cadastrado")
		}
		if conteudo != nil {
			t.Fatal("esperava conteudo=nil sem modelo cadastrado")
		}
	})

	t.Run("placeholder num único <w:r>: substitui corretamente", func(t *testing.T) {
		docxBytes := criarDocxDeTeste(t,
			`<w:p><w:r><w:t>Contrato {numero_contrato}, fiscal {fiscal_nome}.</w:t></w:r></w:p>`,
		)
		modelo := salvarComoVersaoAtiva(t, t.TempDir(), docxBytes)
		modeloRepo := testutil.NewFakeModeloDocumentoRepository(modelo)

		conteudo, usado, err := renderizarComModelo(ctx, modeloRepo, models.GatilhoAtesto, CamposMerge{
			"numero_contrato": "125/2026",
			"fiscal_nome":     "Maria Fiscal",
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !usado {
			t.Fatal("esperava usado=true com modelo cadastrado")
		}

		texto := extrairTextoDocumentXML(t, conteudo)
		if !bytes.Contains([]byte(texto), []byte("Contrato 125/2026, fiscal Maria Fiscal.")) {
			t.Fatalf("placeholder não substituído corretamente, XML resultante: %s", texto)
		}
	})

	// Cenário que justifica a escolha da biblioteca (ver o comentário em
	// modelo_documento_render.go): o Word frequentemente fragmenta um
	// placeholder em múltiplos <w:r> — um strings.Replace ingênuo falharia
	// aqui.
	t.Run("placeholder fragmentado em múltiplos <w:r>: ainda substitui corretamente", func(t *testing.T) {
		docxBytes := criarDocxDeTeste(t,
			`<w:p><w:r><w:t>Número: {numero_</w:t></w:r><w:r><w:t>contrato} — fim.</w:t></w:r></w:p>`,
		)
		modelo := salvarComoVersaoAtiva(t, t.TempDir(), docxBytes)
		modeloRepo := testutil.NewFakeModeloDocumentoRepository(modelo)

		conteudo, usado, err := renderizarComModelo(ctx, modeloRepo, models.GatilhoAtesto, CamposMerge{
			"numero_contrato": "999/2026",
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !usado {
			t.Fatal("esperava usado=true com modelo cadastrado")
		}

		texto := extrairTextoDocumentXML(t, conteudo)
		if !bytes.Contains([]byte(texto), []byte("Número: 999/2026")) {
			t.Fatalf("placeholder fragmentado não substituído corretamente, XML resultante: %s", texto)
		}
	})

	t.Run("modelo sem versão ativa (estado inconsistente): usado=false, sem erro", func(t *testing.T) {
		gatilho := models.GatilhoAtesto
		modelo := &models.ModeloDocumento{ID: uuid.New(), Categoria: "Atesto", Gatilho: &gatilho, VersaoAtiva: nil}
		modeloRepo := testutil.NewFakeModeloDocumentoRepository(modelo)

		_, usado, err := renderizarComModelo(ctx, modeloRepo, models.GatilhoAtesto, CamposMerge{})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if usado {
			t.Fatal("esperava usado=false sem versão ativa")
		}
	})

	t.Run("erro de repositório (não not-found) é propagado", func(t *testing.T) {
		modeloRepo := &modeloDocumentoRepoComErro{err: errors.New("boom")}

		_, _, err := renderizarComModelo(ctx, modeloRepo, models.GatilhoAtesto, CamposMerge{})
		if err == nil {
			t.Fatal("esperava erro propagado")
		}
	})
}

// modeloDocumentoRepoComErro é um dublê mínimo só pra simular um erro de
// infraestrutura (não repository.ErrModeloDocumentoNotFound) vindo de
// FindAtivoByGatilho — o único caso não coberto pelo
// FakeModeloDocumentoRepository, que sempre devolve ErrModeloDocumentoNotFound
// quando não há match.
type modeloDocumentoRepoComErro struct {
	err error
}

func (m *modeloDocumentoRepoComErro) Create(ctx context.Context, modelo *models.ModeloDocumento) error {
	return m.err
}
func (m *modeloDocumentoRepoComErro) Update(ctx context.Context, modelo *models.ModeloDocumento) error {
	return m.err
}
func (m *modeloDocumentoRepoComErro) FindByID(ctx context.Context, id uuid.UUID) (*models.ModeloDocumento, error) {
	return nil, m.err
}
func (m *modeloDocumentoRepoComErro) FindByCategoria(ctx context.Context, categoria string) (*models.ModeloDocumento, error) {
	return nil, m.err
}
func (m *modeloDocumentoRepoComErro) List(ctx context.Context) ([]models.ModeloDocumento, error) {
	return nil, m.err
}
func (m *modeloDocumentoRepoComErro) FindAtivoByGatilho(ctx context.Context, gatilho models.TipoGatilhoModelo) (*models.ModeloDocumento, error) {
	return nil, m.err
}

var _ repository.ModeloDocumentoRepository = (*modeloDocumentoRepoComErro)(nil)
