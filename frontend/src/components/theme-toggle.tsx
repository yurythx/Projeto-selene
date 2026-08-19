"use client";

import { useTheme } from "next-themes";
import { Moon, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useMontado } from "@/lib/use-montado";

/**
 * Alterna direto entre claro/escuro num único clique — sem dropdown nem
 * opção "sistema" (pedido explícito do usuário: só precisa ser clicável e
 * já trocar). O ThemeProvider (next-themes, providers.tsx) continua com
 * defaultTheme="system"/enableSystem — a primeira visita, antes de
 * qualquer clique aqui, ainda respeita o SO; a partir do primeiro clique
 * o usuário fica preso em claro/escuro (não tem como voltar a "sistema"
 * pela UI, por design — é exatamente o comportamento pedido).
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
    <Button
      variant="ghost"
      size="icon"
      aria-label={escuro ? "Mudar para tema claro" : "Mudar para tema escuro"}
      onClick={() => setTheme(escuro ? "light" : "dark")}
    >
      {escuro ? <Moon className="size-4" /> : <Sun className="size-4" />}
    </Button>
  );
}
