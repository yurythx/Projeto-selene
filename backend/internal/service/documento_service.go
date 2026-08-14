package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// DocumentoService implementa o upload de documentos anexos (PDFs) ao
// escopo de um processo de pagamento, incluindo a deduplicação por
// SHA-256 exigida pela Seção 4.2/7.2 da documentação de domínio.
type DocumentoService struct {
	docRepo      repository.DocumentoAnexoRepository
	tipoDocRepo  repository.TipoDocumentoRepository
	processoRepo repository.ProcessoPagamentoRepository
	storageDir   string
}

// NewDocumentoService constrói um DocumentoService. storageDir é o
// diretório raiz (local) onde os arquivos são gravados — em produção,
// pode ser trocado por um backend S3 sem alterar a assinatura pública
// deste service.
func NewDocumentoService(
	docRepo repository.DocumentoAnexoRepository,
	tipoDocRepo repository.TipoDocumentoRepository,
	processoRepo repository.ProcessoPagamentoRepository,
	storageDir string,
) *DocumentoService {
	return &DocumentoService{
		docRepo:      docRepo,
		tipoDocRepo:  tipoDocRepo,
		processoRepo: processoRepo,
		storageDir:   storageDir,
	}
}

// Upload calcula o SHA-256 do conteúdo recebido e, se um documento com o
// mesmo hash já estiver anexado a este processo, retorna o registro
// existente em vez de duplicar (reaproveitamento de arquivos por card —
// Seção 7.2, "sem necessidade de re-upload pelo fiscal"). Caso contrário,
// grava o arquivo em disco e cria o registro.
func (s *DocumentoService) Upload(
	ctx context.Context,
	processoID uuid.UUID,
	tipoDocumentoID int,
	nomeArquivo string,
	conteudo []byte,
	enviadoPorID uuid.UUID,
) (*models.DocumentoAnexo, error) {
	// nomeArquivo vem direto do multipart do cliente (Content-Disposition
	// da requisição) — NUNCA confiável. Sem sanitizar, um filename como
	// "../../../../evil.sh" faria o filepath.Join abaixo escapar de
	// destDir (path traversal, CWE-22); o hash+"_" na frente só anula UM
	// nível de "../", não protege contra vários.
	nomeArquivo = sanitizarNomeArquivo(nomeArquivo)

	if _, err := s.processoRepo.FindByID(ctx, processoID); err != nil {
		return nil, err
	}
	if _, err := s.tipoDocRepo.FindByID(ctx, tipoDocumentoID); err != nil {
		return nil, err
	}

	soma := sha256.Sum256(conteudo)
	hash := hex.EncodeToString(soma[:])

	existente, err := s.docRepo.FindByProcessoAndHash(ctx, processoID, hash)
	if err == nil {
		return existente, nil
	}
	if !errors.Is(err, repository.ErrDocumentoNotFound) {
		return nil, fmt.Errorf("service: checar duplicidade de documento: %w", err)
	}

	// 0o750: dono lê/escreve/executa, grupo só lê/executa, nenhum acesso
	// para "outros" — os documentos aqui incluem CNDs e dados de
	// contratos, não deveriam ficar world-readable no filesystem.
	destDir := filepath.Join(s.storageDir, processoID.String())
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return nil, fmt.Errorf("service: criar diretório de armazenamento: %w", err)
	}

	caminho := filepath.Join(destDir, hash+"_"+nomeArquivo)
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
		return nil, fmt.Errorf("service: gravar arquivo em disco: %w", err)
	}

	documento := &models.DocumentoAnexo{
		ProcessoPagamentoID: processoID,
		TipoDocumentoID:     tipoDocumentoID,
		NomeArquivo:         nomeArquivo,
		CaminhoStorage:      caminho,
		HashArquivo:         hash,
		EnviadoPorID:        enviadoPorID,
	}
	if err := s.docRepo.Create(ctx, documento); err != nil {
		return nil, fmt.Errorf("service: registrar documento anexo: %w", err)
	}

	return documento, nil
}

// Listar retorna todos os documentos anexados a um processo.
func (s *DocumentoService) Listar(ctx context.Context, processoID uuid.UUID) ([]models.DocumentoAnexo, error) {
	documentos, err := s.docRepo.ListByProcesso(ctx, processoID)
	if err != nil {
		return nil, fmt.Errorf("service: listar documentos do processo: %w", err)
	}
	return documentos, nil
}

// sanitizarNomeArquivo neutraliza um filename de upload não confiável
// antes de usá-lo em qualquer caminho de disco ou header (Content-
// Disposition do e-mail em notifier.go). Duas proteções:
//
//  1. filepath.Base descarta qualquer componente de diretório do nome —
//     inclusive sequências "../" — então o resultado nunca contém um
//     separador de caminho, o que por si só impede escapar de destDir em
//     DocumentoService.Upload via filepath.Join.
//  2. Caracteres de controle (CR, LF, NUL, etc.) são removidos — evita
//     que um filename injete headers adicionais se for reaproveitado tal
//     qual em Content-Disposition (MIME) mais adiante.
func sanitizarNomeArquivo(nome string) string {
	const fallback = "arquivo"

	base := filepath.Base(nome)
	if base == "." || base == ".." || base == string(filepath.Separator) || base == "" {
		return fallback
	}

	var limpo strings.Builder
	for _, r := range base {
		if r < 0x20 || r == 0x7f {
			continue
		}
		limpo.WriteRune(r)
	}

	resultado := limpo.String()
	if resultado == "" {
		return fallback
	}
	return resultado
}
