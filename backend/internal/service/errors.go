package service

import (
	"errors"
	"fmt"
	"strings"
)

// ErrChecklistIncompleto é retornado por KanbanService.AvancarEtapa quando
// a etapa de origem ainda tem documentos obrigatórios pendentes. Carrega a
// lista de nomes pendentes para que o handler HTTP possa devolvê-la ao
// fiscal em vez de só um "não autorizado" genérico.
type ErrChecklistIncompleto struct {
	Pendentes []string
}

func (e *ErrChecklistIncompleto) Error() string {
	return fmt.Sprintf("checklist incompleto: documento(s) pendente(s): %s", strings.Join(e.Pendentes, ", "))
}

// ErrEtapaFinal é retornado quando se tenta avançar um processo que já
// está na última etapa do Kanban (Contabilidade/Liquidação) — não há
// próxima etapa; o processo só pode ser concluído (ver ConcluirPagamento).
var ErrEtapaFinal = errors.New("service: processo já está na etapa final do kanban")

// ErrProcessoNaoElegivelParaConclusao é retornado quando se tenta concluir
// um processo que ainda não chegou na última etapa do Kanban.
var ErrProcessoNaoElegivelParaConclusao = errors.New("service: processo só pode ser concluído a partir da etapa final do kanban")

// ErrFiscalInvalido é retornado ao tentar criar um contrato cujo FiscalID
// não corresponde a um usuário com IsFiscal=true — também usado por
// DesignacaoService.Designar para os papéis FISCAL/FISCAL_SUPLENTE, mesma
// regra (ver ErrServidorInvalido para os demais papéis).
var ErrFiscalInvalido = errors.New("service: usuário informado não é um fiscal habilitado")

// ErrServidorInvalido é retornado por DesignacaoService.Designar quando
// ServidorID não corresponde a nenhum usuário cadastrado. ACHADO DE
// REVISÃO: antes desta checagem, Designar não validava o servidor de
// jeito nenhum (ao contrário de ContratoService.Criar, que sempre validou
// o fiscal) — dava pra registrar uma Portaria de designação apontando pra
// um ID inexistente, corrompendo silenciosamente o histórico auditável
// que a norma exige (IN01 Art.6º / IN04 Art.10).
var ErrServidorInvalido = errors.New("service: servidor designado não encontrado")

// ErrTipoObjetoInvalido é retornado quando o TipoObjeto informado não é um
// dos três valores válidos do domínio (CONSUMO, PERMANENTE, SERVICO).
var ErrTipoObjetoInvalido = errors.New("service: tipo de objeto inválido")

// ErrContratoEncerrado é retornado ao tentar abrir um novo processo de
// pagamento para um contrato marcado como inativo (Contrato.Ativo=false).
var ErrContratoEncerrado = errors.New("service: contrato está encerrado, não aceita novos processos de pagamento")

// ErrMotivoObrigatorio é retornado pelo Gerador de Documentos (Fase 2 do
// roadmap) quando a Notificação de Descumprimento ou a Minuta de Aditivo
// são geradas sem um motivo/justificativa — os dois únicos tipos que
// exigem esse campo (o Atesto não pede motivo).
var ErrMotivoObrigatorio = errors.New("service: motivo/justificativa é obrigatório")

// ErrFornecedorNaoEncontrado é retornado pelo Dossiê do Fornecedor (Fase 4
// do roadmap) quando nenhum contrato cadastrado tem o CNPJ informado.
var ErrFornecedorNaoEncontrado = errors.New("service: nenhum contrato encontrado para este CNPJ")

// ErrCredenciaisInvalidas é retornado pelo login local (AuthService) tanto
// pra e-mail inexistente quanto pra senha errada quanto pra conta sem
// senha local (provisionada via Keycloak) — deliberadamente a MESMA
// mensagem nos três casos, pra não revelar por resposta (nem por tempo —
// ver o comentário em AuthService.Login sobre a comparação bcrypt contra
// um hash de preenchimento) se um e-mail está cadastrado no sistema.
var ErrCredenciaisInvalidas = errors.New("service: e-mail ou senha inválidos")

// ErrSenhaFraca é retornado quando a nova senha (troca de senha) não
// atende o tamanho mínimo.
var ErrSenhaFraca = errors.New("service: senha precisa ter pelo menos 8 caracteres")

// ErrContaSemSenhaLocal é retornado ao tentar trocar a senha de uma conta
// provisionada via Keycloak (PasswordHash nulo) — a senha dela é
// gerenciada pelo Keycloak, não pelo Selene.
var ErrContaSemSenhaLocal = errors.New("service: esta conta não usa senha local (login via Keycloak)")

