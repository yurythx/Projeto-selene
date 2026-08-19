import type { ItemRadar } from "@/lib/api/client";

/**
 * Helpers de correlação do Radar de Alertas (Fase 1 do roadmap) — usados
 * tanto pelos badges do Kanban quanto pela página /radar. Vivem fora de
 * "server-only" (client.ts) porque o KanbanBoard ("use client") também
 * precisa deles pra decidir o que mostrar em cada card.
 */

/** Retorna os itens do radar relevantes pra um processo específico:
 * alertas de vigência de contrato (correlacionados pelo contrato) e
 * alertas de certidão/processo-parado (correlacionados pelo processo). */
export function itensDoProcesso(
  itens: ItemRadar[],
  processo: { ID?: string; ContratoID?: string }
): ItemRadar[] {
  return itens.filter((item) =>
    item.tipo === "vigencia_contrato"
      ? item.contrato_id === processo.ContratoID
      : item.processo_id === processo.ID
  );
}

/** Nível mais crítico entre uma lista de itens, ou null se a lista estiver vazia. */
export function nivelMaisCritico(itens: ItemRadar[]): ItemRadar["nivel"] | null {
  if (itens.some((item) => item.nivel === "CRITICO")) return "CRITICO";
  if (itens.some((item) => item.nivel === "ATENCAO")) return "ATENCAO";
  return null;
}

export const RADAR_TIPO_LABEL: Record<NonNullable<ItemRadar["tipo"]>, string> = {
  vigencia_contrato: "Vigência do contrato",
  certidao: "Certidão vencida/vencendo",
  processo_parado: "Processo parado",
};
