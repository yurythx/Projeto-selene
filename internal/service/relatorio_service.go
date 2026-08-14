package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/google/uuid"

	"projeto-selene/internal/repository"
)

// relatorioTemplate é o modelo do Relatório de Pagamento (Seção 5, Coluna
// 5, "Ação Automatizada em Go"). Preenche as tags do fiscal, contrato,
// portaria e itens cadastrados no Selene.
//
// LIMITAÇÃO CONHECIDA: a documentação de domínio não forneceu o arquivo
// oficial do modelo de Relatório de Pagamento usado pela prefeitura (o
// layout jurídico/visual real). Este template HTML é um substituto
// funcional que preenche as mesmas tags — o fiscal imprime, colhe a
// assinatura física do Secretário e o atesto no verso da Nota Fiscal, e
// reanexa o PDF escaneado (documento "Relatório de Pagamento Assinado" no
// checklist da Etapa 5). Trocar por um template oficial (DOCX/PDF) é uma
// mudança isolada nesta função, sem impacto no resto do fluxo.
const relatorioTemplate = `<!DOCTYPE html>
<html lang="pt-BR">
<head><meta charset="utf-8"><title>Relatório de Pagamento — {{.NumeroContrato}}</title></head>
<body>
	<h1>Relatório de Pagamento</h1>
	<table border="1" cellpadding="6" cellspacing="0">
		<tr><th align="left">Contrato</th><td>{{.NumeroContrato}}</td></tr>
		<tr><th align="left">Portaria de Nomeação</th><td>{{.PortariaNomeacao}}</td></tr>
		<tr><th align="left">Fiscal Responsável</th><td>{{.FiscalNome}}</td></tr>
		<tr><th align="left">Contratada</th><td>{{.ContratadaNome}} (CNPJ {{.ContratadaCNPJ}})</td></tr>
		<tr><th align="left">Tipo de Objeto</th><td>{{.TipoObjeto}}</td></tr>
		<tr><th align="left">Mês de Referência</th><td>{{.MesReferencia}}</td></tr>
	</table>

	<h2>Documentos Anexados ao Processo</h2>
	<ul>
	{{range .Documentos}}
		<li>{{.TipoDocumento}} — {{.NomeArquivo}}</li>
	{{else}}
		<li>Nenhum documento anexado ainda.</li>
	{{end}}
	</ul>

	<p>________________________________<br>Assinatura do Secretário da Pasta</p>
	<p><em>Atesto físico no verso da Nota Fiscal — anexar escaneado como "Relatório de Pagamento Assinado".</em></p>
</body>
</html>
`

type relatorioDocumentoView struct {
	TipoDocumento string
	NomeArquivo   string
}

type relatorioView struct {
	NumeroContrato   string
	PortariaNomeacao string
	FiscalNome       string
	ContratadaNome   string
	ContratadaCNPJ   string
	TipoObjeto       string
	MesReferencia    string
	Documentos       []relatorioDocumentoView
}

// RelatorioService gera o Relatório de Pagamento de um processo,
// preenchendo o template com os dados do contrato/fiscal cadastrados no
// Selene e a lista de documentos já anexados.
type RelatorioService struct {
	processoRepo repository.ProcessoPagamentoRepository
	docRepo      repository.DocumentoAnexoRepository
	tmpl         *template.Template
}

// NewRelatorioService constrói um RelatorioService, fazendo o parse do
// template uma única vez na inicialização (falha rápido no boot se o
// template estiver malformado, em vez de falhar silenciosamente na
// primeira requisição).
func NewRelatorioService(processoRepo repository.ProcessoPagamentoRepository, docRepo repository.DocumentoAnexoRepository) (*RelatorioService, error) {
	tmpl, err := template.New("relatorio").Parse(relatorioTemplate)
	if err != nil {
		return nil, fmt.Errorf("service: falha ao compilar template do relatório de pagamento: %w", err)
	}

	return &RelatorioService{processoRepo: processoRepo, docRepo: docRepo, tmpl: tmpl}, nil
}

// Gerar renderiza o Relatório de Pagamento (HTML) do processo informado.
func (s *RelatorioService) Gerar(ctx context.Context, processoID uuid.UUID) ([]byte, error) {
	processo, err := s.processoRepo.FindByID(ctx, processoID)
	if err != nil {
		return nil, fmt.Errorf("service: carregar processo para relatório: %w", err)
	}

	documentos, err := s.docRepo.ListByProcesso(ctx, processoID)
	if err != nil {
		return nil, fmt.Errorf("service: carregar documentos para relatório: %w", err)
	}

	view := relatorioView{
		NumeroContrato:   processo.Contrato.NumeroContrato,
		PortariaNomeacao: processo.Contrato.PortariaNomeacao,
		ContratadaNome:   processo.Contrato.ContratadaNome,
		ContratadaCNPJ:   processo.Contrato.ContratadaCNPJ,
		TipoObjeto:       string(processo.Contrato.TipoObjeto),
		MesReferencia:    processo.MesReferencia,
	}
	if processo.Contrato.Fiscal != nil {
		view.FiscalNome = processo.Contrato.Fiscal.Nome
	}
	for _, doc := range documentos {
		nomeTipo := ""
		if doc.TipoDocumento != nil {
			nomeTipo = doc.TipoDocumento.Nome
		}
		view.Documentos = append(view.Documentos, relatorioDocumentoView{
			TipoDocumento: nomeTipo,
			NomeArquivo:   doc.NomeArquivo,
		})
	}

	var buf bytes.Buffer
	if err := s.tmpl.Execute(&buf, view); err != nil {
		return nil, fmt.Errorf("service: renderizar relatório de pagamento: %w", err)
	}

	return buf.Bytes(), nil
}
