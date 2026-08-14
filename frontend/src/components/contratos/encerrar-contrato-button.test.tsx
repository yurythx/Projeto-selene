import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { EncerrarContratoButton } from "./encerrar-contrato-button";

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

function renderWithClient() {
  const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <EncerrarContratoButton contratoId="contrato-1" />
    </QueryClientProvider>
  );
}

describe("EncerrarContratoButton", () => {
  const originalFetch = global.fetch;
  const originalConfirm = window.confirm;

  beforeEach(() => {
    refreshMock.mockClear();
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  afterEach(() => {
    global.fetch = originalFetch;
    window.confirm = originalConfirm;
  });

  it("não chama a API se o usuário cancelar a confirmação", async () => {
    window.confirm = vi.fn().mockReturnValue(false);
    const fetchMock = vi.fn();
    global.fetch = fetchMock as unknown as typeof fetch;
    const user = userEvent.setup();

    renderWithClient();
    await user.click(screen.getByRole("button", { name: "Encerrar contrato" }));

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("chama POST /api/contratos/{id}/encerrar após confirmar", async () => {
    window.confirm = vi.fn().mockReturnValue(true);
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ ID: "contrato-1", Ativo: false }), { status: 200 }));
    global.fetch = fetchMock as unknown as typeof fetch;
    const user = userEvent.setup();

    renderWithClient();
    await user.click(screen.getByRole("button", { name: "Encerrar contrato" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/contratos/contrato-1/encerrar");
    expect(init.method).toBe("POST");
    await waitFor(() => expect(refreshMock).toHaveBeenCalled());
    expect(toastSuccess).toHaveBeenCalled();
  });
});
