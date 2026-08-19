import Link from "next/link";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { buscarConfiguracaoDiarioOficial, requireApi } from "@/lib/api/client";
import { BuscarContratosDiarioOficial } from "@/components/configuracoes/buscar-contratos-diario-oficial";

// Configurações → Diário Oficial → Buscar: a segunda das duas telas
// pedidas explicitamente pelo usuário ("um lugar pra testar a
// configuração com a api e outro pra fazer a busca desses novos
// contratos fazendo a busca por nome, cpf e data").
export default async function BuscarContratosDiarioOficialPage() {
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
      <div>
        <Link
          href="/configuracoes/diario-oficial"
          className="text-muted-foreground text-sm hover:underline"
        >
          ← Diário Oficial
        </Link>
        <h1 className="mt-1 text-2xl font-semibold">Buscar novos contratos</h1>
        <p className="text-muted-foreground text-sm">
          Busca direto na API do Diário Oficial da cidade, por nome, CPF e data de publicação.
        </p>
      </div>

      {!configuracao.BaseURL ? (
        <div className="bg-muted/40 max-w-xl rounded-lg border p-4 text-sm">
          Nenhuma URL configurada ainda.{" "}
          <Link href="/configuracoes/diario-oficial" className="underline">
            Configure a integração primeiro
          </Link>
          .
        </div>
      ) : (
        <BuscarContratosDiarioOficial />
      )}
    </div>
  );
}
