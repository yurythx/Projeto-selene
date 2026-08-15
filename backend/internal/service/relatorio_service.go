package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// RelatorioService gera o Relatório de Pagamento (PDF) de um processo,
// preenchendo as tags do contrato/fiscal cadastrados no Selene e a lista
// de documentos já anexados (Seção 5, Coluna 5, "Ação Automatizada em
// Go"). Desde o SGF-Rondonópolis (Fase 5 do plano), também inclui as
// seções de Ocorrências (IN01 Art.3º-III/Art.5º-IV,IX; IN04 Art.3º-VIII)
// e de Empenho (IN01 Art.5º-VIII; IN04 Art.5º-XXII) quando existirem,
// como apoio documental ao atesto — nenhuma das duas é obrigatória para
// gerar o relatório, ambas ficam omitidas quando não há dado.
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
	processoRepo     repository.ProcessoPagamentoRepository
	docRepo          repository.DocumentoAnexoRepository
	ocorrenciaRepo   repository.OcorrenciaRepository
	empenhoRepo      repository.EmpenhoRepository
	movimentacaoRepo repository.MovimentacaoEmpenhoRepository
}

// NewRelatorioService constrói um RelatorioService.
func NewRelatorioService(
	processoRepo repository.ProcessoPagamentoRepository,
	docRepo repository.DocumentoAnexoRepository,
	ocorrenciaRepo repository.OcorrenciaRepository,
	empenhoRepo repository.EmpenhoRepository,
	movimentacaoRepo repository.MovimentacaoEmpenhoRepository,
) (*RelatorioService, error) {
	return &RelatorioService{
		processoRepo:     processoRepo,
		docRepo:          docRepo,
		ocorrenciaRepo:   ocorrenciaRepo,
		empenhoRepo:      empenhoRepo,
		movimentacaoRepo: movimentacaoRepo,
	}, nil
}

// estadoOcorrenciaLabel traduz o Estado da Ocorrencia pro texto exibido
// no PDF — mesmos rótulos usados no frontend (ver
// components/kanban/ocorrencias-dialog.tsx), duplicados aqui de propósito
// (Go e TypeScript não compartilham constantes).
var estadoOcorrenciaLabel = map[models.EstadoOcorrencia]string{
	models.OcorrenciaRegistrada:   "Registrada",
	models.OcorrenciaNotificada:   "Notificada ao Gestor",
	models.OcorrenciaEmTratamento: "Em tratamento",
	models.OcorrenciaRegularizada: "Regularizada",
}

// formatarCentavos exibe um valor em centavos como "R$ 1.234,56"
// (formato brasileiro) — só usado no texto do PDF, não precisa de uma
// lib de i18n completa. Trata negativo (saldo pode ficar negativo se as
// anulações/faturas apropriadas excederem os reforços registrados).
func formatarCentavos(centavos int64) string {
	negativo := centavos < 0
	if negativo {
		centavos = -centavos
	}

	reais := centavos / 100
	cents := centavos % 100

	reaisStr := fmt.Sprintf("%d", reais)
	var comSeparador []byte
	for i, digito := range []byte(reaisStr) {
		if i > 0 && (len(reaisStr)-i)%3 == 0 {
			comSeparador = append(comSeparador, '.')
		}
		comSeparador = append(comSeparador, digito)
	}

	sinal := ""
	if negativo {
		sinal = "-"
	}
	return fmt.Sprintf("%sR$ %s,%02d", sinal, comSeparador, cents)
}

// calcularSaldoEmpenho reconstrói o saldo somando/subtraindo o histórico
// de movimentações — mesma lógica de EmpenhoService.CalcularSaldo,
// duplicada aqui (4 linhas) em vez de composta via outro *XxxService: o
// pacote service não tem precedente de um service injetar outro, só
// repositórios (ver DesignacaoService, OcorrenciaService etc.).
func calcularSaldoEmpenho(movimentacoes []models.MovimentacaoEmpenho) int64 {
	var saldo int64
	for _, m := range movimentacoes {
		switch m.Tipo {
		case models.MovimentacaoInicial, models.MovimentacaoReforco:
			saldo += m.Valor
		case models.MovimentacaoAnulacao, models.MovimentacaoFaturaApropriada:
			saldo -= m.Valor
		}
	}
	return saldo
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

	ocorrencias, err := s.ocorrenciaRepo.ListByProcesso(ctx, processoID)
	if err != nil {
		return nil, fmt.Errorf("service: carregar ocorrências para relatório: %w", err)
	}

	var empenho *models.Empenho
	var saldoEmpenho int64
	if processo.EmpenhoID != nil {
		empenho, err = s.empenhoRepo.FindByID(ctx, *processo.EmpenhoID)
		if err != nil {
			return nil, fmt.Errorf("service: carregar empenho para relatório: %w", err)
		}
		movimentacoes, err := s.movimentacaoRepo.ListByEmpenho(ctx, empenho.ID)
		if err != nil {
			return nil, fmt.Errorf("service: carregar movimentações de empenho para relatório: %w", err)
		}
		saldoEmpenho = calcularSaldoEmpenho(movimentacoes)
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

	// SGF-Rondonópolis (Fase 5 do plano) — seção de Ocorrências, só
	// aparece quando há alguma registrada (IN01 Art.3º-III/Art.5º-IV,IX;
	// IN04 Art.3º-VIII).
	if len(ocorrencias) > 0 {
		pdf.Ln(8)
		pdf.SetFont("Helvetica", "B", 13)
		pdf.CellFormat(0, 8, tr("Ocorrências Registradas (SGF)"), "", 1, "L", false, 0, "")
		pdf.Ln(2)

		pdf.SetFont("Helvetica", "", 10)
		for _, ocorrencia := range ocorrencias {
			estado := estadoOcorrenciaLabel[ocorrencia.Estado]
			if estado == "" {
				estado = string(ocorrencia.Estado)
			}
			pdf.CellFormat(0, 6, tr(fmt.Sprintf("- [%s] %s", estado, ocorrencia.Descricao)), "", 1, "L", false, 0, "")
		}
	}

	// SGF-Rondonópolis (Fase 5 do plano) — seção de Empenho, só aparece
	// quando o processo tem um Empenho vinculado (IN01 Art.5º-VIII; IN04
	// Art.5º-XXII). Registro PARALELO/informativo, não a fonte de verdade
	// orçamentária — ver o comentário em models.Empenho.
	if empenho != nil {
		pdf.Ln(8)
		pdf.SetFont("Helvetica", "B", 13)
		pdf.CellFormat(0, 8, tr("Empenho (Acompanhamento SGF)"), "", 1, "L", false, 0, "")
		pdf.Ln(2)

		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 6, tr(fmt.Sprintf("Número: %s", empenho.NumeroEmpenho)), "", 1, "L", false, 0, "")
		pdf.CellFormat(0, 6, tr(fmt.Sprintf("Saldo atual: %s", formatarCentavos(saldoEmpenho))), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "I", 8)
		pdf.CellFormat(0, 5, tr("Acompanhamento informativo do Selene — não substitui o saldo do sistema orçamentário oficial."), "", 1, "L", false, 0, "")
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
