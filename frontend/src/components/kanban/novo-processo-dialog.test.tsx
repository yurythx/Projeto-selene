import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NovoProcessoDialog } from "./novo-processo-dialog";
import type { Contrato } from "@/lib/api/client";

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

const contratosAtivos: Contrato[] = [
  { ID: "contrato-1", NumeroContrato: "10/2026", ContratadaNome: "Fornecedora Ltda", Ativo: true },
];

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe("NovoProcessoDialog", () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    refreshMock.mockClear();
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("valida o formato do mês de referência antes de enviar", async () => {
    const fetchMock = vi.fn();
    global.fetch = fetchMock as unknown as typeof fetch;
    const user = userEvent.setup();

    renderWithClient(
      <NovoProcessoDialog open onOpenChange={() => {}} contratosAtivos={contratosAtivos} />
    );

    await user.type(screen.getByLabelText("Mês de referência"), "13/26");
    await user.click(screen.getByRole("button", { name: "Abrir processo" }));

    await waitFor(() => {
      expect(screen.getByText(/Formato MM\/AAAA/)).toBeInTheDocument();
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("envia POST /api/processos e fecha o dialog com sucesso", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ ID: "novo" }), { status: 201 }));
    global.fetch = fetchMock as unknown as typeof fetch;
    const onOpenChange = vi.fn();
    // Base UI mantém `pointer-events: none` no item da lista durante a
    // animação de abertura do Select — em jsdom (sem timers reais de
    // CSS/rAF) isso pode fazer o userEvent.click "real" (que respeita
    // pointer-events, de propósito, pra pegar bugs de UI de verdade)
    // falhar de forma intermitente. Desabilitar essa checagem aqui é
    // seguro: não estamos testando a animação, só o fluxo de submissão.
    const user = userEvent.setup({ pointerEventsCheck: 0 });

    renderWithClient(
      <NovoProcessoDialog open onOpenChange={onOpenChange} contratosAtivos={contratosAtivos} />
    );

    await user.click(screen.getByRole("combobox", { name: /contrato/i }));
    await user.click(await screen.findByText("10/2026 — Fornecedora Ltda"));
    await user.type(screen.getByLabelText("Mês de referência"), "07/2026");
    await user.click(screen.getByRole("button", { name: "Abrir processo" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/processos");
    const body = JSON.parse(init.body as string);
    expect(body).toEqual({ contrato_id: "contrato-1", mes_referencia: "07/2026" });

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(toastSuccess).toHaveBeenCalled();
    expect(refreshMock).toHaveBeenCalled();
  });
});
