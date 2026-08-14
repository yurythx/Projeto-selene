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
