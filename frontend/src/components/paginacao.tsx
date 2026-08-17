"use client";

import { useRouter, usePathname, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";

/**
 * Controles de paginação genéricos (Anterior/Próxima + "Página X de Y"),
 * dirigidos pelo parâmetro de URL "pagina" — reaproveitável em qualquer
 * listagem paginada pelo backend (hoje só Contratos, ver
 * repository.ResultadoPaginado). Não renderiza nada com 1 página só ou
 * menos (nada pra paginar).
 */
export function Paginacao({
  paginaAtual,
  totalPaginas,
}: {
  paginaAtual: number;
  totalPaginas: number;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  if (totalPaginas <= 1) {
    return null;
  }

  function irPara(pagina: number) {
    const params = new URLSearchParams(searchParams.toString());
    params.set("pagina", String(pagina));
    router.push(`${pathname}?${params.toString()}`);
  }

  return (
    <div className="flex items-center justify-between text-sm">
      <p className="text-muted-foreground">
        Página {paginaAtual} de {totalPaginas}
      </p>
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={paginaAtual <= 1}
          onClick={() => irPara(paginaAtual - 1)}
        >
          <ChevronLeftIcon className="size-4" />
          Anterior
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={paginaAtual >= totalPaginas}
          onClick={() => irPara(paginaAtual + 1)}
        >
          Próxima
          <ChevronRightIcon className="size-4" />
        </Button>
      </div>
    </div>
  );
}
