"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
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

// fiscal_id NÃO é um campo do formulário — o Route Handler
// (app/api/contratos/route.ts) preenche a partir do usuário logado. Ver o
// comentário lá para o porquê (não há hoje como um não-admin listar outros
// fiscais).
const schema = z.object({
  numero_contrato: z.string().min(1, "Obrigatório"),
  data_assinatura: z.string().min(1, "Obrigatório"),
  contratada_nome: z.string().min(1, "Obrigatório"),
  contratada_cnpj: z.string().min(1, "Obrigatório"),
  contratada_email: z.string().email("E-mail inválido").optional().or(z.literal("")),
  portaria_nomeacao: z.string().optional(),
  tipo_objeto: z.enum(["CONSUMO", "PERMANENTE", "SERVICO"]),
  // Opcional — alimenta o Radar de Alertas (Fase 1 do roadmap). Sem essa
  // data, o contrato simplesmente não aparece no radar de vigência.
  data_vigencia_fim: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

// Dialog de cadastro de contrato — fiscalNome é só exibição (o
// fiscal_id de verdade é sempre resolvido server-side a partir da sessão
// pela rota BFF, nunca enviado pelo client, ver route.ts).
export function NovoContratoDialog({ fiscalNome }: { fiscalNome: string }) {
  const [open, setOpen] = useState(false);
  const router = useRouter();

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      numero_contrato: "",
      data_assinatura: "",
      contratada_nome: "",
      contratada_cnpj: "",
      contratada_email: "",
      portaria_nomeacao: "",
      tipo_objeto: "SERVICO",
      data_vigencia_fim: "",
    },
  });

  const mutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const res = await fetch("/api/contratos", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível criar o contrato.");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("Contrato criado com sucesso.");
      setOpen(false);
      form.reset();
      router.refresh();
    },
    onError: (error: Error) => {
      toast.error(error.message);
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button>Novo contrato</Button>} />
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Novo contrato</DialogTitle>
        </DialogHeader>

        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="numero_contrato">Número do contrato</Label>
              <Input
                id="numero_contrato"
                placeholder="125/2026"
                {...form.register("numero_contrato")}
              />
              {form.formState.errors.numero_contrato && (
                <p className="text-destructive text-sm">
                  {form.formState.errors.numero_contrato.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="data_assinatura">Data de assinatura</Label>
              <Input id="data_assinatura" type="date" {...form.register("data_assinatura")} />
              {form.formState.errors.data_assinatura && (
                <p className="text-destructive text-sm">
                  {form.formState.errors.data_assinatura.message}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="data_vigencia_fim">Fim da vigência (opcional)</Label>
            <Input id="data_vigencia_fim" type="date" {...form.register("data_vigencia_fim")} />
            <p className="text-muted-foreground text-xs">
              Alimenta o Radar de Alertas de prazos legais.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="contratada_nome">Empresa contratada</Label>
            <Input id="contratada_nome" {...form.register("contratada_nome")} />
            {form.formState.errors.contratada_nome && (
              <p className="text-destructive text-sm">
                {form.formState.errors.contratada_nome.message}
              </p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="contratada_cnpj">CNPJ</Label>
              <Input id="contratada_cnpj" {...form.register("contratada_cnpj")} />
              {form.formState.errors.contratada_cnpj && (
                <p className="text-destructive text-sm">
                  {form.formState.errors.contratada_cnpj.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="contratada_email">E-mail da contratada</Label>
              <Input id="contratada_email" type="email" {...form.register("contratada_email")} />
              {form.formState.errors.contratada_email && (
                <p className="text-destructive text-sm">
                  {form.formState.errors.contratada_email.message}
                </p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="tipo_objeto">Tipo de objeto</Label>
              <Select
                defaultValue={form.getValues("tipo_objeto")}
                onValueChange={(value) =>
                  form.setValue("tipo_objeto", value as FormValues["tipo_objeto"])
                }
              >
                <SelectTrigger id="tipo_objeto" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="CONSUMO">Consumo</SelectItem>
                  <SelectItem value="PERMANENTE">Permanente</SelectItem>
                  <SelectItem value="SERVICO">Serviço</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="portaria_nomeacao">Portaria de nomeação</Label>
              <Input id="portaria_nomeacao" {...form.register("portaria_nomeacao")} />
            </div>
          </div>

          <p className="text-muted-foreground text-sm">
            Fiscal responsável: <span className="font-medium">{fiscalNome}</span> (você)
          </p>

          <DialogFooter>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Salvando..." : "Salvar"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
