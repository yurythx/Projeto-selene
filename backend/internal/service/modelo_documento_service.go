package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	docx "github.com/lukasjarosch/go-docx"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// zipMagicBytes é a assinatura de arquivo de um .docx (na prática, um
// contêiner ZIP/Office Open XML) — checagem rasa que rejeita a maioria
// dos enganos óbvios (ex: subir um .pdf com a extensão trocada) antes de
// gastar um docx.OpenBytes. A validação de verdade é tentar abrir o
// arquivo como .docx logo em seguida (ver validarDocx) — a assinatura
// ZIP sozinha não garante Office Open XML válido (podia ser qualquer
// .zip/.xlsx/.pptx renomeado).
var zipMagicBytes = []byte{0x50, 0x4b, 0x03, 0x04}

// validarDocx confere se conteudo é um .docx que a biblioteca de merge
// consegue abrir — a mensagem de erro fica amigável (não vaza o erro cru
// de parsing XML pro admin), ErrArquivoNaoEDocx.
func validarDocx(conteudo []byte) error {
	if !bytes.HasPrefix(conteudo, zipMagicBytes) {
		return ErrArquivoNaoEDocx
	}
	doc, err := docx.OpenBytes(conteudo)
	if err != nil {
		return ErrArquivoNaoEDocx
	}
	doc.Close()
	return nil
}

// ModeloDocumentoService implementa o CRUD de Modelos de Documentos
// (Configurações, admin-only) — categorias de arquivo .docx com
// histórico de versões auditável. Ver o comentário em
// models.ModeloDocumento sobre a relação com os 4 gatilhos de geração.
type ModeloDocumentoService struct {
	// db só é usado pra abrir transações — leituras/escritas fora de
	// transação usam os repositories injetados, mesmo padrão de
	// KanbanService.
	db         *gorm.DB
	modeloRepo repository.ModeloDocumentoRepository
	versaoRepo repository.ModeloDocumentoVersaoRepository
	storageDir string
}

// NewModeloDocumentoService constrói um ModeloDocumentoService.
func NewModeloDocumentoService(
	db *gorm.DB,
	modeloRepo repository.ModeloDocumentoRepository,
	versaoRepo repository.ModeloDocumentoVersaoRepository,
	storageDir string,
) *ModeloDocumentoService {
	return &ModeloDocumentoService{db: db, modeloRepo: modeloRepo, versaoRepo: versaoRepo, storageDir: storageDir}
}

// CriarCategoria cadastra uma categoria nova de modelo de documento com
// sua primeira versão, atomicamente (categoria + versão nascem juntas ou
// nenhuma das duas).
func (s *ModeloDocumentoService) CriarCategoria(
	ctx context.Context,
	categoria string,
	gatilho *models.TipoGatilhoModelo,
	nomeArquivo string,
	conteudo []byte,
	enviadoPorID uuid.UUID,
) (*models.ModeloDocumento, error) {
	categoria = strings.TrimSpace(categoria)
	if categoria == "" {
		return nil, ErrCategoriaObrigatoria
	}
	if err := validarDocx(conteudo); err != nil {
		return nil, err
	}

	// ID pré-gerado (em vez de deixar BeforeCreate gerar) porque é
	// usado no caminho de armazenamento do arquivo antes do INSERT —
	// mesmo espírito de DocumentoService.Upload usar processoID.String()
	// (um ID já conhecido) na montagem do destDir.
	modelo := &models.ModeloDocumento{ID: uuid.New(), Categoria: categoria, Gatilho: gatilho}

	versao, err := s.gravarArquivoEmDisco(modelo.ID, nomeArquivo, conteudo, enviadoPorID)
	if err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		modeloRepoTx := repository.NewModeloDocumentoRepository(tx)
		versaoRepoTx := repository.NewModeloDocumentoVersaoRepository(tx)

		if err := modeloRepoTx.Create(ctx, modelo); err != nil {
			return err
		}
		if err := versaoRepoTx.Create(ctx, versao); err != nil {
			return err
		}
		modelo.VersaoAtivaID = &versao.ID
		return modeloRepoTx.Update(ctx, modelo)
	})
	if err != nil {
		return nil, fmt.Errorf("service: criar categoria de modelo de documento: %w", err)
	}

	return s.Buscar(ctx, modelo.ID)
}

// NovaVersao substitui o arquivo ativo de uma categoria já existente,
// preservando as versões anteriores no histórico (nunca apaga, ver o
// comentário em models.ModeloDocumentoVersao).
func (s *ModeloDocumentoService) NovaVersao(
	ctx context.Context,
	modeloID uuid.UUID,
	nomeArquivo string,
	conteudo []byte,
	enviadoPorID uuid.UUID,
) (*models.ModeloDocumento, error) {
	if err := validarDocx(conteudo); err != nil {
		return nil, err
	}

	modelo, err := s.modeloRepo.FindByID(ctx, modeloID)
	if err != nil {
		return nil, err
	}

	versao, err := s.gravarArquivoEmDisco(modelo.ID, nomeArquivo, conteudo, enviadoPorID)
	if err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		modeloRepoTx := repository.NewModeloDocumentoRepository(tx)
		versaoRepoTx := repository.NewModeloDocumentoVersaoRepository(tx)

		if err := versaoRepoTx.Create(ctx, versao); err != nil {
			return err
		}
		modelo.VersaoAtivaID = &versao.ID
		return modeloRepoTx.Update(ctx, modelo)
	})
	if err != nil {
		return nil, fmt.Errorf("service: publicar nova versão de modelo de documento: %w", err)
	}

	return s.Buscar(ctx, modelo.ID)
}

