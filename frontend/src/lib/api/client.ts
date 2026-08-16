import "server-only";
import { redirect } from "next/navigation";
import type { components } from "./schema";

export type Contrato = components["schemas"]["Contrato"];
export type NovoContratoRequest = components["schemas"]["NovoContratoRequest"];
export type AtualizarContratoRequest = components["schemas"]["AtualizarContratoRequest"];
export type Usuario = components["schemas"]["Usuario"];
export type ErroSimples = components["schemas"]["ErroSimples"];
export type KanbanEtapa = components["schemas"]["KanbanEtapa"];
export type TipoDocumento = components["schemas"]["TipoDocumento"];
export type ProcessoPagamento = components["schemas"]["ProcessoPagamento"];
export type ProcessoComFiscalizacao = components["schemas"]["ProcessoComFiscalizacao"];
export type AllowedAction = "AVANCAR_ETAPA" | "CONCLUIR_PAGAMENTO" | "ANEXAR_DOCUMENTO" | "REGISTRAR_OCORRENCIA" | "REGISTRAR_MOVIMENTACAO_EMPENHO";
export type DocumentoAnexo = components["schemas"]["DocumentoAnexo"];
export type ItemRadar = components["schemas"]["ItemRadar"];
export type MinutaAditivoRequest = components["schemas"]["MinutaAditivoRequest"];
export type VerificarDocumentoResponse = components["schemas"]["VerificarDocumentoResponse"];
export type RegistroVistoria = components["schemas"]["RegistroVistoria"];
export type FotoVistoria = components["schemas"]["FotoVistoria"];
export type FornecedorResumo = components["schemas"]["FornecedorResumo"];
export type FornecedorDossie = components["schemas"]["FornecedorDossie"];

// --- SGF-Rondonópolis (adequação às IN SCL 01/2019 e 04/2021) ---
export type ServidorOpcao = components["schemas"]["ServidorOpcao"];
export type PortariaDesignacao = components["schemas"]["PortariaDesignacao"];
export type DesignarRequest = components["schemas"]["DesignarRequest"];
export type Empenho = components["schemas"]["Empenho"];
export type EmpenhoComSaldo = components["schemas"]["EmpenhoComSaldo"];
export type CriarEmpenhoRequest = components["schemas"]["CriarEmpenhoRequest"];
export type MovimentacaoEmpenho = components["schemas"]["MovimentacaoEmpenho"];
export type RegistrarMovimentacaoRequest = components["schemas"]["RegistrarMovimentacaoRequest"];
export type Ocorrencia = components["schemas"]["Ocorrencia"];
export type RegistrarOcorrenciaRequest = components["schemas"]["RegistrarOcorrenciaRequest"];

/** Corpo de POST /processos/{id}/vistorias. */
export interface NovaVistoriaRequest {
  latitude?: number | null;
  longitude?: number | null;
  observacoes?: string;
}

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

/**
 * Envolve uma chamada de apiFetch feita por um Server Component de
 * PÁGINA — nunca por um Route Handler (ver o aviso abaixo) — pra tratar
 * 401 ("token inválido ou expirado") como o que ele é: sessão inválida,
 * não uma falha passageira do backend. Sem isso, o ApiError borbulhava
 * pro error.tsx da rota, que mostra "backend pode estar indisponível,
 * tente de novo" — mensagem enganosa aqui, já que "tentar de novo"
 * reenviaria o mesmo token quebrado e cairia no mesmo 401 de novo.
 *
 * Cenário real que expôs a falta disto: a chave que assina os tokens de
 * login local é gerada em memória a cada boot do backend (ver a
 * LIMITAÇÃO CONHECIDA em internal/localauth/localauth.go) — todo
 * restart/redeploy invalida silenciosamente sessões locais já abertas, e
 * o usuário só via a tela de erro genérica em vez de ser levado pro
 * login de novo.
 *
 * Redireciona pra /api/auth/sessao-invalida (não direto pra /login) —
 * essa rota limpa o cookie de sessão antes de mandar pro /login. Um
 * redirect("/login") direto NÃO basta: o cookie de sessão continua
 * criptograficamente válido pro Auth.js (só o accessToken embutido é que
 * o backend rejeitou), então proxy.ts (GUEST_ONLY_ROUTES) via sessão
 * presente manda de volta pra "/" → /kanban → 401 de novo → aqui de
 * novo — loop infinito, reproduzido antes desta versão do fix (ver
 * e2e/session.spec.ts e o comentário na própria rota).
 *
 * NUNCA usar em Route Handlers (app/api/**\/route.ts): redirect() lança
 * um sinal específico de renderização de página que eles não devem
 * capturar — precisam continuar devolvendo o JSON de erro original (ver
 * o catch em cada route.ts) pro fetch() do client interpretar.
 */
