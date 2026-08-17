import { describe, it, expect } from "vitest";
import { filtrarProcessos, processoCasaFiltro, type FiltroKanban } from "./kanban-filtro";
import type { ProcessoPagamento } from "@/lib/api/client";

function processo(overrides: Partial<ProcessoPagamento> = {}): ProcessoPagamento {
  return {
    ID: "p1",
    MesReferencia: "07/2026",
    EtapaAtualID: 1,
    Status: "Ativo",
    Contrato: {
      NumeroContrato: "10/2026",
      ContratadaNome: "Fornecedora Alfa Ltda",
      ContratadaCNPJ: "11.111.111/0001-11",
      TipoObjeto: "SERVICO",
      Fiscal: { Nome: "Fiscal Um" },
    },
    ...overrides,
  } as ProcessoPagamento;
}

const SEM_FILTRO: FiltroKanban = { busca: "", tipoObjeto: "" };

describe("processoCasaFiltro", () => {
  it("sem filtro nenhum, sempre casa", () => {
    expect(processoCasaFiltro(processo(), SEM_FILTRO)).toBe(true);
  });

  it("busca casa número do contrato", () => {
    expect(processoCasaFiltro(processo(), { busca: "10/2026", tipoObjeto: "" })).toBe(true);
  });

  it("busca casa nome da contratada, sem diferenciar caixa", () => {
    expect(processoCasaFiltro(processo(), { busca: "fornecedora alfa", tipoObjeto: "" })).toBe(true);
  });

  it("busca casa CNPJ", () => {
    expect(processoCasaFiltro(processo(), { busca: "11.111.111", tipoObjeto: "" })).toBe(true);
  });

  it("busca casa nome do fiscal", () => {
    expect(processoCasaFiltro(processo(), { busca: "fiscal um", tipoObjeto: "" })).toBe(true);
  });

  it("busca casa mês de referência", () => {
    expect(processoCasaFiltro(processo(), { busca: "07/2026", tipoObjeto: "" })).toBe(true);
  });

  it("busca sem correspondência não casa", () => {
    expect(processoCasaFiltro(processo(), { busca: "não existe", tipoObjeto: "" })).toBe(false);
  });

  it("filtro de tipo de objeto exato", () => {
    expect(processoCasaFiltro(processo(), { busca: "", tipoObjeto: "SERVICO" })).toBe(true);
    expect(processoCasaFiltro(processo(), { busca: "", tipoObjeto: "CONSUMO" })).toBe(false);
  });

  it("busca e tipo combinados: os dois precisam casar", () => {
    expect(processoCasaFiltro(processo(), { busca: "Alfa", tipoObjeto: "SERVICO" })).toBe(true);
    expect(processoCasaFiltro(processo(), { busca: "Alfa", tipoObjeto: "CONSUMO" })).toBe(false);
  });

  it("processo sem Contrato carregado não quebra, só não casa busca por texto", () => {
    expect(processoCasaFiltro(processo({ Contrato: undefined }), { busca: "Alfa", tipoObjeto: "" })).toBe(false);
    expect(processoCasaFiltro(processo({ Contrato: undefined }), SEM_FILTRO)).toBe(true);
  });
});

describe("filtrarProcessos", () => {
  it("filtra uma lista mantendo só os que casam", () => {
    const lista = [
      processo({ ID: "a", Contrato: { NumeroContrato: "1/2026", ContratadaNome: "Alfa", TipoObjeto: "SERVICO" } }),
      processo({ ID: "b", Contrato: { NumeroContrato: "2/2026", ContratadaNome: "Beta", TipoObjeto: "CONSUMO" } }),
    ];
    const resultado = filtrarProcessos(lista, { busca: "Beta", tipoObjeto: "" });
    expect(resultado.map((p) => p.ID)).toEqual(["b"]);
  });

  it("sem filtro, devolve a mesma lista (mesma referência, sem cópia desnecessária)", () => {
    const lista = [processo()];
    expect(filtrarProcessos(lista, SEM_FILTRO)).toBe(lista);
  });
});
