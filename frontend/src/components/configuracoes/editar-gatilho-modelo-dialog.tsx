"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
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
import { GATILHO_LABEL, GATILHOS } from "@/lib/modelos-documento";
import type { ModeloDocumento } from "@/lib/api/client";

const SEM_GATILHO = "NENHUM";

/** Renomeia a categoria e/ou troca o gatilho de geração associado — não mexe no arquivo. */
export function EditarGatilhoModeloDialog({ modelo }: { modelo: ModeloDocumento }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [categoria, setCategoria] = useState(modelo.Categoria ?? "");
  const [gatilho, setGatilho] = useState(modelo.Gatilho ?? SEM_GATILHO);

  const mutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(`/api/configuracoes/modelos-documento/${modelo.ID}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ categoria, gatilho }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível salvar as alterações.");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("Categoria atualizada.");
      setOpen(false);
      router.refresh();
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline">Editar</Button>} />
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Editar categoria</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="editar-categoria">Categoria</Label>
            <Input id="editar-categoria" value={categoria} onChange={(e) => setCategoria(e.target.value)} />
          </div>

          <div className="space-y-2">
            <Label htmlFor="editar-gatilho">Gatilho de geração</Label>
            <Select value={gatilho} onValueChange={(v) => setGatilho(v ?? SEM_GATILHO)}>
              <SelectTrigger id="editar-gatilho" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={SEM_GATILHO}>Nenhum (só biblioteca de referência)</SelectItem>
                {GATILHOS.map((g) => (
                  <SelectItem key={g} value={g}>
                    {GATILHO_LABEL[g]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button onClick={() => mutation.mutate()} disabled={!categoria || mutation.isPending}>
            {mutation.isPending ? "Salvando..." : "Salvar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
