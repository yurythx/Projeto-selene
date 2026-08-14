package service

import (
	"context"
	"fmt"
	"log"
	"net/smtp"

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
		log.Println("service: SMTP_HOST não configurado — notificações às empresas contratadas serão apenas registradas em log, não enviadas de fato")
		return &logOnlyNotifier{}
	}

	return &smtpNotifier{cfg: cfg}
}

type logOnlyNotifier struct{}

func (n *logOnlyNotifier) EnviarPacoteEmpresa(ctx context.Context, processo *models.ProcessoPagamento, anexos []models.DocumentoAnexo) error {
	log.Printf(
		"service: [SMTP desabilitado] pacote digital do processo %s (contrato %s) seria enviado à empresa contratada agora, com %d anexo(s)",
		processo.ID, processo.Contrato.NumeroContrato, len(anexos),
	)
	return nil
}

type smtpNotifier struct {
	cfg SMTPConfig
}

// EnviarPacoteEmpresa monta um e-mail simples (texto) com a lista dos
// documentos do pacote e o envia via SMTP autenticado.
//
// LIMITAÇÃO CONHECIDA: esta implementação envia os NOMES/caminhos dos
// documentos do pacote em texto, não os arquivos anexados de fato (SMTP
// com anexos multipart/MIME reais fica para uma iteração futura, quando
// houver um endereço de e-mail real da contratada e um formato de pacote
// definido). O objetivo deste passo é ter a ação assíncrona (goroutine)
// e o ponto de extensão (Notifier) prontos e funcionais end-to-end.
func (n *smtpNotifier) EnviarPacoteEmpresa(ctx context.Context, processo *models.ProcessoPagamento, anexos []models.DocumentoAnexo) error {
	destinatario := processo.Contrato.ContratadaEmail
	if destinatario == "" {
		return fmt.Errorf("service: contrato %s não tem ContratadaEmail cadastrado — não é possível notificar a empresa", processo.Contrato.NumeroContrato)
	}

	corpo := fmt.Sprintf(
		"Prezados,\r\n\r\nO processo de pagamento referente ao contrato %s (mês %s) avançou para a etapa de emissão de OS.\r\n\r\nDocumentos do pacote:\r\n",
		processo.Contrato.NumeroContrato, processo.MesReferencia,
	)
	for _, anexo := range anexos {
		corpo += fmt.Sprintf("- %s (%s)\r\n", anexo.NomeArquivo, anexo.TipoDocumento.Nome)
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: Processo Selene %s - Pacote de OS\r\n\r\n%s",
		n.cfg.From, destinatario, processo.Contrato.NumeroContrato, corpo,
	)

	auth := smtp.PlainAuth("", n.cfg.User, n.cfg.Password, n.cfg.Host)
	addr := fmt.Sprintf("%s:%s", n.cfg.Host, n.cfg.Port)

	if err := smtp.SendMail(addr, auth, n.cfg.From, []string{destinatario}, []byte(msg)); err != nil {
		return fmt.Errorf("service: falha ao enviar e-mail via SMTP: %w", err)
	}

	return nil
}
