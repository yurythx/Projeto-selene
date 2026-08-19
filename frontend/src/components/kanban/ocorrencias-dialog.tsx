"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { Ocorrencia } from "@/lib/api/client";

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw Object.assign(new Error(body.error ?? "Erro na requisição."), { body, status: res.status });
  }
  return body as T;
}

function formatarData(iso?: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("pt-BR");
}

const estadoLabel: Record<string, string> = {
  REGISTRADA: "Registrada",
  NOTIFICADA: "Notificada ao Gestor",
  EM_TRATAMENTO: "Em tratamento",
  REGULARIZADA: "Regularizada",
};

const estadoVariant: Record<string, "secondary" | "default" | "destructive" | "outline"> = {
  REGISTRADA: "destructive",
  NOTIFICADA: "secondary",
  EM_TRATAMENTO: "secondary",
  REGULARIZADA: "default",
};

/**
 * SGF-Rondonópolis: dialog aninhado no drawer do Kanban, mesmo padrão de
 * VistoriasDialog — registro e acompanhamento de Ocorrencia (IN01
 * Art.3º-III/Art.5º-IV,IX; IN04 Art.3º-VIII/Art.5º-VIII,XVI). Enquanto
 * houver uma ocorrência não regularizada aqui, o backend bloqueia de
 * verdade o avanço de etapa deste processo (ver
 * FiscalizacaoService.VerificarAvancoPermitido) — não é só um aviso
 * visual.
 */
export function OcorrenciasDialog({
  processoId,
  isFiscal,
  open,
  onOpenChange,
}: {
  processoId: string;
  isFiscal: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [descricao, setDescricao] = useState("");

  function invalidar() {
    queryClient.invalidateQueries({ queryKey: ["ocorrencias", processoId] });
    // O bloqueio de avanço depende do estado das ocorrências — a leitura
    // decorada do processo (allowed_actions/estado_fiscalizacao) também
    // precisa ser refeita.
    queryClient.invalidateQueries({ queryKey: ["processo", processoId] });
  }

  const ocorrenciasQuery = useQuery({
    queryKey: ["ocorrencias", processoId],
    queryFn: () => fetchJSON<Ocorrencia[]>(`/api/processos/${processoId}/ocorrencias`),
    enabled: open,
  });

  const registrarMutation = useMutation({
    mutationFn: () =>
      fetchJSON<Ocorrencia>(`/api/processos/${processoId}/ocorrencias`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ descricao }),
      }),
    onSuccess: () => {
      toast.success("Ocorrência registrada. O avanço de etapa fica bloqueado até regularizar.");
      invalidar();
      setDescricao("");
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  const transicaoMutation = useMutation({
    mutationFn: ({ id, acao }: { id: string; acao: "notificar" | "tratar" | "regularizar" }) =>
      fetchJSON<Ocorrencia>(`/api/ocorrencias/${id}/${acao}`, { method: "POST" }),
    onSuccess: () => {
      toast.success("Ocorrência atualizada.");
      invalidar();
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  const proximaAcao: Record<string, { acao: "notificar" | "tratar" | "regularizar"; label: string }> = {
    REGISTRADA: { acao: "notificar", label: "Notificar Gestor" },
    NOTIFICADA: { acao: "tratar", label: "Iniciar tratamento" },
    EM_TRATAMENTO: { acao: "regularizar", label: "Regularizar" },
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[90vh] flex-col sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Ocorrências</DialogTitle>
        </DialogHeader>

        <div className="flex-1 space-y-4 overflow-y-auto">
          {isFiscal && (
            <Card className="space-y-3 p-4">
              <div className="space-y-2">
                <Label htmlFor="descricao_ocorrencia">Nova ocorrência</Label>
                <Textarea
                  id="descricao_ocorrencia"
                  rows={3}
                  value={descricao}
                  onChange={(e) => setDescricao(e.target.value)}
                  placeholder="Atraso, irregularidade ou outra ocorrência relacionada à execução do contrato..."
                />
              </div>
              <Button
                type="button"
                className="w-full"
                onClick={() => registrarMutation.mutate()}
                disabled={registrarMutation.isPending || !descricao.trim()}
              >
                {registrarMutation.isPending ? "Registrando..." : "Registrar ocorrência"}
              </Button>
            </Card>
          )}

          <div className="space-y-3">
            {ocorrenciasQuery.isLoading && (
              <p className="text-muted-foreground text-sm">Carregando...</p>
            )}
            {ocorrenciasQuery.data?.length === 0 && (
              <p className="text-muted-foreground text-sm">Nenhuma ocorrência registrada.</p>
            )}
            {ocorrenciasQuery.data?.map((ocorrencia) => {
              const proxima = ocorrencia.Estado ? proximaAcao[ocorrencia.Estado] : undefined;
              return (
                <Card key={ocorrencia.ID} className="space-y-2 p-4 text-sm">
                  <div className="flex items-center justify-between gap-2">
                    <Badge variant={estadoVariant[ocorrencia.Estado ?? ""] ?? "outline"}>
                      {estadoLabel[ocorrencia.Estado ?? ""] ?? ocorrencia.Estado}
                    </Badge>
                    <span className="text-muted-foreground text-xs">
                      {formatarData(ocorrencia.CreatedAt)}
                    </span>
                  </div>
                  <p>{ocorrencia.Descricao}</p>
                  {ocorrencia.DataNotificacaoGestor && (
                    <p className="text-muted-foreground text-xs">
                      Notificada ao Gestor em {formatarData(ocorrencia.DataNotificacaoGestor)}
                    </p>
                  )}
                  {ocorrencia.DataRegularizacao && (
                    <p className="text-muted-foreground text-xs">
                      Regularizada em {formatarData(ocorrencia.DataRegularizacao)}
                    </p>
                  )}
                  {isFiscal && proxima && (
                    <Button
                      type="button"
                      size="sm"
                      variant="secondary"
                      onClick={() =>
                        transicaoMutation.mutate({ id: ocorrencia.ID!, acao: proxima.acao })
                      }
                      disabled={transicaoMutation.isPending}
                    >
                      {proxima.label}
                    </Button>
                  )}
                </Card>
              );
            })}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
