"use client";

import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { BellIcon, CheckCheckIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { RadarNivelBadge } from "@/components/radar/radar-badge";
import type { Notificacao } from "@/lib/api/client";

async function buscarNaoLidas(): Promise<number> {
  const res = await fetch("/api/notificacoes/nao-lidas");
  if (!res.ok) return 0;
  const body = await res.json();
  return body.total ?? 0;
}

async function buscarNotificacoes(): Promise<Notificacao[]> {
  const res = await fetch("/api/notificacoes");
  if (!res.ok) return [];
  return res.json();
}

function formatarData(iso: string) {
  return new Date(iso).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

/**
 * Sino de notificações in-app na TopBar — pedido explícito do usuário:
 * "precisamos ter os alertas/notificacoes a respeito de prazos e
 * vencimentos", canal "dentro do app" confirmado junto com e-mail (ver
 * NotificacaoService.GerarAlertas, backend, que gera as duas). As
 * notificações em si são geradas por um processo em segundo plano no
 * backend (a cada NOTIFICACOES_INTERVALO_HORAS) — esta tela só lê e
 * marca como lida, nunca gera.
 *
 * refetchInterval: 60s — sem WebSocket/SSE nesta stack, um poll simples
 * já é suficiente pro volume de alertas esperado (não é um chat, alguns
 * minutos de atraso pra ver um alerta novo é aceitável).
 */
export function NotificacoesBell() {
  const queryClient = useQueryClient();

  const { data: naoLidas = 0 } = useQuery({
    queryKey: ["notificacoes-nao-lidas"],
    queryFn: buscarNaoLidas,
    refetchInterval: 60_000,
  });

  const { data: notificacoes = [] } = useQuery({
    queryKey: ["notificacoes"],
    queryFn: buscarNotificacoes,
    refetchInterval: 60_000,
  });

  function invalidar() {
    queryClient.invalidateQueries({ queryKey: ["notificacoes-nao-lidas"] });
    queryClient.invalidateQueries({ queryKey: ["notificacoes"] });
  }

  const marcarLida = useMutation({
    mutationFn: (id: string) => fetch(`/api/notificacoes/${id}/marcar-lida`, { method: "POST" }),
    onSuccess: invalidar,
  });

  const marcarTodasLidas = useMutation({
    mutationFn: () => fetch("/api/notificacoes/marcar-todas-lidas", { method: "POST" }),
    onSuccess: invalidar,
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="ghost" size="icon" className="relative" aria-label="Notificações">
            <BellIcon className="size-5" />
            {naoLidas > 0 && (
              <span className="bg-destructive absolute top-1 right-1 flex size-4 items-center justify-center rounded-full text-[10px] font-semibold text-white">
                {naoLidas > 9 ? "9+" : naoLidas}
              </span>
            )}
          </Button>
        }
      />
      <DropdownMenuContent align="end" side="bottom" className="w-96 p-0">
        <div className="flex items-center justify-between border-b p-3">
          <p className="text-sm font-semibold">Notificações</p>
          {naoLidas > 0 && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => marcarTodasLidas.mutate()}
              disabled={marcarTodasLidas.isPending}
            >
              <CheckCheckIcon />
              Marcar todas como lidas
            </Button>
          )}
        </div>

        <div className="max-h-96 overflow-y-auto">
          {notificacoes.length === 0 ? (
            <p className="text-muted-foreground p-4 text-sm">Nenhuma notificação por enquanto.</p>
          ) : (
            notificacoes.map((n, indice) => (
              <div key={n.id}>
                {indice > 0 && <DropdownMenuSeparator className="my-0" />}
                <Link
                  href={`/contratos/${n.contrato_id}`}
                  onClick={() => {
                    if (!n.lida) marcarLida.mutate(n.id!);
                  }}
                  className={`hover:bg-accent block p-3 text-sm transition-colors ${n.lida ? "opacity-60" : ""}`}
                >
                  <div className="flex items-center gap-2">
                    <RadarNivelBadge nivel={n.nivel!} />
                    {n.numero_contrato && (
                      <span className="text-muted-foreground text-xs">Contrato {n.numero_contrato}</span>
                    )}
                    {!n.lida && <Badge className="ml-auto h-4.5 px-1.5 text-[10px]">novo</Badge>}
                  </div>
                  <p className="mt-1">{n.mensagem}</p>
                  <p className="text-muted-foreground mt-1 text-xs">{formatarData(n.criada_em!)}</p>
                </Link>
              </div>
            ))
          )}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
