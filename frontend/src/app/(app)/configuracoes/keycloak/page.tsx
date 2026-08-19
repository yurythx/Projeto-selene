import Link from "next/link";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { buscarConfiguracaoKeycloak, requireApi } from "@/lib/api/client";
import { KeycloakConfigForm } from "@/components/configuracoes/keycloak-config-form";

// Configurações → Keycloak/SSO: pedido explícito do usuário — "já
// usamos [Keycloak] hoje mas não temos no front, e se eu quiser mudar
// ou implementar um novo, crie uma opção". Ver KeycloakConfigForm pro
// formulário e lib/keycloak-config.ts pra como o frontend passa a usar
// a configuração salva aqui, sem reiniciar o container.
export default async function KeycloakConfigPage() {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  if (!session?.user?.isAdmin) {
    return (
      <div className="text-muted-foreground text-sm">
        Você não tem permissão para acessar esta página.
      </div>
    );
  }

  const configuracao = await requireApi(buscarConfiguracaoKeycloak(accessToken));

  return (
    <div className="space-y-6">
      <div>
        <Link href="/configuracoes" className="text-muted-foreground text-sm hover:underline">
          ← Configurações
        </Link>
        <h1 className="mt-1 text-2xl font-semibold">Keycloak / SSO</h1>
        <p className="text-muted-foreground text-sm">
          Configuração do login institucional (SSO) — editável aqui sem precisar mexer em variáveis de
          ambiente nem reiniciar os containers.
        </p>
      </div>

      <KeycloakConfigForm configuracaoInicial={configuracao} />
    </div>
  );
}
