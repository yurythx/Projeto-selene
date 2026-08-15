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
// não corresponde a um usuário com IsFiscal=true.
var ErrFiscalInvalido = errors.New("service: usuário informado não é um fiscal habilitado")

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

// ErrOcorrenciaAbertaBloqueiaAvanco é retornado por
// FiscalizacaoService.VerificarAvancoPermitido quando existe uma
// Ocorrencia vinculada ao processo cujo Estado ainda não é REGULARIZADA —
// trava de ESCRITA de Camada 2 (regra do SGF, não da norma; confirmada
// explicitamente com o usuário), aplicada pelo handler ANTES de chamar
// KanbanService.AvancarEtapa.
var ErrOcorrenciaAbertaBloqueiaAvanco = errors.New("service: existe ocorrência aberta vinculada a este processo — regularize antes de avançar")
