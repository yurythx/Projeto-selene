"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";

// Boundary de erro do grupo (app) — captura falhas em Server Components
// dessas rotas (ex: backend indisponível, timeout) e mostra uma mensagem
// amigável com opção de tentar de novo, em vez da tela de erro genérica
// do Next. Precisa ser Client Component (convenção do App Router).
export default function AppError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-4 text-center">
      <div>
        <h1 className="text-xl font-semibold">Não foi possível carregar esta página</h1>
        <p className="text-muted-foreground mt-1 text-sm">
          O backend pode estar indisponível no momento. Tente novamente em instantes.
        </p>
      </div>
      <Button onClick={reset}>Tentar de novo</Button>
    </div>
  );
}
