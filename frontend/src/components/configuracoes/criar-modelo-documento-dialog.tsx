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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { GATILHO_LABEL, GATILHOS } from "@/lib/modelos-documento";

const SEM_GATILHO = "NENHUM";

// items do Select de gatilho — ver o comentário em contratos-filtro.tsx
// sobre por quê (sem isso, <SelectValue> mostra o value cru depois de
// escolher, não o rótulo).
const ITEMS_GATILHO = {
  [SEM_GATILHO]: "Nenhum (só biblioteca de referência)",
  ...GATILHO_LABEL,
};

/**
 * Cadastra uma categoria nova de modelo de documento — não usa
 * react-hook-form (diferente da maioria dos dialogs deste projeto)
 * porque o campo de arquivo precisa virar FormData de qualquer jeito
 * (multipart), então os demais campos (categoria/gatilho) são geridos
 * como estado simples também, mesmo espírito de
 * kanban/processo-dialog.tsx (upload de documento do checklist).
 */
export function CriarModeloDocumentoDialog() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [categoria, setCategoria] = useState("");
  const [gatilho, setGatilho] = useState(SEM_GATILHO);
  const [arquivo, setArquivo] = useState<File | null>(null);
  const inputArquivoRef = useRef<HTMLInputElement>(null);

  const limpar = () => {
    setCategoria("");
    setGatilho(SEM_GATILHO);
    setArquivo(null);
    if (inputArquivoRef.current) inputArquivoRef.current.value = "";
  };

  const mutation = useMutation({
    mutationFn: async () => {
      if (!arquivo) throw new Error("Selecione um arquivo .docx.");
      const formData = new FormData();
      formData.append("categoria", categoria);
      if (gatilho !== SEM_GATILHO) formData.append("gatilho", gatilho);
      formData.append("arquivo", arquivo);

      const res = await fetch("/api/configuracoes/modelos-documento", {
        method: "POST",
        body: formData,
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível cadastrar o modelo.");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("Modelo cadastrado.");
      setOpen(false);
      limpar();
      router.refresh();
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button>Novo modelo</Button>} />
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Novo modelo de documento</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="categoria">Categoria</Label>
            <Input
              id="categoria"
              placeholder="Ofício, Relatório Quadrimestral, OF..."
              value={categoria}
              onChange={(e) => setCategoria(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="gatilho">Gatilho de geração (opcional)</Label>
            <Select items={ITEMS_GATILHO} value={gatilho} onValueChange={(v) => setGatilho(v ?? SEM_GATILHO)}>
              <SelectTrigger id="gatilho" className="w-full">
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
            <p className="text-muted-foreground text-xs">
              Associar um gatilho faz esse fluxo de geração preencher este modelo de
              verdade em vez do PDF padrão.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="arquivo">Arquivo (.docx)</Label>
            <Input
              id="arquivo"
              ref={inputArquivoRef}
              type="file"
              accept=".docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
              onChange={(e) => setArquivo(e.target.files?.[0] ?? null)}
            />
            <p className="text-muted-foreground text-xs">
              Use campos como {"{numero_contrato}"}, {"{fiscal_nome}"}, {"{data_emissao}"} no
              texto do Word — são substituídos na hora de gerar.
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            onClick={() => mutation.mutate()}
            disabled={!categoria || !arquivo || mutation.isPending}
          >
            {mutation.isPending ? "Enviando..." : "Cadastrar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
