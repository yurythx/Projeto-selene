/**
 * Paleta de cores por etapa do Kanban — uma cor de destaque fixa por
 * posição de coluna (não por nome, pra sobreviver a uma eventual
 * renomeação de etapa), no espírito visual do Monday.com: cada grupo/
 * coluna tem uma cor própria e bem saturada, usada tanto na barra do
 * cabeçalho da coluna quanto na borda de destaque dos cards dentro dela.
 *
 * São 6 cores fixas porque o funil tem 6 etapas fixas (ver
 * KanbanEtapa/checklist.go) — não precisa ser dinâmico.
 */
export interface CorEtapa {
  /** Borda de destaque do card (mesma cor da coluna, "etiqueta" discreta). */
  accent: string;
  /** Fundo SÓLIDO e saturado do cabeçalho da coluna — no espírito Monday.com,
   * onde cada grupo tem uma faixa de cor cheia, não um tom pastel. */
  headerBg: string;
  /** Texto sobre headerBg (sempre branco — todas as cores da paleta são
   * escuras o bastante em ambos os temas pra manter contraste). */
  headerText: string;
  /** Anel de destaque usado na coluna de destino válida durante o arraste. */
  ring: string;
}

const PALETA: CorEtapa[] = [
  {
    accent: "bg-sky-500",
    headerBg: "bg-sky-600 dark:bg-sky-700",
    headerText: "text-white",
    ring: "ring-sky-400",
  },
  {
    accent: "bg-violet-500",
    headerBg: "bg-violet-600 dark:bg-violet-700",
    headerText: "text-white",
    ring: "ring-violet-400",
  },
  {
    accent: "bg-amber-500",
    headerBg: "bg-amber-500 dark:bg-amber-600",
    headerText: "text-white",
    ring: "ring-amber-400",
  },
  {
    accent: "bg-rose-500",
    headerBg: "bg-rose-600 dark:bg-rose-700",
    headerText: "text-white",
    ring: "ring-rose-400",
  },
  {
    accent: "bg-teal-500",
    headerBg: "bg-teal-600 dark:bg-teal-700",
    headerText: "text-white",
    ring: "ring-teal-400",
  },
  {
    accent: "bg-emerald-500",
    headerBg: "bg-emerald-600 dark:bg-emerald-700",
    headerText: "text-white",
    ring: "ring-emerald-400",
  },
];

/** Cor da etapa pela posição (0-based) na lista de etapas — nunca lança, cicla se houver mais de 6 (não deveria acontecer no funil fixo). */
export function corDaEtapa(posicao: number): CorEtapa {
  return PALETA[posicao % PALETA.length];
}
