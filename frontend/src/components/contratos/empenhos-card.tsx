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
import type { Empenho, EmpenhoComSaldo } from "@/lib/api/client";

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw Object.assign(new Error(body.error ?? "Erro na requisição."), { body, status: res.status });
  }
  return body as T;
}

function formatarCentavos(centavos?: number) {
  if (centavos == null) return "—";
  return (centavos / 100).toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
}

function formatarData(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("pt-BR");
}

/**
 * Converte um valor em reais digitado como string ("1.234,56" ou
 * "1234.56") pra centavos (int) — mesma convenção monetária do backend
 * (models.Empenho.ValorInicial, int64 em centavos, ver o plano).
 */
function paraCentavos(valor: string): number | null {
  const normalizado = valor.trim().replace(/\./g, "").replace(",", ".");
  const numero = Number(normalizado);
  if (!Number.isFinite(numero) || numero <= 0) return null;
  return Math.round(numero * 100);
}

function SaldoEmpenho({ empenhoId }: { empenhoId: string }) {
  const query = useQuery({
    queryKey: ["empenho", empenhoId],
    queryFn: () => fetchJSON<EmpenhoComSaldo>(`/api/empenhos/${empenhoId}`),
  });
  if (query.isLoading) return <span className="text-muted-foreground text-xs">calculando...</span>;
  return <Badge variant="outline">Saldo: {formatarCentavos(query.data?.saldo)}</Badge>;
}

function NovoEmpenhoDialog({ contratoId }: { contratoId: string }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [numero, setNumero] = useState("");
  const [dataEmissao, setDataEmissao] = useState("");
  const [valor, setValor] = useState("");

  const mutation = useMutation({
    mutationFn: () => {
      const valorCentavos = paraCentavos(valor);
      if (!valorCentavos) throw new Error("Valor inválido.");
      return fetchJSON<Empenho>(`/api/contratos/${contratoId}/empenhos`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          numero_empenho: numero,
          data_emissao: dataEmissao,
          valor_inicial: valorCentavos,
        }),
      });
    },
    onSuccess: () => {
      toast.success("Empenho registrado.");
      queryClient.invalidateQueries({ queryKey: ["empenhos", contratoId] });
      setOpen(false);
      setNumero("");
      setDataEmissao("");
      setValor("");
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" size="sm">Novo empenho</Button>} />
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Registrar empenho</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div className="space-y-1">
            <Label htmlFor="numero_empenho">Número do empenho</Label>
            <Input id="numero_empenho" value={numero} onChange={(e) => setNumero(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="data_emissao">Data de emissão</Label>
            <Input
              id="data_emissao"
              type="date"
              value={dataEmissao}
              onChange={(e) => setDataEmissao(e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="valor_inicial">Valor inicial (R$)</Label>
            <Input
              id="valor_inicial"
              inputMode="decimal"
              placeholder="0,00"
              value={valor}
              onChange={(e) => setValor(e.target.value)}
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            onClick={() => mutation.mutate()}
            disabled={!numero || !dataEmissao || !valor || mutation.isPending}
          >
            {mutation.isPending ? "Registrando..." : "Registrar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function NovaMovimentacaoDialog({ empenhoId }: { empenhoId: string }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [tipo, setTipo] = useState<"REFORCO" | "ANULACAO" | "">("");
  const [valor, setValor] = useState("");

  const mutation = useMutation({
    mutationFn: () => {
      const valorCentavos = paraCentavos(valor);
      if (!valorCentavos || !tipo) throw new Error("Preencha tipo e valor.");
      return fetchJSON(`/api/empenhos/${empenhoId}/movimentacoes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tipo, valor: valorCentavos }),
      });
    },
    onSuccess: () => {
      toast.success("Movimentação registrada.");
      queryClient.invalidateQueries({ queryKey: ["empenho", empenhoId] });
      setOpen(false);
      setTipo("");
      setValor("");
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="ghost" size="sm">Reforço/Anulação</Button>} />
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Registrar movimentação</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div className="space-y-1">
            <Label htmlFor="tipo_movimentacao">Tipo</Label>
            <Select value={tipo} onValueChange={(value) => setTipo((value as typeof tipo) ?? "")}>
              <SelectTrigger id="tipo_movimentacao" className="w-full">
                <SelectValue placeholder="Selecione" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="REFORCO">Reforço (soma ao saldo)</SelectItem>
                <SelectItem value="ANULACAO">Anulação (subtrai do saldo)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="valor_movimentacao">Valor (R$)</Label>
            <Input
              id="valor_movimentacao"
              inputMode="decimal"
              placeholder="0,00"
              value={valor}
              onChange={(e) => setValor(e.target.value)}
            />
          </div>
        </div>
        <DialogFooter>
          <Button onClick={() => mutation.mutate()} disabled={!tipo || !valor || mutation.isPending}>
            {mutation.isPending ? "Registrando..." : "Registrar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/**
 * SGF-Rondonópolis: acompanhamento PARALELO/informativo de saldo de
 * empenho (IN01 Art.5º-VIII; IN04 Art.5º-XXII) — Empenho no backend.
 * NÃO é a fonte de verdade orçamentária (essa é exclusiva dos sistemas
 * corporativos da prefeitura, ver o comentário em
 * backend/internal/models/empenho.go); é o apoio de acompanhamento do
 * fiscal dentro do Selene.
 */
export function EmpenhosCard({ contratoId, isFiscal }: { contratoId: string; isFiscal: boolean }) {
  const query = useQuery({
    queryKey: ["empenhos", contratoId],
    queryFn: () => fetchJSON<Empenho[]>(`/api/contratos/${contratoId}/empenhos`),
  });

  return (
    <Card className="shadow-sm">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-base">Empenho (acompanhamento SGF)</CardTitle>
        {isFiscal && <NovoEmpenhoDialog contratoId={contratoId} />}
      </CardHeader>
      <CardContent className="space-y-3">
        {query.isLoading && <p className="text-muted-foreground text-sm">Carregando...</p>}
        {query.data?.length === 0 && (
          <p className="text-muted-foreground text-sm">Nenhum empenho registrado ainda.</p>
        )}
        {query.data?.map((empenho) => (
          <div key={empenho.ID} className="space-y-1 border-b pb-2 text-sm last:border-b-0 last:pb-0">
            <div className="flex items-center justify-between gap-2">
              <p className="font-medium">Empenho nº {empenho.NumeroEmpenho}</p>
              <SaldoEmpenho empenhoId={empenho.ID!} />
            </div>
            <div className="flex items-center justify-between gap-2">
              <p className="text-muted-foreground text-xs">
                Emitido em {formatarData(empenho.DataEmissao)} — valor inicial{" "}
                {formatarCentavos(empenho.ValorInicial)}
              </p>
              {isFiscal && <NovaMovimentacaoDialog empenhoId={empenho.ID!} />}
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
