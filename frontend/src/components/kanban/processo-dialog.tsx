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
import type {
  TipoDocumento,
  ProcessoPagamento,
  ProcessoComFiscalizacao,
  DocumentoAnexo,
  ItemRadar,
  AllowedAction,
} from "@/lib/api/client";
import { RadarNivelBadge } from "@/components/radar/radar-badge";
import { abrirPDFDeResposta } from "@/lib/abrir-pdf";
import { VistoriasDialog } from "./vistorias-dialog";
import { OcorrenciasDialog } from "./ocorrencias-dialog";
import { TriangleAlertIcon } from "lucide-react";

const estadoFiscalizacaoLabel: Record<string, string> = {
  A_EXECUTAR_CONFERIR: "A executar / conferir",
  EM_ANALISE_EXTERNA: "Em análise externa",
  DOCUMENTAR_ATESTAR: "Documentar / atestar",
  PENDENCIA_DEVOLVIDO: "Pendência — ocorrência aberta",
  CONCLUIDO: "Concluído",
};

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw Object.assign(new Error(body.error ?? "Erro na requisição."), { body, status: res.status });
  }
  return body as T;
}

/**
 * Drawer aberto ao clicar num card do Kanban — mostra dados do
 * contrato/processo, alertas do Radar, documentos anexados (com upload),
 * e os botões de ação (avançar etapa, marcar como pago, gerar atesto),
 * além dos dois dialogs aninhados (Vistorias, Ocorrências — botão
 * "Ocorrências" e "Vistorias de Campo" no rodapé).
 */
export function ProcessoDialog({
  processo,
  tiposDocumento,
  alertasRadar = [],
  isFiscal,
  open,
  onOpenChange,
}: {
  processo: ProcessoPagamento;
  tiposDocumento: TipoDocumento[];
  alertasRadar?: ItemRadar[];
  isFiscal: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [tipoSelecionado, setTipoSelecionado] = useState<string>("");
  const [pendentes, setPendentes] = useState<string[] | null>(null);
  const [vistoriasOpen, setVistoriasOpen] = useState(false);
  const [ocorrenciasOpen, setOcorrenciasOpen] = useState(false);

  const tipoDocumentoSelecionado = tiposDocumento.find(
    (tipo) => String(tipo.ID) === tipoSelecionado
  );

  const documentosQuery = useQuery({
    queryKey: ["documentos", processo.ID],
    queryFn: () => fetchJSON<DocumentoAnexo[]>(`/api/processos/${processo.ID}/documentos`),
    enabled: open,
  });

  // SGF-Rondonópolis: allowed_actions/estado_fiscalizacao/acao_ou_espera só
  // existem na resposta de GET /processos/{id} (ver
  // FiscalizacaoService.Decorar no backend), não na listagem que alimentou
  // o `processo` recebido por prop — refeito aqui a cada abertura do
  // drawer. Enquanto carrega/se falhar, cai para a leitura antiga
  // (EtapaAtualID/Status) via podeAvancarFallback/podeConcluirFallback
  // abaixo, pra nunca deixar os botões desaparecerem por um problema de
  // rede momentâneo.
  const processoDetalhadoQuery = useQuery({
    queryKey: ["processo", processo.ID],
    queryFn: () => fetchJSON<ProcessoComFiscalizacao>(`/api/processos/${processo.ID}`),
    enabled: open,
  });
  const allowedActions: AllowedAction[] = processoDetalhadoQuery.data?.allowed_actions ?? [];

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

  const atestoMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(`/api/processos/${processo.ID}/atesto`, { method: "POST" });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível gerar o atesto.");
      }
      return res;
    },
    onSuccess: async (res) => {
      await abrirPDFDeResposta(res);
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
  }

  // Fallback (a leitura antiga, EtapaAtualID/Status) só entra em jogo
  // enquanto processoDetalhadoQuery ainda não respondeu — assim que
  // responde, allowed_actions manda: já reflete checklist pendente E
  // ocorrência aberta (SGF), nenhum dos dois a leitura antiga considerava.
  const decorado = processoDetalhadoQuery.data;
  const podeConcluir = decorado
    ? allowedActions.includes("CONCLUIR_PAGAMENTO")
    : processo.EtapaAtualID === 6 && processo.Status !== "Concluido";
  const podeAvancar = decorado
    ? allowedActions.includes("AVANCAR_ETAPA")
    : processo.Status !== "Concluido" && processo.EtapaAtualID !== 6;

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
          {decorado?.estado_fiscalizacao && (
            <p>
              <span className="text-muted-foreground">Estado (SGF): </span>
              <Badge variant={decorado.estado_fiscalizacao === "PENDENCIA_DEVOLVIDO" ? "destructive" : "outline"}>
                {estadoFiscalizacaoLabel[decorado.estado_fiscalizacao] ?? decorado.estado_fiscalizacao}
              </Badge>
            </p>
          )}
        </div>

        {alertasRadar.length > 0 && (
          <div className="space-y-1">
            {alertasRadar.map((alerta, i) => (
              <div
                key={i}
                className="flex items-center justify-between gap-2 rounded-md border p-2 text-sm"
              >
                <span>{alerta.mensagem}</span>
                {alerta.nivel && <RadarNivelBadge nivel={alerta.nivel} />}
              </div>
            ))}
          </div>
        )}

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
          <div className="flex gap-2">
            <Button type="button" variant="outline" onClick={() => setVistoriasOpen(true)}>
              Vistorias de Campo
            </Button>
            <Button type="button" variant="outline" onClick={() => setOcorrenciasOpen(true)}>
              Ocorrências
            </Button>
            {isFiscal && (
              <>
                <Button
                  variant="outline"
                  onClick={() => atestoMutation.mutate()}
                  disabled={atestoMutation.isPending}
                >
                  {atestoMutation.isPending ? "Gerando..." : "Gerar Atesto"}
                </Button>
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
              </>
            )}
          </div>
        </DialogFooter>
      </DialogContent>

      <VistoriasDialog
        processoId={processo.ID!}
        isFiscal={isFiscal}
        open={vistoriasOpen}
        onOpenChange={setVistoriasOpen}
      />
      <OcorrenciasDialog
        processoId={processo.ID!}
        isFiscal={isFiscal}
        open={ocorrenciasOpen}
        onOpenChange={setOcorrenciasOpen}
      />
    </Dialog>
  );
}