// ErrValorInvalido é retornado quando um valor monetário informado
// (Empenho.ValorInicial ou MovimentacaoEmpenho.Valor) não é positivo —
// ver EmpenhoService.
var ErrValorInvalido = errors.New("service: valor precisa ser maior que zero")

// ErrTipoMovimentacaoInvalido é retornado ao tentar registrar
// manualmente uma movimentação do tipo INICIAL — esse tipo só é criado
// automaticamente por EmpenhoService.CriarEmpenho, nunca via
// RegistrarMovimentacao.
var ErrTipoMovimentacaoInvalido = errors.New("service: tipo de movimentação inválido para este lançamento")

// ErrSaldoInsuficiente é retornado por EmpenhoService.RegistrarMovimentacao
// ao tentar lançar uma ANULACAO ou FATURA_APROPRIADA maior que o saldo
// atual do empenho — sem esta checagem, o saldo reconstruído por
// CalcularSaldo podia virar negativo, o que não tem significado
// orçamentário (não dá pra anular ou apropriar mais do que o empenho tem
// disponível). ACHADO DE REVISÃO: nenhuma validação de saldo existia antes
// desta constante, único ponto do domínio financeiro do SGF sem trava
// equivalente ao que já protegia Contrato/Ocorrência/Designação.
var ErrSaldoInsuficiente = errors.New("service: valor da movimentação excede o saldo disponível do empenho")

// ErrTransicaoOcorrenciaInvalida é retornado ao tentar mover uma
// Ocorrencia para um estado que não é o próximo passo válido do fluxo
// linear REGISTRADA→NOTIFICADA→EM_TRATAMENTO→REGULARIZADA (ver
// OcorrenciaService e o comentário em models.EstadoOcorrencia).
var ErrTransicaoOcorrenciaInvalida = errors.New("service: transição de estado da ocorrência não permitida a partir do estado atual")

// ErrDataInvalida é retornado quando uma data recebida da API (formato
// esperado "AAAA-MM-DD", ver parseData em util.go) não pôde ser
// interpretada — usado via `fmt.Errorf("...: %w: %w", ErrDataInvalida,
// errOriginal)` para que errors.Is funcione e o handler mapeie pra 400,
// preservando o erro original de time.Parse na mensagem/log.
//
// ACHADO DE REVISÃO: antes desta constante existir, os três pontos de
// parseData em ContratoService (DataAssinatura/DataVigenciaFim em Criar e
// Atualizar) só envolviam o erro de time.Parse com fmt.Errorf, sem
// nenhum sentinel — respondError (handler/errors.go) não reconhecia esse
// erro e caía no default (500 "erro interno"), quando o correto é 400
// (erro do cliente, data mal formatada). Corrigido nos três call sites
// junto com a introdução desta constante.
var ErrDataInvalida = errors.New("service: data inválida")

// ErrCNPJInvalido é retornado quando ContratadaCNPJ não tem o formato de
// um CNPJ (14 dígitos, com ou sem máscara) — ver cnpjValido em util.go.
// ACHADO DE REVISÃO: antes desta checagem existir, qualquer string não
// vazia era aceita como CNPJ (só o `binding:"required"` do handler barrava
// o campo em branco), o que corrompia silenciosamente o agrupamento por
// CNPJ do Dossiê do Fornecedor (FornecedorService, Fase 4).
var ErrCNPJInvalido = errors.New("service: CNPJ inválido — informe os 14 dígitos")

// ErrEmailInvalido é retornado quando ContratadaEmail é preenchido mas não
// tem formato de e-mail — o zod do formulário já valida isso no client,
// esta é a defesa em profundidade no backend (mesmo espírito do
// bff-schemas.ts no frontend) para quem chamar a API diretamente.
var ErrEmailInvalido = errors.New("service: e-mail da contratada inválido")

// ErrVigenciaAntesDaAssinatura é retornado quando DataVigenciaFim é
// anterior (ou igual) a DataAssinatura — um contrato não pode vencer antes
// (ou no mesmo dia) de ser assinado. Sem esta checagem, um erro de
// digitação na data de vigência silenciosamente corrompe o cálculo de
// "dias restantes" do Radar de Alertas (RadarService).
var ErrVigenciaAntesDaAssinatura = errors.New("service: data de vigência final precisa ser posterior à data de assinatura")

// ErrCategoriaObrigatoria é retornado ao criar/renomear uma categoria de
// modelo de documento com nome vazio (só espaços em branco também conta).
var ErrCategoriaObrigatoria = errors.New("service: categoria é obrigatória")

// ErrArquivoNaoEDocx é retornado quando o arquivo enviado como modelo de
// documento não é um .docx válido (assinatura ZIP ausente, ou a
// biblioteca de merge não conseguiu abri-lo como Office Open XML) —
// mensagem amigável em vez de vazar o erro cru de parsing XML pro admin.
var ErrArquivoNaoEDocx = errors.New("service: arquivo enviado não é um .docx válido")

