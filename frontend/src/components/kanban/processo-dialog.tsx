"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { TipoDocumento, ProcessoPagamento, DocumentoAnexo } from "@/lib/api/client";
import { TriangleAlertIcon } from "lucide-react";

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw Object.assign(new Error(body.error ?? "Erro na requisição."), { body, status: res.status });
  }
  return body as T;
}

export function ProcessoDialog({
  processo,
  tiposDocumento,
  isFiscal,
  open,
  onOpenChange,
}: {
  processo: ProcessoPagamento;
  tiposDocumento: TipoDocumento[];
  isFiscal: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [tipoSelecionado, setTipoSelecionado] = useState<string>("");
  const [pendentes, setPendentes] = useState<string[] | null>(null);

  const documentosQuery = useQuery({
    queryKey: ["documentos", processo.ID],
    queryFn: () => fetchJSON<DocumentoAnexo[]>(`/api/processos/${processo.ID}/documentos`),
    enabled: open,
  });

  const uploadMutation = useMutation({
    mutationFn: async (formData: FormData) =>
      fetchJSON<DocumentoAnexo>(`/api/processos/${processo.ID}/documentos`, {
        method: "POST",
        body: formData,
      }),
    onSuccess: () => {
      toast.success("Documento anexado.");
      queryClient.invalidateQueries({ queryKey: ["documentos", processo.ID] });
      setPendentes(null);
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  const avancarMutation = useMutation({
    mutationFn: () => fetchJSON(`/api/processos/${processo.ID}/avancar`, { method: "POST" }),
    onSuccess: () => {
      toast.success("Processo avançou de etapa.");
      onOpenChange(false);
      router.refresh();
    },
    onError: (erro: Error & { body?: { documentos_pendentes?: string[] } }) => {
      if (erro.body?.documentos_pendentes) {
        setPendentes(erro.body.documentos_pendentes);
      } else {
        toast.error(erro.message);
      }
    },
  });

  const concluirMutation = useMutation({
    mutationFn: () => fetchJSON(`/api/processos/${processo.ID}/concluir`, { method: "POST" }),
    onSuccess: () => {
      toast.success("Processo marcado como pago.");
      onOpenChange(false);
      router.refresh();
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  function handleUploadSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const arquivoInput = form.elements.namedItem("arquivo") as HTMLInputElement;
    const arquivo = arquivoInput.files?.[0];
    if (!tipoSelecionado || !arquivo) {
      toast.error("Selecione o tipo de documento e o arquivo.");
      return;
    }
    const formData = new FormData();
    formData.append("tipo_documento_id", tipoSelecionado);
    formData.append("arquivo", arquivo);
    uploadMutation.mutate(formData);
    form.reset();
  }

  const podeConcluir = processo.EtapaAtualID === 6 && processo.Status !== "Concluido";
  const podeAvancar = processo.Status !== "Concluido" && processo.EtapaAtualID !== 6;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>
            {processo.Contrato?.NumeroContrato} — {processo.MesReferencia}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-1 text-sm">
          <p>
            <span className="text-muted-foreground">Contratada: </span>
            {processo.Contrato?.ContratadaNome}
          </p>
          <p>
            <span className="text-muted-foreground">Etapa atual: </span>
            {processo.EtapaAtual?.Nome}
          </p>
          <p>
            <span className="text-muted-foreground">Situação: </span>
            <Badge variant={processo.Status === "Concluido" ? "default" : "secondary"}>
              {processo.Status === "Concluido" ? "Pago" : "Em andamento"}
            </Badge>
          </p>
        </div>

        {pendentes && pendentes.length > 0 && (
          <Alert variant="destructive">
            <TriangleAlertIcon />
            <AlertTitle>Checklist incompleto</AlertTitle>
            <AlertDescription>
              <p>Documentos pendentes pra avançar desta etapa:</p>
              <ul className="list-disc pl-4">
                {pendentes.map((nome) => (
                  <li key={nome}>{nome}</li>
                ))}
              </ul>
            </AlertDescription>
          </Alert>
        )}

        <div className="space-y-2">
          <h3 className="text-sm font-medium">Documentos anexados</h3>
          {documentosQuery.isLoading && (
            <p className="text-muted-foreground text-sm">Carregando...</p>
          )}
          {documentosQuery.data && documentosQuery.data.length === 0 && (
            <p className="text-muted-foreground text-sm">Nenhum documento anexado ainda.</p>
          )}
          <ul className="space-y-1">
            {documentosQuery.data?.map((doc) => (
              <li key={doc.ID} className="flex items-center justify-between text-sm">
                <span>{doc.TipoDocumento?.Nome}</span>
                <span className="text-muted-foreground text-xs">{doc.NomeArquivo}</span>
              </li>
            ))}
          </ul>
        </div>

        {isFiscal && (
          <form onSubmit={handleUploadSubmit} className="flex items-end gap-2">
            <div className="flex-1 space-y-1">
              <Label htmlFor="tipo_documento_id">Tipo de documento</Label>
              <Select value={tipoSelecionado} onValueChange={(value) => setTipoSelecionado(value ?? "")}>
                <SelectTrigger id="tipo_documento_id" className="w-full">
                  <SelectValue placeholder="Selecione" />
                </SelectTrigger>
                <SelectContent>
                  {tiposDocumento.map((tipo) => (
                    <SelectItem key={tipo.ID} value={String(tipo.ID)}>
                      {tipo.Nome}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label htmlFor="arquivo">Arquivo</Label>
              <input
                id="arquivo"
                name="arquivo"
                type="file"
                className="text-sm"
                accept="application/pdf,image/*"
              />
            </div>
            <Button type="submit" variant="secondary" disabled={uploadMutation.isPending}>
              {uploadMutation.isPending ? "Enviando..." : "Anexar"}
            </Button>
          </form>
        )}

        <DialogFooter className="items-center sm:justify-between">
          <a
            href={`/api/processos/${processo.ID}/relatorio`}
            target="_blank"
            rel="noreferrer"
            className="text-sm underline underline-offset-2"
          >
            Baixar relatório (PDF)
          </a>
          {isFiscal && (
            <div className="flex gap-2">
              {podeAvancar && (
                <Button onClick={() => avancarMutation.mutate()} disabled={avancarMutation.isPending}>
                  {avancarMutation.isPending ? "Avançando..." : "Avançar etapa"}
                </Button>
              )}
              {podeConcluir && (
                <Button
                  variant="default"
                  onClick={() => concluirMutation.mutate()}
                  disabled={concluirMutation.isPending}
                >
                  {concluirMutation.isPending ? "Concluindo..." : "Marcar como pago"}
                </Button>
              )}
            </div>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
