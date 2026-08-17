"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  ArrowLeftIcon,
  TriangleAlertIcon,
  CheckCircle2Icon,
  XCircleIcon,
  FileTextIcon,
  UploadIcon,
  Trash2Icon,
} from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import type {
  TipoDocumento,
  ProcessoComFiscalizacao,
  DocumentoAnexo,
  ItemRadar,
  AllowedAction,
} from "@/lib/api/client";
import { RadarNivelBadge } from "@/components/radar/radar-badge";
import { abrirOuBaixarDocumento } from "@/lib/abrir-documento";
import { tiposDocumentoAplicaveis } from "@/lib/tipos-documento";
import { montarChecklist } from "@/lib/checklist";
import { cn } from "@/lib/utils";
import { VistoriasDialog } from "./vistorias-dialog";
import { OcorrenciasDialog } from "./ocorrencias-dialog";

const ESTADO_FISCALIZACAO_LABEL: Record<string, string> = {
  A_EXECUTAR_CONFERIR: "A executar / conferir",
  EM_ANALISE_EXTERNA: "Em análise externa",
  DOCUMENTAR_ATESTAR: "Documentar / atestar",
  PENDENCIA_DEVOLVIDO: "Pendência — ocorrência aberta",
  CONCLUIDO: "Concluído",
};

const TIPO_OBJETO_LABEL: Record<string, string> = {
  CONSUMO: "Consumo",
  PERMANENTE: "Permanente",
  SERVICO: "Serviço",
};

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    // Corpo de erro cru no console — é exatamente o que falta pra
    // diagnosticar um 4xx/5xx de upload sem depender só do toast (que só
    // mostra `body.error`, uma frase; o corpo completo pode ter mais
    // contexto, ex: detalhes de validação).
    console.error(`[Selene] ${init?.method ?? "GET"} ${url} → ${res.status}`, body);
    throw Object.assign(new Error(body.error ?? "Erro na requisição."), { body, status: res.status });
  }
  return body as T;
}

/**
 * Página dedicada do processo (substituiu o drawer/modal em cima do
 * Kanban — pedido explícito: "abra uma página inteira com todos os
 * elementos"). Vive em /kanban/[id]. Mesma funcionalidade de
 * ProcessoDialog (removido), mais o checklist visual (✓/x) que faltava —
 * antes só dava pra saber o que faltava tentando avançar e lendo o erro
 * 422; agora GET /processos/{id} já traz `documentos_requeridos`
 * (FiscalizacaoService.Decorar) e este componente cruza com os
 * documentos já anexados.
 */
