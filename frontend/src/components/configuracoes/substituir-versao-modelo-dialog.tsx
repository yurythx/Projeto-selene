"use client";

import { useRef, useState } from "react";
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

/** Publica uma nova versão do arquivo, substituindo a ativa — o histórico anterior é preservado. */
export function SubstituirVersaoModeloDialog({ modeloId }: { modeloId: string }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [arquivo, setArquivo] = useState<File | null>(null);
  const inputArquivoRef = useRef<HTMLInputElement>(null);

  const mutation = useMutation({
    mutationFn: async () => {
      if (!arquivo) throw new Error("Selecione um arquivo .docx.");
      const formData = new FormData();
      formData.append("arquivo", arquivo);

      const res = await fetch(`/api/configuracoes/modelos-documento/${modeloId}/versoes`, {
        method: "POST",
        body: formData,
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível publicar a nova versão.");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("Nova versão publicada.");
      setOpen(false);
      setArquivo(null);
      if (inputArquivoRef.current) inputArquivoRef.current.value = "";
      router.refresh();
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" size="sm">Substituir arquivo</Button>} />
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Publicar nova versão</DialogTitle>
        </DialogHeader>

        <div className="space-y-2">
          <Label htmlFor="arquivo-nova-versao">Arquivo (.docx)</Label>
          <Input
            id="arquivo-nova-versao"
            ref={inputArquivoRef}
            type="file"
            accept=".docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
            onChange={(e) => setArquivo(e.target.files?.[0] ?? null)}
          />
          <p className="text-muted-foreground text-xs">
            A versão atual continua disponível no histórico, não é apagada.
          </p>
        </div>

        <DialogFooter>
          <Button onClick={() => mutation.mutate()} disabled={!arquivo || mutation.isPending}>
            {mutation.isPending ? "Enviando..." : "Publicar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
