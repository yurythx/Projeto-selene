import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NovoContratoDialog } from "./novo-contrato-dialog";

const refreshMock = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: refreshMock }),
}));

const toastSuccess = vi.fn();
const toastError = vi.fn();
vi.mock("sonner", () => ({
  toast: { success: (...args: unknown[]) => toastSuccess(...args), error: (...args: unknown[]) => toastError(...args) },
}));

function renderDialog() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <NovoContratoDialog fiscalNome="Maria Fiscal" />
    </QueryClientProvider>
  );
}

describe("NovoContratoDialog", () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    refreshMock.mockClear();
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("não envia o formulário quando campos obrigatórios estão vazios", async () => {
    const fetchMock = vi.fn();
    global.fetch = fetchMock as unknown as typeof fetch;
    const user = userEvent.setup();

    renderDialog();
    await user.click(screen.getByRole("button", { name: "Novo contrato" }));
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    await waitFor(() => {
      expect(screen.getAllByText("Obrigatório").length).toBeGreaterThan(0);
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("nunca mostra ou envia um campo de fiscal — fiscal_id é resolvido no servidor", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Novo contrato" }));

    expect(screen.queryByLabelText(/fiscal/i)).not.toBeInTheDocument();
    expect(screen.getByText("Maria Fiscal")).toBeInTheDocument();
  });

  it("envia POST /api/contratos com o corpo preenchido e fecha o dialog ao ter sucesso", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ID: "novo-id" }), { status: 201 })
    );
    global.fetch = fetchMock as unknown as typeof fetch;
    const user = userEvent.setup();

    renderDialog();
    await user.click(screen.getByRole("button", { name: "Novo contrato" }));

    await user.type(screen.getByLabelText("Número do contrato"), "10/2026");
    await user.type(screen.getByLabelText("Data de assinatura"), "2026-02-01");
    await user.type(screen.getByLabelText("Empresa contratada"), "Fornecedora Ltda");
    await user.type(screen.getByLabelText("CNPJ"), "11.111.111/0001-11");

    await user.click(screen.getByRole("button", { name: "Salvar" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/contratos");
    expect(init.method).toBe("POST");
    const body = JSON.parse(init.body as string);
    expect(body).toMatchObject({
      numero_contrato: "10/2026",
      contratada_nome: "Fornecedora Ltda",
      contratada_cnpj: "11.111.111/0001-11",
    });
    expect(body).not.toHaveProperty("fiscal_id");

    await waitFor(() => expect(toastSuccess).toHaveBeenCalled());
    await waitFor(() => expect(refreshMock).toHaveBeenCalled());
  });

  it("mostra um toast de erro e mantém o dialog aberto quando a API falha", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: "contrato duplicado" }), { status: 400 })
    );
    global.fetch = fetchMock as unknown as typeof fetch;
    const user = userEvent.setup();

    renderDialog();
    await user.click(screen.getByRole("button", { name: "Novo contrato" }));
    await user.type(screen.getByLabelText("Número do contrato"), "10/2026");
    await user.type(screen.getByLabelText("Data de assinatura"), "2026-02-01");
    await user.type(screen.getByLabelText("Empresa contratada"), "Fornecedora Ltda");
    await user.type(screen.getByLabelText("CNPJ"), "11.111.111/0001-11");
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    await waitFor(() => expect(toastError).toHaveBeenCalledWith("contrato duplicado"));
    expect(screen.getByRole("button", { name: "Salvar" })).toBeInTheDocument();
  });
});
