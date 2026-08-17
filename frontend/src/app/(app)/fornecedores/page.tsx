import { getAccessToken } from "@/lib/auth-token";
import { listarFornecedores, requireApi } from "@/lib/api/client";
import { FornecedoresLista } from "@/components/fornecedores/fornecedores-lista";

export default async function FornecedoresPage() {
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  const fornecedores = await requireApi(listarFornecedores(accessToken));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Dossiê do Fornecedor</h1>
        <p className="text-muted-foreground text-sm">
          {fornecedores.length} fornecedor{fornecedores.length === 1 ? "" : "es"} com contrato
          cadastrado.
        </p>
      </div>

      <FornecedoresLista fornecedores={fornecedores} />
    </div>
  );
}