export async function requireApi<T>(promise: Promise<T>): Promise<T> {
  try {
    return await promise;
  } catch (erro) {
    if (erro instanceof ApiError && erro.status === 401) {
      redirect("/api/auth/sessao-invalida");
    }
    throw erro;
  }
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

export function buscarContrato(accessToken: string, id: string) {
  return apiFetch<Contrato>(`/api/v1/contratos/${id}`, accessToken);
}

export function atualizarContrato(accessToken: string, id: string, dados: AtualizarContratoRequest) {
  return apiFetch<Contrato>(`/api/v1/contratos/${id}`, accessToken, {
    method: "PATCH",
    body: JSON.stringify(dados),
  });
}

export function encerrarContrato(accessToken: string, id: string) {
  return apiFetch<Contrato>(`/api/v1/contratos/${id}/encerrar`, accessToken, {
    method: "POST",
  });
}

export interface AtualizarUsuarioRequest {
  is_fiscal?: boolean;
  is_admin?: boolean;
  matricula?: string;
}

export function listarUsuarios(accessToken: string) {
  return apiFetch<Usuario[]>("/api/v1/admin/users", accessToken);
}

export function atualizarUsuario(accessToken: string, id: string, dados: AtualizarUsuarioRequest) {
  return apiFetch<Usuario>(`/api/v1/admin/users/${id}`, accessToken, {
    method: "PATCH",
    body: JSON.stringify(dados),
  });
}

export interface CriarUsuarioLocalRequest {
  nome: string;
  email: string;
  senha_temporaria: string;
  is_fiscal?: boolean;
  is_admin?: boolean;
}

/** Login tradicional (usuário/senha) — só um admin cria contas locais. */
export function criarUsuarioLocal(accessToken: string, dados: CriarUsuarioLocalRequest) {
  return apiFetch<Usuario>("/api/v1/admin/users/local", accessToken, {
    method: "POST",
    body: JSON.stringify(dados),
  });
}

/** Troca a senha da PRÓPRIA conta (autenticado) — POST /auth/trocar-senha. */
export function trocarSenha(accessToken: string, senhaAtual: string, senhaNova: string) {
  return apiFetch<void>("/api/v1/auth/trocar-senha", accessToken, {
    method: "POST",
    body: JSON.stringify({ senha_atual: senhaAtual, senha_nova: senhaNova }),
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
  // ProcessoComFiscalizacao, não ProcessoPagamento: GET /processos/{id} é o
  // único endpoint decorado com a leitura de Camada 2 do SGF-Rondonópolis
  // (estado_fiscalizacao/acao_ou_espera/allowed_actions) — ver
  // backend/internal/service/fiscalizacao_service.go. Os campos de
  // ProcessoPagamento continuam todos presentes (allOf no OpenAPI).
  return apiFetch<ProcessoComFiscalizacao>(`/api/v1/processos/${id}`, accessToken);
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
  arquivo: File,
  // dataValidade ("AAAA-MM-DD") é opcional — só relevante pra tipos com
  // ExigeValidade=true (certidões), mas o backend aceita pra qualquer
  // tipo sem reclamar; quem decide se pede o campo é a UI.
  dataValidade?: string
) {
  const formData = new FormData();
  formData.append("tipo_documento_id", String(tipoDocumentoId));
  formData.append("arquivo", arquivo);
  if (dataValidade) {
    formData.append("data_validade", dataValidade);
  }
  return apiFetch<DocumentoAnexo>(`/api/v1/processos/${processoId}/documentos`, accessToken, {
    method: "POST",
    body: formData,
  });
}

export function listarRadar(accessToken: string) {
  return apiFetch<ItemRadar[]>("/api/v1/radar", accessToken);
}

export function listarVistorias(accessToken: string, processoId: string) {
  return apiFetch<RegistroVistoria[]>(`/api/v1/processos/${processoId}/vistorias`, accessToken);
}

export function registrarVistoria(accessToken: string, processoId: string, dados: NovaVistoriaRequest) {
  return apiFetch<RegistroVistoria>(`/api/v1/processos/${processoId}/vistorias`, accessToken, {
    method: "POST",
    body: JSON.stringify(dados),
  });
}

export function anexarFotoVistoria(accessToken: string, vistoriaId: string, foto: File) {
  const formData = new FormData();
  formData.append("foto", foto);
  return apiFetch<FotoVistoria>(`/api/v1/vistorias/${vistoriaId}/fotos`, accessToken, {
    method: "POST",
    body: formData,
  });
}

export function listarFornecedores(accessToken: string) {
  return apiFetch<FornecedorResumo[]>("/api/v1/fornecedores", accessToken);
}

export function buscarFornecedor(accessToken: string, cnpj: string) {
  return apiFetch<FornecedorDossie>(`/api/v1/fornecedores/${cnpj}`, accessToken);
}

// --- SGF-Rondonópolis: designação, empenho, ocorrência ---

/** Projeção mínima de todos os usuários (ID/Nome/Email) — popula o seletor de servidor de "Nova designação". Não é admin-only, ver o handler. */
export function listarServidores(accessToken: string) {
  return apiFetch<ServidorOpcao[]>("/api/v1/servidores", accessToken);
}

export function listarDesignacoes(accessToken: string, contratoId: string) {
  return apiFetch<PortariaDesignacao[]>(`/api/v1/contratos/${contratoId}/designacoes`, accessToken);
}

export function designar(accessToken: string, contratoId: string, dados: DesignarRequest) {
  return apiFetch<PortariaDesignacao>(`/api/v1/contratos/${contratoId}/designacoes`, accessToken, {
    method: "POST",
    body: JSON.stringify(dados),
  });
}

export function listarEmpenhos(accessToken: string, contratoId: string) {
  return apiFetch<Empenho[]>(`/api/v1/contratos/${contratoId}/empenhos`, accessToken);
}

export function criarEmpenho(accessToken: string, contratoId: string, dados: CriarEmpenhoRequest) {
  return apiFetch<Empenho>(`/api/v1/contratos/${contratoId}/empenhos`, accessToken, {
    method: "POST",
    body: JSON.stringify(dados),
  });
}

export function buscarEmpenho(accessToken: string, empenhoId: string) {
  return apiFetch<EmpenhoComSaldo>(`/api/v1/empenhos/${empenhoId}`, accessToken);
}

export function registrarMovimentacaoEmpenho(accessToken: string, empenhoId: string, dados: RegistrarMovimentacaoRequest) {
  return apiFetch<MovimentacaoEmpenho>(`/api/v1/empenhos/${empenhoId}/movimentacoes`, accessToken, {
    method: "POST",
    body: JSON.stringify(dados),
  });
}

export function listarOcorrencias(accessToken: string, processoId: string) {
  return apiFetch<Ocorrencia[]>(`/api/v1/processos/${processoId}/ocorrencias`, accessToken);
}

export function registrarOcorrencia(accessToken: string, processoId: string, dados: RegistrarOcorrenciaRequest) {
  return apiFetch<Ocorrencia>(`/api/v1/processos/${processoId}/ocorrencias`, accessToken, {
    method: "POST",
    body: JSON.stringify(dados),
  });
}

export function notificarOcorrencia(accessToken: string, ocorrenciaId: string) {
  return apiFetch<Ocorrencia>(`/api/v1/ocorrencias/${ocorrenciaId}/notificar`, accessToken, { method: "POST" });
}

export function tratarOcorrencia(accessToken: string, ocorrenciaId: string) {
  return apiFetch<Ocorrencia>(`/api/v1/ocorrencias/${ocorrenciaId}/tratar`, accessToken, { method: "POST" });
}

export function regularizarOcorrencia(accessToken: string, ocorrenciaId: string) {
  return apiFetch<Ocorrencia>(`/api/v1/ocorrencias/${ocorrenciaId}/regularizar`, accessToken, { method: "POST" });
}

/**
 * Baixa o Relatório de Campo (PDF) de uma vistoria. Não passa por
 * apiFetch — resposta binária, mesmo padrão de baixarRelatorio.
 */
export async function baixarRelatorioCampo(accessToken: string, vistoriaId: string): Promise<Response> {
  const res = await fetch(`${API_URL}/api/v1/vistorias/${vistoriaId}/relatorio`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    cache: "no-store",
  });

  if (!res.ok) {
    const body = await res.json().catch(() => undefined);
    throw new ApiError(res.status, body);
  }

  return res;
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

/**
 * POST binário genérico (Módulo 2 do roadmap: os 3 geradores de PDF) —
 * mesma lógica de baixarRelatorio, mas com corpo JSON opcional.
 */
async function postPDF(path: string, accessToken: string, body?: unknown): Promise<Response> {
  const res = await fetch(`${API_URL}${path}`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${accessToken}`,
      ...(body ? { "Content-Type": "application/json" } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    cache: "no-store",
  });

  if (!res.ok) {
    const responseBody = await res.json().catch(() => undefined);
    throw new ApiError(res.status, responseBody);
  }

  return res;
}

/** Gera a Notificação de Descumprimento (PDF) do contrato informado. */
export function gerarNotificacao(accessToken: string, contratoId: string, motivo: string) {
  return postPDF(`/api/v1/contratos/${contratoId}/notificacao`, accessToken, { motivo });
}

/** Gera o Atesto (PDF, com QR code de verificação) do processo informado. */
export function gerarAtesto(accessToken: string, processoId: string) {
  return postPDF(`/api/v1/processos/${processoId}/atesto`, accessToken);
}

/** Gera a Minuta de Aditivo (PDF) do contrato informado. */
export function gerarMinutaAditivo(accessToken: string, contratoId: string, dados: MinutaAditivoRequest) {
  return postPDF(`/api/v1/contratos/${contratoId}/minuta-aditivo`, accessToken, dados);
}

/**
 * Verifica a autenticidade de um documento emitido pelo código do QR code.
 * Diferente das demais funções deste arquivo, NÃO recebe accessToken: a
 * rota do backend é pública de propósito (ver GeradorDocumentosHandler.
 * Verificar) — quem escaneia o QR de um papel impresso não tem login no
 * Selene.
 */
export async function verificarDocumento(codigo: string): Promise<VerificarDocumentoResponse> {
  const res = await fetch(`${API_URL}/api/v1/verificar/${encodeURIComponent(codigo)}`, {
    cache: "no-store",
  });

  if (!res.ok) {
    // A rota do backend não deveria responder erro (sempre 200, ver
    // comentário no handler) — se acontecer mesmo assim (backend fora do
    // ar etc.), trata como "inválido" em vez de propagar um 5xx pra uma
    // página pública.
    return { valido: false };
  }

  return (await res.json()) as VerificarDocumentoResponse;
}
