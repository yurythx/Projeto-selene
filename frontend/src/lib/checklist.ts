import type { DocumentoAnexo } from "@/lib/api/client";

/**
 * Um item do checklist visual da página do processo — nome do
 * TipoDocumento exigido, se já foi anexado, e (quando satisfeito) o
 * próprio DocumentoAnexo correspondente, pra permitir pré-visualizá-lo
 * direto do checklist (mesmo dialog usado em "Documentos anexados") sem
 * precisar procurar o mesmo nome na outra lista.
 */
export interface ItemChecklist {
  nome: string;
  satisfeito: boolean;
  documento?: DocumentoAnexo;
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
  const anexadosPorNome = new Map<string, DocumentoAnexo>();
  for (const doc of documentosAnexados) {
    const nome = doc.TipoDocumento?.Nome;
    // Regra de unicidade por tipo (ver ErrTipoDocumentoJaAnexado no
    // backend) garante no máximo um documento por nome — não há
    // ambiguidade de qual usar aqui.
    if (nome && !anexadosPorNome.has(nome)) {
      anexadosPorNome.set(nome, doc);
    }
  }

  return documentosRequeridos.map((nome) => {
    const documento = anexadosPorNome.get(nome);
    return { nome, satisfeito: Boolean(documento), documento };
  });
}
