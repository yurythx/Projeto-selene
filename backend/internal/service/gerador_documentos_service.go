package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// GeradorDocumentosService implementa o Módulo 2 do roadmap ("Gerador
// Inteligente de Documentos Legais"): 3 geradores de PDF oficiais
// pré-preenchidos a partir dos dados já cadastrados no Selene —
// Notificação de Descumprimento, Atesto (com QR code de verificação) e
// Minuta de Aditivo. Cada geração cria um registro em DocumentoEmitido,
// que alimenta o Dossiê do Fornecedor (Fase 4) e permite auditoria
// (quem gerou o quê, quando, com qual motivo).
//
// Reaproveita o mesmo padrão de PDF de RelatorioService (fpdf +
// UnicodeTranslatorFromDescriptor + helper campo()) em vez de introduzir
// uma abordagem nova.
type GeradorDocumentosService struct {
	contratoRepo   repository.ContratoRepository
	processoRepo   repository.ProcessoPagamentoRepository
	docEmitidoRepo repository.DocumentoEmitidoRepository

	// publicURL é a URL pública do FRONTEND (config.PublicURL) usada pra
	// montar o link embutido no QR code do Atesto. Vazio = o QR embute só
	// o código de verificação em texto puro (ver gerarConteudoQR).
	publicURL string
}

// NewGeradorDocumentosService constrói um GeradorDocumentosService.
func NewGeradorDocumentosService(
	contratoRepo repository.ContratoRepository,
	processoRepo repository.ProcessoPagamentoRepository,
	docEmitidoRepo repository.DocumentoEmitidoRepository,
	publicURL string,
) *GeradorDocumentosService {
	return &GeradorDocumentosService{
		contratoRepo:   contratoRepo,
		processoRepo:   processoRepo,
		docEmitidoRepo: docEmitidoRepo,
		publicURL:      strings.TrimRight(publicURL, "/"),
	}
}

// MinutaAditivoInput agrupa as respostas do questionário curto usado para
// gerar a justificativa técnica da Minuta de Aditivo.
type MinutaAditivoInput struct {
	// TipoAditivo: "VALOR", "PRAZO" ou "VALOR_E_PRAZO".
	TipoAditivo   string
	Justificativa string
	// NovoValor e NovoPrazo são texto livre (ex: "R$ 150.000,00",
	// "31/12/2026") — preenchidos conforme TipoAditivo; a minuta é uma
	// justificativa técnica preliminar, não uma peça jurídica final, então
	// não há validação rígida de formato aqui.
	NovoValor string
	NovoPrazo string
}

// gerarCodigoVerificacao produz um código curto e não-sequencial (não dá
// pra adivinhar o próximo a partir de um existente) usado tanto na URL de
// verificação quanto no QR code. Prefixo "SEL-" só pra ficar reconhecível
// visualmente como um código do Selene.
func gerarCodigoVerificacao() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("service: gerar código de verificação: %w", err)
	}
	return "SEL-" + strings.ToUpper(hex.EncodeToString(buf)), nil
}

// novoTradutor monta um *fpdf.Fpdf em branco com o mesmo setup usado em
// RelatorioService — fontes core só entendem cp1252/WinAnsi, então todo
// texto (majoritariamente em português acentuado) precisa passar pelo
// tradutor antes de ir pro PDF.
func novoPDF(titulo string) (*fpdf.Fpdf, func(string) string) {
	pdf := fpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetTitle(tr(titulo), false)
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()
	return pdf, tr
}

func campoPDF(pdf *fpdf.Fpdf, tr func(string) string, rotulo, valor string) {
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(50, 8, tr(rotulo), "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 8, tr(valor), "1", 1, "L", false, 0, "")
}

// GerarNotificacao emite a Notificação de Descumprimento do contrato
// informado, registra o histórico em DocumentoEmitido e retorna o PDF
// pronto pra download.
func (s *GeradorDocumentosService) GerarNotificacao(ctx context.Context, contratoID uuid.UUID, motivo string, geradoPorID uuid.UUID) ([]byte, *models.DocumentoEmitido, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, nil, ErrMotivoObrigatorio
	}

	contrato, err := s.contratoRepo.FindByID(ctx, contratoID)
	if err != nil {
		return nil, nil, err
	}

	codigo, err := gerarCodigoVerificacao()
	if err != nil {
		return nil, nil, err
	}

	registro := &models.DocumentoEmitido{
		ContratoID:        contrato.ID,
		Tipo:              models.TipoDocumentoEmitidoNotificacao,
		Motivo:            motivo,
		GeradoPorID:       geradoPorID,
		CodigoVerificacao: codigo,
	}
	if err := s.docEmitidoRepo.Create(ctx, registro); err != nil {
		return nil, nil, fmt.Errorf("service: registrar notificação emitida: %w", err)
	}

	pdf, tr := novoPDF(fmt.Sprintf("Notificacao de Descumprimento - %s", contrato.NumeroContrato))

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, tr("Notificação de Descumprimento Contratual"), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	campoPDF(pdf, tr, "Contrato", contrato.NumeroContrato)
	campoPDF(pdf, tr, "Contratada", fmt.Sprintf("%s (CNPJ %s)", contrato.ContratadaNome, contrato.ContratadaCNPJ))
	campoPDF(pdf, tr, "Data de emissão", time.Now().Format("02/01/2006"))
	campoPDF(pdf, tr, "Código de verificação", codigo)

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 8, tr("Motivo da notificação"), "", 1, "L", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "", 11)
	pdf.MultiCell(0, 6, tr(motivo), "1", "L", false)

	pdf.Ln(10)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5, tr(fmt.Sprintf(
		"Fica a empresa %s NOTIFICADA para, no prazo legal cabível, regularizar a situação apontada acima, "+
			"sob pena das sanções administrativas previstas no instrumento contratual e na Lei nº 14.133/2021.",
		contrato.ContratadaNome,
	)), "", "L", false)

	pdf.Ln(16)
	larguraAssinatura := 90.0
	pdf.CellFormat(larguraAssinatura, 0, "", "T", 1, "C", false, 0, "")
	pdf.CellFormat(larguraAssinatura, 6, tr("Assinatura do Fiscal do Contrato"), "", 1, "C", false, 0, "")

	corpo, err := renderizarPDF(pdf)
	if err != nil {
		return nil, nil, err
	}
	return corpo, registro, nil
}

