import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import {
  listarEtapas,
  listarProcessos,
  listarContratos,
  listarRadar,
  requireApi,
} from "@/lib/api/client";
import { KanbanBoard } from "@/components/kanban/kanban-board";

export default async function KanbanPage() {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  const [etapas, contratos, radarItens] = await requireApi(
    Promise.all([
      listarEtapas(accessToken),
      // tamanho máximo permitido pela API — ver limitação documentada no
      // NovoProcessoDialog sobre contratos além da primeira página.
      listarContratos(accessToken, 1, 100),
      listarRadar(accessToken),
    ])
  );

  const colunas = await requireApi(
    Promise.all(etapas.map((etapa) => listarProcessos(accessToken, etapa.ID!)))
  );

  const contratosAtivos = contratos.dados.filter((c) => c.Ativo);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Kanban</h1>
        <p className="text-muted-foreground text-sm">
          Funil de compliance dos processos de pagamento — 6 etapas.
        </p>
      </div>

      <KanbanBoard
        etapas={etapas}
        colunasIniciais={colunas}
        contratosAtivos={contratosAtivos}
        radarItens={radarItens}
        isFiscal={Boolean(session?.user?.isFiscal)}
      />
    </div>
  );
}
