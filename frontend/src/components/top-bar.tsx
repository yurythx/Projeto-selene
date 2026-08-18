"use client";

import { useSession, signOut } from "next-auth/react";
import { Menu, LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ThemeToggle } from "@/components/theme-toggle";
import { useSidebarContext } from "@/components/sidebar-context";

/**
 * Barra superior PERSISTENTE — desktop e mobile, ao contrário da antiga
 * MobileTopBar (que só existia em telas < md e só tinha hambúrguer+logo,
 * sem nada de sessão). Tema e usuário logado moraram no rodapé da
 * Sidebar até aqui; saíram de lá porque o pedido explícito do usuário
 * foi "barra lateral com o menu" + "barra superior com o usuário logado
 * e o tema" — duas responsabilidades separadas, não uma sidebar fazendo
 * as duas coisas.
 *
 * Faz parte do fluxo normal da coluna de conteúdo (não é `fixed`), então
 * empurra o `<main>` pra baixo sozinha — sem padding-top mágico.
 */
export function TopBar() {
  const { data: session } = useSession();
  const { setMobileOpen } = useSidebarContext();

  const iniciais = (session?.user?.name ?? session?.user?.email ?? "?")
    .trim()
    .split(/\s+/)
    .map((parte) => parte[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();

  return (
    // h-16, vidro fosco (bg-sidebar/90 + backdrop-blur) — mesma altura e
    // "textura" de painel translúcido do header real de papermoon.cloud
    // (ver sidebar.tsx pro mesmo raciocínio, já documentado lá).
    <header className="bg-sidebar/90 text-sidebar-foreground border-sidebar-border sticky top-0 z-30 flex h-16 items-center gap-3 border-b px-4 shadow-sm backdrop-blur-md supports-[backdrop-filter]:bg-sidebar/75 sm:px-6">
      {/* Hambúrguer + logo só em mobile — a Sidebar já mostra a logo em
          telas md+, não precisa duplicar aqui. */}
      <button
        type="button"
        onClick={() => setMobileOpen(true)}
        aria-label="Abrir menu"
        className="hover:bg-sidebar-accent -ml-1.5 flex size-9 shrink-0 items-center justify-center rounded-lg outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-sidebar active:translate-y-px md:hidden"
      >
        <Menu className="size-5" />
      </button>
      <div className="flex items-center gap-2 md:hidden">
        <div className="flex size-6 items-center justify-center rounded-md bg-gradient-to-br from-amber-300 to-amber-500 text-xs font-bold text-slate-900 shadow-lg shadow-amber-400/40 ring-1 ring-white/25">
          S
        </div>
        <span className="text-base font-bold tracking-tight">Selene</span>
      </div>

      <div className="flex-1" />

      <ThemeToggle />
      {session?.user && (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon"
                className="rounded-full"
                aria-label={`Menu de ${session.user.name ?? session.user.email}`}
              >
                <Avatar className="size-8">
                  <AvatarFallback>{iniciais}</AvatarFallback>
                </Avatar>
              </Button>
            }
          />
          <DropdownMenuContent align="end" side="bottom">
            {/* DropdownMenuLabel (Menu.GroupLabel do base-ui) exige um
                Menu.Group ancestral — sem ele, useMenuGroupRootContext()
                lança "Base UI error #31: MenuGroupContext is missing" e
                quebra a página inteira (ErrorBoundary) toda vez que
                alguém abre este menu. Bug real já corrigido uma vez
                nesta sessão — não reintroduzir. */}
            <DropdownMenuGroup>
              <DropdownMenuLabel className="font-normal">
                <div className="flex flex-col space-y-1">
                  <p className="text-sm font-medium leading-none">{session.user.name}</p>
                  <p className="text-muted-foreground text-xs leading-none">
                    {session.user.email}
                  </p>
                </div>
              </DropdownMenuLabel>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            {/* onClick, não onSelect — @base-ui/react/menu (não Radix): a
                prop que existe de verdade é onClick; onSelect é ignorado
                (não é um evento nativo de clique). Mesmo bug já corrigido
                uma vez nesta sessão — não reintroduzir. */}
            <DropdownMenuItem onClick={() => signOut({ redirectTo: "/login" })}>
              <LogOut />
              Sair
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </header>
  );
}
