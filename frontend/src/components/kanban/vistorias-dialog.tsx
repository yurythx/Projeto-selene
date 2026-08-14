"use client";

import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { RegistroVistoria } from "@/lib/api/client";
import { MapPinIcon, CameraIcon, FileTextIcon } from "lucide-react";

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw Object.assign(new Error(body.error ?? "Erro na requisição."), { body, status: res.status });
  }
  return body as T;
}

/**
 * Captura a localização atual via navigator.geolocation — API do browser,
 * sem dependência nova (Módulo 3 do roadmap). O usuário pode negar a
 * permissão ou o dispositivo pode não suportar; nos dois casos a Promise
 * rejeita e a vistoria segue sem coordenadas (ver o comentário no model
 * RegistroVistoria no backend — graceful degradation, não bloqueio).
 */
function capturarLocalizacao(): Promise<{ latitude: number; longitude: number }> {
  return new Promise((resolve, reject) => {
    if (!("geolocation" in navigator)) {
      reject(new Error("Geolocalização não é suportada neste dispositivo/navegador."));
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => resolve({ latitude: pos.coords.latitude, longitude: pos.coords.longitude }),
      (erro) => reject(new Error(erro.message || "Não foi possível obter a localização.")),
      { enableHighAccuracy: true, timeout: 10_000 }
    );
  });
}

function formatarDataHora(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("pt-BR");
}

/**
 * Dialog mobile-first de vistorias de campo — a primeira tela do projeto
 * pensada primariamente pra celular (fiscal em campo, não na mesa):
 * botões grandes, um controle por linha, captura de localização e de foto
 * direto da câmera (`capture="environment"`).
 */
export function VistoriasDialog({
  processoId,
  isFiscal,
  open,
  onOpenChange,
}: {
  processoId: string;
  isFiscal: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [observacoes, setObservacoes] = useState("");
  const [coordenadas, setCoordenadas] = useState<{ latitude: number; longitude: number } | null>(null);
  const [capturandoLocalizacao, setCapturandoLocalizacao] = useState(false);
  const fotoInputRefs = useRef<Record<string, HTMLInputElement | null>>({});

  const vistoriasQuery = useQuery({
    queryKey: ["vistorias", processoId],
    queryFn: () => fetchJSON<RegistroVistoria[]>(`/api/processos/${processoId}/vistorias`),
    enabled: open,
  });

  const registrarMutation = useMutation({
    mutationFn: () =>
      fetchJSON<RegistroVistoria>(`/api/processos/${processoId}/vistorias`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          latitude: coordenadas?.latitude ?? null,
          longitude: coordenadas?.longitude ?? null,
          observacoes,
        }),
      }),
    onSuccess: () => {
      toast.success("Vistoria registrada.");
      queryClient.invalidateQueries({ queryKey: ["vistorias", processoId] });
      setObservacoes("");
      setCoordenadas(null);
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  const anexarFotoMutation = useMutation({
    mutationFn: async ({ vistoriaId, arquivo }: { vistoriaId: string; arquivo: File }) => {
      const formData = new FormData();
      formData.append("foto", arquivo);
      return fetchJSON(`/api/vistorias/${vistoriaId}/fotos`, { method: "POST", body: formData });
    },
    onSuccess: () => {
      toast.success("Foto anexada.");
      queryClient.invalidateQueries({ queryKey: ["vistorias", processoId] });
    },
    onError: (erro: Error) => toast.error(erro.message),
  });

  async function handleCapturarLocalizacao() {
    setCapturandoLocalizacao(true);
    try {
      setCoordenadas(await capturarLocalizacao());
    } catch (erro) {
      toast.error(erro instanceof Error ? erro.message : "Falha ao capturar localização.");
    } finally {
      setCapturandoLocalizacao(false);
    }
  }

  function handleFotoSelecionada(vistoriaId: string, arquivo: File | undefined) {
    if (!arquivo) return;
    anexarFotoMutation.mutate({ vistoriaId, arquivo });
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[90vh] flex-col sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Vistorias de Campo</DialogTitle>
        </DialogHeader>

        <div className="flex-1 space-y-4 overflow-y-auto">
          {isFiscal && (
            <Card className="space-y-3 p-4">
              <div className="space-y-2">
                <Label htmlFor="observacoes_vistoria">Nova vistoria — observações</Label>
                <Textarea
                  id="observacoes_vistoria"
                  rows={3}
                  value={observacoes}
                  onChange={(e) => setObservacoes(e.target.value)}
                  placeholder="O que foi observado em campo..."
                />
              </div>

              <Button
                type="button"
                variant="outline"
                className="w-full"
                onClick={handleCapturarLocalizacao}
                disabled={capturandoLocalizacao}
              >
                <MapPinIcon />
                {capturandoLocalizacao
                  ? "Obtendo localização..."
                  : coordenadas
                    ? `Localização capturada (${coordenadas.latitude.toFixed(4)}, ${coordenadas.longitude.toFixed(4)})`
                    : "Capturar localização atual"}
              </Button>

              <Button
                type="button"
                className="w-full"
                onClick={() => registrarMutation.mutate()}
                disabled={registrarMutation.isPending}
              >
                {registrarMutation.isPending ? "Registrando..." : "Registrar vistoria"}
              </Button>
            </Card>
          )}

          <div className="space-y-3">
            {vistoriasQuery.isLoading && (
              <p className="text-muted-foreground text-sm">Carregando...</p>
            )}
            {vistoriasQuery.data?.length === 0 && (
              <p className="text-muted-foreground text-sm">Nenhuma vistoria registrada ainda.</p>
            )}
            {vistoriasQuery.data?.map((vistoria) => (
              <Card key={vistoria.ID} className="space-y-2 p-4 text-sm">
                <div className="flex items-center justify-between">
                  <span className="font-medium">{formatarDataHora(vistoria.DataHora)}</span>
                  {vistoria.Latitude != null && vistoria.Longitude != null && (
                    <span className="text-muted-foreground flex items-center gap-1 text-xs">
                      <MapPinIcon className="size-3" />
                      {vistoria.Latitude.toFixed(4)}, {vistoria.Longitude.toFixed(4)}
                    </span>
                  )}
                </div>
                {vistoria.Observacoes && <p>{vistoria.Observacoes}</p>}
                <p className="text-muted-foreground text-xs">
                  {vistoria.Fotos?.length ?? 0} foto(s) anexada(s)
                </p>

                <div className="flex flex-wrap items-center gap-2 pt-1">
                  {isFiscal && (
                    <>
                      <input
                        ref={(el) => {
                          fotoInputRefs.current[vistoria.ID!] = el;
                        }}
                        type="file"
                        accept="image/*"
                        capture="environment"
                        className="hidden"
                        onChange={(e) =>
                          handleFotoSelecionada(vistoria.ID!, e.target.files?.[0])
                        }
                      />
                      <Button
                        type="button"
                        size="sm"
                        variant="secondary"
                        onClick={() => fotoInputRefs.current[vistoria.ID!]?.click()}
                        disabled={anexarFotoMutation.isPending}
                      >
                        <CameraIcon />
                        Anexar foto
                      </Button>
                    </>
                  )}
                  <a
                    href={`/api/vistorias/${vistoria.ID}/relatorio`}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 text-sm underline underline-offset-2"
                  >
                    <FileTextIcon className="size-3.5" />
                    Relatório de Campo (PDF)
                  </a>
                </div>
              </Card>
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
