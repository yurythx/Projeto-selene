import Link from "next/link";
import { getAccessToken } from "@/lib/auth-token";
import { listarRadar, type ItemRadar } from "@/lib/api/client";
import { RADAR_TIPO_LABEL } from "@/lib/radar";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { RadarNivelBadge } from "@/components/radar/radar-badge";

// CRITICO antes de ATENCAO; dentro do mesmo nível, o prazo mais urgente
// (menor dias_restantes, incluindo já vencidos/negativos) primeiro.
function ordenar(itens: ItemRadar[]) {
  return [...itens].sort((a, b) => {
    if (a.nivel !== b.nivel) return a.nivel === "CRITICO" ? -1 : 1;
    return (a.dias_restantes ?? 0) - (b.dias_restantes ?? 0);
  });
}

export default async function RadarPage() {
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  const itens = ordenar(await listarRadar(accessToken));
  const criticos = itens.filter((i) => i.nivel === "CRITICO").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Radar de Alertas</h1>
        <p className="text-muted-foreground text-sm">
          {itens.length} alerta{itens.length === 1 ? "" : "s"} em aberto
          {criticos > 0 ? ` — ${criticos} crítico${criticos === 1 ? "" : "s"}` : ""}.
          Vigência de contrato, certidões vencidas/vencendo e processos parados há muito tempo.
        </p>
      </div>

      <div className="overflow-x-auto rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nível</TableHead>
              <TableHead>Tipo</TableHead>
              <TableHead>Contrato</TableHead>
              <TableHead>Mensagem</TableHead>
              <TableHead>Dias</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {itens.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-muted-foreground text-center">
                  Nenhum alerta em aberto no momento.
                </TableCell>
              </TableRow>
            )}
            {itens.map((item, i) => (
              <TableRow key={i} className="hover:bg-accent">
                <TableCell>{item.nivel && <RadarNivelBadge nivel={item.nivel} />}</TableCell>
                <TableCell className="text-sm">
                  {item.tipo ? RADAR_TIPO_LABEL[item.tipo] : "—"}
                </TableCell>
                <TableCell className="font-medium">
                  <Link href={`/contratos/${item.contrato_id}`} className="hover:underline">
                    {item.numero_contrato}
                  </Link>
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">{item.mensagem}</TableCell>
                <TableCell className="text-sm">{item.dias_restantes}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
