import { describe, it, expect } from "vitest";
import { montarChecklist } from "./checklist";
import type { DocumentoAnexo } from "@/lib/api/client";

function doc(nome: string): DocumentoAnexo {
  return { ID: nome, TipoDocumento: { Nome: nome } } as DocumentoAnexo;
}

describe("montarChecklist", () => {
  it("marca como satisfeito só o que está anexado", () => {
    const resultado = montarChecklist(
      ["Ordem de Fornecimento (OF)", "Pré-Empenho", "Ofício de Solicitação"],
      [doc("Ordem de Fornecimento (OF)")]
    );
    expect(resultado).toEqual([
      { nome: "Ordem de Fornecimento (OF)", satisfeito: true },
      { nome: "Pré-Empenho", satisfeito: false },
      { nome: "Ofício de Solicitação", satisfeito: false },
    ]);
  });

  it("lista vazia de exigidos produz checklist vazio", () => {
    expect(montarChecklist([], [doc("Nota Fiscal / Fatura")])).toEqual([]);
  });

  it("documento anexado sem TipoDocumento carregado não quebra", () => {
    const semTipo = { ID: "x", TipoDocumento: undefined } as DocumentoAnexo;
    const resultado = montarChecklist(["Pré-Empenho"], [semTipo]);
    expect(resultado).toEqual([{ nome: "Pré-Empenho", satisfeito: false }]);
  });

  it("todos satisfeitos", () => {
    const resultado = montarChecklist(["A", "B"], [doc("A"), doc("B")]);
    expect(resultado.every((item) => item.satisfeito)).toBe(true);
  });
});