// AtualizarCategoriaInput agrupa os campos editáveis de uma categoria.
// Ponteiros distinguem "não informado" de "definido" — mesmo padrão de
// AtualizarContratoInput.
type AtualizarCategoriaInput struct {
	Categoria *string
	// Gatilho, se não-nil, atualiza a associação — um ponteiro pra
	// ponteiro porque *models.TipoGatilhoModelo(nil) é um valor válido
	// (remover a associação), distinto de "não informar este campo".
	Gatilho **models.TipoGatilhoModelo
}

// AtualizarCategoria aplica AtualizarCategoriaInput a uma categoria
// existente — não mexe no arquivo/versão ativa, só nos metadados.
func (s *ModeloDocumentoService) AtualizarCategoria(ctx context.Context, modeloID uuid.UUID, input AtualizarCategoriaInput) (*models.ModeloDocumento, error) {
	modelo, err := s.modeloRepo.FindByID(ctx, modeloID)
	if err != nil {
		return nil, err
	}

	if input.Categoria != nil {
		categoria := strings.TrimSpace(*input.Categoria)
		if categoria == "" {
			return nil, ErrCategoriaObrigatoria
		}
		modelo.Categoria = categoria
	}
	if input.Gatilho != nil {
		modelo.Gatilho = *input.Gatilho
	}

	if err := s.modeloRepo.Update(ctx, modelo); err != nil {
		return nil, err
	}

	return s.Buscar(ctx, modelo.ID)
}

// Listar retorna todas as categorias cadastradas.
func (s *ModeloDocumentoService) Listar(ctx context.Context) ([]models.ModeloDocumento, error) {
	modelos, err := s.modeloRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: listar modelos de documento: %w", err)
	}
	return modelos, nil
}

// Buscar retorna uma categoria pelo ID, com a versão ativa e o histórico
// completo pré-carregados (ver ModeloDocumentoRepository.FindByID).
func (s *ModeloDocumentoService) Buscar(ctx context.Context, id uuid.UUID) (*models.ModeloDocumento, error) {
	return s.modeloRepo.FindByID(ctx, id)
}

// Baixar lê do disco o conteúdo de uma versão — a ativa (versaoID=nil)
// ou uma específica do histórico.
func (s *ModeloDocumentoService) Baixar(ctx context.Context, modeloID uuid.UUID, versaoID *uuid.UUID) (conteudo []byte, nomeArquivo string, err error) {
	modelo, err := s.modeloRepo.FindByID(ctx, modeloID)
	if err != nil {
		return nil, "", err
	}

	var versao *models.ModeloDocumentoVersao
	if versaoID == nil {
		versao = modelo.VersaoAtiva
		if versao == nil {
			return nil, "", repository.ErrModeloDocumentoVersaoNotFound
		}
	} else {
		versao, err = s.versaoRepo.FindByID(ctx, *versaoID)
		if err != nil {
			return nil, "", err
		}
		if versao.ModeloDocumentoID != modeloID {
			return nil, "", repository.ErrModeloDocumentoVersaoNotFound
		}
	}

	conteudo, err = os.ReadFile(versao.CaminhoStorage)
	if err != nil {
		return nil, "", fmt.Errorf("service: ler arquivo do modelo de documento: %w", err)
	}

	return conteudo, versao.NomeArquivo, nil
}

// gravarArquivoEmDisco sanitiza o nome, calcula o hash e grava o arquivo
// em storageDir/modelos/<modeloID>/<hash>_<nome> — mesmo padrão de
// DocumentoService.Upload/VistoriaService.AnexarFoto, mas SEM checar
// duplicidade por hash: aqui cada upload é "publicar uma versão nova" de
// propósito, mesmo com conteúdo idêntico ao anterior (ver o comentário
// em models.ModeloDocumentoVersao).
func (s *ModeloDocumentoService) gravarArquivoEmDisco(modeloID uuid.UUID, nomeArquivo string, conteudo []byte, enviadoPorID uuid.UUID) (*models.ModeloDocumentoVersao, error) {
	nomeArquivo = sanitizarNomeArquivo(nomeArquivo)

	soma := sha256.Sum256(conteudo)
	hash := hex.EncodeToString(soma[:])

	// 0o750/0o600: mesmo raciocínio de DocumentoService.Upload — os
	// modelos podem conter dados institucionais (timbre, layout oficial),
	// não deveriam ficar world-readable no filesystem.
	destDir := filepath.Join(s.storageDir, "modelos", modeloID.String())
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return nil, fmt.Errorf("service: criar diretório de armazenamento do modelo: %w", err)
	}

	caminho := filepath.Join(destDir, hash+"_"+nomeArquivo)
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
		return nil, fmt.Errorf("service: gravar arquivo de modelo em disco: %w", err)
	}

	return &models.ModeloDocumentoVersao{
		ModeloDocumentoID: modeloID,
		NomeArquivo:       nomeArquivo,
		CaminhoStorage:    caminho,
		HashArquivo:       hash,
		TamanhoBytes:      int64(len(conteudo)),
		EnviadoPorID:      enviadoPorID,
	}, nil
}
