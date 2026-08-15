"use client";

import { useQuery } from "@tanstack/react-query";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { PortariaDesignacao } from "@/lib/api/client";

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
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

/**
 * SGF-Rondonópolis: histórico auditável de designação de
 * fiscal/suplente/gestor/fiscal setorial (IN01 Art.4º-I/Art.6º; IN04
 * Art.4º-I/Art.10) — PortariaDesignacao no backend. Só leitura por
 * enquanto: criar uma nova designação exige escolher um servidor por ID,
 * e não existe hoje uma listagem de usuários acessível a um fiscal comum
 * (GET /admin/users é admin-only) pra montar esse seletor — ver o plano,
 * Fase 6, sobre modelagem de papéis ainda incompleta. Quando essa
 * listagem existir, um formulário "Nova designação" entra aqui do mesmo
 * jeito que EmpenhosCard já tem pra Empenho.
 */
export function DesignacoesCard({ contratoId }: { contratoId: string }) {
  const query = useQuery({
    queryKey: ["designacoes", contratoId],
    queryFn: () => fetchJSON<PortariaDesignacao[]>(`/api/contratos/${contratoId}/designacoes`),
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Histórico de designação (SGF)</CardTitle>
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
