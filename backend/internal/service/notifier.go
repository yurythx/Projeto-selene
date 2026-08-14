package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"

	"projeto-selene/internal/models"
)

// Notifier envia o pacote digital unificado (OF + Pré-Empenho + Empenho +
// OS) para a empresa contratada ao confirmar a Etapa 3 do Kanban (Seção 5,
// "Ação Assíncrona em Go"). É chamado em uma goroutine separada pelo
// KanbanService — falhas aqui não devem nem podem reverter a transição de
// etapa já confirmada, apenas ficam registradas em log.
type Notifier interface {
	EnviarPacoteEmpresa(ctx context.Context, processo *models.ProcessoPagamento, anexos []models.DocumentoAnexo) error
}

// SMTPConfig agrupa as credenciais de envio de e-mail. Todos os campos são
// opcionais: se SMTPHost estiver vazio, NewNotifier retorna um
// logOnlyNotifier em vez de falhar — permite rodar a aplicação sem SMTP
// configurado (ex: ambiente de desenvolvimento) sem crashar no boot,
// apenas avisando no log que o envio real está desabilitado.
type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

// NewNotifier decide qual implementação de Notifier usar de acordo com a
// presença de configuração SMTP — decisão explícita e documentada aqui, em
// vez de um fallback silencioso escondido dentro do envio.
func NewNotifier(cfg SMTPConfig) Notifier {
	if cfg.Host == "" {
		slog.Warn("SMTP_HOST não configurado — notificações às empresas contratadas serão apenas registradas em log, não enviadas de fato")
		return &logOnlyNotifier{}
	}

	return &smtpNotifier{cfg: cfg}
}

type logOnlyNotifier struct{}

func (n *logOnlyNotifier) EnviarPacoteEmpresa(ctx context.Context, processo *models.ProcessoPagamento, anexos []models.DocumentoAnexo) error {
	slog.InfoContext(ctx, "[SMTP desabilitado] pacote digital seria enviado à empresa contratada agora",
		"processo_id", processo.ID,
		"contrato", processo.Contrato.NumeroContrato,
		"anexos", len(anexos),
	)
	return nil
}

type smtpNotifier struct {
	cfg SMTPConfig
}

// EnviarPacoteEmpresa monta um e-mail multipart/mixed com os documentos do
// pacote (OF, Pré-Empenho, Nota de Empenho, etc.) anexados de verdade —
// lidos do storage local (DocumentoAnexo.CaminhoStorage) e codificados em
// base64 — e envia via SMTP autenticado.
func (n *smtpNotifier) EnviarPacoteEmpresa(ctx context.Context, processo *models.ProcessoPagamento, anexos []models.DocumentoAnexo) error {
	destinatario := processo.Contrato.ContratadaEmail
	if destinatario == "" {
		return fmt.Errorf("service: contrato %s não tem ContratadaEmail cadastrado — não é possível notificar a empresa", processo.Contrato.NumeroContrato)
	}

	msg, err := montarMensagem(n.cfg.From, destinatario, processo, anexos)
	if err != nil {
		return fmt.Errorf("service: montar e-mail do pacote digital: %w", err)
	}

	auth := smtp.PlainAuth("", n.cfg.User, n.cfg.Password, n.cfg.Host)
	addr := fmt.Sprintf("%s:%s", n.cfg.Host, n.cfg.Port)

	if err := smtp.SendMail(addr, auth, n.cfg.From, []string{destinatario}, msg); err != nil {
		return fmt.Errorf("service: falha ao enviar e-mail via SMTP: %w", err)
	}

	return nil
}

