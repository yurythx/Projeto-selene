"use client";

import { createContext, useContext, useState, type ReactNode } from "react";

interface SidebarContextValue {
  /** Drawer da sidebar aberto no mobile (< md) — irrelevante em telas md+, onde a sidebar é sempre visível (colapsada ou não). */
  mobileOpen: boolean;
  setMobileOpen: (open: boolean) => void;
}

const SidebarContext = createContext<SidebarContextValue | null>(null);

/**
 * Compartilha o estado "drawer mobile aberto" entre o botão de hambúrguer
 * (MobileTopBar, que fica na coluna de conteúdo) e a própria `<aside>`
 * (Sidebar, fora dessa coluna) — os dois vivem em subárvores irmãs no
 * layout (ver app/(app)/layout.tsx), então um Context é mais simples que
 * subir o estado pra um Server Component (que não pode ter estado) ou
 * fundir os dois componentes num só.
 */
export function SidebarProvider({ children }: { children: ReactNode }) {
  const [mobileOpen, setMobileOpen] = useState(false);
  return (
    <SidebarContext.Provider value={{ mobileOpen, setMobileOpen }}>
      {children}
    </SidebarContext.Provider>
  );
}

export function useSidebarContext(): SidebarContextValue {
  const ctx = useContext(SidebarContext);
  if (!ctx) {
    throw new Error("useSidebarContext precisa ser usado dentro de um SidebarProvider");
  }
  return ctx;
}
