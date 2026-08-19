import Link from "next/link";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { buscarConfiguracaoDiarioOficial, requireApi } from "@/lib/api/client";
import { DiarioOficialConfigForm } from "@/components/configuracoes/diario-oficial-config-form";

// Configurações → Diário Oficial: pedido explícito do usuário — "uma
// sessão nas configurações onde vamos pesquisar novos contratos [...]
// direto do diário oficial da cidade, temos uma api pra fazer isso, [quero]
// um lugar pra testar a configuração com a api e outro pra fazer a
// busca". ESTRUTURA GENÉRICA (decisão de escopo confirmada com o
// usuário) — a API real da cidade ainda não está definida; ver o
// comentário no topo de
// backend/internal/service/diario_oficial_service.go pro contrato
// assumido de request/response.
export default async function DiarioOficialConfigPage() {
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

  const configuracao = await requireApi(buscarConfiguracaoDiarioOficial(accessToken));

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <Link href="/configuracoes" className="text-muted-foreground text-sm hover:underline">
            ← Configurações
          </Link>
          <h1 className="mt-1 text-2xl font-semibold">Diário Oficial</h1>
          <p className="text-muted-foreground text-sm">
            Conexão com a API do Diário Oficial da cidade — configure, teste, e depois busque novos
            contratos publicados na tela de busca.
          </p>
        </div>
        <Link
          href="/configuracoes/diario-oficial/buscar"
          className="border-input hover:bg-muted shrink-0 rounded-lg border px-4 py-2 text-sm font-medium whitespace-nowrap"
        >
          Ir para a busca →
        </Link>
      </div>

      <DiarioOficialConfigForm configuracaoInicial={configuracao} />
    </div>
  );
}
