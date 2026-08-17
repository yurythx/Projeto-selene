import type { DocumentoAnexo } from "@/lib/api/client";

/** Um item do checklist visual da página do processo — nome do TipoDocumento exigido + se já foi anexado. */
export interface ItemChecklist {
  nome: string;
  satisfeito: boolean;
}

/**
 * Cruza a lista completa de documentos exigidos pra etapa atual
 * (ProcessoComFiscalizacao.documentos_requeridos, calculada pelo backend
 * via service.RequisitosEtapa — única fonte de verdade da regra, ver o
 * comentário lá) com os documentos já anexados (por TipoDocumento.Nome,
 * mesmo critério que service.ChecklistPendente usa no backend) — produz
 * a lista ✓/x mostrada na página do processo. Documentos anexados que
 * NÃO estão na lista de exigidos (ex: anexados numa etapa anterior, ou
 * um tipo sem restrição nenhuma) não aparecem aqui — este checklist é
 * "o que falta pra sair da etapa atual", não um inventário geral.
 */
export function montarChecklist(
  documentosRequeridos: string[],
  documentosAnexados: DocumentoAnexo[]
): ItemChecklist[] {
  const anexadosPorNome = new Set(
    documentosAnexados
      .map((doc) => doc.TipoDocumento?.Nome)
      .filter((nome): nome is string => Boolean(nome))
  );

  return documentosRequeridos.map((nome) => ({
    nome,
    satisfeito: anexadosPorNome.has(nome),
  }));
}
