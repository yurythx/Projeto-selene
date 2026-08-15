import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ProcessoDialog } from "./processo-dialog";
import type { ProcessoPagamento, TipoDocumento, DocumentoAnexo, ProcessoComFiscalizacao } from "@/lib/api/client";

const refreshMock = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: refreshMock }),
}));

const toastSuccess = vi.fn();
const toastError = vi.fn();
vi.mock("sonner", () => ({
  toast: {
    success: (...args: unknown[]) => toastSuccess(...args),
    error: (...args: unknown[]) => toastError(...args),
  },
}));

const processo: ProcessoPagamento = {
  ID: "processo-1",
  MesReferencia: "07/2026",
  EtapaAtualID: 1,
  EtapaAtual: { ID: 1, Nome: "Elaborar OF / Pré-Empenho", Posicao: 1 },
  Status: "Ativo",
  Contrato: { NumeroContrato: "10/2026", ContratadaNome: "Fornecedora Ltda" },
};

// Resposta padrão de GET /api/processos/{id} (decorada, ver
// FiscalizacaoService.Decorar no backend) — allowed_actions inclui
// AVANCAR_ETAPA por padrão, pra não depender de timing de fetch nos
// testes que clicam em "Avançar etapa" (ver o comentário em
// processo-dialog.tsx sobre o fallback baseado em EtapaAtualID/Status
// enquanto essa query não resolveu).
const processoDecorado: ProcessoComFiscalizacao = {
  ...processo,
  estado_fiscalizacao: "A_EXECUTAR_CONFERIR",
  acao_ou_espera: "ACAO_FISCAL",
  allowed_actions: ["AVANCAR_ETAPA", "ANEXAR_DOCUMENTO", "REGISTRAR_OCORRENCIA", "REGISTRAR_MOVIMENTACAO_EMPENHO"],
};

const tiposDocumento: TipoDocumento[] = [{ ID: 1, Nome: "Ofício de Solicitação" }];

/**
 * Mock de global.fetch que roteia por URL, não só por método — desde o
 * SGF-Rondonópolis o drawer faz DOIS GETs concorrentes ao abrir
 * (documentos e a leitura decorada do processo), então um mock genérico
 * "qualquer GET vira []" fazia a segunda query também resolver `[]`,
 * derrubando allowed_actions e criando uma corrida entre o clique e o
 * botão sumir do DOM. /documentos vira lista vazia; o resto (a leitura
 * decorada do processo) vira processoDecorado.
 */
function mockFetchPadrao() {
  return vi.fn((url: string) => {
    if (url.includes("/documentos")) {
      return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
    }
    return Promise.resolve(new Response(JSON.stringify(processoDecorado), { status: 200 }));
  }) as unknown as typeof fetch;
}

function renderDialog(overrides: Partial<ProcessoPagamento> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ProcessoDialog
        processo={{ ...processo, ...overrides }}
        tiposDocumento={tiposDocumento}
        isFiscal
        open
        onOpenChange={() => {}}
      />
    </QueryClientProvider>
  );
}

describe("ProcessoDialog", () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    refreshMock.mockClear();
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("busca e lista os documentos anexados ao abrir", async () => {
    const documentos: DocumentoAnexo[] = [
      { ID: "doc-1", TipoDocumento: { ID: 1, Nome: "Ofício de Solicitação" }, NomeArquivo: "oficio.pdf" },
    ];
    global.fetch = vi.fn((url: string) => {
      if (url.includes("/documentos")) {
        return Promise.resolve(new Response(JSON.stringify(documentos), { status: 200 }));
      }
      return Promise.resolve(new Response(JSON.stringify(processoDecorado), { status: 200 }));
    }) as unknown as typeof fetch;

    renderDialog();

    expect(await screen.findByText("Ofício de Solicitação")).toBeInTheDocument();
    expect(screen.getByText("oficio.pdf")).toBeInTheDocument();
  });

  it("ao avançar com checklist incompleto (422), mostra os documentos pendentes sem fechar o dialog", async () => {
    const onOpenChange = vi.fn();
    global.fetch = vi.fn((url: string, init?: RequestInit) => {
      if (init?.method === "POST") {
        return Promise.resolve(
          new Response(
            JSON.stringify({ error: "checklist incompleto", documentos_pendentes: ["Pré-Empenho"] }),
            { status: 422 }
          )
        );
      }
      if (url.includes("/documentos")) {
        return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
      }
      return Promise.resolve(new Response(JSON.stringify(processoDecorado), { status: 200 }));
    }) as unknown as typeof fetch;
    const user = userEvent.setup();

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <ProcessoDialog
          processo={processo}
          tiposDocumento={tiposDocumento}
          isFiscal
          open
          onOpenChange={onOpenChange}
        />
      </QueryClientProvider>
    );

    await user.click(await screen.findByRole("button", { name: "Avançar etapa" }));

    await waitFor(() => {
      expect(screen.getByText("Checklist incompleto")).toBeInTheDocument();
      expect(screen.getByText("Pré-Empenho")).toBeInTheDocument();
    });
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(refreshMock).not.toHaveBeenCalled();
  });

  it("ao avançar com sucesso, fecha o dialog e atualiza a página", async () => {
    const onOpenChange = vi.fn();
    global.fetch = vi.fn((url: string, init?: RequestInit) => {
      if (init?.method === "POST") {
        return Promise.resolve(new Response(JSON.stringify(processo), { status: 200 }));
      }
      if (url.includes("/documentos")) {
        return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
      }
      return Promise.resolve(new Response(JSON.stringify(processoDecorado), { status: 200 }));
    }) as unknown as typeof fetch;
    const user = userEvent.setup();

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <ProcessoDialog
          processo={processo}
          tiposDocumento={tiposDocumento}
          isFiscal
          open
          onOpenChange={onOpenChange}
        />
      </QueryClientProvider>
    );

    await user.click(await screen.findByRole("button", { name: "Avançar etapa" }));

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(refreshMock).toHaveBeenCalled();
    expect(toastSuccess).toHaveBeenCalled();
  });

  it("não mostra o botão 'Marcar como pago' fora da etapa 6", () => {
    global.fetch = mockFetchPadrao();
    renderDialog({ EtapaAtualID: 1 });
    expect(screen.queryByRole("button", { name: "Marcar como pago" })).not.toBeInTheDocument();
  });

  it("mostra 'Marcar como pago' na etapa 6 e nenhuma ação de avanço quando já concluído", () => {
    global.fetch = mockFetchPadrao();
    renderDialog({ EtapaAtualID: 6, Status: "Concluido" });
    expect(screen.queryByRole("button", { name: "Avançar etapa" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Marcar como pago" })).not.toBeInTheDocument();
  });
});
