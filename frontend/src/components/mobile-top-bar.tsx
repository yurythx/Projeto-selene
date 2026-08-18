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
    <header className="bg-sidebar text-sidebar-foreground border-sidebar-border sticky top-0 z-30 flex h-14 items-center gap-3 border-b px-4 shadow-sm md:hidden">
      <button
        type="button"
        onClick={() => setMobileOpen(true)}
        aria-label="Abrir menu"
        className="hover:bg-sidebar-accent -ml-1.5 flex size-9 items-center justify-center rounded-lg"
      >
        <Menu className="size-5" />
      </button>
      <div className="flex items-center gap-2">
        {/* Mesma cor de marca do badge da sidebar (ver sidebar.tsx) —
            âmbar, o accent real do papermoon.cloud, igual nos dois temas. */}
        <div className="flex size-6 items-center justify-center rounded-md bg-amber-400 text-xs font-bold text-slate-900">
          S
        </div>
        <span className="font-semibold">Selene</span>
      </div>
    </header>
  );
}
