"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
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
import { abrirPDFDeResposta } from "@/lib/abrir-pdf";

// Motivos comuns pré-preenchidos ("motivo selecionado de uma lista", como
// pedido no roadmap) — "OUTRO" revela um campo de texto livre. Não vem do
// backend: é uma lista de conveniência da UI, o backend aceita qualquer
// texto não-vazio como motivo.
const MOTIVOS_COMUNS = [
  "Atraso na entrega/execução do objeto contratado",
  "Descumprimento de especificação técnica",
  "Não comparecimento à vistoria agendada",
  "Documentação obrigatória não apresentada no prazo",
  "OUTRO",
];

export function GerarNotificacaoDialog({ contratoId }: { contratoId: string }) {
  const [open, setOpen] = useState(false);
  const [motivoSelecionado, setMotivoSelecionado] = useState("");
  const [motivoLivre, setMotivoLivre] = useState("");

  const mutation = useMutation({
    mutationFn: async (motivo: string) => {
      const res = await fetch(`/api/contratos/${contratoId}/notificacao`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ motivo }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível gerar a notificação.");
      }
      return res;
    },
    onSuccess: async (res) => {
      await abrirPDFDeResposta(res);
      toast.success("Notificação gerada.");
      setOpen(false);
      setMotivoSelecionado("");
      setMotivoLivre("");
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const motivoFinal = motivoSelecionado === "OUTRO" ? motivoLivre.trim() : motivoSelecionado;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline">Notificação de Descumprimento</Button>} />
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Gerar Notificação de Descumprimento</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="motivo">Motivo</Label>
            <Select value={motivoSelecionado} onValueChange={(value) => setMotivoSelecionado(value ?? "")}>
              <SelectTrigger id="motivo" className="w-full">
                <SelectValue placeholder="Selecione o motivo" />
              </SelectTrigger>
              <SelectContent>
                {MOTIVOS_COMUNS.map((motivo) => (
                  <SelectItem key={motivo} value={motivo}>
                    {motivo === "OUTRO" ? "Outro (especifique)" : motivo}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {motivoSelecionado === "OUTRO" && (
            <div className="space-y-2">
              <Label htmlFor="motivo_livre">Descreva o motivo</Label>
              <Textarea
                id="motivo_livre"
                value={motivoLivre}
                onChange={(e) => setMotivoLivre(e.target.value)}
                rows={4}
              />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            onClick={() => mutation.mutate(motivoFinal)}
            disabled={!motivoFinal || mutation.isPending}
          >
            {mutation.isPending ? "Gerando..." : "Gerar PDF"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
