import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { EditarUsuarioDialog } from "./editar-usuario-dialog";
import type { Usuario } from "@/lib/api/client";

const refreshMock = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: refreshMock }),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const usuario: Usuario = {
  ID: "user-1",
  Nome: "Ana Fiscal",
  Email: "ana@exemplo.gov.br",
  IsFiscal: false,
  IsAdmin: false,
  Matricula: "123",
};

function renderWithClient() {
  const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <EditarUsuarioDialog usuario={usuario} />
    </QueryClientProvider>
  );
}

describe("EditarUsuarioDialog", () => {
  const originalFetch = global.fetch;

  beforeEach(() => refreshMock.mockClear());
  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("envia PATCH /api/admin/usuarios/{id} com os flags atualizados", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify(usuario), { status: 200 }));
    global.fetch = fetchMock as unknown as typeof fetch;
    const user = userEvent.setup();

    renderWithClient();
    await user.click(screen.getByRole("button", { name: "Editar" }));
    await user.click(screen.getByRole("switch", { name: /é fiscal/i }));
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/admin/usuarios/user-1");
    expect(init.method).toBe("PATCH");
    expect(JSON.parse(init.body as string)).toEqual({
      is_fiscal: true,
      is_admin: false,
      matricula: "123",
    });
  });
});
