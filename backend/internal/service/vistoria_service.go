package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// VistoriaService implementa o Módulo 3 do roadmap (Vistorias e Registro
// Fotográfico com Geolocalização): registro de uma vistoria de campo,
// upload das fotos (reaproveitando a mesma infra de storage local e
// deduplicação por SHA-256 de DocumentoService — só generalizada pra um
// segundo tipo de anexo) e geração do "Relatório de Campo" em PDF.
type VistoriaService struct {
	vistoriaRepo repository.VistoriaRepository
	fotoRepo     repository.FotoVistoriaRepository
	processoRepo repository.ProcessoPagamentoRepository
	storageDir   string
}

// NewVistoriaService constrói um VistoriaService.
func NewVistoriaService(
	vistoriaRepo repository.VistoriaRepository,
	fotoRepo repository.FotoVistoriaRepository,
	processoRepo repository.ProcessoPagamentoRepository,
	storageDir string,
) *VistoriaService {
	return &VistoriaService{
		vistoriaRepo: vistoriaRepo,
		fotoRepo:     fotoRepo,
		processoRepo: processoRepo,
		storageDir:   storageDir,
	}
}

// RegistrarInput agrupa os dados de abertura de uma vistoria — as fotos
// são anexadas depois, uma a uma, via AnexarFoto (mesmo padrão em duas
// etapas de ProcessoPagamento/DocumentoAnexo: primeiro cria o registro,
// depois anexa arquivos a ele).
type RegistrarInput struct {
	ProcessoPagamentoID uuid.UUID
	FiscalID            uuid.UUID
	// Latitude/Longitude são opcionais — ver o comentário no model
	// RegistroVistoria sobre por que a geolocalização pode faltar.
	Latitude    *float64
	Longitude   *float64
	Observacoes string
}

// Registrar abre um novo registro de vistoria pro processo informado.
func (s *VistoriaService) Registrar(ctx context.Context, input RegistrarInput) (*models.RegistroVistoria, error) {
	if _, err := s.processoRepo.FindByID(ctx, input.ProcessoPagamentoID); err != nil {
		return nil, err
	}

	vistoria := &models.RegistroVistoria{
		ProcessoPagamentoID: input.ProcessoPagamentoID,
		FiscalID:            input.FiscalID,
		DataHora:            time.Now(),
		Latitude:            input.Latitude,
		Longitude:           input.Longitude,
		Observacoes:         input.Observacoes,
	}
	if err := s.vistoriaRepo.Create(ctx, vistoria); err != nil {
		return nil, fmt.Errorf("service: registrar vistoria: %w", err)
	}

	return vistoria, nil
}

// AnexarFoto grava uma foto em disco e a associa à vistoria informada —
// mesma lógica de deduplicação por SHA-256 de DocumentoService.Upload
// (ver o comentário lá): reenviar o mesmo arquivo pra mesma vistoria
// reaproveita o registro existente.
func (s *VistoriaService) AnexarFoto(ctx context.Context, vistoriaID uuid.UUID, nomeArquivo string, conteudo []byte) (*models.FotoVistoria, error) {
	nomeArquivo = sanitizarNomeArquivo(nomeArquivo)

	if _, err := s.vistoriaRepo.FindByID(ctx, vistoriaID); err != nil {
		return nil, err
	}

	soma := sha256.Sum256(conteudo)
	hash := hex.EncodeToString(soma[:])

	existente, err := s.fotoRepo.FindByVistoriaAndHash(ctx, vistoriaID, hash)
	if err == nil {
		return existente, nil
	}
	if !errors.Is(err, repository.ErrFotoVistoriaNotFound) {
		return nil, fmt.Errorf("service: checar duplicidade de foto de vistoria: %w", err)
	}

	// Mesmas permissões restritivas de DocumentoService.Upload — ver o
	// comentário lá (0o750/0o600, não world-readable).
	destDir := filepath.Join(s.storageDir, "vistorias", vistoriaID.String())
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return nil, fmt.Errorf("service: criar diretório de armazenamento de fotos: %w", err)
	}

	caminho := filepath.Join(destDir, hash+"_"+nomeArquivo)
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
		return nil, fmt.Errorf("service: gravar foto em disco: %w", err)
	}

	foto := &models.FotoVistoria{
		VistoriaID:     vistoriaID,
		NomeArquivo:    nomeArquivo,
		CaminhoStorage: caminho,
		HashArquivo:    hash,
	}
	if err := s.fotoRepo.Create(ctx, foto); err != nil {
		return nil, fmt.Errorf("service: registrar foto de vistoria: %w", err)
	}

	return foto, nil
}

// ListarPorProcesso retorna todas as vistorias de um processo, mais
// recente primeiro, com as fotos já carregadas.
func (s *VistoriaService) ListarPorProcesso(ctx context.Context, processoID uuid.UUID) ([]models.RegistroVistoria, error) {
	vistorias, err := s.vistoriaRepo.ListByProcesso(ctx, processoID)
	if err != nil {
		return nil, fmt.Errorf("service: listar vistorias do processo: %w", err)
	}
	return vistorias, nil
}

