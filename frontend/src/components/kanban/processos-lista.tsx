"use client";

import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { corDaEtapa } from "@/lib/kanban-colors";
import { itensDoProcesso, nivelMaisCritico } from "@/lib/radar";
import { RadarNivelBadge } from "@/components/radar/radar-badge";
import type { KanbanEtapa, ProcessoPagamento, ItemRadar } from "@/lib/api/client";

const TIPO_OBJETO_LABEL: Record<string, string> = {
  CONSUMO: "Consumo",
  PERMANENTE: "Permanente",
  SERVICO: "Serviço",
};

function iniciaisDoFiscal(nome?: string): string {
  if (!nome) return "?";
  return nome
    .trim()
    .split(/\s+/)
    .map((parte) => parte[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

/**
 * A mesma coleção de processos do quadro Kanban, achatada numa tabela —
 * "a versão em lista que todo CRM tem" (Monday/Trello também oferecem os
 * dois modos sobre o mesmo dado). Reaproveita ProcessoDialog através do
 * callback onOpen (o dialog em si continua vivendo em KanbanBoard, que já
 * o usa pro modo Kanban) — clicar numa linha abre exatamente o mesmo
 * drawer que clicar num card.
 */
export function ProcessosLista({
  processos,
  etapas,
  radarItens,
  onOpen,
}: {
  processos: ProcessoPagamento[];
  etapas: KanbanEtapa[];
  radarItens: ItemRadar[];
  onOpen: (processo: ProcessoPagamento) => void;
}) {
  const posicaoPorEtapa = new Map(etapas.map((etapa, i) => [etapa.ID, i]));

  if (processos.length === 0) {
    return (
      <p className="text-muted-foreground rounded-lg border p-8 text-center text-sm">
        Nenhum processo encontrado.
      </p>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border shadow-sm">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Contrato</TableHead>
            <TableHead>Contratada</TableHead>
            <TableHead>Tipo</TableHead>
            <TableHead>Etapa</TableHead>
            <TableHead>Fiscal</TableHead>
            <TableHead>Mês ref.</TableHead>
            <TableHead>Situação</TableHead>
            <TableHead>Radar</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {processos.map((processo) => {
            const posicao = processo.EtapaAtualID != null ? posicaoPorEtapa.get(processo.EtapaAtualID) : undefined;
            const cor = corDaEtapa(posicao ?? 0);
            const nivel = nivelMaisCritico(itensDoProcesso(radarItens, processo));

            return (
              <TableRow
                key={processo.ID}
                className="hover:bg-accent cursor-pointer"
                onClick={() => onOpen(processo)}
              >
                <TableCell className="font-medium">{processo.Contrato?.NumeroContrato}</TableCell>
                <TableCell className="max-w-[220px] truncate">{processo.Contrato?.ContratadaNome}</TableCell>
                <TableCell>
                  {processo.Contrato?.TipoObjeto ? TIPO_OBJETO_LABEL[processo.Contrato.TipoObjeto] : "—"}
                </TableCell>
                <TableCell>
                  <span className={cn("inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap", cor.headerBg, cor.headerText)}>
                    {processo.EtapaAtual?.Nome ?? "—"}
                  </span>
                </TableCell>
                <TableCell>
                  {processo.Contrato?.Fiscal?.Nome ? (
                    <div className="flex items-center gap-1.5">
                      <Avatar size="sm">
                        <AvatarFallback>{iniciaisDoFiscal(processo.Contrato.Fiscal.Nome)}</AvatarFallback>
                      </Avatar>
                      <span className="truncate">{processo.Contrato.Fiscal.Nome}</span>
                    </div>
                  ) : (
                    "—"
                  )}
                </TableCell>
                <TableCell>{processo.MesReferencia}</TableCell>
                <TableCell>
                  <Badge variant={processo.Status === "Concluido" ? "success" : "secondary"}>
                    {processo.Status === "Concluido" ? "Pago" : "Em andamento"}
                  </Badge>
                </TableCell>
                <TableCell>{nivel ? <RadarNivelBadge nivel={nivel} className="text-xs" /> : "—"}</TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
