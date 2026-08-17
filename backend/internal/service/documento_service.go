package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	// dataValidade alimenta o Radar de Alertas (Fase 1 do roadmap) —
	// nil quando o cliente não informou (ex: tipo de documento que não
	// vence) ou quando optou por não preencher. Não é exigido mesmo
	// quando TipoDocumento.ExigeValidade=true: sem essa data o
	// documento simplesmente não aparece no radar de certidões, o que é
	// preferível a travar o upload por um dado que o fiscal pode não ter
	// em mãos naquele momento.
	dataValidade *time.Time,
) (*models.DocumentoAnexo, error) {
	// nomeArquivo vem direto do multipart do cliente (Content-Disposition
	// da requisição) — NUNCA confiável. Sem sanitizar, um filename como
	// "../../../../evil.sh" faria o filepath.Join abaixo escapar de
	// destDir (path traversal, CWE-22); o hash+"_" na frente só anula UM
	// nível de "../", não protege contra vários.
	nomeArquivo = sanitizarNomeArquivo(nomeArquivo)

	processo, err := s.processoRepo.FindByID(ctx, processoID)
	if err != nil {
		return nil, err
	}
	tipoDoc, err := s.tipoDocRepo.FindByID(ctx, tipoDocumentoID)
	if err != nil {
		return nil, err
	}

	// processo.Contrato só vem nil nos dublês de teste que não preload
	// (FindByID real sempre traz o Contrato) — nesse caso não há como
	// avaliar a restrição, então não bloqueia (fail-open) em vez de
	// derrubar um nil pointer.
	if processo.Contrato != nil && !TipoDocumentoAplicavel(*tipoDoc, processo.Contrato.TipoObjeto, processo.Contrato.ExigeFiscalizacaoTerceirizacao) {
		return nil, ErrTipoDocumentoNaoAplicavel
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

	// Regra pedida pelo usuário: no máximo UM documento de cada tipo por
	// processo (ex: não pode ter dois "Pré-Empenho") — diferente da
	// checagem de hash acima (que só reaproveita quando é o MESMO
	// arquivo). Aqui, mesmo um arquivo diferente do mesmo tipo é
	// rejeitado: quem quiser substituir precisa excluir o anterior
	// primeiro (ver DocumentoService.Excluir) e reenviar o correto — sem
	// isso, o checklist ficaria ambíguo sobre qual dos dois vale.
	if _, err := s.docRepo.FindByProcessoAndTipo(ctx, processoID, tipoDocumentoID); err == nil {
		return nil, ErrTipoDocumentoJaAnexado
	} else if !errors.Is(err, repository.ErrDocumentoNotFound) {
		return nil, fmt.Errorf("service: checar documento existente do mesmo tipo: %w", err)
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
		DataValidade:        dataValidade,
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

// Baixar carrega o conteúdo (do disco) de um documento anexo específico —
// usado tanto pra download quanto pra pré-visualização inline na página
// do processo (ver GeradorDocumentosHandler... não, ver DocumentoHandler.
// Baixar). Confere que o documento realmente pertence a processoID
// (defesa em profundidade: os dois IDs vêm de segmentos distintos da
// rota, GET /processos/:id/documentos/:docId/download — sem essa
// checagem, adivinhar o UUID de um documento de outro processo bastaria
// pra baixá-lo por uma URL de processo errada).
func (s *DocumentoService) Baixar(ctx context.Context, processoID, documentoID uuid.UUID) ([]byte, *models.DocumentoAnexo, error) {
	documento, err := s.docRepo.FindByID(ctx, documentoID)
	if err != nil {
		return nil, nil, err
	}
	if documento.ProcessoPagamentoID != processoID {
		return nil, nil, repository.ErrDocumentoNotFound
	}

	conteudo, err := os.ReadFile(documento.CaminhoStorage)
	if err != nil {
		return nil, nil, fmt.Errorf("service: ler arquivo do documento anexo: %w", err)
	}

	return conteudo, documento, nil
}

// Excluir remove um documento anexo — o registro e o arquivo físico.
//
// Sem trava de "já foi usado pra satisfazer o checklist de uma etapa já
// concluída": o pedido é justamente poder corrigir um upload errado
// (arquivo trocado, tipo errado). A auditoria do avanço em si continua
// íntegra em kanban_logs — apagar o anexo não desfaz uma transição de
// etapa já ocorrida, só remove o arquivo da lista atual. Decisão
// deliberada, documentada aqui por não haver uma regra de domínio
// pedindo o contrário; se um dia precisar bloquear isso, é um `if`
// checando EtapaAtualID/histórico antes do Delete abaixo.
func (s *DocumentoService) Excluir(ctx context.Context, processoID, documentoID uuid.UUID) error {
	documento, err := s.docRepo.FindByID(ctx, documentoID)
	if err != nil {
		return err
	}
	if documento.ProcessoPagamentoID != processoID {
		return repository.ErrDocumentoNotFound
	}

	if err := s.docRepo.Delete(ctx, documentoID); err != nil {
		return err
	}

	// Best-effort: a exclusão lógica (a linha no banco, fonte de verdade
	// da aplicação) já se consolidou nesse ponto — se o arquivo físico
	// não puder ser removido (permissão, já apagado manualmente antes),
	// só registra o aviso; não faz sentido devolver erro pro cliente por
	// uma falha de limpeza de disco depois que o que importa já foi feito.
	if err := os.Remove(documento.CaminhoStorage); err != nil && !os.IsNotExist(err) {
		slog.WarnContext(ctx, "falha ao remover arquivo físico do documento excluído", "caminho", documento.CaminhoStorage, "erro", err)
	}

	return nil
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
