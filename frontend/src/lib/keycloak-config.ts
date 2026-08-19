/**
 * Busca a configuração de Keycloak (Client ID/Secret/Issuer) que o
 * provider do NextAuth deve usar AGORA — server-side only, nunca
 * importado por código que roda no browser.
 *
 * Antes, esses três valores só vinham de variáveis de ambiente
 * (AUTH_KEYCLOAK_ID/SECRET/ISSUER), lidas uma vez no boot do processo —
 * pra mudar era preciso editar o ambiente e reiniciar o container.
 * Pedido do usuário: "já usamos [Keycloak] hoje mas não temos no front,
 * e se eu quiser mudar ou implementar um novo, crie uma opção" — agora
 * um admin edita em Configurações → Keycloak/SSO (ver
 * app/(app)/configuracoes/keycloak/page.tsx) e a mudança precisa valer
 * sem reiniciar o frontend.
 *
 * Isso só é possível porque src/auth.ts usa a forma "preguiçosa" do
 * NextAuth (`NextAuth(async (req) => ({...}))`, ver o comentário lá) —
 * ela reavalia a configuração a cada requisição de autenticação, em vez
 * de fixá-la uma única vez no primeiro import do módulo.
 *
 * Consultar o backend a cada requisição de auth seria um desperdício
 * (o foco pedido explicitamente é otimização) — um cache em memória do
 * próprio processo, com TTL curto, equilibra "quase em tempo real" (o
 * admin vê o efeito da mudança em até KEYCLOAK_CONFIG_TTL_MS) com não
 * bater no backend toda hora.
 */

export interface KeycloakRuntimeConfig {
  clientId: string;
  clientSecret: string;
  issuer: string;
}

const KEYCLOAK_CONFIG_TTL_MS = 60_000;

let cache: { valor: KeycloakRuntimeConfig; buscadoEm: number } | null = null;

function fallbackDasVariaveisDeAmbiente(): KeycloakRuntimeConfig {
  return {
    clientId: process.env.AUTH_KEYCLOAK_ID ?? "",
    clientSecret: process.env.AUTH_KEYCLOAK_SECRET ?? "",
    issuer: process.env.AUTH_KEYCLOAK_ISSUER ?? "",
  };
}

export async function obterConfiguracaoKeycloakRuntime(): Promise<KeycloakRuntimeConfig> {
  const agora = Date.now();
  if (cache && agora - cache.buscadoEm < KEYCLOAK_CONFIG_TTL_MS) {
    return cache.valor;
  }

  const apiUrl = process.env.API_URL;
  const segredoInterno = process.env.INTERNAL_API_SECRET;
  if (!apiUrl || !segredoInterno) {
    // Ambiente sem as variáveis novas configuradas ainda (deploy antigo,
    // ou alguém rodando só o frontend fora do compose) — comportamento
    // idêntico ao de antes desta mudança, sem quebrar nada.
    const fallback = fallbackDasVariaveisDeAmbiente();
    cache = { valor: fallback, buscadoEm: agora };
    return fallback;
  }

  try {
    const res = await fetch(`${apiUrl}/internal/keycloak-config`, {
      headers: { "X-Internal-Secret": segredoInterno },
      cache: "no-store",
      signal: AbortSignal.timeout(3000),
    });

    if (res.status === 404) {
      // Nenhum admin salvou uma configuração customizada ainda — cai
      // pras variáveis de ambiente, o comportamento de sempre.
      const fallback = fallbackDasVariaveisDeAmbiente();
      cache = { valor: fallback, buscadoEm: agora };
      return fallback;
    }
    if (!res.ok) {
      throw new Error(`backend respondeu ${res.status}`);
    }

    const body = await res.json();
    const valor: KeycloakRuntimeConfig = {
      clientId: body.client_id ?? "",
      clientSecret: body.client_secret ?? "",
      issuer: body.issuer_url ?? "",
    };
    cache = { valor, buscadoEm: agora };
    return valor;
  } catch (erro) {
    console.error(
      "[Selene] falha ao buscar configuração de Keycloak do backend — usando o último valor conhecido (cache) ou variáveis de ambiente",
      erro
    );
    // Um backend momentaneamente fora do ar não deve derrubar o login:
    // mantém o valor em cache (mesmo "vencido") se existir, só cai pro
    // fallback de variáveis de ambiente se nunca buscamos nada ainda.
    if (cache) {
      return cache.valor;
    }
    const fallback = fallbackDasVariaveisDeAmbiente();
    cache = { valor: fallback, buscadoEm: agora };
    return fallback;
  }
}
