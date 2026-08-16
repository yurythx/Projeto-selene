"use client";

import { useSyncExternalStore } from "react";
import { useTheme } from "next-themes";
import { Moon, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

// useMontado distingue "renderizando no server (ou antes da hidratação)"
// de "já rodando no client" sem o padrão setState-em-useEffect (o
// react-hooks/set-state-in-effect do lint reprova isso — cascata de
// renders evitável). useSyncExternalStore com getServerSnapshot=false e
// getSnapshot=true é o jeito recomendado pelo próprio React para esse
// caso: o valor "muda" exatamente uma vez, na hidratação, sem precisar de
// um efeito disparando um novo render manualmente.
function useMontado() {
  return useSyncExternalStore(
    () => () => {},
    () => true,
    () => false
  );
}

/**
 * Alterna entre tema claro/escuro/sistema — ThemeProvider (next-themes)
 * está em components/providers.tsx, attribute="class", casando com o
 * seletor `.dark` de globals.css.
 *
 * `montado` evita o ícone errado no primeiro render: o server sempre
 * renderiza com resolvedTheme=undefined (o tema real só é conhecido no
 * client, depois de ler localStorage/prefers-color-scheme via
 * ThemeProvider) — sem essa guarda, o ícone "piscaria" de Sol pra Lua (ou
 * vice-versa) assim que a hidratação terminasse.
 */
export function ThemeToggle() {
  const { setTheme, resolvedTheme } = useTheme();
  const montado = useMontado();

  const escuro = montado && resolvedTheme === "dark";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="ghost" size="icon" aria-label="Alternar tema">
            {escuro ? <Moon className="size-4" /> : <Sun className="size-4" />}
          </Button>
        }
      />
      <DropdownMenuContent align="end">
        {/* onClick, não onSelect — ver o comentário equivalente em nav.tsx
            sobre DropdownMenuItem ser @base-ui/react/menu, não Radix. */}
        <DropdownMenuItem onClick={() => setTheme("light")}>Claro</DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("dark")}>Escuro</DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("system")}>Sistema</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
