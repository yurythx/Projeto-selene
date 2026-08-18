"use client";

import { Menu } from "lucide-react";
import { useSidebarContext } from "@/components/sidebar-context";

/**
 * Barra fixa no topo, só em telas < md — a sidebar some completamente
 * nesse breakpoint (ver sidebar.tsx) e esta barra é o único jeito de
 * reabri-la, além de manter a marca visível o tempo todo. Faz parte do
 * fluxo normal da coluna de conteúdo (não é `fixed`), então empurra o
 * `<main>` pra baixo sozinha — sem precisar de padding-top mágico pra
 * compensar um elemento flutuante sobrepondo o conteúdo.
 */
export function MobileTopBar() {
  const { setMobileOpen } = useSidebarContext();

  return (
    // h-16 (era h-14) — mesma altura do header real de papermoon.cloud.
    // bg-sidebar/90 + backdrop-blur — não cor chapada: o header deles
    // (fixed, sobre o conteúdo) e os painéis internos (ex.: o card de
    // formulário, `bg-surface-1/60 backdrop-blur-sm`) usam esse mesmo
    // vidro fosco translúcido como assinatura visual, confirmado no CSS
    // compilado deles. supports-[...] cobre navegador sem suporte a
    // backdrop-filter (fica sólido, sem o vidro, mas legível).
    <header className="bg-sidebar/90 text-sidebar-foreground border-sidebar-border sticky top-0 z-30 flex h-16 items-center gap-3 border-b px-4 shadow-sm backdrop-blur-md supports-[backdrop-filter]:bg-sidebar/75 md:hidden">
      <button
        type="button"
        onClick={() => setMobileOpen(true)}
        aria-label="Abrir menu"
        className="hover:bg-sidebar-accent -ml-1.5 flex size-9 items-center justify-center rounded-lg outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-sidebar active:translate-y-px"
      >
        <Menu className="size-5" />
      </button>
      <div className="flex items-center gap-2">
        {/* Mesmo selo com gradiente + glow do badge da sidebar (ver
            sidebar.tsx) — âmbar, o accent real do papermoon.cloud, igual
            nos dois temas. */}
        <div className="flex size-6 items-center justify-center rounded-md bg-gradient-to-br from-amber-300 to-amber-500 text-xs font-bold text-slate-900 shadow-lg shadow-amber-400/40 ring-1 ring-white/25">
          S
        </div>
        {/* text-base font-bold tracking-tight — mesmo tratamento da
            wordmark aplicado em sidebar.tsx, pra consistência entre as
            duas barras (era só font-semibold). */}
        <span className="text-base font-bold tracking-tight">Selene</span>
      </div>
    </header>
  );
}
