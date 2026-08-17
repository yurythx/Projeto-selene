import type { ProcessoPagamento } from "@/lib/api/client";

/** Critérios de busca/filtro do quadro Kanban (Kanban e Lista compartilham
 * o mesmo filtro, ver kanban-board.tsx) — aplicado inteiramente no
 * cliente, sobre os dados que a página já buscou no servidor (todas as
 * etapas de uma vez, ver kanban/page.tsx): o volume esperado (processos
 * em andamento de uma prefeitura) é pequeno o bastante pra isso ser mais
 * simples e mais rápido (sem round-trip a cada tecla) do que levar busca
 * pro backend, diferente de Contratos (ver ContratoRepository.List), que
 * pagina de verdade porque a listagem ali cresce sem limite ao longo dos
 * anos. */
export interface FiltroKanban {
  /** Texto livre, casa contra número do contrato, contratada, CNPJ, fiscal e mês de referência. */
  busca: string;
  /** "" = qualquer tipo. */
  tipoObjeto: string;
}

export const FILTRO_KANBAN_VAZIO: FiltroKanban = { busca: "", tipoObjeto: "" };

function normalizar(texto: string): string {
  return texto.trim().toLowerCase();
}

/** Reporta se `processo` casa com `filtro` — true pra ambos os critérios vazios (nenhum filtro aplicado). */
export function processoCasaFiltro(processo: ProcessoPagamento, filtro: FiltroKanban): boolean {
  if (filtro.tipoObjeto && processo.Contrato?.TipoObjeto !== filtro.tipoObjeto) {
    return false;
  }

  const busca = normalizar(filtro.busca);
  if (!busca) {
    return true;
  }

  const campos = [
    processo.Contrato?.NumeroContrato,
    processo.Contrato?.ContratadaNome,
    processo.Contrato?.ContratadaCNPJ,
    processo.Contrato?.Fiscal?.Nome,
    processo.MesReferencia,
  ];
  return campos.some((campo) => Boolean(campo) && normalizar(campo!).includes(busca));
}

/** Filtra uma lista de processos — usado tanto por coluna (visão Kanban) quanto na lista achatada (visão Lista). */
export function filtrarProcessos(processos: ProcessoPagamento[], filtro: FiltroKanban): ProcessoPagamento[] {
  if (!filtro.busca && !filtro.tipoObjeto) {
    return processos;
  }
  return processos.filter((processo) => processoCasaFiltro(processo, filtro));
}
