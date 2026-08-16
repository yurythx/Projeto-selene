"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSession, signOut } from "next-auth/react";
import { KanbanSquare, FileText, Radar as RadarIcon, Building2, ShieldCheck, Settings, LogOut } from "lucide-react";
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
  { href: "/kanban", label: "Kanban", icon: KanbanSquare },
  { href: "/contratos", label: "Contratos", icon: FileText },
  { href: "/radar", label: "Radar", icon: RadarIcon },
  { href: "/fornecedores", label: "Fornecedores", icon: Building2 },
];

// Cor de marca única e consistente (estilo Monday.com: um roxo/índigo
// vibrante marca o item ativo, independente do tema) — mesmo tom já
// usado como --sidebar-primary no modo escuro em globals.css
// (oklch(0.488 0.243 264.376) ≈ faixa índigo/violeta), aplicado aqui
// explicitamente pra funcionar igual nos dois temas (o token claro é só
// cinza neutro, sem "cor de marca" de verdade).
const ITEM_ATIVO = "bg-indigo-600 text-white dark:bg-indigo-500";
const ITEM_INATIVO =
  "text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground";

/**
 * Barra de navegação LATERAL (estilo Monday.com) — substitui a antiga nav
 * horizontal (ver histórico do componente Nav). Fixa à esquerda, cor de
 * marca única no item ativo, avatar/tema na base. "Administração" e
 * "Configurações" só aparecem pra `session.user.isAdmin`.
 */
export function Sidebar() {
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
    <aside className="bg-sidebar text-sidebar-foreground border-sidebar-border sticky top-0 flex h-screen w-60 shrink-0 flex-col border-r">
      <div className="flex h-14 items-center gap-2 px-4">
        <div className="flex size-7 items-center justify-center rounded-lg bg-indigo-600 text-sm font-bold text-white dark:bg-indigo-500">
          S
        </div>
        <span className="font-semibold">Selene</span>
      </div>

      <nav className="flex-1 space-y-1 px-3 py-2">
        {NAV_ITEMS.map((item) => {
          const ativo = pathname.startsWith(item.href);
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                ativo ? ITEM_ATIVO : ITEM_INATIVO
              )}
            >
              <Icon className="size-4 shrink-0" />
              {item.label}
            </Link>
          );
        })}

        {session?.user?.isAdmin && (
          <>
            <div
              className="text-sidebar-foreground/40 px-3 pt-4 pb-1 text-xs font-semibold tracking-wide uppercase"
              aria-hidden="true"
            >
              Admin
            </div>
            {/* Rótulo "Administração" preservado ao pé da letra — é o
                accessible name que e2e/admin.spec.ts procura via
                getByRole("link", {name: "Administração"}). */}
            <Link
              href="/admin/usuarios"
              className={cn(
                "flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                pathname.startsWith("/admin") ? ITEM_ATIVO : ITEM_INATIVO
              )}
            >
              <ShieldCheck className="size-4 shrink-0" />
              Administração
            </Link>
            <Link
              href="/configuracoes/modelos-documentos"
              className={cn(
                "flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                pathname.startsWith("/configuracoes") ? ITEM_ATIVO : ITEM_INATIVO
              )}
            >
              <Settings className="size-4 shrink-0" />
              Configurações
            </Link>
          </>
        )}
      </nav>

      <div className="border-sidebar-border flex items-center justify-between border-t p-3">
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
            <DropdownMenuContent align="end" side="top">
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
              {/* onClick, não onSelect — DropdownMenuItem é @base-ui/react/menu
                  (não Radix): a prop que existe de verdade é onClick; onSelect
                  é ignorado (não é um evento nativo de clique). Mesmo bug já
                  corrigido uma vez nesta sessão — não reintroduzir. */}
              <DropdownMenuItem onClick={() => signOut({ redirectTo: "/login" })}>
                <LogOut />
                Sair
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </aside>
  );
}
