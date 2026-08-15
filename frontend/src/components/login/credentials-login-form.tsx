"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { signIn } from "next-auth/react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { TriangleAlertIcon } from "lucide-react";
import { sanitizeCallbackUrl } from "@/lib/safe-redirect";

const schema = z.object({
  email: z.string().trim().min(1, "Obrigatório").email("E-mail inválido"),
  senha: z.string().min(1, "Obrigatório"),
});

type FormValues = z.infer<typeof schema>;

/**
 * Login tradicional (e-mail/senha) — alternativa ao botão "Entrar com
 * Keycloak" na mesma página. Client Component (ao contrário do resto de
 * /login, que é Server Component com uma server action): signIn() do
 * provider Credentials precisa rodar no client, com redirect:false, pra
 * dar pra mostrar a mensagem de erro sem sair da página em caso de
 * credenciais inválidas.
 *
 * router.push(callbackUrl) abaixo é navegação client-side — ao contrário
 * de signIn(..., { redirectTo }) do Auth.js, NÃO passa pela validação
 * same-origin que o framework aplica por padrão. callbackUrl já chega
 * sanitizado de LoginPage (page.tsx), mas sanitizeCallbackUrl() é
 * chamado de novo aqui: este componente não deveria depender de quem o
 * renderiza pra não abrir um Open Redirect (CWE-601) se um dia ganhar
 * outro call site.
 */
export function CredentialsLoginForm({ callbackUrl }: { callbackUrl: string }) {
  const router = useRouter();
  const [erro, setErro] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", senha: "" },
  });

  async function onSubmit(values: FormValues) {
    setErro(null);
    setEnviando(true);
    try {
      const resultado = await signIn("credentials", {
        email: values.email,
        senha: values.senha,
        redirect: false,
      });

      if (resultado?.error) {
        setErro("E-mail ou senha inválidos.");
        return;
      }

      router.push(sanitizeCallbackUrl(callbackUrl));
      router.refresh();
    } finally {
      setEnviando(false);
    }
  }

  return (
    <form className="space-y-4 text-left" onSubmit={form.handleSubmit(onSubmit)}>
      {erro && (
        <Alert variant="destructive">
          <TriangleAlertIcon />
          <AlertDescription>{erro}</AlertDescription>
        </Alert>
      )}

      <div className="space-y-2">
        <Label htmlFor="email">E-mail</Label>
        <Input id="email" type="email" autoComplete="username" {...form.register("email")} />
        {form.formState.errors.email && (
          <p className="text-destructive text-sm">{form.formState.errors.email.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="senha">Senha</Label>
        <Input id="senha" type="password" autoComplete="current-password" {...form.register("senha")} />
        {form.formState.errors.senha && (
          <p className="text-destructive text-sm">{form.formState.errors.senha.message}</p>
        )}
      </div>

      <Button type="submit" variant="outline" className="w-full" disabled={enviando}>
        {enviando ? "Entrando..." : "Entrar com e-mail e senha"}
      </Button>
    </form>
  );
}
