"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { SearchIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";

// Schema PRÓPRIO (mesmo motivo já documentado em
// diario-oficial-config-form.tsx) — espelha
// buscarContratosDiarioOficialSchema (lib/validation/bff-schemas.ts).
const schema = z.object({
  nome: z.string().trim().max(255).optional(),
  cpf: z.string().trim().max(20).optional(),
  data: z.string().trim().max(20).optional(),
});

type FormValues = z.infer<typeof schema>;

/**
 * Tenta achar uma lista de resultados dentro de um JSON de formato
 * desconhecido — ESTRUTURA GENÉRICA (ver o comentário de escopo em
 * backend/internal/service/diario_oficial_service.go): sem um schema
 * real confirmado, a única forma honesta de renderizar é aceitar tanto
 * "a resposta já é a lista" quanto "a lista está dentro de uma chave
 * comum" (resultados/items/data/contratos), com fallback pro JSON bruto
 * quando nada disso bate.
 */
function extrairItens(resultado: unknown): Record<string, unknown>[] | null {
  if (Array.isArray(resultado)) {
    return resultado.every((item) => item !== null && typeof item === "object")
      ? (resultado as Record<string, unknown>[])
      : null;
  }
  if (resultado && typeof resultado === "object") {
    for (const chave of ["resultados", "items", "data", "contratos", "results"]) {
      const valor = (resultado as Record<string, unknown>)[chave];
      if (Array.isArray(valor)) {
        return valor.every((item) => item !== null && typeof item === "object")
          ? (valor as Record<string, unknown>[])
          : null;
      }
    }
  }
  return null;
}

function formatarValor(valor: unknown): string {
  if (valor === null || valor === undefined) return "—";
  if (typeof valor === "object") return JSON.stringify(valor);
  return String(valor);
}

export function BuscarContratosDiarioOficial() {
  const [buscando, setBuscando] = useState(false);
  const [resultado, setResultado] = useState<unknown>(undefined);
  const [buscou, setBuscou] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { nome: "", cpf: "", data: "" },
  });

  async function onSubmit(values: FormValues) {
    setBuscando(true);
    setBuscou(false);
    try {
      const params = new URLSearchParams();
      if (values.nome) params.set("nome", values.nome);
      if (values.cpf) params.set("cpf", values.cpf);
      if (values.data) params.set("data", values.data);

      const res = await fetch(`/api/configuracoes/diario-oficial/buscar?${params.toString()}`);
      const body = await res.json().catch(() => null);
      if (!res.ok) {
        toast.error(body?.error ?? "Não foi possível buscar no Diário Oficial.");
        return;
      }
      setResultado(body.resultado);
      setBuscou(true);
    } finally {
      setBuscando(false);
    }
  }

  const itens = buscou ? extrairItens(resultado) : null;

  return (
    <div className="space-y-6">
      <form
        className="flex flex-wrap items-end gap-4"
        onSubmit={form.handleSubmit(onSubmit)}
        // Sem "role=search" explícito — este form já é o único de busca
        // nesta página, o rótulo "Buscar" no botão já deixa a intenção
        // clara sem precisar de mais um atributo ARIA redundante.
      >
        <div className="space-y-2">
          <Label htmlFor="nome">Nome</Label>
          <Input id="nome" placeholder="Nome da contratada" {...form.register("nome")} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="cpf">CPF</Label>
          <Input id="cpf" placeholder="000.000.000-00" {...form.register("cpf")} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="data">Data</Label>
          <Input id="data" type="date" {...form.register("data")} />
        </div>
        <Button type="submit" disabled={buscando}>
          <SearchIcon />
          {buscando ? "Buscando..." : "Buscar"}
        </Button>
      </form>

      {buscou && (
        <div>
          {itens !== null ? (
            itens.length === 0 ? (
              <p className="text-muted-foreground text-sm">Nenhum resultado encontrado.</p>
            ) : (
              <div className="space-y-3">
                <p className="text-muted-foreground text-sm">
                  {itens.length} resultado{itens.length === 1 ? "" : "s"}.
                </p>
                {itens.map((item, indice) => (
                  // Sem um id confiável no formato assumido — indice é
                  // estável o bastante aqui (a lista é substituída
                  // inteira a cada busca, nunca reordenada in-place).
                  <Card key={indice} className="shadow-sm">
                    <CardContent>
                      <dl className="grid grid-cols-1 gap-x-6 gap-y-1 text-sm sm:grid-cols-2">
                        {Object.entries(item).map(([chave, valor]) => (
                          <div key={chave} className="flex gap-2">
                            <dt className="text-muted-foreground shrink-0 font-medium">{chave}:</dt>
                            <dd className="min-w-0 break-words">{formatarValor(valor)}</dd>
                          </div>
                        ))}
                      </dl>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )
          ) : (
            <div className="space-y-2">
              <p className="text-muted-foreground text-sm">
                A resposta da API não bateu com o formato esperado (lista de itens) — mostrando o JSON
                bruto pra debug:
              </p>
              <pre className="max-h-96 overflow-auto rounded-lg border bg-black/5 p-4 text-xs whitespace-pre-wrap dark:bg-white/5">
                {JSON.stringify(resultado, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
