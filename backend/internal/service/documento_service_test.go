package service

import (
	"context"
	"errors"
	"io"
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

// fakeProcessoPagamentoRepositoryComContrato devolve um processo com
// Contrato preenchido (TipoObjeto/ExigeFiscalizacaoTerceirizacao
// configuráveis), pra exercitar a checagem de TipoDocumentoAplicavel em
// DocumentoService.Upload — o fake "OK" acima devolve Contrato nil de
// propósito (caminho fail-open), este aqui testa o caminho real.
// etapaAtualID alimenta a checagem de RequisitosAcumulados (só é possível
// anexar um tipo que já faz parte do checklist até a etapa atual) — zero
// value (0) faz RequisitosAcumulados devolver uma lista vazia, então os
// testes que exercitam documentos condicionais de Etapa 5 (Boleto DAM
// etc.) precisam setar etapaAtualID: 5 explicitamente.
type fakeProcessoPagamentoRepositoryComContrato struct {
	fakeProcessoPagamentoRepositoryOK
	tipoObjeto                     models.TipoObjeto
	exigeFiscalizacaoTerceirizacao bool
	etapaAtualID                   int
}

func (f *fakeProcessoPagamentoRepositoryComContrato) FindByID(ctx context.Context, id uuid.UUID) (*models.ProcessoPagamento, error) {
	return &models.ProcessoPagamento{
		ID:           id,
		EtapaAtualID: f.etapaAtualID,
		Contrato: &models.Contrato{
			TipoObjeto:                     f.tipoObjeto,
			ExigeFiscalizacaoTerceirizacao: f.exigeFiscalizacaoTerceirizacao,
		},
	}, nil
}

// fakeTipoDocumentoRepositoryRestrito devolve um TipoDocumento restrito a
// SERVICO — simula "Boleto DAM"/"Planilha de Medição de Serviços".
type fakeTipoDocumentoRepositoryRestrito struct{}

func (f *fakeTipoDocumentoRepositoryRestrito) List(ctx context.Context) ([]models.TipoDocumento, error) {
	return nil, nil
}
func (f *fakeTipoDocumentoRepositoryRestrito) FindByID(ctx context.Context, id int) (*models.TipoDocumento, error) {
	restrito := models.TipoObjetoServico
	return &models.TipoDocumento{ID: id, Nome: "Boleto DAM", RestritoTipoObjeto: &restrito}, nil
}
func (f *fakeTipoDocumentoRepositoryRestrito) FindByNome(ctx context.Context, nome string) (*models.TipoDocumento, error) {
	restrito := models.TipoObjetoServico
	return &models.TipoDocumento{Nome: nome, RestritoTipoObjeto: &restrito}, nil
}

// TestDocumentoServiceUpload_TipoNaoAplicavel é o teste de regressão pro
// filtro por tipo de contrato: um documento restrito a SERVICO (ex:
// "Boleto DAM") não pode ser anexado a um processo de contrato
// CONSUMO/PERMANENTE, mesmo que o cliente force o tipo_documento_id no
// request (o select do frontend já filtra, mas a regra de verdade
// precisa estar no backend).
func TestDocumentoServiceUpload_TipoNaoAplicavel(t *testing.T) {
	storageDir := t.TempDir()
	docRepo := &fakeDocumentoAnexoRepository{}
	tipoDocRepo := &fakeTipoDocumentoRepositoryRestrito{}

	t.Run("contrato CONSUMO rejeita documento restrito a SERVICO", func(t *testing.T) {
		processoRepo := &fakeProcessoPagamentoRepositoryComContrato{tipoObjeto: models.TipoObjetoConsumo, etapaAtualID: 5}
		svc := NewDocumentoService(docRepo, tipoDocRepo, processoRepo, storageDir)

		_, err := svc.Upload(context.Background(), uuid.New(), 1, "boleto.pdf", []byte("conteúdo"), uuid.New(), nil)
		if !errors.Is(err, ErrTipoDocumentoNaoAplicavel) {
			t.Fatalf("esperava ErrTipoDocumentoNaoAplicavel, veio %v", err)
		}
	})

	t.Run("contrato SERVICO aceita o mesmo documento", func(t *testing.T) {
		// etapaAtualID: 5 — "Boleto DAM" só entra no checklist acumulado
		// (RequisitosAcumulados) como condicional da Etapa 5 pra contratos
		// SERVICO; sem isso a nova checagem de "documento ainda não exigido
		// nesta etapa" rejeitaria mesmo um contrato SERVICO válido.
		processoRepo := &fakeProcessoPagamentoRepositoryComContrato{tipoObjeto: models.TipoObjetoServico, etapaAtualID: 5}
		svc := NewDocumentoService(docRepo, tipoDocRepo, processoRepo, storageDir)

		_, err := svc.Upload(context.Background(), uuid.New(), 1, "boleto.pdf", []byte("conteúdo servico"), uuid.New(), nil)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
	})
}

// TestDocumentoServiceUpload_UmPorTipo cobre a regra pedida pelo usuário:
// no máximo um documento anexado de cada TipoDocumento por processo (ex:
// não pode ter dois "Pré-Empenho"). Também confirma que essa regra é
// DISTINTA da deduplicação por hash já existente: reenviar o MESMO
// arquivo (mesmo conteúdo, logo mesmo hash) continua reaproveitando o
// registro existente via FindByProcessoAndHash, sem cair no erro de
// uniqueness — só um arquivo DIFERENTE do mesmo tipo é rejeitado.
func TestDocumentoServiceUpload_UmPorTipo(t *testing.T) {
	storageDir := t.TempDir()
	docRepo := &fakeDocumentoAnexoRepository{}
	svc := NewDocumentoService(docRepo, &fakeTipoDocumentoRepositoryOK{}, &fakeProcessoPagamentoRepositoryOK{}, storageDir)

	processoID := uuid.New()

	primeiro, err := svc.Upload(context.Background(), processoID, 1, "pre-empenho.pdf", []byte("conteúdo do primeiro pré-empenho"), uuid.New(), nil)
	if err != nil {
		t.Fatalf("falha ao preparar documento de teste: %v", err)
	}

	t.Run("reenviar o mesmo arquivo (mesmo hash) reaproveita o registro existente", func(t *testing.T) {
		doc, err := svc.Upload(context.Background(), processoID, 1, "pre-empenho.pdf", []byte("conteúdo do primeiro pré-empenho"), uuid.New(), nil)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if doc.ID != primeiro.ID {
			t.Fatalf("esperava reaproveitar o registro existente (dedup por hash), veio um ID diferente: %v != %v", doc.ID, primeiro.ID)
		}
	})

	t.Run("enviar um arquivo diferente do mesmo tipo é rejeitado", func(t *testing.T) {
		_, err := svc.Upload(context.Background(), processoID, 1, "pre-empenho-v2.pdf", []byte("conteúdo diferente, mesmo tipo"), uuid.New(), nil)
		if !errors.Is(err, ErrTipoDocumentoJaAnexado) {
			t.Fatalf("esperava ErrTipoDocumentoJaAnexado, veio %v", err)
		}
	})

	t.Run("mesmo tipo em outro processo não é bloqueado", func(t *testing.T) {
		outroProcessoID := uuid.New()
		_, err := svc.Upload(context.Background(), outroProcessoID, 1, "pre-empenho.pdf", []byte("conteúdo em outro processo"), uuid.New(), nil)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
	})

	t.Run("depois de excluído, um novo documento do mesmo tipo pode ser enviado", func(t *testing.T) {
		if err := svc.Excluir(context.Background(), processoID, primeiro.ID); err != nil {
			t.Fatalf("falha ao excluir documento de teste: %v", err)
		}
		_, err := svc.Upload(context.Background(), processoID, 1, "pre-empenho-v2.pdf", []byte("conteúdo diferente, mesmo tipo"), uuid.New(), nil)
		if err != nil {
			t.Fatalf("erro inesperado depois de excluir o anterior: %v", err)
		}
	})
}

// fakeTipoDocumentoRepositoryComNome devolve um TipoDocumento com o nome
// configurado (sem nenhuma restrição de contrato) — usado pra testar a
// checagem de "documento ainda não exigido nesta etapa" com nomes reais
// do checklist (ver checklist.go), diferente de fakeTipoDocumentoRepositoryOK
// (nome fixo "Tipo Teste", que nunca bate com nenhum checklist de verdade).
type fakeTipoDocumentoRepositoryComNome struct {
	nome string
}

func (f *fakeTipoDocumentoRepositoryComNome) List(ctx context.Context) ([]models.TipoDocumento, error) {
	return nil, nil
}
func (f *fakeTipoDocumentoRepositoryComNome) FindByID(ctx context.Context, id int) (*models.TipoDocumento, error) {
	return &models.TipoDocumento{ID: id, Nome: f.nome}, nil
}
func (f *fakeTipoDocumentoRepositoryComNome) FindByNome(ctx context.Context, nome string) (*models.TipoDocumento, error) {
	return &models.TipoDocumento{Nome: nome}, nil
}

// TestDocumentoServiceUpload_NaoExigidoNestaEtapa cobre a regra pedida
// pelo usuário: "quero que apareça somente os documentos que realmente
// podem ser inseridos na etapa" — o backend rejeita o upload de um tipo
// que ainda não faz parte do checklist cumulado até a etapa atual do
// processo (ex: uma CND, só exigida na Etapa 5, enquanto o processo
// ainda está na Etapa 1), mas continua aceitando tipos de etapas já
// concluídas (a regra cumulativa de RequisitosAcumulados/checklist
// precisa continuar permitindo reenviar um obrigatório antigo excluído).
func TestDocumentoServiceUpload_NaoExigidoNestaEtapa(t *testing.T) {
	storageDir := t.TempDir()
	docRepo := &fakeDocumentoAnexoRepository{}

	t.Run("documento de uma etapa futura é rejeitado", func(t *testing.T) {
		tipoDocRepo := &fakeTipoDocumentoRepositoryComNome{nome: "CND Federal"} // só exigido na Etapa 5
		processoRepo := &fakeProcessoPagamentoRepositoryComContrato{tipoObjeto: models.TipoObjetoConsumo, etapaAtualID: 1}
		svc := NewDocumentoService(docRepo, tipoDocRepo, processoRepo, storageDir)

		_, err := svc.Upload(context.Background(), uuid.New(), 1, "cnd.pdf", []byte("conteúdo"), uuid.New(), nil)
		if !errors.Is(err, ErrTipoDocumentoNaoExigidoAinda) {
			t.Fatalf("esperava ErrTipoDocumentoNaoExigidoAinda, veio %v", err)
		}
	})

	t.Run("documento da etapa atual é aceito", func(t *testing.T) {
		tipoDocRepo := &fakeTipoDocumentoRepositoryComNome{nome: "Nota de Empenho"} // exigido na Etapa 3
		processoRepo := &fakeProcessoPagamentoRepositoryComContrato{tipoObjeto: models.TipoObjetoConsumo, etapaAtualID: 3}
		svc := NewDocumentoService(docRepo, tipoDocRepo, processoRepo, storageDir)

		_, err := svc.Upload(context.Background(), uuid.New(), 1, "empenho.pdf", []byte("conteúdo"), uuid.New(), nil)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
	})

	t.Run("documento de uma etapa já concluída continua aceito (regra cumulativa)", func(t *testing.T) {
		tipoDocRepo := &fakeTipoDocumentoRepositoryComNome{nome: "Pré-Empenho"} // exigido na Etapa 1
		processoRepo := &fakeProcessoPagamentoRepositoryComContrato{tipoObjeto: models.TipoObjetoConsumo, etapaAtualID: 4}
		svc := NewDocumentoService(docRepo, tipoDocRepo, processoRepo, storageDir)

		_, err := svc.Upload(context.Background(), uuid.New(), 1, "pre-empenho.pdf", []byte("conteúdo"), uuid.New(), nil)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
	})
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

// TestDocumentoServiceBaixar cobre o caminho feliz e as duas rejeições de
// DocumentoService.Baixar: documento inexistente e documento que existe
// mas pertence a outro processo (a defesa em profundidade documentada no
// comentário do método).
func TestDocumentoServiceBaixar(t *testing.T) {
	storageDir := t.TempDir()
	docRepo := &fakeDocumentoAnexoRepository{}
	svc := NewDocumentoService(docRepo, &fakeTipoDocumentoRepositoryOK{}, &fakeProcessoPagamentoRepositoryOK{}, storageDir)

	processoID := uuid.New()
	documento, err := svc.Upload(context.Background(), processoID, 1, "boleto.pdf", []byte("conteúdo de teste"), uuid.New(), nil)
	if err != nil {
		t.Fatalf("falha ao preparar documento de teste: %v", err)
	}

	t.Run("caminho feliz devolve o arquivo aberto", func(t *testing.T) {
		arquivo, doc, err := svc.Baixar(context.Background(), processoID, documento.ID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		defer func() { _ = arquivo.Close() }()

		conteudo, err := io.ReadAll(arquivo)
		if err != nil {
			t.Fatalf("falha ao ler arquivo devolvido: %v", err)
		}
		if string(conteudo) != "conteúdo de teste" {
			t.Fatalf("conteúdo inesperado: %q", conteudo)
		}
		if doc.NomeArquivo != "boleto.pdf" {
			t.Fatalf("nome de arquivo inesperado: %q", doc.NomeArquivo)
		}
	})

	t.Run("processoID que não é o dono real é rejeitado", func(t *testing.T) {
		_, _, err := svc.Baixar(context.Background(), uuid.New(), documento.ID)
		if !errors.Is(err, repository.ErrDocumentoNotFound) {
			t.Fatalf("esperava ErrDocumentoNotFound, veio %v", err)
		}
	})

	t.Run("documento inexistente é rejeitado", func(t *testing.T) {
		_, _, err := svc.Baixar(context.Background(), processoID, uuid.New())
		if !errors.Is(err, repository.ErrDocumentoNotFound) {
			t.Fatalf("esperava ErrDocumentoNotFound, veio %v", err)
		}
	})
}

// TestDocumentoServiceExcluir cobre o caminho feliz (registro E arquivo
// físico removidos) e as rejeições — incluindo o caso em que o
// processoID informado não é o dono real, onde nem o registro nem o
// arquivo podem ter sido tocados.
func TestDocumentoServiceExcluir(t *testing.T) {
	storageDir := t.TempDir()
	docRepo := &fakeDocumentoAnexoRepository{}
	svc := NewDocumentoService(docRepo, &fakeTipoDocumentoRepositoryOK{}, &fakeProcessoPagamentoRepositoryOK{}, storageDir)

	processoID := uuid.New()
	documento, err := svc.Upload(context.Background(), processoID, 1, "boleto.pdf", []byte("conteúdo de teste"), uuid.New(), nil)
	if err != nil {
		t.Fatalf("falha ao preparar documento de teste: %v", err)
	}

	t.Run("processoID que não é o dono real é rejeitado, nada é removido", func(t *testing.T) {
		err := svc.Excluir(context.Background(), uuid.New(), documento.ID)
		if !errors.Is(err, repository.ErrDocumentoNotFound) {
			t.Fatalf("esperava ErrDocumentoNotFound, veio %v", err)
		}
		if _, err := os.Stat(documento.CaminhoStorage); err != nil {
			t.Fatalf("arquivo não deveria ter sido removido: %v", err)
		}
	})

	t.Run("excluir documento inexistente é rejeitado", func(t *testing.T) {
		err := svc.Excluir(context.Background(), processoID, uuid.New())
		if !errors.Is(err, repository.ErrDocumentoNotFound) {
			t.Fatalf("esperava ErrDocumentoNotFound, veio %v", err)
		}
	})

	t.Run("caminho feliz remove o registro e o arquivo físico", func(t *testing.T) {
		if err := svc.Excluir(context.Background(), processoID, documento.ID); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if _, err := docRepo.FindByID(context.Background(), documento.ID); !errors.Is(err, repository.ErrDocumentoNotFound) {
			t.Fatalf("esperava documento removido do repositório, veio %v", err)
		}
		if _, err := os.Stat(documento.CaminhoStorage); !os.IsNotExist(err) {
			t.Fatalf("esperava arquivo físico removido, stat err=%v", err)
		}
	})
}
