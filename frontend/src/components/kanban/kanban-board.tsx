"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import type { KanbanEtapa, TipoDocumento, Contrato, ProcessoPagamento } from "@/lib/api/client";
import { ProcessoDialog } from "./processo-dialog";
import { NovoProcessoDialog } from "./novo-processo-dialog";

export function KanbanBoard({
  etapas,
  colunasIniciais,
  tiposDocumento,
  contratosAtivos,
  isFiscal,
}: {
  etapas: KanbanEtapa[];
  colunasIniciais: ProcessoPagamento[][];
  tiposDocumento: TipoDocumento[];
  contratosAtivos: Contrato[];
  isFiscal: boolean;
}) {
  const [selecionado, setSelecionado] = useState<ProcessoPagamento | null>(null);
  const [novoOpen, setNovoOpen] = useState(false);

  return (
    <div className="space-y-4">
      {isFiscal && (
        <div className="flex justify-end">
          <Button onClick={() => setNovoOpen(true)}>Novo processo</Button>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 overflow-x-auto sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        {etapas.map((etapa, i) => (
          <div key={etapa.ID} className="min-w-[220px] space-y-3">
            <div className="flex items-center justify-between px-1">
              <h2 className="text-sm font-semibold">{etapa.Nome}</h2>
              <Badge variant="secondary">{colunasIniciais[i]?.length ?? 0}</Badge>
            </div>

            <div className="space-y-2">
              {(colunasIniciais[i] ?? []).map((processo) => (
                <Card
                  key={processo.ID}
                  className="cursor-pointer p-3 transition-colors hover:bg-accent"
                  onClick={() => setSelecionado(processo)}
                >
                  <p className="text-sm font-medium">{processo.Contrato?.NumeroContrato}</p>
                  <p className="text-muted-foreground text-xs">{processo.Contrato?.ContratadaNome}</p>
                  <div className="mt-2 flex items-center justify-between">
                    <span className="text-muted-foreground text-xs">{processo.MesReferencia}</span>
                    {processo.Status === "Concluido" && (
                      <Badge variant="default" className="text-xs">
                        Pago
                      </Badge>
                    )}
                  </div>
                </Card>
              ))}
              {(colunasIniciais[i] ?? []).length === 0 && (
                <p className="text-muted-foreground px-1 text-xs">Nenhum processo.</p>
              )}
            </div>
          </div>
        ))}
      </div>

      {selecionado && (
        <ProcessoDialog
          processo={selecionado}
          tiposDocumento={tiposDocumento}
          isFiscal={isFiscal}
          open={Boolean(selecionado)}
          onOpenChange={(open) => !open && setSelecionado(null)}
        />
      )}

      {isFiscal && (
        <NovoProcessoDialog
          open={novoOpen}
          onOpenChange={setNovoOpen}
          contratosAtivos={contratosAtivos}
        />
      )}
    </div>
  );
}
