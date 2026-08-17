"use client";

import { SearchIcon, KanbanSquareIcon, ListIcon } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import type { Visualizacao } from "./kanban-board";

// Sentinela — mesmo padrão de contratos-filtro.tsx (base-ui Select não
// aceita value="").
const TODOS_TIPOS = "TODOS";

const TIPO_OBJETO_OPCOES = [
  { value: TODOS_TIPOS, label: "Todos os tipos" },
  { value: "CONSUMO", label: "Consumo" },
  { value: "PERMANENTE", label: "Permanente" },
  { value: "SERVICO", label: "Serviço" },
];

/**
 * Barra acima do quadro: busca + filtro de tipo de contrato (aplicados em
 * memória, ver lib/kanban-filtro.ts) e o alternador Kanban/Lista — os dois
 * modos de visualização compartilham o mesmo filtro, então trocar de modo
 * não perde o que o fiscal já tinha digitado.
 */
export function KanbanToolbar({
  busca,
  onBuscaChange,
  tipoObjeto,
  onTipoObjetoChange,
  visualizacao,
  onVisualizacaoChange,
}: {
  busca: string;
  onBuscaChange: (valor: string) => void;
  tipoObjeto: string;
  onTipoObjetoChange: (valor: string) => void;
  visualizacao: Visualizacao;
  onVisualizacaoChange: (valor: Visualizacao) => void;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-[240px] flex-1">
          <SearchIcon className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
          <Input
            placeholder="Buscar por contrato, contratada, CNPJ ou fiscal..."
            className="pl-8"
            value={busca}
            onChange={(e) => onBuscaChange(e.target.value)}
          />
        </div>
        <Select
          value={tipoObjeto || TODOS_TIPOS}
          onValueChange={(valor) => onTipoObjetoChange(valor === TODOS_TIPOS ? "" : (valor ?? ""))}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Tipo" />
          </SelectTrigger>
          <SelectContent>
            {TIPO_OBJETO_OPCOES.map((opcao) => (
              <SelectItem key={opcao.value} value={opcao.value}>
                {opcao.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex items-center gap-1 rounded-lg border p-0.5" role="group" aria-label="Modo de visualização">
        <Button
          type="button"
          variant={visualizacao === "kanban" ? "secondary" : "ghost"}
          size="sm"
          aria-pressed={visualizacao === "kanban"}
          onClick={() => onVisualizacaoChange("kanban")}
        >
          <KanbanSquareIcon className={cn("size-4", visualizacao !== "kanban" && "opacity-60")} />
          Kanban
        </Button>
        <Button
          type="button"
          variant={visualizacao === "lista" ? "secondary" : "ghost"}
          size="sm"
          aria-pressed={visualizacao === "lista"}
          onClick={() => onVisualizacaoChange("lista")}
        >
          <ListIcon className={cn("size-4", visualizacao !== "lista" && "opacity-60")} />
          Lista
        </Button>
      </div>
    </div>
  );
}
