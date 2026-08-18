"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSession, signOut } from "next-auth/react";
import {
  LayoutDashboard,
  KanbanSquare,
  FileText,
  Radar as RadarIcon,
  Building2,
  ShieldCheck,
  Settings,
  LogOut,
  X,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
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
import { useMontado } from "@/lib/use-montado";
import { useSidebarContext } from "@/components/sidebar-context";

const NAV_ITEMS = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/kanban", label: "Kanban", icon: KanbanSquare },
  { href: "/contratos", label: "Contratos", icon: FileText },
  { href: "/radar", label: "Radar", icon: RadarIcon },
  { href: "/fornecedores", label: "Fornecedores", icon: Building2 },
];

// Cor de marca única e consistente (estilo Monday.com: uma cor vibrante
// marca o item ativo, independente do tema) — âmbar (#fbbf24), o accent
// real da marca PaperMoon (ver globals.css, extraído do CSS compilado de
// papermoon.cloud), a MESMA cor exata nos dois temas — antes daqui era
// um índigo/violeta sem relação nenhuma com a marca real. Texto escuro
// (não branco) sobre o accent: âmbar é uma cor clara, branco nela
// reprova contraste — mesmo padrão usado nos botões reais do
// papermoon.cloud. shadow-lg shadow-amber-400/30 é o halo/glow colorido
// visto de verdade no CTA principal deles (`shadow-lg
// shadow-glow-accent`) — shadow-sm sozinho (usado antes) só dava um
// relevo neutro de "pílula", sem o brilho de marca que salta aos olhos
// no site real.
const ITEM_ATIVO = "bg-amber-400 text-slate-900 shadow-lg shadow-amber-400/30";
const ITEM_INATIVO =
  "text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground";

const CHAVE_COLAPSADO = "selene:sidebar:colapsado";

function lerColapsadoSalvo(): boolean {
  return localStorage.getItem(CHAVE_COLAPSADO) === "1";
}

/**
 * Barra de navegação LATERAL (estilo Monday.com). Três estados:
 *
 * - Desktop (md+), expandida: 256px, ícone + rótulo, é o default.
 * - Desktop (md+), colapsada: 64px, só ícone (com tooltip nativo via
 *   `title`) — alternado pelo botão-pílula preso na borda direita,
 *   preferência persistida em localStorage (mesmo truque de useMontado
 *   usado em kanban-board.tsx pra visualização Kanban/Lista, evita
 *   mismatch de hidratação).
 * - Mobile (< md): vira um drawer fora do fluxo (`fixed`), escondido por
 *   padrão (`-translate-x-full`) e aberto pelo hambúrguer em
 *   MobileTopBar — os dois compartilham o estado via SidebarProvider
 *   (ver sidebar-context.tsx). Sempre em largura total quando aberta,
 *   independente da preferência de colapso do desktop (64px não faz
 *   sentido num drawer que já é overlay, não ocupa espaço permanente).
 */
