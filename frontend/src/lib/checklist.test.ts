import { describe, it, expect } from "vitest";
import { montarChecklist } from "./checklist";
import type { DocumentoAnexo } from "@/lib/api/client";

function doc(nome: string): DocumentoAnexo {
  return { ID: nome, TipoDocumento: { Nome: nome } } as DocumentoAnexo;
}

describe("montarChecklist", () => {
  it("marca como satisfeito só o que está anexado, e carrega o documento correspondente", () => {
    const documentoOF = doc("Ordem de Fornecimento (OF)");
    const resultado = montarChecklist(
      ["Ordem de Fornecimento (OF)", "Pré-Empenho", "Ofício de Solicitação"],
      [documentoOF]
    );
    expect(resultado).toEqual([
      { nome: "Ordem de Fornecimento (OF)", satisfeito: true, documento: documentoOF },
      { nome: "Pré-Empenho", satisfeito: false, documento: undefined },
      { nome: "Ofício de Solicitação", satisfeito: false, documento: undefined },
    ]);
  });

  it("lista vazia de exigidos produz checklist vazio", () => {
    expect(montarChecklist([], [doc("Nota Fiscal / Fatura")])).toEqual([]);
  });

  it("documento anexado sem TipoDocumento carregado não quebra, e fica sem documento pra pré-visualizar", () => {
    const semTipo = { ID: "x", TipoDocumento: undefined } as DocumentoAnexo;
    const resultado = montarChecklist(["Pré-Empenho"], [semTipo]);
    expect(resultado).toEqual([{ nome: "Pré-Empenho", satisfeito: false, documento: undefined }]);
  });

  it("todos satisfeitos, cada um com seu documento", () => {
    const documentoA = doc("A");
    const documentoB = doc("B");
    const resultado = montarChecklist(["A", "B"], [documentoA, documentoB]);
    expect(resultado.every((item) => item.satisfeito)).toBe(true);
    expect(resultado.map((item) => item.documento)).toEqual([documentoA, documentoB]);
  });
});
