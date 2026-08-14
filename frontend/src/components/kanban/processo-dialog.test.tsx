import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ProcessoDialog } from "./processo-dialog";
import type { ProcessoPagamento, TipoDocumento, DocumentoAnexo } from "@/lib/api/client";

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

const tiposDocumento: TipoDocumento[] = [{ ID: 1, Nome: "Ofício de Solicitação" }];

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
    global.fetch = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify(documentos), { status: 200 })) as unknown as typeof fetch;

    renderDialog();

    expect(await screen.findByText("Ofício de Solicitação")).toBeInTheDocument();
    expect(screen.getByText("oficio.pdf")).toBeInTheDocument();
  });

  it("ao avançar com checklist incompleto (422), mostra os documentos pendentes sem fechar o dialog", async () => {
    const onOpenChange = vi.fn();
    const fetchMock = vi.fn((url: string, init?: RequestInit) => {
      if (init?.method === "POST") {
        return Promise.resolve(
          new Response(
            JSON.stringify({ error: "checklist incompleto", documentos_pendentes: ["Pré-Empenho"] }),
            { status: 422 }
          )
        );
      }
      return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
    });
    global.fetch = fetchMock as unknown as typeof fetch;
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
    const fetchMock = vi.fn((url: string, init?: RequestInit) => {
      if (init?.method === "POST") {
        return Promise.resolve(new Response(JSON.stringify(processo), { status: 200 }));
      }
      return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
    });
    global.fetch = fetchMock as unknown as typeof fetch;
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
    renderDialog({ EtapaAtualID: 1 });
    expect(screen.queryByRole("button", { name: "Marcar como pago" })).not.toBeInTheDocument();
  });

  it("mostra 'Marcar como pago' na etapa 6 e nenhuma ação de avanço quando já concluído", () => {
    renderDialog({ EtapaAtualID: 6, Status: "Concluido" });
    expect(screen.queryByRole("button", { name: "Avançar etapa" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Marcar como pago" })).not.toBeInTheDocument();
  });
});
