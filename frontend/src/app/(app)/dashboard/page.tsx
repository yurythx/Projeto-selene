import Link from "next/link";
import {
  FileTextIcon,
  KanbanSquareIcon,
  TriangleAlertIcon,
  ShieldAlertIcon,
} from "lucide-react";
import { getAccessToken } from "@/lib/auth-token";
import {
  listarRadar,
  listarContratos,
  listarEtapas,
  listarProcessos,
  requireApi,
  type ItemRadar,
} from "@/lib/api/client";
import { RADAR_TIPO_LABEL } from "@/lib/radar";
import { corDaEtapa } from "@/lib/kanban-colors";
import { cn } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { RadarNivelBadge } from "@/components/radar/radar-badge";

// CRITICO antes de ATENCAO; dentro do mesmo nível, o prazo mais urgente
// primeiro — mesmo critério de ordenação de /radar (ver radar/page.tsx).
function ordenarPorUrgencia(itens: ItemRadar[]) {
  return [...itens].sort((a, b) => {
    if (a.nivel !== b.nivel) return a.nivel === "CRITICO" ? -1 : 1;
    return (a.dias_restantes ?? 0) - (b.dias_restantes ?? 0);
  });
}

function KpiCard({
  titulo,
  valor,
  descricao,
  Icon,
  tone = "default",
}: {
  titulo: string;
  valor: number;
  descricao: string;
  Icon: React.ComponentType<{ className?: string }>;
  tone?: "default" | "destructive";
}) {
  return (
    <Card className="shadow-sm">
      <CardContent className="flex items-start justify-between gap-3">
        <div>
          <p className="text-muted-foreground text-sm">{titulo}</p>
          <p className={cn("text-3xl font-semibold", tone === "destructive" && "text-destructive")}>
            {valor}
          </p>
          <p className="text-muted-foreground mt-1 text-xs">{descricao}</p>
        </div>
        <Icon
          className={cn(
            "size-8 shrink-0 opacity-70",
            tone === "destructive" ? "text-destructive" : "text-muted-foreground"
          )}
        />
      </CardContent>
    </Card>
  );
}

/**
 * Painel inicial ("/" redireciona pra cá, ver sidebar.tsx e
 * app/(app)/page.tsx) — o que todo CRM tem antes de mandar o usuário pra
 * uma lista de trabalho: um panorama rápido sem precisar abrir Kanban,
 * Contratos e Radar separadamente pra montar esse quadro na cabeça.
 * Inteiramente derivado de endpoints que já existiam (Radar, Contratos
 * com o novo filtro por situação, Kanban por etapa) — nenhum endpoint
 * novo no backend só pra isto.
 */
