"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  closestCenter,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";
import type {
  KanbanEtapa,
  Contrato,
  ProcessoPagamento,
  ItemRadar,
  ResultadoPaginado,
} from "@/lib/api/client";
import { itensDoProcesso, nivelMaisCritico } from "@/lib/radar";
import { RadarNivelBadge } from "@/components/radar/radar-badge";
import { corDaEtapa } from "@/lib/kanban-colors";
import { filtrarProcessos, FILTRO_KANBAN_VAZIO, type FiltroKanban } from "@/lib/kanban-filtro";
import { useMontado } from "@/lib/use-montado";
import { NovoProcessoDialog } from "./novo-processo-dialog";
import { KanbanToolbar } from "./kanban-toolbar";
import { ProcessosLista } from "./processos-lista";

export type Visualizacao = "kanban" | "lista";

// Preferência de visualização (Kanban/Lista) sobrevive a um reload — mesmo
// truque de useMontado (ver lib/use-montado.ts, extraído de
// theme-toggle.tsx): o valor default (renderizado no servidor E na
// primeira pintura no cliente, antes da hidratação assentar) é sempre
// "kanban", a leitura do localStorage só acontece depois de `montado`
// virar true — sem isso, tanto o SSR quanto um useEffect+setState
// causariam cascata de render/mismatch de hidratação.
const CHAVE_VISUALIZACAO = "selene:kanban:visualizacao";

function lerVisualizacaoSalva(): Visualizacao {
  const salvo = localStorage.getItem(CHAVE_VISUALIZACAO);
  return salvo === "lista" ? "lista" : "kanban";
}

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw Object.assign(new Error(body.error ?? "Erro na requisição."), { body, status: res.status });
  }
  return body as T;
}

function iniciaisDoFiscal(nome?: string): string {
  if (!nome) return "?";
  return nome
    .trim()
    .split(/\s+/)
    .map((parte) => parte[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

/**
 * Card individual, arrastável via dnd-kit. `arrastavel` controla se o
 * card participa do drag-and-drop (fiscal + processo ainda não pago/na
 * etapa final) — um card não-arrastável ainda navega pra página do
 * processo ao clicar, só não entra no fluxo de drag.
 */
function ProcessoCard({
  processo,
  cor,
  nivel,
  arrastavel,
  onOpen,
}: {
  processo: ProcessoPagamento;
  cor: ReturnType<typeof corDaEtapa>;
  nivel: ReturnType<typeof nivelMaisCritico>;
  arrastavel: boolean;
  onOpen: () => void;
}) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: processo.ID!,
    data: { processo },
    disabled: !arrastavel,
  });

  const iniciais = iniciaisDoFiscal(processo.Contrato?.Fiscal?.Nome);

  return (
    <Card
      ref={setNodeRef}
      onClick={onOpen}
      className={cn(
        "group relative cursor-pointer overflow-hidden border-l-4 p-3 shadow-sm transition-all duration-150",
        "hover:-translate-y-0.5 hover:shadow-md",
        isDragging && "opacity-30",
        arrastavel && "active:cursor-grabbing"
      )}
      style={{ borderLeftColor: "transparent" }}
      {...(arrastavel ? { ...attributes, ...listeners } : {})}
    >
      <span className={cn("absolute inset-y-0 left-0 w-1", cor.accent)} aria-hidden />
      <div className="flex items-start justify-between gap-2">
        <p className="text-sm font-semibold">{processo.Contrato?.NumeroContrato}</p>
        {nivel && <RadarNivelBadge nivel={nivel} className="text-xs" />}
      </div>
      <p className="text-muted-foreground mt-0.5 truncate text-xs">
        {processo.Contrato?.ContratadaNome}
      </p>
      <div className="mt-3 flex items-center justify-between gap-2">
        <Badge variant="secondary" className="text-xs font-normal">
          {processo.MesReferencia}
        </Badge>
        <div className="flex items-center gap-1.5">
          {processo.Status === "Concluido" && (
            <Badge variant="success" className="text-xs">
              Pago
            </Badge>
          )}
          {processo.Contrato?.Fiscal?.Nome && (
            <Avatar size="sm" title={processo.Contrato.Fiscal.Nome}>
              <AvatarFallback>{iniciais}</AvatarFallback>
            </Avatar>
          )}
        </div>
      </div>
    </Card>
  );
}

/**
 * O quadro `/kanban`: 6 colunas fixas (uma por KanbanEtapa), cada uma com
 * cor própria (ver lib/kanban-colors.ts, estilo Monday.com — grupo/coluna
 * colorido). Cards são arrastáveis (dnd-kit) de uma etapa pra próxima —
 * SÓ pra próxima: o funil é linear e passa pelo mesmo checklist/validação
 * de sempre (KanbanService.AvancarEtapa no backend), soltar num card não
 * move ele por mágica, dispara a MESMA chamada que o botão "Avançar
 * etapa" do drawer — se o checklist estiver incompleto ou houver
 * ocorrência aberta bloqueando, o card simplesmente não se move e o erro
 * aparece num toast, exatamente como já acontecia pelo botão.
 */
