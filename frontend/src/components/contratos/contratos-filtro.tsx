"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname, useSearchParams } from "next/navigation";
import { SearchIcon } from "lucide-react";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// Sentinelas — base-ui Select não aceita value="" num SelectItem (mesmo
// padrão já usado em editar-gatilho-modelo-dialog.tsx com "NENHUM"): aqui
// "TODOS"/"TODAS" representam "sem filtro nesse campo" e são convertidos
// pra ausência do parâmetro de URL antes de navegar.
const TODOS_TIPOS = "TODOS";
const TODAS_SITUACOES = "TODAS";

// Passadas como prop `items` do Select (não só usadas no .map() do
// SelectContent) — sem isso, <SelectValue> só resolve o rótulo enquanto o
// popup está aberto (o registro interno do base-ui é perdido quando o
// Portal do SelectContent desmonta ao fechar); sem o `items` persistente,
// depois de escolher a tela mostra o value cru ("SERVICO") em vez do
// rótulo ("Serviço"). Achado real em produção, não só teórico.
const TIPO_OBJETO_OPCOES = [
  { value: TODOS_TIPOS, label: "Todos os tipos" },
  { value: "CONSUMO", label: "Consumo" },
  { value: "PERMANENTE", label: "Permanente" },
  { value: "SERVICO", label: "Serviço" },
];

const SITUACAO_OPCOES = [
  { value: TODAS_SITUACOES, label: "Todas as situações" },
  { value: "ativo", label: "Ativos" },
  { value: "encerrado", label: "Encerrados" },
];

/**
 * Barra de busca/filtro de Contratos — reflete o estado na URL (?busca=,
 * ?tipo_objeto=, ?situacao=) pra ser compartilhável/voltar-com-o-back-
 * button funcionar, e pra deixar a página Server Component (o filtro é
 * aplicado no backend via repository.FiltroContrato, não em memória no
 * cliente). Qualquer mudança de filtro reseta ?pagina pra 1 — senão dava
 * pra ficar numa página 3 que não existe mais no resultado filtrado.
 */
export function ContratosFiltro({
  buscaInicial,
  tipoObjetoInicial,
  situacaoInicial,
}: {
  buscaInicial: string;
  tipoObjetoInicial: string;
  situacaoInicial: string;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [busca, setBusca] = useState(buscaInicial);

  // Debounce de 400ms — sem isso, cada tecla digitada dispararia uma
  // navegação (e uma nova busca no backend) por letra.
  useEffect(() => {
    const handle = setTimeout(() => {
      if (busca !== (searchParams.get("busca") ?? "")) {
        atualizarFiltro({ busca });
      }
    }, 400);
    return () => clearTimeout(handle);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- só reagir a `busca`; searchParams/router mudam a cada navegação e reiniciariam o debounce
  }, [busca]);

  function atualizarFiltro(mudancas: Record<string, string>) {
    const params = new URLSearchParams(searchParams.toString());
    for (const [chave, valor] of Object.entries(mudancas)) {
      if (valor) {
        params.set(chave, valor);
      } else {
        params.delete(chave);
      }
    }
    params.delete("pagina");
    router.push(`${pathname}?${params.toString()}`);
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="relative min-w-[220px] flex-1">
        <SearchIcon className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
        <Input
          placeholder="Buscar por número, contratada ou CNPJ..."
          className="pl-8"
          value={busca}
          onChange={(e) => setBusca(e.target.value)}
        />
      </div>
      <Select
        items={TIPO_OBJETO_OPCOES}
        value={tipoObjetoInicial || TODOS_TIPOS}
        onValueChange={(valor) => atualizarFiltro({ tipo_objeto: valor === TODOS_TIPOS ? "" : (valor ?? "") })}
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
      <Select
        items={SITUACAO_OPCOES}
        value={situacaoInicial || TODAS_SITUACOES}
        onValueChange={(valor) => atualizarFiltro({ situacao: valor === TODAS_SITUACOES ? "" : (valor ?? "") })}
      >
        <SelectTrigger className="w-[180px]">
          <SelectValue placeholder="Situação" />
        </SelectTrigger>
        <SelectContent>
          {SITUACAO_OPCOES.map((opcao) => (
            <SelectItem key={opcao.value} value={opcao.value}>
              {opcao.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
