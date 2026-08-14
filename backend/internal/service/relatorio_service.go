package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"

	"projeto-selene/internal/repository"
)

// RelatorioService gera o Relatório de Pagamento (PDF) de um processo,
// preenchendo as tags do contrato/fiscal cadastrados no Selene e a lista
// de documentos já anexados (Seção 5, Coluna 5, "Ação Automatizada em
// Go").
//
// LIMITAÇÃO CONHECIDA: a documentação de domínio não forneceu o arquivo
// oficial do modelo de Relatório de Pagamento usado pela prefeitura (o
// layout jurídico/visual real). Este PDF é um substituto funcional que
// preenche as mesmas tags em um layout simples e legível — o fiscal
// imprime, colhe a assinatura física do Secretário e o atesto no verso da
// Nota Fiscal, e reanexa o PDF escaneado (documento "Relatório de
// Pagamento Assinado" no checklist da Etapa 5). Trocar pelo layout
// oficial da prefeitura é uma mudança isolada nesta função.
type RelatorioService struct {
	processoRepo repository.ProcessoPagamentoRepository
	docRepo      repository.DocumentoAnexoRepository
}

// NewRelatorioService constrói um RelatorioService.
func NewRelatorioService(processoRepo repository.ProcessoPagamentoRepository, docRepo repository.DocumentoAnexoRepository) (*RelatorioService, error) {
	return &RelatorioService{processoRepo: processoRepo, docRepo: docRepo}, nil
}

// Gerar renderiza o Relatório de Pagamento (PDF, A4) do processo
// informado.
func (s *RelatorioService) Gerar(ctx context.Context, processoID uuid.UUID) ([]byte, error) {
	processo, err := s.processoRepo.FindByID(ctx, processoID)
	if err != nil {
		return nil, fmt.Errorf("service: carregar processo para relatório: %w", err)
	}

	documentos, err := s.docRepo.ListByProcesso(ctx, processoID)
	if err != nil {
		return nil, fmt.Errorf("service: carregar documentos para relatório: %w", err)
	}

	fiscalNome := ""
	if processo.Contrato.Fiscal != nil {
		fiscalNome = processo.Contrato.Fiscal.Nome
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	// As fontes padrão do PDF (core 14, ex: Helvetica) só entendem bytes
	// em cp1252/WinAnsi, não UTF-8 — sem este translator, qualquer
	// acento nos dados (que são praticamente todos em português: "Ofício
	// de Solicitação", "Pré-Empenho" etc.) sairia corrompido no PDF.
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetTitle(tr(fmt.Sprintf("Relatorio de Pagamento - %s", processo.Contrato.NumeroContrato)), false)
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, tr("Relatório de Pagamento"), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	campo := func(rotulo, valor string) {
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(50, 8, tr(rotulo), "1", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 11)
		pdf.CellFormat(0, 8, tr(valor), "1", 1, "L", false, 0, "")
	}

	campo("Contrato", processo.Contrato.NumeroContrato)
	campo("Portaria de Nomeação", processo.Contrato.PortariaNomeacao)
	campo("Fiscal Responsável", fiscalNome)
	campo("Contratada", fmt.Sprintf("%s (CNPJ %s)", processo.Contrato.ContratadaNome, processo.Contrato.ContratadaCNPJ))
	campo("Tipo de Objeto", string(processo.Contrato.TipoObjeto))
	campo("Mês de Referência", processo.MesReferencia)

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 8, tr("Documentos Anexados ao Processo"), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "", 10)
	if len(documentos) == 0 {
		pdf.CellFormat(0, 6, tr("Nenhum documento anexado ainda."), "", 1, "L", false, 0, "")
	}
	for _, doc := range documentos {
		nomeTipo := ""
		if doc.TipoDocumento != nil {
			nomeTipo = doc.TipoDocumento.Nome
		}
		pdf.CellFormat(0, 6, tr(fmt.Sprintf("- %s: %s", nomeTipo, doc.NomeArquivo)), "", 1, "L", false, 0, "")
	}

	pdf.Ln(16)
	pdf.SetFont("Helvetica", "", 11)
	larguraAssinatura := 90.0
	pdf.CellFormat(larguraAssinatura, 0, "", "T", 1, "C", false, 0, "")
	pdf.CellFormat(larguraAssinatura, 6, tr("Assinatura do Secretário da Pasta"), "", 1, "C", false, 0, "")

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.MultiCell(0, 5, tr("Atesto físico no verso da Nota Fiscal - anexar escaneado como \"Relatório de Pagamento Assinado\"."), "", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("service: renderizar relatório de pagamento em PDF: %w", err)
	}

	return buf.Bytes(), nil
}
