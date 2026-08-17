"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ConfiguracaoKeycloak } from "@/lib/api/client";

// Schema PRÓPRIO (não importado de lib/validation/bff-schemas.ts) — esse
// módulo tem `import "server-only"` no topo (é validação do lado do
// Route Handler), então não pode ser importado por um Client Component
// como este. As regras espelham atualizarKeycloakConfigSchema de
// propósito — a validação de verdade continua acontecendo no servidor
// (Route Handler + backend Go), esta aqui é só feedback imediato na UI.
const schema = z.object({
  client_id: z.string().trim().min(1, "obrigatório").max(255),
  client_secret: z.string().trim().max(500).optional(),
  issuer_url: z.string().trim().url("precisa ser uma URL válida (ex: https://keycloak.exemplo.gov.br/realms/selene)"),
  audience: z.string().trim().max(255).optional(),
});

type FormValues = z.infer<typeof schema>;

function formatarData(iso?: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("pt-BR");
}

/**
 * Formulário de Configurações → Keycloak/SSO — pedido explícito do
 * usuário: "já usamos [Keycloak] hoje mas não temos no front, e se eu
 * quiser mudar ou implementar um novo, crie uma opção". Aplica em
 * runtime (sem reiniciar os containers): o backend testa o novo
 * issuer/JWKS ANTES de salvar (ver KeycloakConfigService.Salvar) e o
 * frontend passa a usar o novo Client ID/Secret/Issuer no login SSO no
 * próximo request de autenticação (cache de até 60s, ver
 * lib/keycloak-config.ts).
 */
export function KeycloakConfigForm({ configuracaoInicial }: { configuracaoInicial: ConfiguracaoKeycloak }) {
  const router = useRouter();
  const [enviando, setEnviando] = useState(false);
  const [configuracao, setConfiguracao] = useState(configuracaoInicial);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      client_id: configuracaoInicial.ClientID ?? "",
      client_secret: "",
      issuer_url: configuracaoInicial.IssuerURL ?? "",
      audience: configuracaoInicial.Audience ?? "",
    },
  });

  async function onSubmit(values: FormValues) {
    setEnviando(true);
    try {
      const res = await fetch("/api/configuracoes/keycloak", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) {
        toast.error(body.error ?? "Não foi possível salvar a configuração.");
        return;
      }

      toast.success("Configuração de Keycloak salva e aplicada.");
      setConfiguracao(body as ConfiguracaoKeycloak);
      form.reset({
        client_id: body.ClientID ?? "",
        client_secret: "",
        issuer_url: body.IssuerURL ?? "",
        audience: body.Audience ?? "",
      });
      router.refresh();
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-muted/40 rounded-lg border p-4 text-sm">
        <p>
          <span className="font-medium">Origem ativa agora: </span>
          {configuracao.Origem === "banco_de_dados" ? (
            <>configuração salva pela tela abaixo</>
          ) : (
            <>variáveis de ambiente do container (nenhum admin salvou nada por aqui ainda)</>
          )}
        </p>
        {configuracao.Origem === "banco_de_dados" && (
          <p className="text-muted-foreground mt-1">
            Última atualização: {formatarData(configuracao.AtualizadoEm)}
            {configuracao.AtualizadoPorNome ? ` por ${configuracao.AtualizadoPorNome}` : ""}
          </p>
        )}
        <p className="text-muted-foreground mt-1">
          Segredo (Client Secret): {configuracao.TemSegredoConfigurado ? "configurado" : "não configurado"}
        </p>
      </div>

      <form className="max-w-xl space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
        <div className="space-y-2">
          <Label htmlFor="client_id">Client ID</Label>
          <Input id="client_id" {...form.register("client_id")} />
          {form.formState.errors.client_id && (
            <p className="text-destructive text-sm">{form.formState.errors.client_id.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="client_secret">Client Secret</Label>
          <Input
            id="client_secret"
            type="password"
            autoComplete="off"
            placeholder={configuracao.TemSegredoConfigurado ? "•••••••• (deixe em branco pra manter o atual)" : ""}
            {...form.register("client_secret")}
          />
          {form.formState.errors.client_secret && (
            <p className="text-destructive text-sm">{form.formState.errors.client_secret.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="issuer_url">Issuer URL</Label>
          <Input
            id="issuer_url"
            placeholder="https://keycloak.prefeitura.gov.br/realms/selene"
            {...form.register("issuer_url")}
          />
          <p className="text-muted-foreground text-xs">
            URL do realm, sem barra final. O JWKS é derivado automaticamente
            (.../protocol/openid-connect/certs).
          </p>
          {form.formState.errors.issuer_url && (
            <p className="text-destructive text-sm">{form.formState.errors.issuer_url.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="audience">Audience (opcional)</Label>
          <Input id="audience" {...form.register("audience")} />
          <p className="text-muted-foreground text-xs">
            Deixe em branco se o client do Keycloak não popular o claim &quot;aud&quot; — a validação de
            audience fica desligada nesse caso.
          </p>
        </div>

        <p className="text-muted-foreground text-xs">
          A nova configuração é testada (o backend busca o JWKS do novo issuer) antes de ser salva — se o
          Keycloak informado não responder, nada é alterado. Trocar Client ID/Secret pode exigir que quem já
          estiver logado via SSO Keycloak entre de novo; contas de login local não são afetadas.
        </p>

        <Button type="submit" disabled={enviando}>
          {enviando ? "Salvando..." : "Salvar e aplicar"}
        </Button>
      </form>
    </div>
  );
}
