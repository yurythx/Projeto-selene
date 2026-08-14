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
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Contrato } from "@/lib/api/client";

const schema = z.object({
  contrato_id: z.string().min(1, "Selecione um contrato"),
  mes_referencia: z
    .string()
    .regex(/^(0[1-9]|1[0-2])\/\d{4}$/, "Formato MM/AAAA, ex: 07/2026"),
});

type FormValues = z.infer<typeof schema>;

export function NovoProcessoDialog({
  open,
  onOpenChange,
  contratosAtivos,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  contratosAtivos: Contrato[];
}) {
  const router = useRouter();
  const [contratoSelecionado, setContratoSelecionado] = useState("");

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { contrato_id: "", mes_referencia: "" },
  });

  const mutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const res = await fetch("/api/processos", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível abrir o processo.");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("Processo aberto na Etapa 1.");
      onOpenChange(false);
      form.reset();
      setContratoSelecionado("");
      router.refresh();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Novo processo de pagamento</DialogTitle>
        </DialogHeader>

        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <div className="space-y-2">
            <Label htmlFor="contrato_id">Contrato</Label>
            <Select
              value={contratoSelecionado}
              onValueChange={(value) => {
                setContratoSelecionado(value ?? "");
                form.setValue("contrato_id", value ?? "");
              }}
            >
              <SelectTrigger id="contrato_id" className="w-full">
                <SelectValue placeholder="Selecione um contrato ativo" />
              </SelectTrigger>
              <SelectContent>
                {contratosAtivos.map((contrato) => (
                  <SelectItem key={contrato.ID} value={contrato.ID!}>
                    {contrato.NumeroContrato} — {contrato.ContratadaNome}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {form.formState.errors.contrato_id && (
              <p className="text-destructive text-sm">
                {form.formState.errors.contrato_id.message}
              </p>
            )}
            {contratosAtivos.length === 0 && (
              <p className="text-muted-foreground text-sm">
                Nenhum contrato ativo cadastrado ainda.
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="mes_referencia">Mês de referência</Label>
            <Input
              id="mes_referencia"
              placeholder="07/2026"
              {...form.register("mes_referencia")}
            />
            {form.formState.errors.mes_referencia && (
              <p className="text-destructive text-sm">
                {form.formState.errors.mes_referencia.message}
              </p>
            )}
          </div>

          <DialogFooter>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Abrindo..." : "Abrir processo"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
