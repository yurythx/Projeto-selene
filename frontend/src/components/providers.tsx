"use client";

import { useState } from "react";
import { SessionProvider } from "next-auth/react";
import type { Session } from "next-auth";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "next-themes";
import { Toaster } from "@/components/ui/sonner";

// Providers agrupa os client-side providers que envolvem toda a árvore
// (ver app/layout.tsx): tema claro/escuro, sessão do Auth.js e cache do
// TanStack Query.
export function Providers({
  children,
  session,
  themeNonce,
}: {
  children: React.ReactNode;
  // Resolvida server-side (app/layout.tsx via auth()) e repassada aqui —
  // ver o comentário lá sobre a corrida que isso evita em
  // useSession().update().
  session: Session | null;
  // Nonce da CSP da requisição atual (app/layout.tsx, header x-nonce) —
  // repassado pro <script> inline que o next-themes injeta pra evitar
  // flash de tema; sem ele, script-src bloqueia esse script. Ver o
  // comentário em layout.tsx.
  themeNonce?: string;
}) {
  // useState garante uma instância por árvore de componentes (não
  // compartilhada entre requisições no server, recomendação do TanStack
  // Query para App Router).
  const [queryClient] = useState(() => new QueryClient());

  return (
    // attribute="class" casa com o seletor `.dark` já definido em
    // globals.css (@custom-variant dark (&:is(.dark *))) — next-themes
    // alterna essa classe no <html>. enableSystem respeita
    // prefers-color-scheme quando o usuário nunca escolheu manualmente
    // (tema default "system"); disableTransitionOnChange evita um flash de
    // transição CSS em todos os elementos no instante da troca.
    <ThemeProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
      nonce={themeNonce}
    >
      <SessionProvider session={session}>
        <QueryClientProvider client={queryClient}>
          {children}
          <Toaster richColors />
        </QueryClientProvider>
      </SessionProvider>
    </ThemeProvider>
  );
}
