"use client";

import { useState } from "react";
import { SessionProvider } from "next-auth/react";
import type { Session } from "next-auth";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";

// Providers agrupa os client-side providers que envolvem toda a árvore
// (ver app/layout.tsx): sessão do Auth.js e cache do TanStack Query.
export function Providers({
  children,
  session,
}: {
  children: React.ReactNode;
  // Resolvida server-side (app/layout.tsx via auth()) e repassada aqui —
  // ver o comentário lá sobre a corrida que isso evita em
  // useSession().update().
  session: Session | null;
}) {
  // useState garante uma instância por árvore de componentes (não
  // compartilhada entre requisições no server, recomendação do TanStack
  // Query para App Router).
  const [queryClient] = useState(() => new QueryClient());

  return (
    <SessionProvider session={session}>
      <QueryClientProvider client={queryClient}>
        {children}
        <Toaster richColors />
      </QueryClientProvider>
    </SessionProvider>
  );
}