export default async function DashboardPage() {
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  const [radarItens, contratosAtivos, contratosEncerrados, etapas] = await requireApi(
    Promise.all([
      listarRadar(accessToken),
      // tamanho=1: só o total importa aqui, não os dados da página — o
      // novo filtro `situacao` (ver repository.FiltroContrato) faz a
      // contagem no banco em vez de baixar tudo pra contar no cliente.
      listarContratos(accessToken, 1, 1, { situacao: "ativo" }),
      listarContratos(accessToken, 1, 1, { situacao: "encerrado" }),
      listarEtapas(accessToken),
    ])
  );

  const colunas = await requireApi(
    Promise.all(etapas.map((etapa) => listarProcessos(accessToken, etapa.ID!)))
  );

  const totalContratosAtivos = contratosAtivos.total ?? 0;
  const totalContratosEncerrados = contratosEncerrados.total ?? 0;
  const processosEmAndamento = colunas.reduce((soma, coluna) => soma + (coluna.dados?.length ?? 0), 0);
  const maiorColuna = Math.max(1, ...colunas.map((coluna) => coluna.dados?.length ?? 0));

  const certidoes = radarItens.filter((item) => item.tipo === "certidao");
  const vigencias = radarItens.filter((item) => item.tipo === "vigencia_contrato");
  const parados = radarItens.filter((item) => item.tipo === "processo_parado");
  const criticos = radarItens.filter((item) => item.nivel === "CRITICO");
  const itensMaisCriticos = ordenarPorUrgencia(radarItens).slice(0, 6);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Dashboard</h1>
        <p className="text-muted-foreground text-sm">
          Panorama geral dos contratos e processos sob fiscalização.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          titulo="Contratos ativos"
          valor={totalContratosAtivos}
          descricao={`${totalContratosEncerrados} encerrado${totalContratosEncerrados === 1 ? "" : "s"}`}
          Icon={FileTextIcon}
        />
        <KpiCard
          titulo="Processos em andamento"
          valor={processosEmAndamento}
          descricao="Nas 6 etapas do funil de compliance"
          Icon={KanbanSquareIcon}
        />
        <KpiCard
          titulo="Alertas críticos"
          valor={criticos.length}
          descricao={`${radarItens.length} alerta${radarItens.length === 1 ? "" : "s"} no total`}
          Icon={TriangleAlertIcon}
          tone={criticos.length > 0 ? "destructive" : "default"}
        />
        <KpiCard
          titulo="Certidões vencidas/vencendo"
          valor={certidoes.length}
          descricao={`${parados.length} processo${parados.length === 1 ? "" : "s"} parado${parados.length === 1 ? "" : "s"} há muito tempo`}
          Icon={ShieldAlertIcon}
          tone={certidoes.length > 0 ? "destructive" : "default"}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="shadow-sm lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">Processos por etapa</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {etapas.map((etapa, i) => {
              const cor = corDaEtapa(i);
              const qtd = colunas[i]?.dados?.length ?? 0;
              return (
                <Link key={etapa.ID} href="/kanban" className="block">
                  <div className="flex items-center justify-between text-sm">
                    <span>{etapa.Nome}</span>
                    <span className="text-muted-foreground">{qtd}</span>
                  </div>
                  <div className="bg-muted mt-1 h-2 overflow-hidden rounded-full">
                    <div
                      className={cn("h-full rounded-full transition-all", cor.headerBg)}
                      style={{ width: `${(qtd / maiorColuna) * 100}%` }}
                    />
                  </div>
                </Link>
              );
            })}
          </CardContent>
        </Card>

        <Card className="shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">Vigência de contratos</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <p>
              <span className="text-muted-foreground">Ativos: </span>
              {totalContratosAtivos}
            </p>
            <p>
              <span className="text-muted-foreground">Encerrados: </span>
              {totalContratosEncerrados}
            </p>
            <p>
              <span className="text-muted-foreground">Vencendo/vencidos: </span>
              {vigencias.length}
            </p>
            <Link href="/contratos" className="text-sm underline underline-offset-2">
              Ver todos os contratos
            </Link>
          </CardContent>
        </Card>
      </div>

      <Card className="shadow-sm">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Itens mais críticos do Radar</CardTitle>
          <Link href="/radar" className="text-sm underline underline-offset-2">
            Ver todos
          </Link>
        </CardHeader>
        <CardContent className="space-y-2">
          {itensMaisCriticos.length === 0 && (
            <p className="text-muted-foreground text-sm">Nenhum alerta em aberto no momento.</p>
          )}
          {itensMaisCriticos.map((item, i) => (
            <div
              key={i}
              className="flex flex-wrap items-center justify-between gap-2 border-b pb-2 text-sm last:border-b-0 last:pb-0"
            >
              <div className="flex flex-wrap items-center gap-2">
                {item.nivel && <RadarNivelBadge nivel={item.nivel} />}
                <Link href={`/contratos/${item.contrato_id}`} className="font-medium hover:underline">
                  {item.numero_contrato}
                </Link>
                <span className="text-muted-foreground text-xs">
                  {item.tipo ? RADAR_TIPO_LABEL[item.tipo] : ""}
                </span>
                <span className="text-muted-foreground">{item.mensagem}</span>
              </div>
              <span className="text-muted-foreground text-xs whitespace-nowrap">
                {item.dias_restantes}d
              </span>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
