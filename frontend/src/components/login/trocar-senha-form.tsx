"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "next-auth/react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const schema = z
  .object({
    senhaAtual: z.string().min(1, "Obrigatório"),
    senhaNova: z.string().min(8, "Mínimo 8 caracteres"),
    confirmarSenhaNova: z.string().min(1, "Obrigatório"),
  })
  .refine((v) => v.senhaNova === v.confirmarSenhaNova, {
    message: "As senhas não coincidem",
    path: ["confirmarSenhaNova"],
  });

type FormValues = z.infer<typeof schema>;

/**
 * Formulário de troca de senha — usado tanto na troca obrigatória de
 * primeiro login (redirecionada pelo proxy.ts quando mustChangePassword)
 * quanto numa troca voluntária futura. Depois do POST ter sucesso, chama
 * update() do useSession() pra atualizar a claim mustChangePassword na
 * sessão SEM precisar logar de novo (ver o trigger==="update" em
 * src/auth.ts) — sem isso, o proxy.ts continuaria redirecionando pra cá
 * eternamente, mesmo com a senha já trocada.
 */
export function TrocarSenhaForm({ obrigatoria }: { obrigatoria: boolean }) {
  const router = useRouter();
  const { update } = useSession();
  const [enviando, setEnviando] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { senhaAtual: "", senhaNova: "", confirmarSenhaNova: "" },
  });

  async function onSubmit(values: FormValues) {
    setEnviando(true);
    try {
      const res = await fetch("/api/auth/trocar-senha", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ senha_atual: values.senhaAtual, senha_nova: values.senhaNova }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        toast.error(body.error ?? "Não foi possível trocar a senha.");
        return;
      }

      // update() SEM argumento faz next-auth mandar um GET (só releitura
      // da sessão) — o servidor só aplica trigger:"update" no callback
      // jwt() (ver src/auth.ts) quando a requisição é POST, o que só
      // acontece se passarmos ALGO como argumento (mesmo objeto vazio:
      // `typeof data === "undefined"` é o que decide GET vs POST em
      // next-auth/react.js).
      await update({});
      toast.success("Senha trocada com sucesso.");
      router.push("/");
      router.refresh();
    } finally {
      setEnviando(false);
    }
  }

  return (
    <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
      {obrigatoria && (
        <p className="text-muted-foreground text-sm">
          Sua conta tem uma senha temporária — troque-a antes de continuar.
        </p>
      )}

      <div className="space-y-2">
        <Label htmlFor="senhaAtual">Senha atual</Label>
        <Input id="senhaAtual" type="password" autoComplete="current-password" {...form.register("senhaAtual")} />
        {form.formState.errors.senhaAtual && (
          <p className="text-destructive text-sm">{form.formState.errors.senhaAtual.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="senhaNova">Nova senha</Label>
        <Input id="senhaNova" type="password" autoComplete="new-password" {...form.register("senhaNova")} />
        {form.formState.errors.senhaNova && (
          <p className="text-destructive text-sm">{form.formState.errors.senhaNova.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="confirmarSenhaNova">Confirmar nova senha</Label>
        <Input
          id="confirmarSenhaNova"
          type="password"
          autoComplete="new-password"
          {...form.register("confirmarSenhaNova")}
        />
        {form.formState.errors.confirmarSenhaNova && (
          <p className="text-destructive text-sm">{form.formState.errors.confirmarSenhaNova.message}</p>
        )}
      </div>

      <Button type="submit" className="w-full" disabled={enviando}>
        {enviando ? "Salvando..." : "Trocar senha"}
      </Button>
    </form>
  );
}