// GerarAtesto emite o Atesto (Folha de Rosto + Termo de Recebimento) do
// processo informado, com QR code de verificação de autenticidade.
func (s *GeradorDocumentosService) GerarAtesto(ctx context.Context, processoID uuid.UUID, geradoPorID uuid.UUID) ([]byte, *models.DocumentoEmitido, error) {
	processo, err := s.processoRepo.FindByID(ctx, processoID)
	if err != nil {
		return nil, nil, err
	}

	codigo, err := gerarCodigoVerificacao()
	if err != nil {
		return nil, nil, err
	}

	registro := &models.DocumentoEmitido{
		ContratoID:          processo.ContratoID,
		ProcessoPagamentoID: &processo.ID,
		Tipo:                models.TipoDocumentoEmitidoAtesto,
		GeradoPorID:         geradoPorID,
		CodigoVerificacao:   codigo,
	}
	if err := s.docEmitidoRepo.Create(ctx, registro); err != nil {
		return nil, nil, fmt.Errorf("service: registrar atesto emitido: %w", err)
	}

	qrPNG, err := qrcode.Encode(s.conteudoQR(codigo), qrcode.Medium, 256)
	if err != nil {
		return nil, nil, fmt.Errorf("service: gerar QR code do atesto: %w", err)
	}

	fiscalNome := ""
	if processo.Contrato != nil && processo.Contrato.Fiscal != nil {
		fiscalNome = processo.Contrato.Fiscal.Nome
	}
	numeroContrato, contratadaNome, contratadaCNPJ := "", "", ""
	if processo.Contrato != nil {
		numeroContrato = processo.Contrato.NumeroContrato
		contratadaNome = processo.Contrato.ContratadaNome
		contratadaCNPJ = processo.Contrato.ContratadaCNPJ
	}

	pdf, tr := novoPDF(fmt.Sprintf("Atesto - %s - %s", numeroContrato, processo.MesReferencia))

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, tr("Folha de Rosto e Termo de Recebimento"), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	campoPDF(pdf, tr, "Contrato", numeroContrato)
	campoPDF(pdf, tr, "Contratada", fmt.Sprintf("%s (CNPJ %s)", contratadaNome, contratadaCNPJ))
	campoPDF(pdf, tr, "Fiscal Responsável", fiscalNome)
	campoPDF(pdf, tr, "Mês de Referência", processo.MesReferencia)
	campoPDF(pdf, tr, "Data de emissão", time.Now().Format("02/01/2006"))

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5, tr(
		"Atesto, para os devidos fins, que os serviços/materiais objeto deste contrato, referentes ao mês "+
			"acima indicado, foram recebidos e conferidos, encontrando-se em conformidade com o pactuado, "+
			"nos termos do art. 140 da Lei nº 14.133/2021.",
	), "", "L", false)

	pdf.Ln(16)
	larguraAssinatura := 90.0
	pdf.CellFormat(larguraAssinatura, 0, "", "T", 1, "C", false, 0, "")
	pdf.CellFormat(larguraAssinatura, 6, tr("Assinatura do Fiscal do Contrato"), "", 1, "C", false, 0, "")

	// QR code de verificação — canto inferior direito, com o código
	// também em texto (nem todo leitor de QR está à mão, e o código serve
	// pra digitação manual na página de verificação).
	pdf.Ln(14)
	imgOpts := fpdf.ImageOptions{ImageType: "PNG"}
	pdf.RegisterImageOptionsReader("qr-atesto", imgOpts, bytes.NewReader(qrPNG))
	pdf.ImageOptions("qr-atesto", 150, 240, 30, 30, false, imgOpts, 0, "")
	pdf.SetXY(20, 272)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, tr(fmt.Sprintf("Verificação de autenticidade: código %s", codigo)), "", 1, "L", false, 0, "")

	corpo, err := renderizarPDF(pdf)
	if err != nil {
		return nil, nil, err
	}
	return corpo, registro, nil
}

