package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// checklistBase mapeia o ID de cada etapa do Kanban (a etapa de ORIGEM de
// uma transição, i.e. a etapa que o card está deixando) para os nomes dos
// TipoDocumento exigidos antes de permitir o avanço — Seção 5 da
// documentação de domínio. Etapas sem entrada aqui (2 e 6) não têm
// checklist de saída: a Coluna 2 é apenas tramitação externa read-only, e
// a Coluna 6 é terminal (não avança, apenas é marcada como paga).
var checklistBase = map[int][]string{
	1: {
		"Ordem de Fornecimento (OF)",
		"Pré-Empenho",
		"Ofício de Solicitação",
	},
	3: {
		"Nota de Empenho",
	},
	4: {
		"Nota Fiscal / Fatura",
		"Ordem de Recepção",
	},
	5: {
		"Extrato do Empenho",
		"Declaração do Simples Nacional",
		"CND Trabalhista",
		"CND FGTS",
		"CND Municipal",
		"CND Estadual",
		"CND Federal",
		"CND INSS",
		// Aprovação física: o relatório gerado, assinado pelo Secretário e
		// com o atesto escaneado, precisa ser reanexado antes do avanço.
		"Relatório de Pagamento Assinado",
	},
}

// checklistCondicionalServico são os documentos exigidos ADICIONALMENTE na
// Etapa 5 quando o contrato é da categoria SERVICO (Seção 5, "Regra
// Condicional por Tipo").
var checklistCondicionalServico = []string{
	"Planilha de Medição de Serviços",
	"Boleto DAM",
}

// checklistCondicionalTerceirizacao são os documentos mensais exigidos
// ADICIONALMENTE na Etapa 5 quando o contrato está marcado com
// ExigeFiscalizacaoTerceirizacao (Camada 2, ver models.Contrato) —
// documentação trabalhista/previdenciária específica de mão de obra
// terceirizada, IN SCL Nº 04/2021 Art.9º-XXXII, alíneas a/b.1/b.2/b.3.
// Este é um subconjunto deliberadamente estreito do Anexo III da IN04
// (42 itens no total, questionário de autoavaliação do fiscal) — só os
// itens genuinamente documentais (comprovante/protocolo/guia/relação a
// anexar) entraram aqui; o restante do Anexo III não tem formato de
// checklist de anexo e não deve ser forçado neste modelo.
var checklistCondicionalTerceirizacao = []string{
	"Comprovante de Pagamento de Salário",
	"Protocolo GFIP",
	"Guia GRF/GPS",
	"Relação de Trabalhadores (SEFIP)",
}

// RequisitosEtapa retorna a lista completa de nomes de TipoDocumento
// exigidos para sair da etapaOrigemID, já incluindo os itens condicionais
// de acordo com o tipoObjeto e o exigeFiscalizacaoTerceirizacao do
// contrato. Etapas sem checklist (2 e 6) retornam uma lista vazia.
func RequisitosEtapa(etapaOrigemID int, tipoObjeto models.TipoObjeto, exigeFiscalizacaoTerceirizacao bool) []string {
	base, ok := checklistBase[etapaOrigemID]
	if !ok {
		return nil
	}

	requisitos := make([]string, len(base))
	copy(requisitos, base)

	if etapaOrigemID == 5 {
		if tipoObjeto == models.TipoObjetoServico {
			requisitos = append(requisitos, checklistCondicionalServico...)
		}
		if exigeFiscalizacaoTerceirizacao {
			requisitos = append(requisitos, checklistCondicionalTerceirizacao...)
		}
	}

	return requisitos
}

// RequisitosAcumulados retorna a união dos requisitos de TODAS as etapas
// já percorridas até etapaAtualID (inclusive) — etapas 1..etapaAtualID —
// não só os da etapa atual isolada (RequisitosEtapa). Existe porque
// DocumentoService.Excluir permite apagar qualquer documento anexado,
// inclusive um que satisfazia o checklist de uma etapa JÁ CONCLUÍDA
// (decisão deliberada — ver o comentário lá: a auditoria em kanban_logs
// continua íntegra, só o anexo em si some) — sem verificar de forma
// cumulativa, o processo continuaria avançando com essa lacuna aberta.
// Pedido explícito do usuário: "os arquivos que foram obrigatórios nas
// etapas anteriores podem ser apagados, mas precisamos subir outro do
// mesmo modelo, senão não iremos prosseguir — fazer essa verificação em
// todas as etapas".
func RequisitosAcumulados(etapaAtualID int, tipoObjeto models.TipoObjeto, exigeFiscalizacaoTerceirizacao bool) []string {
	var todos []string
	for etapa := 1; etapa <= etapaAtualID; etapa++ {
		todos = append(todos, RequisitosEtapa(etapa, tipoObjeto, exigeFiscalizacaoTerceirizacao)...)
	}
	return todos
}

// TipoDocumentoAplicavel reporta se `tipo` pode ser anexado (e deve
// aparecer no select de upload) num processo cujo contrato tem o
// tipoObjeto e a flag de terceirização informados. Usa
// TipoDocumento.RestritoTipoObjeto/RestritoTerceirizacao — dados
// semeados a partir das mesmas duas listas condicionais acima
// (checklistCondicionalServico/checklistCondicionalTerceirizacao), então
// nunca diverge do que RequisitosEtapa exige pra avançar a Etapa 5.
// Documentos sem nenhuma restrição (a maioria — os da Coluna 1/3/4 e as
// CNDs da Coluna 5) são sempre aplicáveis, em qualquer tipo de contrato.
func TipoDocumentoAplicavel(tipo models.TipoDocumento, tipoObjeto models.TipoObjeto, exigeFiscalizacaoTerceirizacao bool) bool {
	if tipo.RestritoTipoObjeto != nil && *tipo.RestritoTipoObjeto != tipoObjeto {
		return false
	}
	if tipo.RestritoTerceirizacao && !exigeFiscalizacaoTerceirizacao {
		return false
	}
	return true
}

// ChecklistPendente compara os documentos exigidos ATÉ a etapaOrigemID
// (cumulativo — RequisitosAcumulados, não só o da etapa atual isolada;
// ver o comentário lá) com os documentos já anexados ao processo, e
// retorna os nomes dos que ainda faltam. Uma lista vazia significa
// checklist satisfeito — o card pode avançar.
//
// Documentos anexados em etapas anteriores contam automaticamente aqui
// (reaproveitamento de arquivos por card, Seção 7.2): a checagem é sobre
// TODOS os documentos do processo, não apenas os anexados na etapa atual.
func ChecklistPendente(
	ctx context.Context,
	docRepo repository.DocumentoAnexoRepository,
	processoID uuid.UUID,
	etapaOrigemID int,
	tipoObjeto models.TipoObjeto,
	exigeFiscalizacaoTerceirizacao bool,
) ([]string, error) {
	requisitos := RequisitosAcumulados(etapaOrigemID, tipoObjeto, exigeFiscalizacaoTerceirizacao)
	if len(requisitos) == 0 {
		return nil, nil
	}

	documentos, err := docRepo.ListByProcesso(ctx, processoID)
	if err != nil {
		return nil, fmt.Errorf("service: carregar documentos anexados para checklist: %w", err)
	}

	anexados := make(map[string]bool, len(documentos))
	for _, doc := range documentos {
		if doc.TipoDocumento != nil {
			anexados[doc.TipoDocumento.Nome] = true
		}
	}

	var pendentes []string
	for _, nome := range requisitos {
		if !anexados[nome] {
			pendentes = append(pendentes, nome)
		}
	}

	return pendentes, nil
}