/** Estado local de uma coluna — os processos carregados até agora (pode ser mais de uma página) mais os metadados de paginação que guiam o botão "Carregar mais". */
interface EstadoColuna {
  processos: ProcessoPagamento[];
  pagina: number;
  total: number;
  tamanhoPagina: number;
  carregandoMais: boolean;
}

function colunaInicial(resultado: ResultadoPaginado<ProcessoPagamento>): EstadoColuna {
  return {
    processos: resultado.dados,
    pagina: resultado.pagina ?? 1,
    total: resultado.total ?? resultado.dados.length,
    tamanhoPagina: resultado.tamanho_pagina ?? resultado.dados.length,
    carregandoMais: false,
  };
}

export function KanbanBoard({
  etapas,
  colunasIniciais,
  contratosAtivos,
  radarItens,
  isFiscal,
}: {
  etapas: KanbanEtapa[];
  colunasIniciais: ResultadoPaginado<ProcessoPagamento>[];
  contratosAtivos: Contrato[];
  radarItens: ItemRadar[];
  isFiscal: boolean;
}) {
  const router = useRouter();
  const [novoOpen, setNovoOpen] = useState(false);
  const [activeProcesso, setActiveProcesso] = useState<ProcessoPagamento | null>(null);
  const [filtro, setFiltro] = useState<FiltroKanban>(FILTRO_KANBAN_VAZIO);
  const [visualizacaoEscolhida, setVisualizacaoEscolhida] = useState<Visualizacao | null>(null);
  const montado = useMontado();
  const visualizacao: Visualizacao = visualizacaoEscolhida ?? (montado ? lerVisualizacaoSalva() : "kanban");

  function mudarVisualizacao(nova: Visualizacao) {
    setVisualizacaoEscolhida(nova);
    localStorage.setItem(CHAVE_VISUALIZACAO, nova);
  }

  // Paginação real por coluna (pedido explícito do usuário) — cada
  // coluna busca até 100 processos de cada vez (o tamanho de página já
  // usado antes desta mudança) e ganha um botão "Carregar mais" quando
  // sobra mais no backend. Estado próprio (não só um useMemo em cima da
  // prop) porque "carregar mais" MUTA a lista carregada — precisa
  // acumular entre cliques, não recalcular do zero a cada render.
  const [colunas, setColunas] = useState<EstadoColuna[]>(() => colunasIniciais.map(colunaInicial));

  // Ressincroniza quando o Server Component acima refaz o fetch (ex:
  // router.refresh() depois de arrastar um card) — colunasIniciais muda
  // de identidade, e sem isto o estado local ficaria preso na primeira
  // carga pra sempre (useState só roda o inicializador uma vez). Ajustar
  // o state DURANTE o render (não num useEffect) é o padrão documentado
  // do próprio React pra "resetar state quando uma prop muda"
  // (react.dev/reference/react/useState#storing-information-from-previous-renders)
  // — evita o frame extra com dado velho que um useEffect teria. Reset
  // pra página 1 em cada coluna é o comportamento certo aqui: depois de
  // uma mudança real nos dados, não dá pra saber se as páginas 2+ que já
  // tinham sido carregadas continuam válidas.
  const [colunasIniciaisAnterior, setColunasIniciaisAnterior] = useState(colunasIniciais);
  if (colunasIniciais !== colunasIniciaisAnterior) {
    setColunasIniciaisAnterior(colunasIniciais);
    setColunas(colunasIniciais.map(colunaInicial));
  }

  async function carregarMais(etapaId: number, indice: number) {
    const coluna = colunas[indice];
    if (!coluna || coluna.carregandoMais) return;

    setColunas((prev) => prev.map((c, i) => (i === indice ? { ...c, carregandoMais: true } : c)));
    try {
      const proximaPagina = coluna.pagina + 1;
      const resultado = await fetchJSON<ResultadoPaginado<ProcessoPagamento>>(
        `/api/processos?etapa=${etapaId}&pagina=${proximaPagina}&tamanho=${coluna.tamanhoPagina}`
      );
      setColunas((prev) =>
        prev.map((c, i) =>
          i === indice
            ? {
                ...c,
                processos: [...c.processos, ...resultado.dados],
                pagina: resultado.pagina ?? proximaPagina,
                total: resultado.total ?? c.total,
                carregandoMais: false,
              }
            : c
        )
      );
    } catch (erro) {
      toast.error(erro instanceof Error ? erro.message : "Não foi possível carregar mais processos.");
      setColunas((prev) => prev.map((c, i) => (i === indice ? { ...c, carregandoMais: false } : c)));
    }
  }

  // Filtrado por coluna (visão Kanban, contagem no cabeçalho reflete o
  // filtro) e achatado (visão Lista) a partir do MESMO filtro — trocar de
  // modo não perde busca/filtro em andamento. Filtra só o que já foi
  // CARREGADO (busca é client-side) — "Carregar mais" continua visível
  // nesse caso, com um aviso (ver o JSX abaixo) de que a busca pode não
  // cobrir processos ainda não carregados.
  const colunasFiltradas = useMemo(
    () => colunas.map((coluna) => filtrarProcessos(coluna.processos, filtro)),
    [colunas, filtro]
  );
  const todosProcessosFiltrados = useMemo(() => colunasFiltradas.flat(), [colunasFiltradas]);
  const filtroAtivo = Boolean(filtro.busca || filtro.tipoObjeto);

  const sensors = useSensors(
    // distance: 8 — um clique simples (sem mover o ponteiro) não deve
    // iniciar um arraste, senão o onClick de abrir o drawer nunca dispara.
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor)
  );

  const ultimaEtapaId = etapas[etapas.length - 1]?.ID;

  // etapaSeguinte[etapaAtualId] = ID da próxima etapa — usado tanto pra
  // decidir se um card é arrastável (tem próxima etapa) quanto pra
  // validar, no drop, se a coluna de destino é exatamente essa.
  const etapaSeguinte = useMemo(() => {
    const mapa = new Map<number, number>();
    etapas.forEach((etapa, i) => {
      const proxima = etapas[i + 1];
      if (etapa.ID != null && proxima?.ID != null) {
        mapa.set(etapa.ID, proxima.ID);
      }
    });
    return mapa;
  }, [etapas]);

  const avancarMutation = useMutation({
    mutationFn: (processoId: string) =>
      fetchJSON(`/api/processos/${processoId}/avancar`, { method: "POST" }),
    onSuccess: () => {
      toast.success("Processo avançou de etapa.");
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

  function handleDragStart(event: DragStartEvent) {
    setActiveProcesso((event.active.data.current?.processo as ProcessoPagamento) ?? null);
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveProcesso(null);
    const { active, over } = event;
    if (!over) return;

    const processo = active.data.current?.processo as ProcessoPagamento | undefined;
    if (!processo?.ID || processo.EtapaAtualID == null) return;

    const destinoValido = etapaSeguinte.get(processo.EtapaAtualID);
    if (destinoValido == null || over.id !== destinoValido) {
      // Solto fora da única coluna válida (a próxima) — não faz nada,
      // o card simplesmente reaparece na coluna de origem.
      return;
    }

    avancarMutation.mutate(processo.ID);
  }

  const proximaEtapaValidaId = activeProcesso?.EtapaAtualID != null
    ? etapaSeguinte.get(activeProcesso.EtapaAtualID)
    : undefined;

  // Visão Lista achata todas as colunas numa tabela só — não tem "por
  // coluna" pra oferecer um botão em cada uma (ver o board acima), então
  // um botão só carrega a próxima página de QUALQUER coluna que ainda
  // tenha mais, em paralelo.
  const indicesComMaisPaginas = colunas
    .map((c, i) => (c.processos.length < c.total ? i : -1))
    .filter((i) => i >= 0);
  const carregandoMaisNaLista = indicesComMaisPaginas.some((i) => colunas[i]?.carregandoMais);

  function carregarMaisNaLista() {
    for (const indice of indicesComMaisPaginas) {
      carregarMais(etapas[indice].ID!, indice);
    }
  }

  return (
    <div className="space-y-4">
      {isFiscal && (
        <div className="flex justify-end">
          <Button onClick={() => setNovoOpen(true)}>Novo processo</Button>
        </div>
      )}

      <KanbanToolbar
        busca={filtro.busca}
        onBuscaChange={(busca) => setFiltro((f) => ({ ...f, busca }))}
        tipoObjeto={filtro.tipoObjeto}
        onTipoObjetoChange={(tipoObjeto) => setFiltro((f) => ({ ...f, tipoObjeto }))}
        visualizacao={visualizacao}
        onVisualizacaoChange={mudarVisualizacao}
      />

      {visualizacao === "lista" ? (
        <div className="space-y-2">
          <ProcessosLista
            processos={todosProcessosFiltrados}
            etapas={etapas}
            radarItens={radarItens}
            onOpen={(processo) => router.push(`/kanban/${processo.ID}`)}
          />
          {indicesComMaisPaginas.length > 0 && (
            <div className="flex flex-col items-center gap-1">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={carregandoMaisNaLista}
                onClick={carregarMaisNaLista}
              >
                {carregandoMaisNaLista ? "Carregando..." : "Carregar mais processos"}
              </Button>
              {filtroAtivo && (
                <p className="text-muted-foreground text-xs">A busca só cobre os processos já carregados.</p>
              )}
            </div>
          )}
        </div>
      ) : (
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="grid grid-cols-1 gap-4 overflow-x-auto sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
          {etapas.map((etapa, i) => {
            const cor = corDaEtapa(i);
            const processos = colunasFiltradas[i] ?? [];
            const colunaEstado = colunas[i];
            const temMaisPaginas = Boolean(colunaEstado && colunaEstado.processos.length < colunaEstado.total);
            const arrastandoAlgo = activeProcesso != null;
            const ehDestinoValido = arrastandoAlgo && etapa.ID === proximaEtapaValidaId;
            const ehOrigem = arrastandoAlgo && etapa.ID === activeProcesso?.EtapaAtualID;

            return (
              <KanbanColumn
                key={etapa.ID}
                etapaId={etapa.ID!}
                destacada={ehDestinoValido}
                esmaecida={arrastandoAlgo && !ehDestinoValido && !ehOrigem}
                cor={cor}
              >
                <div className={cn("flex items-center justify-between gap-2 rounded-t-md px-3 py-2", cor.headerBg)}>
                  <h2 className={cn("text-sm font-semibold", cor.headerText)}>{etapa.Nome}</h2>
                  <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-white/25 text-xs font-medium text-white">
                    {processos.length}
                  </span>
                </div>

                <div className="space-y-2 px-2 pt-2 pb-1">
                  {processos.map((processo) => {
                    const alertas = itensDoProcesso(radarItens, processo);
                    const nivel = nivelMaisCritico(alertas);
                    const podeArrastar =
                      isFiscal &&
                      processo.Status !== "Concluido" &&
                      processo.EtapaAtualID !== ultimaEtapaId;
                    return (
                      <ProcessoCard
                        key={processo.ID}
                        processo={processo}
                        cor={cor}
                        nivel={nivel}
                        arrastavel={podeArrastar}
                        onOpen={() => router.push(`/kanban/${processo.ID}`)}
                      />
                    );
                  })}
                  {processos.length === 0 && (
                    <p className="text-muted-foreground px-1.5 py-4 text-center text-xs">
                      Nenhum processo.
                    </p>
                  )}
                </div>

                {temMaisPaginas && (
                  <div className="space-y-1 px-2 pb-2">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="w-full"
                      disabled={colunaEstado.carregandoMais}
                      onClick={() => carregarMais(etapa.ID!, i)}
                    >
                      {colunaEstado.carregandoMais
                        ? "Carregando..."
                        : `Carregar mais (${colunaEstado.total - colunaEstado.processos.length} restantes)`}
                    </Button>
                    {/* Busca/filtro é aplicado só sobre o que já foi
                        carregado (client-side) — avisa que pode faltar
                        processo em vez de deixar parecer que a busca já
                        cobriu a coluna inteira. */}
                    {filtroAtivo && (
                      <p className="text-muted-foreground px-1 text-center text-[11px]">
                        A busca só cobre os processos já carregados.
                      </p>
                    )}
                  </div>
                )}
              </KanbanColumn>
            );
          })}
        </div>

        <DragOverlay>
          {activeProcesso && (
            <div className="rotate-2 opacity-95">
              <ProcessoCard
                processo={activeProcesso}
                cor={corDaEtapa(etapas.findIndex((e) => e.ID === activeProcesso.EtapaAtualID))}
                nivel={nivelMaisCritico(itensDoProcesso(radarItens, activeProcesso))}
                arrastavel={false}
                onOpen={() => {}}
              />
            </div>
          )}
        </DragOverlay>
      </DndContext>
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

/** Coluna droppable — o realce (`destacada`)/esmaecimento (`esmaecida`) só aparecem durante um arraste em andamento, pra deixar claro pro fiscal em qual coluna o card pode ser solto. */
function KanbanColumn({
  etapaId,
  cor,
  destacada,
  esmaecida,
  children,
}: {
  etapaId: number;
  cor: ReturnType<typeof corDaEtapa>;
  destacada: boolean;
  esmaecida: boolean;
  children: React.ReactNode;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: etapaId });

  return (
    <div
      ref={setNodeRef}
      data-testid={`kanban-coluna-${etapaId}`}
      className={cn(
        "min-w-[220px] overflow-hidden rounded-lg border bg-muted/20 pb-2 shadow-sm transition-all duration-150",
        destacada && cn("ring-2", cor.ring, isOver && "bg-muted/40"),
        esmaecida && "opacity-50"
      )}
    >
      {children}
    </div>
  );
}
