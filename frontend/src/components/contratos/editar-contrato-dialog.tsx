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
import type { Contrato } from "@/lib/api/client";

// Campos alinhados com AtualizarContratoRequest do backend —
// numero_contrato, fiscal_id e tipo_objeto não são editáveis aqui de
// propósito (identidade/classificação do contrato, ver comentário no
// backend em ContratoService.Atualizar).
const schema = z.object({
  portaria_nomeacao: z.string().optional(),
  contratada_nome: z.string().min(1, "Obrigatório"),
  contratada_cnpj: z.string().min(1, "Obrigatório"),
  contratada_email: z.string().email("E-mail inválido").optional().or(z.literal("")),
  // Opcional — alimenta o Radar de Alertas (Fase 1 do roadmap). Campo
  // vazio limpa a vigência cadastrada (ver AtualizarContratoInput no
  // backend: ponteiro-pra-string distingue "não enviado" de "enviado
  // vazio", e este form sempre envia o valor atual do input).
  data_vigencia_fim: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

export function EditarContratoDialog({ contrato }: { contrato: Contrato }) {
  const [open, setOpen] = useState(false);
  const router = useRouter();

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      portaria_nomeacao: contrato.PortariaNomeacao ?? "",
      contratada_nome: contrato.ContratadaNome ?? "",
      contratada_cnpj: contrato.ContratadaCNPJ ?? "",
      contratada_email: contrato.ContratadaEmail ?? "",
      data_vigencia_fim: contrato.DataVigenciaFim ?? "",
    },
  });

  const mutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const res = await fetch(`/api/contratos/${contrato.ID}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível salvar as alterações.");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("Contrato atualizado.");
      setOpen(false);
      router.refresh();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline">Editar</Button>} />
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Editar contrato</DialogTitle>
        </DialogHeader>

        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
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

          <div className="space-y-2">
            <Label htmlFor="portaria_nomeacao">Portaria de nomeação</Label>
            <Input id="portaria_nomeacao" {...form.register("portaria_nomeacao")} />
          </div>

          <div className="space-y-2">
            <Label htmlFor="data_vigencia_fim">Fim da vigência (opcional)</Label>
            <Input id="data_vigencia_fim" type="date" {...form.register("data_vigencia_fim")} />
            <p className="text-muted-foreground text-xs">
              Alimenta o Radar de Alertas de prazos legais. Deixe em branco pra limpar.
            </p>
          </div>

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