// tipoImagemPDF mapeia a extensão do arquivo pro identificador de tipo que
// o fpdf espera ("JPG", "PNG") — só esses dois formatos são aceitos no
// upload de fotos (ver VistoriaHandler.AnexarFoto), então um formato fora
// desse conjunto aqui indica um bug de validação anterior, não uma
// condição a ser tratada silenciosamente.
func tipoImagemPDF(nomeArquivo string) string {
	ext := strings.ToLower(filepath.Ext(nomeArquivo))
	switch ext {
	case ".png":
		return "PNG"
	default:
		return "JPG"
	}
}

// GerarRelatorioCampo renderiza o "Relatório de Campo" (PDF, A4) de uma
// vistoria: dados do contrato/processo, coordenadas, observações e as
// fotos anexadas — pensado pra instruir o processo perante o TCE. Reusa o
// mesmo padrão fpdf de RelatorioService/GeradorDocumentosService.
func (s *VistoriaService) GerarRelatorioCampo(ctx context.Context, vistoriaID uuid.UUID) ([]byte, error) {
	vistoria, err := s.vistoriaRepo.FindByID(ctx, vistoriaID)
	if err != nil {
		return nil, err
	}

	numeroContrato, contratadaNome := "", ""
	if vistoria.ProcessoPagamento != nil && vistoria.ProcessoPagamento.Contrato != nil {
		numeroContrato = vistoria.ProcessoPagamento.Contrato.NumeroContrato
		contratadaNome = vistoria.ProcessoPagamento.Contrato.ContratadaNome
	}
	fiscalNome := ""
	if vistoria.Fiscal != nil {
		fiscalNome = vistoria.Fiscal.Nome
	}

	pdf, tr := novoPDF(fmt.Sprintf("Relatorio de Campo - %s", numeroContrato))

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, tr("Relatório de Campo — Vistoria"), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	campoPDF(pdf, tr, "Contrato", numeroContrato)
	campoPDF(pdf, tr, "Contratada", contratadaNome)
	campoPDF(pdf, tr, "Fiscal Responsável", fiscalNome)
	campoPDF(pdf, tr, "Data/Hora da Vistoria", vistoria.DataHora.Format("02/01/2006 15:04"))
	campoPDF(pdf, tr, "Coordenadas", formatarCoordenadas(vistoria.Latitude, vistoria.Longitude))

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, tr("Observações do Fiscal"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	observacoes := vistoria.Observacoes
	if observacoes == "" {
		observacoes = "Nenhuma observação registrada."
	}
	pdf.MultiCell(0, 5, tr(observacoes), "", "L", false)

	if len(vistoria.Fotos) > 0 {
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(0, 8, tr(fmt.Sprintf("Registro Fotográfico (%d foto(s))", len(vistoria.Fotos))), "", 1, "L", false, 0, "")
		pdf.Ln(2)

		_, pageHeight := pdf.GetPageSize()
		_, top, _, bottom := pdf.GetMargins()
		usableHeight := pageHeight - top - bottom
		const larguraFoto = 80.0
		const alturaFoto = 60.0
		const espacamento = 6.0

		for i, foto := range vistoria.Fotos {
			conteudo, err := os.ReadFile(foto.CaminhoStorage)
			if err != nil {
				// Arquivo ausente/corrompido não deve derrubar o relatório
				// inteiro — as demais fotos e o resto do documento
				// continuam válidos; só esta entrada fica marcada.
				pdf.SetFont("Helvetica", "I", 9)
				pdf.CellFormat(0, 6, tr(fmt.Sprintf("- %s: falha ao carregar imagem.", foto.NomeArquivo)), "", 1, "L", false, 0, "")
				continue
			}

			if pdf.GetY()+alturaFoto+10 > usableHeight+top {
				pdf.AddPage()
			}

			nomeImagem := fmt.Sprintf("foto-%d", i)
			imgOpts := fpdf.ImageOptions{ImageType: tipoImagemPDF(foto.NomeArquivo)}
			pdf.RegisterImageOptionsReader(nomeImagem, imgOpts, bytes.NewReader(conteudo))
			pdf.ImageOptions(nomeImagem, pdf.GetX(), pdf.GetY(), larguraFoto, alturaFoto, false, imgOpts, 0, "")
			pdf.SetY(pdf.GetY() + alturaFoto + 2)
			pdf.SetFont("Helvetica", "I", 8)
			pdf.CellFormat(0, 5, tr(foto.NomeArquivo), "", 1, "L", false, 0, "")
			pdf.Ln(espacamento)
		}
	}

	return renderizarPDF(pdf)
}

// formatarCoordenadas exibe lat/lng com 6 casas decimais (~11cm de
// precisão) ou "Não capturadas" quando a geolocalização faltou (ver o
// comentário em models.RegistroVistoria sobre isso ser esperado, não erro).
func formatarCoordenadas(lat, lng *float64) string {
	if lat == nil || lng == nil {
		return "Não capturadas"
	}
	return fmt.Sprintf("%.6f, %.6f", *lat, *lng)
}
