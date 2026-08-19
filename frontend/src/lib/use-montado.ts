import { useSyncExternalStore } from "react";

/**
 * Distingue "renderizando no server (ou antes da hidratação)" de "já
 * rodando no client" sem o padrão setState-em-useEffect (o
 * react-hooks/set-state-in-effect do lint reprova isso — cascata de
 * renders evitável). useSyncExternalStore com getServerSnapshot=false e
 * getSnapshot=true é o jeito recomendado pelo próprio React pra esse
 * caso: o valor "muda" exatamente uma vez, na hidratação, sem precisar de
 * um efeito disparando um novo render manualmente.
 *
 * Extraído de theme-toggle.tsx (mesmo problema: ler um estado só
 * disponível no client — lá é resolvedTheme, aqui é localStorage direto,
 * ver kanban-board.tsx) pra não duplicar o truque.
 */
export function useMontado(): boolean {
  return useSyncExternalStore(
    () => () => {},
    () => true,
    () => false
  );
}