// ErrOcorrenciaAbertaBloqueiaAvanco é retornado por
// FiscalizacaoService.VerificarAvancoPermitido quando existe uma
// Ocorrencia vinculada ao processo cujo Estado ainda não é REGULARIZADA —
// trava de ESCRITA de Camada 2 (regra do SGF, não da norma; confirmada
// explicitamente com o usuário), aplicada pelo handler ANTES de chamar
// KanbanService.AvancarEtapa.
var ErrOcorrenciaAbertaBloqueiaAvanco = errors.New("service: existe ocorrência aberta vinculada a este processo — regularize antes de avançar")

// ErrTipoDocumentoNaoAplicavel é retornado por DocumentoService.Upload
// quando o TipoDocumento escolhido é restrito a um tipo de contrato (ex:
// "Boleto DAM" só se aplica a SERVICO) ou a contratos com
// ExigeFiscalizacaoTerceirizacao=true, e o contrato do processo não
// atende essa condição — ver service.TipoDocumentoAplicavel.
var ErrTipoDocumentoNaoAplicavel = errors.New("service: este tipo de documento não se aplica ao tipo de contrato deste processo")

// ErrTipoDocumentoJaAnexado é retornado por DocumentoService.Upload
// quando o processo já tem um documento anexado do mesmo TipoDocumento —
// pedido explícito do usuário: no máximo um de cada tipo por processo
// (ex: não pode ter dois "Pré-Empenho"). Exclua o anterior antes de
// enviar outro.
var ErrTipoDocumentoJaAnexado = errors.New("service: já existe um documento deste tipo anexado a este processo — exclua o anterior antes de enviar outro")

// ErrTipoDocumentoNaoExigidoAinda é retornado por DocumentoService.Upload
// quando o TipoDocumento escolhido não faz parte do checklist cumulado
// até a etapa atual do processo (ver RequisitosAcumulados) — ex: tentar
// anexar uma CND (só exigida na Etapa 5) enquanto o processo ainda está
// na Etapa 1. Pedido explícito do usuário: só oferecer/aceitar os
// documentos que "realmente podem ser inseridos na etapa".
var ErrTipoDocumentoNaoExigidoAinda = errors.New("service: este tipo de documento não faz parte do checklist até a etapa atual deste processo")

// ErrKeycloakClientIDObrigatorio é retornado por
// KeycloakConfigService.Salvar quando client_id vem vazio.
var ErrKeycloakClientIDObrigatorio = errors.New("service: client_id é obrigatório")

// ErrKeycloakSegredoObrigatorio é retornado quando client_secret vem
// vazio na PRIMEIRA configuração salva (não há um segredo anterior pra
// manter) — em atualizações seguintes, vazio é interpretado como "manter
// o atual", ver o comentário em KeycloakConfigService.Salvar.
var ErrKeycloakSegredoObrigatorio = errors.New("service: client_secret é obrigatório (não há uma configuração anterior pra manter o segredo atual)")

// ErrKeycloakIssuerInvalido é retornado quando issuer_url não é uma URL
// http(s) bem formada, ou quando o Keycloak nesse endereço não responde
// ao buscar o JWKS (ver KeycloakConfigService.Salvar — a validação de
// rede acontece ANTES de persistir, fail-closed).
var ErrKeycloakIssuerInvalido = errors.New("service: issuer_url inválido")

// ErrDiarioOficialURLInvalida é retornado quando base_url não é uma URL
// http(s) bem formada.
var ErrDiarioOficialURLInvalida = errors.New("service: base_url inválida")

// ErrDiarioOficialChaveObrigatoria é retornado quando api_key vem vazia
// na PRIMEIRA configuração salva — mesmo espírito de
// ErrKeycloakSegredoObrigatorio (vazio em atualizações seguintes = manter
// a chave atual).
var ErrDiarioOficialChaveObrigatoria = errors.New("service: api_key é obrigatória (não há uma configuração anterior pra manter a chave atual)")

// ErrDiarioOficialNaoConfigurado é retornado por TestarConexao/
// BuscarContratos quando nenhum admin salvou uma configuração ainda.
var ErrDiarioOficialNaoConfigurado = errors.New("service: integração com o diário oficial ainda não foi configurada")

// ErrDiarioOficialFalhaNaBusca é retornado quando a API externa do
// Diário Oficial não responde, responde com erro, ou devolve algo que
// não é JSON válido — engloba o erro original via %w em cada call site
// (DiarioOficialService.BuscarContratos), não é usado sozinho.
var ErrDiarioOficialFalhaNaBusca = errors.New("service: falha ao consultar a API do diário oficial")
