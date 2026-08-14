"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

const schema = z.object({
  nome: z.string().min(1, "Obrigatório"),
  email: z.string().email("E-mail inválido"),
  senha_temporaria: z.string().min(8, "Mínimo 8 caracteres"),
  is_fiscal: z.boolean(),
  is_admin: z.boolean(),
});

type FormValues = z.infer<typeof schema>;

/**
 * Cria uma conta de login local (usuário/senha) — a alternativa ao
 * Keycloak. Sem autocadastro público: só um admin cria, por aqui. A conta
 * nasce com uma senha temporária que o próprio usuário troca no primeiro
 * login (ver /trocar-senha, forçado pelo proxy.ts).
 */
export function CriarUsuarioLocalDialog() {
  const [open, setOpen] = useState(false);
  const router = useRouter();

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { nome: "", email: "", senha_temporaria: "", is_fiscal: false, is_admin: false },
  });

  // useWatch (não form.watch()) — ver o comentário equivalente em
  // gerar-minuta-aditivo-dialog.tsx sobre o React Compiler não conseguir
  // memoizar com segurança a função retornada por watch().
  const isFiscal = useWatch({ control: form.control, name: "is_fiscal" });
  const isAdmin = useWatch({ control: form.control, name: "is_admin" });

  const mutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const res = await fetch("/api/admin/usuarios/local", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível criar a conta.");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("Conta local criada.");
      setOpen(false);
      form.reset();
      router.refresh();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline">Nova conta local</Button>} />
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Criar conta de login local</DialogTitle>
        </DialogHeader>

        <form className="space-y-4" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
          <div className="space-y-2">
            <Label htmlFor="nome">Nome</Label>
            <Input id="nome" {...form.register("nome")} />
            {form.formState.errors.nome && (
              <p className="text-destructive text-sm">{form.formState.errors.nome.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="email">E-mail</Label>
            <Input id="email" type="email" {...form.register("email")} />
            {form.formState.errors.email && (
              <p className="text-destructive text-sm">{form.formState.errors.email.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="senha_temporaria">Senha temporária</Label>
            <Input id="senha_temporaria" type="text" {...form.register("senha_temporaria")} />
            {form.formState.errors.senha_temporaria && (
              <p className="text-destructive text-sm">{form.formState.errors.senha_temporaria.message}</p>
            )}
            <p className="text-muted-foreground text-xs">
              O usuário precisará trocá-la no primeiro login.
            </p>
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="is_fiscal">É fiscal de contratos</Label>
            <Switch
              id="is_fiscal"
              checked={isFiscal}
              onCheckedChange={(checked) => form.setValue("is_fiscal", checked)}
            />
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="is_admin">É administrador</Label>
            <Switch
              id="is_admin"
              checked={isAdmin}
              onCheckedChange={(checked) => form.setValue("is_admin", checked)}
            />
          </div>

          <DialogFooter>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Criando..." : "Criar conta"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