export function ProcessoPage({
  processoInicial,
  documentosIniciais,
  tiposDocumento,
  alertasRadar = [],
  isFiscal,
}: {
  processoInicial: ProcessoComFiscalizacao;
  documentosIniciais: DocumentoAnexo[];
  tiposDocumento: TipoDocumento[];
  alertasRadar?: ItemRadar[];
  isFiscal: boolean;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const processoId = processoInicial.ID!;
  const [tipoSelecionado, setTipoSelecionado] = useState<string>("");
  const [vistoriasOpen, setVistoriasOpen] = useState(false);
  const [ocorrenciasOpen, setOcorrenciasOpen] = useState(false);
  const [documentoPreview, setDocumentoPreview] = useState<DocumentoAnexo | null>(null);
  const [documentoParaExcluir, setDocumentoParaExcluir] = useState<DocumentoAnexo | null>(null);

  const processoQuery = useQuery({
    queryKey: ["processo", processoId],
    queryFn: () => fetchJSON<ProcessoComFiscalizacao>(`/api/processos/${processoId}`),
    initialData: processoInicial,
  });
  const processo = processoQuery.data;

  const documentosQuery = useQuery({
    queryKey: ["documentos", processoId],
    queryFn: () => fetchJSON<DocumentoAnexo[]>(`/api/processos/${processoId}/documentos`),
    initialData: documentosIniciais,
  });
  const documentos = documentosQuery.data;

  const checklist = montarChecklist(processo.documentos_requeridos ?? [], documentos);
  const checklistCompleto = checklist.every((item) => item.satisfeito);

  // Só oferece no select os tipos que REALMENTE podem ser inseridos nesta
  // etapa — pedido explícito do usuário. Dois filtros combinados:
  // 1. Tipo de contrato (já existia): documentos restritos a SERVICO
  //    (Planilha de Medição, Boleto DAM) ou a
  //    ExigeFiscalizacaoTerceirizacao (Comprovante de Salário, GFIP,
  //    GRF/GPS, SEFIP) só aparecem quando o contrato deste processo
  //    realmente se qualifica; ver lib/tipos-documento.ts e
  //    service.TipoDocumentoAplicavel no backend.
  // 2. Checklist cumulado até a etapa atual (novo): um tipo só aparece se
  //    ainda estiver PENDENTE no checklist (documentos_requeridos, que já
  //    é cumulativo entre etapas — ver RequisitosAcumulados no backend).
  //    Isso esconde documentos de etapas futuras (ex: uma CND, só exigida
  //    na Etapa 5, enquanto o processo ainda está na Etapa 1) e também os
  //    tipos já satisfeitos (reenviar um tipo já anexado seria rejeitado
  //    com 409 — pra substituir, exclua o anterior primeiro, o que volta
  //    a liberá-lo aqui). O backend (DocumentoService.Upload) é quem faz
  //    valer as duas regras de verdade; este filtro só evita oferecer uma
  //    opção que o upload rejeitaria.
  const nomesPendentes = new Set(checklist.filter((item) => !item.satisfeito).map((item) => item.nome));
  const tiposAplicaveis = tiposDocumentoAplicaveis(
    tiposDocumento,
    processo.Contrato?.TipoObjeto,
    processo.Contrato?.ExigeFiscalizacaoTerceirizacao
  ).filter((tipo) => Boolean(tipo.Nome) && nomesPendentes.has(tipo.Nome!));
  const tipoDocumentoSelecionado = tiposAplicaveis.find((tipo) => String(tipo.ID) === tipoSelecionado);
  // items — sem isso, <SelectValue> só consegue resolver o rótulo
  // enquanto o popup está aberto (o registro interno do base-ui some
  // quando o Portal desmonta ao fechar); sem o mapa persistente, depois
  // de escolher a tela mostra o ID cru ("16") em vez do nome do
  // documento. Achado real reportado em produção, não só teórico — ver
  // o mesmo padrão aplicado nos outros Selects dinâmicos do app.
  const itemsTipoDocumento = Object.fromEntries(tiposAplicaveis.map((tipo) => [String(tipo.ID), tipo.Nome]));

  const uploadMutation = useMutation({
    mutationFn: async (formData: FormData) =>
      fetchJSON<DocumentoAnexo>(`/api/processos/${processoId}/documentos`, {
        method: "POST",
        body: formData,
      }),
    onSuccess: () => {
      toast.success("Documento anexado.");
      queryClient.invalidateQueries({ queryKey: ["documentos", processoId] });
      queryClient.invalidateQueries({ queryKey: ["processo", processoId] });
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  const excluirDocumentoMutation = useMutation({
    mutationFn: (documentoId: string) =>
      fetchJSON<void>(`/api/processos/${processoId}/documentos/${documentoId}`, { method: "DELETE" }),
    onSuccess: (_data, documentoId) => {
      const excluido = documentos.find((d) => d.ID === documentoId);
      toast.success(`"${excluido?.TipoDocumento?.Nome ?? excluido?.NomeArquivo ?? "Documento"}" excluído.`);
      queryClient.invalidateQueries({ queryKey: ["documentos", processoId] });
      queryClient.invalidateQueries({ queryKey: ["processo", processoId] });
    },
    onError: (erro: Error) => toast.error(erro.message || "Não foi possível excluir o documento."),
  });

  const avancarMutation = useMutation({
    mutationFn: () => fetchJSON(`/api/processos/${processoId}/avancar`, { method: "POST" }),
    onSuccess: () => {
      toast.success("Processo avançou de etapa.");
      queryClient.invalidateQueries({ queryKey: ["processo", processoId] });
      router.refresh();
    },
    onError: (erro: Error & { body?: { documentos_pendentes?: string[] } }) => {
      if (erro.body?.documentos_pendentes) {
        toast.error(`Checklist incompleto: falta ${erro.body.documentos_pendentes.join(", ")}.`);
      } else {
        toast.error(erro.message);
      }
    },
  });

  const concluirMutation = useMutation({
    mutationFn: () => fetchJSON(`/api/processos/${processoId}/concluir`, { method: "POST" }),
    onSuccess: () => {
      toast.success("Processo marcado como pago.");
      queryClient.invalidateQueries({ queryKey: ["processo", processoId] });
      router.refresh();
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  const atestoMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(`/api/processos/${processoId}/atesto`, { method: "POST" });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        console.error(`[Selene] POST atesto → ${res.status}`, body);
        throw new Error(body.error ?? "Não foi possível gerar o atesto.");
      }
      return res;
    },
    onSuccess: async (res) => {
      await abrirOuBaixarDocumento(res);
      toast.success("Atesto gerado.");
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
    if (tipoDocumentoSelecionado?.ExigeValidade) {
      const dataValidadeInput = form.elements.namedItem("data_validade") as HTMLInputElement | null;
      if (dataValidadeInput?.value) {
        formData.append("data_validade", dataValidadeInput.value);
      }
    }
    uploadMutation.mutate(formData);
    form.reset();
    setTipoSelecionado("");
  }

  const allowedActions: AllowedAction[] = processo.allowed_actions ?? [];
  const podeConcluir = allowedActions.includes("CONCLUIR_PAGAMENTO");
  const podeAvancar = allowedActions.includes("AVANCAR_ETAPA");

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      <Link
        href="/kanban"
        className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 text-sm"
      >
        <ArrowLeftIcon className="size-4" />
        Voltar ao Kanban
      </Link>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-1.5">
          <h1 className="text-2xl font-semibold">
            {processo.Contrato?.NumeroContrato} — {processo.MesReferencia}
          </h1>
          <p className="text-muted-foreground">{processo.Contrato?.ContratadaNome}</p>
          <div className="flex flex-wrap items-center gap-2 pt-1">
            <Badge variant="outline">{processo.EtapaAtual?.Nome}</Badge>
            <Badge variant={processo.Status === "Concluido" ? "success" : "secondary"}>
              {processo.Status === "Concluido" ? "Pago" : "Em andamento"}
            </Badge>
            {processo.estado_fiscalizacao && (
              <Badge variant={processo.estado_fiscalizacao === "PENDENCIA_DEVOLVIDO" ? "destructive" : "outline"}>
                {ESTADO_FISCALIZACAO_LABEL[processo.estado_fiscalizacao] ?? processo.estado_fiscalizacao}
              </Badge>
            )}
          </div>
        </div>

        {isFiscal && (
          <div className="flex flex-wrap gap-2">
            {podeAvancar && (
              <Button onClick={() => avancarMutation.mutate()} disabled={avancarMutation.isPending}>
                {avancarMutation.isPending ? "Avançando..." : "Avançar etapa"}
              </Button>
            )}
            {podeConcluir && (
              <Button onClick={() => concluirMutation.mutate()} disabled={concluirMutation.isPending}>
                {concluirMutation.isPending ? "Concluindo..." : "Marcar como pago"}
              </Button>
            )}
          </div>
        )}
      </div>

      {alertasRadar.length > 0 && (
        <div className="space-y-2">
          {alertasRadar.map((alerta, i) => (
            <div
              key={i}
              className="border-l-destructive bg-muted/30 flex items-center justify-between gap-2 rounded-md border-l-4 p-3 text-sm"
            >
              <span>{alerta.mensagem}</span>
              {alerta.nivel && <RadarNivelBadge nivel={alerta.nivel} />}
            </div>
          ))}
        </div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <Card className="shadow-sm">
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-base">Checklist de documentos</CardTitle>
              {checklist.length > 0 && (
                <Badge variant={checklistCompleto ? "success" : "warning"}>
                  {checklist.filter((i) => i.satisfeito).length}/{checklist.length}
                </Badge>
              )}
            </CardHeader>
            <CardContent>
              {checklist.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                  Esta etapa não exige nenhum documento pra avançar.
                </p>
              ) : (
                <ul className="divide-border divide-y" data-testid="checklist-documentos">
                  {checklist.map((item) => (
                    <li
                      key={item.nome}
                      data-satisfeito={item.satisfeito}
                      className="flex items-center gap-2.5 py-2 text-sm"
                    >
                      {item.satisfeito ? (
                        <CheckCircle2Icon className="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
                      ) : (
                        <XCircleIcon className="text-destructive size-4 shrink-0" />
                      )}
                      <span className={cn(!item.satisfeito && "text-muted-foreground")}>{item.nome}</span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>

          <Card className="shadow-sm">
            <CardHeader>
              <CardTitle className="text-base">Documentos anexados</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {documentos.length === 0 ? (
                <p className="text-muted-foreground text-sm">Nenhum documento anexado ainda.</p>
              ) : (
                <ul className="divide-border divide-y" data-testid="documentos-anexados">
                  {documentos.map((doc) => (
                    <li key={doc.ID} className="flex items-center justify-between gap-2 py-2 text-sm">
                      <button
                        type="button"
                        onClick={() => setDocumentoPreview(doc)}
                        aria-label={`Visualizar ${doc.TipoDocumento?.Nome ?? doc.NomeArquivo}`}
                        className="group flex min-w-0 items-center gap-2 text-left"
                      >
                        <FileTextIcon className="text-muted-foreground size-4 shrink-0" />
                        <span className="truncate underline-offset-2 group-hover:text-primary group-hover:underline">
                          {doc.TipoDocumento?.Nome}
                        </span>
                      </button>
                      <span className="flex shrink-0 items-center gap-1">
                        <span className="text-muted-foreground mr-1 truncate text-xs">{doc.NomeArquivo}</span>
                        {isFiscal && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon-sm"
                            aria-label={`Excluir ${doc.NomeArquivo}`}
                            onClick={() => setDocumentoParaExcluir(doc)}
                            disabled={excluirDocumentoMutation.isPending}
                          >
                            <Trash2Icon className="text-destructive size-4" />
                          </Button>
                        )}
                      </span>
                    </li>
                  ))}
                </ul>
              )}

              {isFiscal && tiposAplicaveis.length === 0 && (
                <p className="text-muted-foreground border-t pt-4 text-sm">
                  Nenhum documento pendente para anexar nesta etapa.
                </p>
              )}

              {isFiscal && tiposAplicaveis.length > 0 && (
                <form onSubmit={handleUploadSubmit} className="border-t pt-4">
                  <div className="flex flex-wrap items-end gap-2">
                    <div className="min-w-[200px] flex-1 space-y-1">
                      <Label htmlFor="tipo_documento_id">Tipo de documento</Label>
                      <Select
                        items={itemsTipoDocumento}
                        value={tipoSelecionado}
                        onValueChange={(value) => setTipoSelecionado(value ?? "")}
                      >
                        <SelectTrigger id="tipo_documento_id" className="w-full">
                          <SelectValue placeholder="Selecione" />
                        </SelectTrigger>
                        <SelectContent>
                          {tiposAplicaveis.map((tipo) => (
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
                    {tipoDocumentoSelecionado?.ExigeValidade && (
                      <div className="space-y-1">
                        <Label htmlFor="data_validade">Validade</Label>
                        <input
                          id="data_validade"
                          name="data_validade"
                          type="date"
                          className="border-input h-9 rounded-md border bg-transparent px-2 text-sm"
                        />
                      </div>
                    )}
                    <Button type="submit" variant="secondary" disabled={uploadMutation.isPending}>
                      <UploadIcon className="size-4" />
                      {uploadMutation.isPending ? "Enviando..." : "Anexar"}
                    </Button>
                  </div>
                </form>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card className="shadow-sm">
            <CardHeader>
              <CardTitle className="text-base">Informações</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <p>
                <span className="text-muted-foreground">Tipo: </span>
                {processo.Contrato?.TipoObjeto ? TIPO_OBJETO_LABEL[processo.Contrato.TipoObjeto] : "—"}
              </p>
              <p>
                <span className="text-muted-foreground">Fiscal: </span>
                {processo.Contrato?.Fiscal?.Nome ?? "—"}
              </p>
              <p>
                <span className="text-muted-foreground">CNPJ: </span>
                {processo.Contrato?.ContratadaCNPJ ?? "—"}
              </p>
              <Link
                href={`/contratos/${processo.Contrato?.ID}`}
                className="inline-block text-sm underline underline-offset-2"
              >
                Ver contrato completo
              </Link>
            </CardContent>
          </Card>

          <Card className="shadow-sm">
            <CardHeader>
              <CardTitle className="text-base">Ações</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-2">
              <a
                href={`/api/processos/${processoId}/relatorio`}
                target="_blank"
                rel="noreferrer"
                className={cn(buttonVariants({ variant: "outline" }), "justify-start")}
              >
                Baixar relatório
              </a>
              <Button variant="outline" className="justify-start" onClick={() => setVistoriasOpen(true)}>
                Vistorias de Campo
              </Button>
              <Button variant="outline" className="justify-start" onClick={() => setOcorrenciasOpen(true)}>
                Ocorrências
              </Button>
              {isFiscal && (
                <Button
                  variant="outline"
                  className="justify-start"
                  onClick={() => atestoMutation.mutate()}
                  disabled={atestoMutation.isPending}
                >
                  <TriangleAlertIcon className="size-4" />
                  {atestoMutation.isPending ? "Gerando..." : "Gerar Atesto"}
                </Button>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      <VistoriasDialog
        processoId={processoId}
        isFiscal={isFiscal}
        open={vistoriasOpen}
        onOpenChange={setVistoriasOpen}
      />
      <OcorrenciasDialog
        processoId={processoId}
        isFiscal={isFiscal}
        open={ocorrenciasOpen}
        onOpenChange={setOcorrenciasOpen}
      />

      {/* Pré-visualização rápida — "conferência" sem sair da tela (pedido
          explícito): um <iframe> aponta pro proxy de download da rota
          BFF, que repassa o Content-Type/Content-Disposition "inline"
          reais do backend. Funciona pra PDF (visualizador nativo do
          navegador) e imagem (o navegador embrulha num documento simples
          quando servida sozinha num frame) sem precisar detectar o tipo
          aqui no cliente. */}
      <Dialog open={documentoPreview != null} onOpenChange={(open) => !open && setDocumentoPreview(null)}>
        <DialogContent className="flex h-[85vh] flex-col sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>{documentoPreview?.TipoDocumento?.Nome ?? documentoPreview?.NomeArquivo}</DialogTitle>
          </DialogHeader>
          {documentoPreview && (
            <iframe
              src={`/api/processos/${processoId}/documentos/${documentoPreview.ID}`}
              title={documentoPreview.NomeArquivo}
              className="bg-muted/30 min-h-0 flex-1 rounded-md border"
            />
          )}
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={documentoParaExcluir != null}
        onOpenChange={(open) => !open && setDocumentoParaExcluir(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Excluir documento?</AlertDialogTitle>
            <AlertDialogDescription>
              {documentoParaExcluir && (
                <>
                  Isso vai remover{" "}
                  <span className="text-foreground font-medium">
                    &ldquo;{documentoParaExcluir.TipoDocumento?.Nome ?? documentoParaExcluir.NomeArquivo}&rdquo;
                  </span>{" "}
                  ({documentoParaExcluir.NomeArquivo}) permanentemente. Se este documento fazia parte do checklist,
                  ele volta a aparecer como pendente. Essa ação não pode ser desfeita.
                </>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancelar</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => documentoParaExcluir && excluirDocumentoMutation.mutate(documentoParaExcluir.ID!)}
            >
              Excluir documento
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
