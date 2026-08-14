import "server-only";
import type { components } from "./schema";

export type Contrato = components["schemas"]["Contrato"];
export type NovoContratoRequest = components["schemas"]["NovoContratoRequest"];
export type AtualizarContratoRequest = components["schemas"]["AtualizarContratoRequest"];
export type Usuario = components["schemas"]["Usuario"];
export type ErroSimples = components["schemas"]["ErroSimples"];

export type ResultadoPaginado<T> = components["schemas"]["ResultadoPaginado"] & {
  dados: T[];
};

/**
 * Erro lançado quando o backend Go responde com status >= 400. Guarda o
 * status e o corpo (quando é JSON parseável) pra quem chamou decidir como
 * tratar — handlers de Route Handler tipicamente repassam o status ao
 * cliente, Server Components tipicamente deixam o error.tsx da rota cuidar.
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly body: unknown
  ) {
    super(`API respondeu ${status}`);
    this.name = "ApiError";
  }
}

const API_URL = process.env.API_URL;

if (!API_URL) {
  throw new Error("Variável de ambiente API_URL não definida (veja frontend/.env.example).");
}

/**
 * Wrapper fino sobre fetch pro backend Go — usado exclusivamente server-side
 * (Server Components, Route Handlers). `accessToken` vem de
 * lib/auth-token.ts (getAccessToken()), nunca do objeto `session` público.
 */
export async function apiFetch<T>(
  path: string,
  accessToken: string,
  init?: RequestInit
): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
      Authorization: `Bearer ${accessToken}`,
    },
    // Dados de contrato mudam com a interação do usuário — sem cache
    // implícito do Next entre requisições distintas.
    cache: "no-store",
  });

  if (!res.ok) {
    const body = await res.json().catch(() => undefined);
    throw new ApiError(res.status, body);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}

export function listarContratos(accessToken: string, pagina = 1, tamanho = 20) {
  const params = new URLSearchParams({ pagina: String(pagina), tamanho: String(tamanho) });
  return apiFetch<ResultadoPaginado<Contrato>>(`/api/v1/contratos?${params}`, accessToken);
}

export function criarContrato(accessToken: string, dados: NovoContratoRequest) {
  return apiFetch<Contrato>("/api/v1/contratos", accessToken, {
    method: "POST",
    body: JSON.stringify(dados),
  });
}