// GerarMinutaAditivo emite a Minuta de Aditivo de Valor/Prazo do contrato
// informado a partir do questionário curto (input).
func (s *GeradorDocumentosService) GerarMinutaAditivo(ctx context.Context, contratoID uuid.UUID, input MinutaAditivoInput, geradoPorID uuid.UUID) ([]byte, *models.DocumentoEmitido, error) {
	justificativa := strings.TrimSpace(input.Justificativa)
	if justificativa == "" {
		return nil, nil, ErrMotivoObrigatorio
	}

	contrato, err := s.contratoRepo.FindByID(ctx, contratoID)
	if err != nil {
		return nil, nil, err
	}

	codigo, err := gerarCodigoVerificacao()
	if err != nil {
		return nil, nil, err
	}

	dadosExtras, err := json.Marshal(input)
	if err != nil {
		return nil, nil, fmt.Errorf("service: serializar dados da minuta de aditivo: %w", err)
	}

	registro := &models.DocumentoEmitido{
		ContratoID:        contrato.ID,
		Tipo:              models.TipoDocumentoEmitidoAditivo,
		Motivo:            justificativa,
		DadosExtras:       string(dadosExtras),
		GeradoPorID:       geradoPorID,
		CodigoVerificacao: codigo,
	}
	if err := s.docEmitidoRepo.Create(ctx, registro); err != nil {
		return nil, nil, fmt.Errorf("service: registrar minuta de aditivo emitida: %w", err)
	}

	pdf, tr := novoPDF(fmt.Sprintf("Minuta de Aditivo - %s", contrato.NumeroContrato))

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, tr("Minuta de Aditivo Contratual"), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	campoPDF(pdf, tr, "Contrato", contrato.NumeroContrato)
	campoPDF(pdf, tr, "Contratada", fmt.Sprintf("%s (CNPJ %s)", contrato.ContratadaNome, contrato.ContratadaCNPJ))
	campoPDF(pdf, tr, "Tipo de aditivo", rotuloTipoAditivo(input.TipoAditivo))
	if input.NovoValor != "" {
		campoPDF(pdf, tr, "Novo valor proposto", input.NovoValor)
	}
	if input.NovoPrazo != "" {
		campoPDF(pdf, tr, "Novo prazo proposto", input.NovoPrazo)
	}
	campoPDF(pdf, tr, "Data de emissão", time.Now().Format("02/01/2006"))

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 8, tr("Justificativa Técnica"), "", 1, "L", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "", 11)
	pdf.MultiCell(0, 6, tr(justificativa), "1", "L", false)

	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.MultiCell(0, 5, tr(
		"Minuta preliminar gerada pelo Selene a partir da justificativa técnica informada pelo fiscal — "+
			"não substitui a análise jurídica/formalização definitiva do termo aditivo, exigida pela Lei nº 14.133/2021.",
	), "", "L", false)

	pdf.Ln(16)
	larguraAssinatura := 90.0
	pdf.CellFormat(larguraAssinatura, 0, "", "T", 1, "C", false, 0, "")
	pdf.CellFormat(larguraAssinatura, 6, tr("Assinatura do Fiscal do Contrato"), "", 1, "C", false, 0, "")

	corpo, err := renderizarPDF(pdf)
	if err != nil {
		return nil, nil, err
	}
	return corpo, registro, nil
}

// Verificar confirma a autenticidade de um documento emitido a partir do
// código impresso/QR code — usado pela rota pública de verificação (GET
// /api/v1/verificar/:codigo, fora do middleware de autenticação: quem
// escaneia um papel impresso não tem login no Selene).
func (s *GeradorDocumentosService) Verificar(ctx context.Context, codigo string) (*models.DocumentoEmitido, error) {
	return s.docEmitidoRepo.FindByCodigoVerificacao(ctx, strings.TrimSpace(codigo))
}

// conteudoQR monta o valor embutido no QR code: um link clicável pro
// frontend quando PublicURL está configurado, ou só o código em texto
// puro (ainda verificável por digitação manual) quando não está.
func (s *GeradorDocumentosService) conteudoQR(codigo string) string {
	if s.publicURL == "" {
		return codigo
	}
	return fmt.Sprintf("%s/verificar/%s", s.publicURL, codigo)
}

func rotuloTipoAditivo(tipo string) string {
	switch tipo {
	case "VALOR":
		return "Aditivo de Valor"
	case "PRAZO":
		return "Aditivo de Prazo"
	case "VALOR_E_PRAZO":
		return "Aditivo de Valor e Prazo"
	default:
		return tipo
	}
}

func renderizarPDF(pdf *fpdf.Fpdf) ([]byte, error) {
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("service: renderizar documento em PDF: %w", err)
	}
	return buf.Bytes(), nil
}
