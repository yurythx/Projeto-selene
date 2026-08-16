// Rótulos e opções compartilhados entre a listagem, o detalhe e os
// dialogs de Modelos de Documentos (Configurações) — mesmos 4 valores de
// models.TipoGatilhoModelo no backend.
export const GATILHO_LABEL: Record<string, string> = {
  NOTIFICACAO_DESCUMPRIMENTO: "Ofício (Notificação de Descumprimento)",
  MINUTA_ADITIVO: "Minuta de Aditivo",
  ATESTO: "Atesto",
  RELATORIO_PAGAMENTO: "Relatório de Pagamento",
};

export const GATILHOS = Object.keys(GATILHO_LABEL) as (keyof typeof GATILHO_LABEL)[];
