"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSession, signOut } from "next-auth/react";
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
import { cn } from "@/lib/utils";
import { ThemeToggle } from "@/components/theme-toggle";

const NAV_ITEMS = [
  { href: "/kanban", label: "Kanban" },
  { href: "/contratos", label: "Contratos" },
  { href: "/radar", label: "Radar" },
  { href: "/fornecedores", label: "Fornecedores" },
];

/**
 * Barra de navegação do topo (grupo de rotas autenticadas `(app)`) — link
 * "Administração" só aparece pra `session.user.isAdmin`. Avatar mostra as
 * iniciais do nome (ou e-mail, se o nome não vier na sessão) e um menu
 * com "Sair".
 */
export function Nav() {
  const pathname = usePathname();
  const { data: session } = useSession();

  const iniciais = (session?.user?.name ?? session?.user?.email ?? "?")
    .trim()
    .split(/\s+/)
    .map((parte) => parte[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();

  return (
    <header className="border-b">
      <div className="mx-auto flex h-14 max-w-[1600px] items-center justify-between px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-6">
          <span className="font-semibold">Selene</span>
          <nav className="flex items-center gap-4 text-sm">
            {NAV_ITEMS.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "text-muted-foreground hover:text-foreground transition-colors",
                  pathname.startsWith(item.href) && "text-foreground font-medium"
                )}
              >
                {item.label}
              </Link>
            ))}
            {session?.user?.isAdmin && (
              <Link
                href="/admin/usuarios"
                className={cn(
                  "text-muted-foreground hover:text-foreground transition-colors",
                  pathname.startsWith("/admin") && "text-foreground font-medium"
                )}
              >
                Administração
              </Link>
            )}
          </nav>
        </div>

        <div className="flex items-center gap-2">
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
              <DropdownMenuContent align="end">
                {/* DropdownMenuLabel (Menu.GroupLabel do base-ui) exige um
                    Menu.Group ancestral — sem ele, useMenuGroupRootContext()
                    lança "Base UI error #31: MenuGroupContext is missing" e
                    quebra a página inteira (ErrorBoundary) toda vez que
                    alguém abre este menu. Achado testando o toggle de tema
                    ao lado — o mesmo clique que expôs o bug de onSelect vs
                    onClick também revelou este. */}
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
                {/* onClick, não onSelect — DropdownMenuItem é @base-ui/react/menu
                  (não Radix): a prop que existe de verdade é onClick (ver
                  MenuItemProps); onSelect é ignorado (não é um evento nativo
                  de clique), então este botão nunca chamava signOut() antes
                  desta correção. Achado ao testar o toggle de tema abaixo. */}
              <DropdownMenuItem onClick={() => signOut({ redirectTo: "/login" })}>
                  Sair
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>
    </header>
  );
}