export function Sidebar() {
  const pathname = usePathname();
  const { data: session } = useSession();
  const { mobileOpen, setMobileOpen } = useSidebarContext();
  const [colapsadoEscolhido, setColapsadoEscolhido] = useState<boolean | null>(null);
  const montado = useMontado();
  const colapsado = colapsadoEscolhido ?? (montado ? lerColapsadoSalvo() : false);
  const mostrarRotulos = !colapsado || mobileOpen;

  function alternarColapsado() {
    const novo = !colapsado;
    setColapsadoEscolhido(novo);
    localStorage.setItem(CHAVE_COLAPSADO, novo ? "1" : "0");
  }

  const iniciais = (session?.user?.name ?? session?.user?.email ?? "?")
    .trim()
    .split(/\s+/)
    .map((parte) => parte[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();

  // Anel de foco em duas camadas (ring + ring-offset), mesma receita dos
  // links reais de papermoon.cloud (`focus-visible:ring-2
  // focus-visible:ring-border-focus focus-visible:ring-offset-2
  // focus-visible:ring-offset-surface-0`, confirmado no HTML renderizado
  // do header deles) — antes esses links não tinham NENHUM anel próprio,
  // só o outline azul default do navegador.
  const itemClasse = (ativo: boolean) =>
    cn(
      "flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-sidebar",
      !mostrarRotulos && "justify-center px-0",
      ativo ? ITEM_ATIVO : ITEM_INATIVO
    );

  return (
    <>
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/40 md:hidden"
          onClick={() => setMobileOpen(false)}
          aria-hidden="true"
        />
      )}

      <aside
        className={cn(
          "bg-sidebar text-sidebar-foreground border-sidebar-border fixed inset-y-0 left-0 z-50 flex h-screen w-64 shrink-0 flex-col border-r shadow-xl transition-transform duration-200 md:sticky md:top-0 md:translate-x-0 md:shadow-none",
          mobileOpen ? "translate-x-0" : "-translate-x-full",
          colapsado ? "md:w-16" : "md:w-64"
        )}
      >
        {/* Botão de colapsar — só desktop, "pílula" presa na borda direita
            da sidebar (padrão Notion/Linear), fora do fluxo interno pra
            não brigar por espaço com o cabeçalho quando colapsada. */}
        <button
          type="button"
          onClick={alternarColapsado}
          aria-label={colapsado ? "Expandir barra lateral" : "Recolher barra lateral"}
          className="border-sidebar-border bg-sidebar text-sidebar-foreground/60 hover:text-sidebar-foreground absolute top-16 -right-3 z-10 hidden size-6 items-center justify-center rounded-full border shadow-sm outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-sidebar active:translate-y-px md:flex"
        >
          {colapsado ? <ChevronRight className="size-3.5" /> : <ChevronLeft className="size-3.5" />}
        </button>

        {/* h-16 (64px), não h-14 — mesma altura de cabeçalho do header real
            de papermoon.cloud (`h-16`, confirmado no HTML deles). */}
        <div className="border-sidebar-border flex h-16 shrink-0 items-center justify-between gap-2 border-b px-4">
          <div className={cn("flex items-center gap-2 overflow-hidden", !mostrarRotulos && "w-full justify-center")}>
            {/* Gradiente + glow + anel interno — mesma "textura de marca"
                da marca-d'água real de papermoon.cloud (o "P" deles tem
                gradiente diagonal, halo e um stroke sutil, não é uma cor
                chapada única). Não copiamos o desenho deles (é a marca
                deles), só a LINGUAGEM visual do selo. */}
            <div className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-amber-300 to-amber-500 text-sm font-bold text-slate-900 shadow-lg shadow-amber-400/40 ring-1 ring-white/25">
              S
            </div>
            {/* text-base font-bold tracking-tight — mesmo tratamento da
                wordmark "PaperMoon" no header deles (era só font-semibold,
                sem tracking-tight nem tamanho explícito). */}
            {mostrarRotulos && (
              <span className="truncate text-base font-bold tracking-tight">Selene</span>
            )}
          </div>
          <button
            type="button"
            onClick={() => setMobileOpen(false)}
            aria-label="Fechar menu"
            className="hover:bg-sidebar-accent flex size-8 shrink-0 items-center justify-center rounded-lg outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-sidebar active:translate-y-px md:hidden"
          >
            <X className="size-4" />
          </button>
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-3">
          {NAV_ITEMS.map((item) => {
            const ativo = pathname.startsWith(item.href);
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                aria-label={item.label}
                title={!mostrarRotulos ? item.label : undefined}
                onClick={() => setMobileOpen(false)}
                className={itemClasse(ativo)}
              >
                <Icon className="size-4 shrink-0" />
                {mostrarRotulos && item.label}
              </Link>
            );
          })}

          {session?.user?.isAdmin && (
            <>
              {mostrarRotulos ? (
                <div
                  className="text-sidebar-foreground/40 px-3 pt-4 pb-1 text-xs font-semibold tracking-wide uppercase"
                  aria-hidden="true"
                >
                  Admin
                </div>
              ) : (
                <div className="border-sidebar-border mx-2 my-3 border-t" aria-hidden="true" />
              )}
              {/* Rótulo "Administração" preservado ao pé da letra — é o
                  accessible name que e2e/admin.spec.ts procura via
                  getByRole("link", {name: "Administração"}); aria-label
                  explícito garante o mesmo nome acessível mesmo colapsada
                  (sem o texto visível). */}
              <Link
                href="/admin/usuarios"
                aria-label="Administração"
                title={!mostrarRotulos ? "Administração" : undefined}
                onClick={() => setMobileOpen(false)}
                className={itemClasse(pathname.startsWith("/admin"))}
              >
                <ShieldCheck className="size-4 shrink-0" />
                {mostrarRotulos && "Administração"}
              </Link>
              <Link
                href="/configuracoes"
                aria-label="Configurações"
                title={!mostrarRotulos ? "Configurações" : undefined}
                onClick={() => setMobileOpen(false)}
                className={itemClasse(pathname.startsWith("/configuracoes"))}
              >
                <Settings className="size-4 shrink-0" />
                {mostrarRotulos && "Configurações"}
              </Link>
            </>
          )}
        </nav>

        <div
          className={cn(
            "border-sidebar-border flex shrink-0 items-center border-t p-3",
            mostrarRotulos ? "justify-between" : "flex-col gap-2"
          )}
        >
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
              <DropdownMenuContent align={mostrarRotulos ? "end" : "center"} side="top">
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
                {/* onClick, não onSelect — @base-ui/react/menu (não Radix):
                    a prop que existe de verdade é onClick; onSelect é
                    ignorado (não é um evento nativo de clique). Mesmo bug
                    já corrigido uma vez nesta sessão — não reintroduzir. */}
                <DropdownMenuItem onClick={() => signOut({ redirectTo: "/login" })}>
                  <LogOut />
                  Sair
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </aside>
    </>
  );
}
