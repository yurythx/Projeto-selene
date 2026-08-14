import "server-only";
import type { components } from "./schema";

export type Contrato = components["schemas"]["Contrato"];
export type NovoContratoRequest = components["schemas"]["NovoContratoRequest"];
export type AtualizarContratoRequest = components["schemas"]["AtualizarContratoRequest"];
export type Usuario = components["schemas"]["Usuario"];
export type ErroSimples = components["schemas"]["ErroSimples"];
export type KanbanEtapa = components["schemas"]["KanbanEtapa"];
export type TipoDocumento = components["schemas"]["TipoDocumento"];
export type ProcessoPagamento = components["schemas"]["ProcessoPagamento"];
export type DocumentoAnexo = components["schemas"]["DocumentoAnexo"];

/** Corpo do 422 de POST /processos/{id}/avancar quando o checklist da etapa não está completo. */
export interface ChecklistIncompletoBody {
  error: string;
  documentos_pendentes: string[];
}

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
  // FormData (upload de documento) define seu próprio Content-Type com o
  // boundary do multipart — nunca forçar application/json nesse caso.
  const temCorpoJSON = Boolean(init?.body) && !(init?.body instanceof FormData);

  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      ...(temCorpoJSON ? { "Content-Type": "application/json" } : {}),
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

export function listarEtapas(accessToken: string) {
  return apiFetch<KanbanEtapa[]>("/api/v1/kanban/etapas", accessToken);
}

export function listarTiposDocumento(accessToken: string) {
  return apiFetch<TipoDocumento[]>("/api/v1/kanban/tipos-documento", accessToken);
}

export function listarProcessos(accessToken: string, etapaId: number, pagina = 1, tamanho = 100) {
  const params = new URLSearchParams({
    etapa: String(etapaId),
    pagina: String(pagina),
    tamanho: String(tamanho),
  });
  return apiFetch<ResultadoPaginado<ProcessoPagamento>>(`/api/v1/processos?${params}`, accessToken);
}

export function buscarProcesso(accessToken: string, id: string) {
  return apiFetch<ProcessoPagamento>(`/api/v1/processos/${id}`, accessToken);
}

export function criarProcesso(accessToken: string, contratoId: string, mesReferencia: string) {
  return apiFetch<ProcessoPagamento>("/api/v1/processos", accessToken, {
    method: "POST",
    body: JSON.stringify({ contrato_id: contratoId, mes_referencia: mesReferencia }),
  });
}

/**
 * Avança o processo pra próxima etapa. Se o checklist da etapa atual não
 * estiver completo, o backend responde 422 — vira uma ApiError cujo
 * `body` é um ChecklistIncompletoBody (`documentos_pendentes`).
 */
export function avancarProcesso(accessToken: string, id: string) {
  return apiFetch<ProcessoPagamento>(`/api/v1/processos/${id}/avancar`, accessToken, {
    method: "POST",
  });
}

export function concluirProcesso(accessToken: string, id: string) {
  return apiFetch<ProcessoPagamento>(`/api/v1/processos/${id}/concluir`, accessToken, {
    method: "POST",
  });
}

export function listarDocumentos(accessToken: string, processoId: string) {
  return apiFetch<DocumentoAnexo[]>(`/api/v1/processos/${processoId}/documentos`, accessToken);
}

export function anexarDocumento(
  accessToken: string,
  processoId: string,
  tipoDocumentoId: number,
  arquivo: File
) {
  const formData = new FormData();
  formData.append("tipo_documento_id", String(tipoDocumentoId));
  formData.append("arquivo", arquivo);
  return apiFetch<DocumentoAnexo>(`/api/v1/processos/${processoId}/documentos`, accessToken, {
    method: "POST",
    body: formData,
  });
}

/**
 * Baixa o PDF do Relatório de Pagamento. Não passa por apiFetch — a
 * resposta é binária (application/pdf), não JSON.
 */
export async function baixarRelatorio(accessToken: string, processoId: string): Promise<Response> {
  const res = await fetch(`${API_URL}/api/v1/processos/${processoId}/relatorio`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    cache: "no-store",
  });

  if (!res.ok) {
    const body = await res.json().catch(() => undefined);
    throw new ApiError(res.status, body);
  }

  return res;
}
