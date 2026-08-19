package service_test

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/service"
	"projeto-selene/internal/testutil"
)

// docxMinimoDeTeste monta um .docx real (Office Open XML mínimo, mas
// válido o bastante pra github.com/lukasjarosch/go-docx abrir) — usado
// só pra passar pela validação de "é um .docx de verdade"
// (service.ErrArquivoNaoEDocx); o conteúdo do placeholder em si já é
// testado em modelo_documento_render_test.go.
func docxMinimoDeTeste(t *testing.T) []byte {
	t.Helper()

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
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
  <w:p><w:r><w:t>Modelo de teste {numero_contrato}.</w:t></w:r></w:p>
</w:body></w:document>`,
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

func TestModeloDocumentoService_CriarCategoria(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	modeloRepo := repository.NewModeloDocumentoRepository(db)
	versaoRepo := repository.NewModeloDocumentoVersaoRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := service.NewModeloDocumentoService(db, modeloRepo, versaoRepo, t.TempDir())

	keycloakID := "admin-modelos-teste"
	admin := &models.User{KeycloakID: &keycloakID, Nome: "Admin Teste", Email: "admin.modelos@teste.local", IsAdmin: true}
	if err := userRepo.Create(ctx, admin); err != nil {
		t.Fatalf("falha ao criar usuário de teste: %v", err)
	}

	docxValido := docxMinimoDeTeste(t)

	t.Run("categoria vazia é rejeitada", func(t *testing.T) {
		_, err := svc.CriarCategoria(ctx, "   ", nil, "modelo.docx", docxValido, admin.ID)
		if err == nil {
			t.Fatal("esperava erro para categoria vazia")
		}
	})

	t.Run("arquivo que não é .docx é rejeitado", func(t *testing.T) {
		_, err := svc.CriarCategoria(ctx, "Categoria Inválida "+uuid.New().String(), nil, "modelo.docx", []byte("não é um docx"), admin.ID)
		if err == nil {
			t.Fatal("esperava erro para arquivo inválido")
		}
	})

	t.Run("caminho feliz cria categoria com versão ativa", func(t *testing.T) {
		categoria := "Ofício de Teste " + uuid.New().String()
		gatilho := models.GatilhoNotificacaoDescumprimento

		modelo, err := svc.CriarCategoria(ctx, categoria, &gatilho, "oficio.docx", docxValido, admin.ID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if modelo.VersaoAtiva == nil {
			t.Fatal("esperava VersaoAtiva preenchida")
		}
		if modelo.VersaoAtiva.NomeArquivo != "oficio.docx" {
			t.Fatalf("esperava nome de arquivo 'oficio.docx', veio %q", modelo.VersaoAtiva.NomeArquivo)
		}

		conteudo, nome, err := svc.Baixar(ctx, modelo.ID, nil)
		if err != nil {
			t.Fatalf("erro ao baixar versão ativa: %v", err)
		}
		if nome != "oficio.docx" {
			t.Fatalf("esperava nome 'oficio.docx' no download, veio %q", nome)
		}
		if len(conteudo) != len(docxValido) {
			t.Fatalf("conteúdo baixado difere do enviado (tamanhos %d vs %d)", len(conteudo), len(docxValido))
		}
	})

	t.Run("categoria duplicada (case-insensitive) é rejeitada", func(t *testing.T) {
		categoria := "Categoria Duplicada " + uuid.New().String()
		if _, err := svc.CriarCategoria(ctx, categoria, nil, "a.docx", docxValido, admin.ID); err != nil {
			t.Fatalf("falha ao criar a primeira categoria: %v", err)
		}

		_, err := svc.CriarCategoria(ctx, strings.ToUpper(categoria), nil, "b.docx", docxValido, admin.ID)
		if err == nil {
			t.Fatal("esperava erro de categoria duplicada")
		}
	})

	t.Run("gatilho já associado a outra categoria é rejeitado", func(t *testing.T) {
		gatilho := models.GatilhoAtesto
		if _, err := svc.CriarCategoria(ctx, "Atesto Original "+uuid.New().String(), &gatilho, "a.docx", docxValido, admin.ID); err != nil {
			t.Fatalf("falha ao criar a primeira categoria com o gatilho: %v", err)
		}

		_, err := svc.CriarCategoria(ctx, "Atesto Duplicado "+uuid.New().String(), &gatilho, "b.docx", docxValido, admin.ID)
		if err == nil {
			t.Fatal("esperava erro de gatilho já associado")
		}
	})

	t.Run("NovaVersao substitui a versão ativa e preserva o histórico", func(t *testing.T) {
		categoria := "Relatório de Teste " + uuid.New().String()
		modelo, err := svc.CriarCategoria(ctx, categoria, nil, "v1.docx", docxValido, admin.ID)
		if err != nil {
			t.Fatalf("erro ao criar categoria: %v", err)
		}
		versaoOriginalID := modelo.VersaoAtiva.ID

		atualizado, err := svc.NovaVersao(ctx, modelo.ID, "v2.docx", docxValido, admin.ID)
		if err != nil {
			t.Fatalf("erro ao publicar nova versão: %v", err)
		}
		if atualizado.VersaoAtiva.NomeArquivo != "v2.docx" {
			t.Fatalf("esperava versão ativa 'v2.docx', veio %q", atualizado.VersaoAtiva.NomeArquivo)
		}
		if len(atualizado.Versoes) != 2 {
			t.Fatalf("esperava 2 versões no histórico, veio %d", len(atualizado.Versoes))
		}

		_, nomeAntigo, err := svc.Baixar(ctx, modelo.ID, &versaoOriginalID)
		if err != nil {
			t.Fatalf("erro ao baixar versão antiga do histórico: %v", err)
		}
		if nomeAntigo != "v1.docx" {
			t.Fatalf("esperava conseguir baixar a versão antiga 'v1.docx', veio %q", nomeAntigo)
		}
	})

	t.Run("AtualizarCategoria renomeia e troca gatilho", func(t *testing.T) {
		categoria := "Categoria Renomeável " + uuid.New().String()
		modelo, err := svc.CriarCategoria(ctx, categoria, nil, "a.docx", docxValido, admin.ID)
		if err != nil {
			t.Fatalf("erro ao criar categoria: %v", err)
		}

		novoNome := "Categoria Renomeada " + uuid.New().String()
		gatilho := models.GatilhoMinutaAditivo
		gatilhoPtr := &gatilho
		atualizado, err := svc.AtualizarCategoria(ctx, modelo.ID, service.AtualizarCategoriaInput{
			Categoria: &novoNome,
			Gatilho:   &gatilhoPtr,
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if atualizado.Categoria != novoNome {
			t.Fatalf("esperava categoria %q, veio %q", novoNome, atualizado.Categoria)
		}
		if atualizado.Gatilho == nil || *atualizado.Gatilho != models.GatilhoMinutaAditivo {
			t.Fatal("esperava gatilho MINUTA_ADITIVO associado")
		}
	})
}