// montarMensagem monta o e-mail completo (headers + corpo multipart/mixed)
// como bytes prontos para smtp.SendMail. Cada anexo é lido do disco e
// embutido codificado em base64 — não é só uma lista de nomes em texto.
func montarMensagem(from, destinatario string, processo *models.ProcessoPagamento, anexos []models.DocumentoAnexo) ([]byte, error) {
	var corpoBuf bytes.Buffer
	writer := multipart.NewWriter(&corpoBuf)

	textoHeaders := textproto.MIMEHeader{}
	textoHeaders.Set("Content-Type", `text/plain; charset="utf-8"`)
	textoPart, err := writer.CreatePart(textoHeaders)
	if err != nil {
		return nil, fmt.Errorf("criar parte de texto: %w", err)
	}

	corpo := fmt.Sprintf(
		"Prezados,\r\n\r\nO processo de pagamento referente ao contrato %s (mês %s) avançou para a etapa de emissão de OS.\r\n\r\nSegue em anexo o pacote digital com os documentos abaixo:\r\n",
		processo.Contrato.NumeroContrato, processo.MesReferencia,
	)
	for _, anexo := range anexos {
		nomeTipo := ""
		if anexo.TipoDocumento != nil {
			nomeTipo = anexo.TipoDocumento.Nome
		}
		corpo += fmt.Sprintf("- %s (%s)\r\n", anexo.NomeArquivo, nomeTipo)
	}
	if _, err := textoPart.Write([]byte(corpo)); err != nil {
		return nil, fmt.Errorf("escrever corpo de texto: %w", err)
	}

	for _, anexo := range anexos {
		if err := anexarArquivo(writer, anexo); err != nil {
			return nil, fmt.Errorf("anexar %q: %w", anexo.NomeArquivo, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalizar corpo multipart: %w", err)
	}

	header := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: Processo Selene %s - Pacote de OS\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%q\r\n\r\n",
		from, destinatario, processo.Contrato.NumeroContrato, writer.Boundary(),
	)

	msg := make([]byte, 0, len(header)+corpoBuf.Len())
	msg = append(msg, header...)
	msg = append(msg, corpoBuf.Bytes()...)

	return msg, nil
}

// anexarArquivo lê o arquivo do storage local e escreve uma parte MIME de
// anexo (Content-Disposition: attachment) codificada em base64.
func anexarArquivo(writer *multipart.Writer, anexo models.DocumentoAnexo) error {
	conteudo, err := os.ReadFile(anexo.CaminhoStorage)
	if err != nil {
		return fmt.Errorf("ler arquivo do storage (%s): %w", anexo.CaminhoStorage, err)
	}

	contentType := mime.TypeByExtension(filepath.Ext(anexo.NomeArquivo))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	headers := textproto.MIMEHeader{}
	headers.Set("Content-Type", contentType)
	headers.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, anexo.NomeArquivo))
	headers.Set("Content-Transfer-Encoding", "base64")

	part, err := writer.CreatePart(headers)
	if err != nil {
		return fmt.Errorf("criar parte MIME: %w", err)
	}

	// base64.NewEncoder não quebra linha sozinho — envolvemos com um
	// writer que insere \r\n a cada 76 caracteres, conforme RFC 2045
	// (linhas de e-mail muito longas são rejeitadas ou truncadas por
	// vários servidores SMTP).
	encoder := base64.NewEncoder(base64.StdEncoding, &linhaBase64Writer{w: part})
	if _, err := encoder.Write(conteudo); err != nil {
		return fmt.Errorf("codificar anexo em base64: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("finalizar codificação base64: %w", err)
	}

	return nil
}

// linhaBase64Writer insere "\r\n" a cada 76 bytes escritos, sem alterar o
// conteúdo em si — usado para quebrar a saída do base64.Encoder em linhas
// no formato esperado por MIME.
type linhaBase64Writer struct {
	w       io.Writer
	naLinha int
}

func (lw *linhaBase64Writer) Write(p []byte) (int, error) {
	const larguraLinha = 76

	total := 0
	for len(p) > 0 {
		restante := larguraLinha - lw.naLinha
		n := len(p)
		if n > restante {
			n = restante
		}

		written, err := lw.w.Write(p[:n])
		total += written
		if err != nil {
			return total, err
		}

		lw.naLinha += written
		p = p[n:]

		if lw.naLinha == larguraLinha {
			if _, err := lw.w.Write([]byte("\r\n")); err != nil {
				return total, err
			}
			lw.naLinha = 0
		}
	}
	return total, nil
}
