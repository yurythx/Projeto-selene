"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { DesignarRequest, PortariaDesignacao, ServidorOpcao } from "@/lib/api/client";

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

const papelLabel: Record<string, string> = {
  FISCAL: "Fiscal",
  FISCAL_SUPLENTE: "Fiscal Suplente",
  GESTOR: "Gestor",
  FISCAL_SETORIAL: "Fiscal Setorial",
};

type PapelValue = DesignarRequest["papel"];

const PAPEIS: PapelValue[] = ["FISCAL", "FISCAL_SUPLENTE", "GESTOR", "FISCAL_SETORIAL"];

function NovaDesignacaoDialog({ contratoId }: { contratoId: string }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [servidorId, setServidorId] = useState("");
  const [papel, setPapel] = useState<PapelValue | "">("");
  const [numeroPortaria, setNumeroPortaria] = useState("");
  const [publicadoDiorondon, setPublicadoDiorondon] = useState("");
  const [dataDesignacao, setDataDesignacao] = useState("");

  // Só busca a lista de servidores quando o dialog está aberto — evita
  // uma requisição desnecessária em toda carga da página de contrato.
  const servidoresQuery = useQuery({
    queryKey: ["servidores"],
    queryFn: () => fetchJSON<ServidorOpcao[]>("/api/servidores"),
    enabled: open,
  });

  const limpar = () => {
    setServidorId("");
    setPapel("");
    setNumeroPortaria("");
    setPublicadoDiorondon("");
    setDataDesignacao("");
  };

  const mutation = useMutation({
    mutationFn: () => {
      if (!servidorId || !papel) throw new Error("Selecione o servidor e o papel.");
      const body: DesignarRequest = {
        servidor_id: servidorId,
        papel,
        numero_portaria: numeroPortaria || undefined,
        publicado_diorondon: publicadoDiorondon || undefined,
        data_designacao: dataDesignacao || undefined,
      };
      return fetchJSON<PortariaDesignacao>(`/api/contratos/${contratoId}/designacoes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
    },
    onSuccess: () => {
      toast.success("Designação registrada.");
      queryClient.invalidateQueries({ queryKey: ["designacoes", contratoId] });
      setOpen(false);
      limpar();
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" size="sm">Nova designação</Button>} />
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Nova designação</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div className="space-y-1">
            <Label htmlFor="servidor_id">Servidor</Label>
            <Select value={servidorId} onValueChange={(value) => setServidorId(value ?? "")}>
              <SelectTrigger id="servidor_id" className="w-full">
                <SelectValue
                  placeholder={servidoresQuery.isLoading ? "Carregando..." : "Selecione"}
                />
              </SelectTrigger>
              <SelectContent>
                {servidoresQuery.data?.map((servidor) => (
                  <SelectItem key={servidor.ID} value={servidor.ID!}>
                    {servidor.Nome} ({servidor.Email})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="papel">Papel</Label>
            <Select value={papel} onValueChange={(value) => setPapel((value as PapelValue) ?? "")}>
              <SelectTrigger id="papel" className="w-full">
                <SelectValue placeholder="Selecione" />
              </SelectTrigger>
              <SelectContent>
                {PAPEIS.map((p) => (
                  <SelectItem key={p} value={p}>
                    {papelLabel[p]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="numero_portaria">Nº da portaria (opcional)</Label>
            <Input
              id="numero_portaria"
              value={numeroPortaria}
              onChange={(e) => setNumeroPortaria(e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="publicado_diorondon">Publicação no Diorondon (opcional)</Label>
            <Input
              id="publicado_diorondon"
              value={publicadoDiorondon}
              onChange={(e) => setPublicadoDiorondon(e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="data_designacao">Data da designação (opcional)</Label>
            <Input
              id="data_designacao"
              type="date"
              value={dataDesignacao}
              onChange={(e) => setDataDesignacao(e.target.value)}
            />
            <p className="text-muted-foreground text-xs">Em branco usa a data de hoje.</p>
          </div>
        </div>
        <DialogFooter>
          <Button
            onClick={() => mutation.mutate()}
            disabled={!servidorId || !papel || mutation.isPending}
          >
            {mutation.isPending ? "Registrando..." : "Registrar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/**
 * SGF-Rondonópolis: histórico auditável de designação de
 * fiscal/suplente/gestor/fiscal setorial (IN01 Art.4º-I/Art.6º; IN04
 * Art.4º-I/Art.10) — PortariaDesignacao no backend. Uma nova designação
 * do mesmo papel revoga automaticamente a anterior (ver DesignacaoService
 * no backend); designar Papel=FISCAL também atualiza Contrato.FiscalID.
 * O seletor de servidor (NovaDesignacaoDialog) usa GET /servidores — uma
 * projeção mínima de usuário aberta a qualquer autenticado, não só admin
 * (diferente de GET /admin/users), ver UserHandler.ListarServidores.
 */
export function DesignacoesCard({ contratoId, isFiscal }: { contratoId: string; isFiscal: boolean }) {
  const query = useQuery({
    queryKey: ["designacoes", contratoId],
    queryFn: () => fetchJSON<PortariaDesignacao[]>(`/api/contratos/${contratoId}/designacoes`),
  });

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-base">Histórico de designação (SGF)</CardTitle>
        {isFiscal && <NovaDesignacaoDialog contratoId={contratoId} />}
      </CardHeader>
      <CardContent className="space-y-3">
        {query.isLoading && <p className="text-muted-foreground text-sm">Carregando...</p>}
        {query.data?.length === 0 && (
          <p className="text-muted-foreground text-sm">Nenhuma designação registrada ainda.</p>
        )}
        {query.data?.map((designacao) => (
          <div
            key={designacao.ID}
            className="flex items-center justify-between gap-2 border-b pb-2 text-sm last:border-b-0 last:pb-0"
          >
            <div>
              <p className="font-medium">{designacao.Servidor?.Nome ?? "—"}</p>
              <p className="text-muted-foreground text-xs">
                Desde {formatarData(designacao.DataDesignacao)}
                {designacao.NumeroPortaria && ` — Portaria nº ${designacao.NumeroPortaria}`}
              </p>
            </div>
            <Badge variant={designacao.DataRevogacao ? "secondary" : "default"}>
              {papelLabel[designacao.Papel ?? ""] ?? designacao.Papel}
              {designacao.DataRevogacao && " (encerrada)"}
            </Badge>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
