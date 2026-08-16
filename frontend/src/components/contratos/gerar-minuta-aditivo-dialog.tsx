"use client";

import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
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
import { abrirOuBaixarDocumento } from "@/lib/abrir-documento";

const schema = z.object({
  tipo_aditivo: z.enum(["VALOR", "PRAZO", "VALOR_E_PRAZO"]),
  justificativa: z.string().min(1, "Obrigatório"),
  novo_valor: z.string().optional(),
  novo_prazo: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

// Dialog do Gerador de Documentos Legais — questionário curto
// (tipo_aditivo + justificativa técnica, valor/prazo novo conforme o
// tipo) que gera a Minuta de Aditivo (PDF) do contrato.
export function GerarMinutaAditivoDialog({ contratoId }: { contratoId: string }) {
  const [open, setOpen] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      tipo_aditivo: "VALOR",
      justificativa: "",
      novo_valor: "",
      novo_prazo: "",
    },
  });

  // useWatch (não form.watch()) — o React Compiler não consegue memoizar
  // com segurança a função retornada por watch(), o que gera um warning de
  // lint (react-hooks/incompatible-library); useWatch é a alternativa
  // recomendada pelo react-hook-form pra esse mesmo caso de uso.
  const tipoAditivo = useWatch({ control: form.control, name: "tipo_aditivo" });
  const pedeValor = tipoAditivo === "VALOR" || tipoAditivo === "VALOR_E_PRAZO";
  const pedePrazo = tipoAditivo === "PRAZO" || tipoAditivo === "VALOR_E_PRAZO";

  const mutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const res = await fetch(`/api/contratos/${contratoId}/minuta-aditivo`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível gerar a minuta de aditivo.");
      }
      return res;
    },
    onSuccess: async (res) => {
      await abrirOuBaixarDocumento(res);
      toast.success("Minuta de aditivo gerada.");
      setOpen(false);
      form.reset();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline">Minuta de Aditivo</Button>} />
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Gerar Minuta de Aditivo</DialogTitle>
        </DialogHeader>

        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <div className="space-y-2">
            <Label htmlFor="tipo_aditivo">Tipo de aditivo</Label>
            <Select
              defaultValue={form.getValues("tipo_aditivo")}
              onValueChange={(value) =>
                form.setValue("tipo_aditivo", value as FormValues["tipo_aditivo"])
              }
            >
              <SelectTrigger id="tipo_aditivo" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="VALOR">Valor</SelectItem>
                <SelectItem value="PRAZO">Prazo</SelectItem>
                <SelectItem value="VALOR_E_PRAZO">Valor e Prazo</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {(pedeValor || pedePrazo) && (
            <div className="grid grid-cols-2 gap-4">
              {pedeValor && (
                <div className="space-y-2">
                  <Label htmlFor="novo_valor">Novo valor proposto</Label>
                  <Input id="novo_valor" placeholder="R$ 150.000,00" {...form.register("novo_valor")} />
                </div>
              )}
              {pedePrazo && (
                <div className="space-y-2">
                  <Label htmlFor="novo_prazo">Novo prazo proposto</Label>
                  <Input id="novo_prazo" placeholder="31/12/2026" {...form.register("novo_prazo")} />
                </div>
              )}
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="justificativa">Justificativa técnica</Label>
            <Textarea id="justificativa" rows={5} {...form.register("justificativa")} />
            {form.formState.errors.justificativa && (
              <p className="text-destructive text-sm">
                {form.formState.errors.justificativa.message}
              </p>
            )}
          </div>

          <DialogFooter>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Gerando..." : "Gerar PDF"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
