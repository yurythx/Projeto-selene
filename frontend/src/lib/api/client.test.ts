import { describe, it, expect, vi, afterEach } from "vitest";
import {
  apiFetch,
  ApiError,
  listarContratos,
  criarContrato,
  listarProcessos,
  avancarProcesso,
  anexarDocumento,
} from "./client";

describe("apiFetch", () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("injeta o Authorization: Bearer <token> e monta a URL a partir de API_URL", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200 })
    );
    global.fetch = fetchMock as unknown as typeof fetch;

    await apiFetch("/api/v1/contratos", "meu-token");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://backend.test/api/v1/contratos");
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer meu-token");
  });

  it("define Content-Type: application/json só quando há corpo", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }));
    global.fetch = fetchMock as unknown as typeof fetch;

    await apiFetch("/x", "token", { method: "POST", body: JSON.stringify({ a: 1 }) });

    const [, init] = fetchMock.mock.calls[0];
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
  });

  it("lança ApiError com status e corpo quando a resposta não é ok", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: "não autorizado" }), { status: 403 })
    );
    global.fetch = fetchMock as unknown as typeof fetch;

    await expect(apiFetch("/x", "token")).rejects.toMatchObject({
      status: 403,
      body: { error: "não autorizado" },
    });
  });

  it("propaga ApiError como instância de ApiError (verificável via instanceof)", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValue(new Response("{}", { status: 500 })) as unknown as typeof fetch;

    try {
      await apiFetch("/x", "token");
      expect.unreachable("deveria ter lançado");
    } catch (erro) {
      expect(erro).toBeInstanceOf(ApiError);
    }
  });

  it("retorna undefined em respostas 204 sem tentar fazer parse de JSON", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 })) as unknown as typeof fetch;

    await expect(apiFetch("/x", "token")).resolves.toBeUndefined();
  });
});

describe("listarContratos", () => {
  it("monta a query string de paginação corretamente", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ total: 0, pagina: 2, tamanho_pagina: 10, dados: [] }), {
          status: 200,
        })
      );
    global.fetch = fetchMock as unknown as typeof fetch;

    await listarContratos("token", 2, 10);

    const [url] = fetchMock.mock.calls[0];
    expect(url).toBe("http://backend.test/api/v1/contratos?pagina=2&tamanho=10");
  });
});

describe("criarContrato", () => {
  it("faz POST com o corpo serializado em JSON", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ ID: "1" }), { status: 201 }));
    global.fetch = fetchMock as unknown as typeof fetch;

    await criarContrato("token", {
      numero_contrato: "1/2026",
      data_assinatura: "2026-01-01",
      contratada_nome: "Empresa X",
      contratada_cnpj: "00.000.000/0001-00",
      fiscal_id: "uuid-fiscal",
      tipo_objeto: "SERVICO",
    });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://backend.test/api/v1/contratos");
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toMatchObject({ numero_contrato: "1/2026" });
  });
});

describe("listarProcessos", () => {
  it("inclui o parâmetro obrigatório 'etapa' na query string", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ total: 0, pagina: 1, tamanho_pagina: 100, dados: [] }), {
          status: 200,
        })
      );
    global.fetch = fetchMock as unknown as typeof fetch;

    await listarProcessos("token", 3);

    const [url] = fetchMock.mock.calls[0];
    expect(url).toBe("http://backend.test/api/v1/processos?etapa=3&pagina=1&tamanho=100");
  });
});

describe("avancarProcesso", () => {
  it("propaga o body do 422 (documentos_pendentes) via ApiError", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ error: "checklist incompleto", documentos_pendentes: ["Nota Fiscal"] }),
        { status: 422 }
      )
    );
    global.fetch = fetchMock as unknown as typeof fetch;

    await expect(avancarProcesso("token", "id-1")).rejects.toMatchObject({
      status: 422,
      body: { documentos_pendentes: ["Nota Fiscal"] },
    });
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://backend.test/api/v1/processos/id-1/avancar");
    expect(init.method).toBe("POST");
  });
});

describe("anexarDocumento", () => {
  it("envia FormData sem forçar Content-Type (o browser define o boundary)", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ ID: "doc-1" }), { status: 201 }));
    global.fetch = fetchMock as unknown as typeof fetch;

    const arquivo = new File(["conteudo"], "nota.pdf", { type: "application/pdf" });
    await anexarDocumento("token", "processo-1", 4, arquivo);

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://backend.test/api/v1/processos/processo-1/documentos");
    expect(init.body).toBeInstanceOf(FormData);
    expect((init.headers as Record<string, string>)["Content-Type"]).toBeUndefined();
    expect((init.body as FormData).get("tipo_documento_id")).toBe("4");
  });
});
