import { Badge } from "@/components/ui/badge";
import type { ItemRadar } from "@/lib/api/client";
import { cn } from "@/lib/utils";

/**
 * Badge de nível de alerta do Radar (Fase 1 do roadmap). CRITICO reaproveita
 * a variante "destructive" do design system; ATENCAO usa uma cor âmbar
 * dedicada — não existe uma variante "warning" no Badge base ainda, então
 * as classes ficam aqui em vez de inventar uma variante global só pra isso.
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
    <Badge
      variant="outline"
      className={cn(
        "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:bg-amber-500/20 dark:text-amber-400",
        className
      )}
    >
      Atenção
    </Badge>
  );
}
