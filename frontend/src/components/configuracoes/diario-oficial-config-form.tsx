"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { CheckCircle2Icon, XCircleIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ConfiguracaoDiarioOficial, ResultadoTesteConexao } from "@/lib/api/client";

// Schema PRÓPRIO (não importado de lib/validation/bff-schemas.ts) — esse
// módulo tem `import "server-only"` no topo, mesmo motivo já documentado
// em keycloak-config-form.tsx. Espelha atualizarDiarioOficialConfigSchema
// de propósito — a validação de verdade continua no servidor.
const schema = z.object({
  base_url: z.string().trim().url("precisa ser uma URL válida (ex: https://diario.example.gov.br/api)"),
  api_key: z.string().trim().max(500).optional(),
});

type FormValues = z.infer<typeof schema>;

function formatarData(iso?: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("pt-BR");
}

/**
 * Formulário de Configurações → Diário Oficial — URL base + chave de
 * API, com um botão de "Testar conexão" separado do salvar (ver o
 * comentário em DiarioOficialService.Salvar, backend, sobre por que não
 * testamos antes de salvar aqui, ao contrário do formulário de
 * Keycloak).
 */
export function DiarioOficialConfigForm({
  configuracaoInicial,
}: {
  configuracaoInicial: ConfiguracaoDiarioOficial;
}) {
  const router = useRouter();
  const [enviando, setEnviando] = useState(false);
  const [testando, setTestando] = useState(false);
  const [resultadoTeste, setResultadoTeste] = useState<ResultadoTesteConexao | null>(null);
  const [configuracao, setConfiguracao] = useState(configuracaoInicial);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      base_url: configuracaoInicial.BaseURL ?? "",
      api_key: "",
    },
  });

  async function onSubmit(values: FormValues) {
    setEnviando(true);
    try {
      const res = await fetch("/api/configuracoes/diario-oficial", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) {
        toast.error(body.error ?? "Não foi possível salvar a configuração.");
        return;
      }

      toast.success("Configuração do Diário Oficial salva.");
      setConfiguracao(body as ConfiguracaoDiarioOficial);
      setResultadoTeste(null);
      form.reset({ base_url: body.BaseURL ?? "", api_key: "" });
      router.refresh();
    } finally {
      setEnviando(false);
    }
  }

  async function testarConexao() {
    setTestando(true);
    setResultadoTeste(null);
    try {
      const res = await fetch("/api/configuracoes/diario-oficial/testar", { method: "POST" });
      const body = await res.json().catch(() => null);
      if (!res.ok) {
        toast.error(body?.error ?? "Não foi possível testar a conexão.");
        return;
      }
      setResultadoTeste(body as ResultadoTesteConexao);
    } finally {
      setTestando(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-muted/40 rounded-lg border p-4 text-sm">
        <p>
          <span className="font-medium">URL configurada agora: </span>
          {configuracao.BaseURL || "nenhuma ainda"}
        </p>
        {configuracao.AtualizadoEm && (
          <p className="text-muted-foreground mt-1">
            Última atualização: {formatarData(configuracao.AtualizadoEm)}
            {configuracao.AtualizadoPorNome ? ` por ${configuracao.AtualizadoPorNome}` : ""}
          </p>
        )}
        <p className="text-muted-foreground mt-1">
          Chave de API: {configuracao.TemChaveConfigurada ? "configurada" : "não configurada"}
        </p>
      </div>

      <form className="max-w-xl space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
        <div className="space-y-2">
          <Label htmlFor="base_url">URL base da API</Label>
          <Input id="base_url" placeholder="https://diario.example.gov.br/api" {...form.register("base_url")} />
          {form.formState.errors.base_url && (
            <p className="text-destructive text-sm">{form.formState.errors.base_url.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="api_key">Chave de API</Label>
          <Input
            id="api_key"
            type="password"
            autoComplete="off"
            placeholder={configuracao.TemChaveConfigurada ? "•••••••• (deixe em branco pra manter a atual)" : ""}
            {...form.register("api_key")}
          />
          {form.formState.errors.api_key && (
            <p className="text-destructive text-sm">{form.formState.errors.api_key.message}</p>
          )}
        </div>

        <p className="text-muted-foreground text-xs">
          Estrutura genérica: a API real do Diário Oficial da cidade ainda não está definida. Esta tela
          guarda URL/chave e assume um contrato simples (busca em <code>/contratos</code>, autenticação via{" "}
          <code>Authorization: Bearer</code>) — ajustável quando a API real for confirmada.
        </p>

        <div className="flex flex-wrap items-center gap-3">
          <Button type="submit" disabled={enviando}>
            {enviando ? "Salvando..." : "Salvar"}
          </Button>
          <Button type="button" variant="outline" onClick={testarConexao} disabled={testando}>
            {testando ? "Testando..." : "Testar conexão"}
          </Button>
        </div>
      </form>

      {resultadoTeste && (
        <div
          className={`max-w-xl rounded-lg border p-4 text-sm ${
            resultadoTeste.Sucesso
              ? "border-emerald-500/30 bg-emerald-500/10"
              : "border-destructive/30 bg-destructive/10"
          }`}
        >
          <div className="flex items-center gap-2 font-medium">
            {resultadoTeste.Sucesso ? (
              <CheckCircle2Icon className="size-4 text-emerald-600 dark:text-emerald-400" />
            ) : (
              <XCircleIcon className="text-destructive size-4" />
            )}
            {resultadoTeste.Sucesso ? "O servidor respondeu" : "Não foi possível conectar"}
          </div>
          <dl className="text-muted-foreground mt-2 space-y-1">
            {resultadoTeste.Sucesso ? (
              <>
                <div>
                  <dt className="inline font-medium">Status HTTP: </dt>
                  <dd className="inline">{resultadoTeste.StatusHTTP}</dd>
                </div>
                <div>
                  <dt className="inline font-medium">Latência: </dt>
                  <dd className="inline">{resultadoTeste.LatenciaMS}ms</dd>
                </div>
                {resultadoTeste.TrechoCorpo && (
                  <div>
                    <dt className="font-medium">Trecho da resposta:</dt>
                    <dd className="mt-1 max-h-32 overflow-auto rounded border bg-black/5 p-2 font-mono text-xs whitespace-pre-wrap dark:bg-white/5">
                      {resultadoTeste.TrechoCorpo}
                    </dd>
                  </div>
                )}
                <p className="pt-1 text-xs">
                  Um status de erro (401/404/etc) ainda conta como conexão bem-sucedida — só confirma que o
                  servidor respondeu, não que o endpoint/autenticação estão corretos pro contrato assumido.
                </p>
              </>
            ) : (
              <div>
                <dt className="inline font-medium">Erro: </dt>
                <dd className="inline">{resultadoTeste.Erro}</dd>
              </div>
            )}
          </dl>
        </div>
      )}
    </div>
  );
}
