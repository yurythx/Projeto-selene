import { Badge } from "@/components/ui/badge";
import type { ItemRadar } from "@/lib/api/client";

/**
 * Badge de nível de alerta do Radar (Fase 1 do roadmap). CRITICO usa a
 * variante "destructive", ATENCAO usa "warning" — ambas do design system
 * central (ver components/ui/badge.tsx).
 */
export function RadarNivelBadge({
  nivel,
  className,
}: {
  nivel: NonNullable<ItemRadar["nivel"]>;
  className?: string;
}) {
  if (nivel === "CRITICO") {
    return (
      <Badge variant="destructive" className={className}>
        Crítico
      </Badge>
    );
  }

  return (
    <Badge variant="warning" className={className}>
      Atenção
    </Badge>
  );
}
