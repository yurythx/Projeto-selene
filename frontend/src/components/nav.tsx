"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSession, signOut } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  { href: "/kanban", label: "Kanban" },
  { href: "/contratos", label: "Contratos" },
  { href: "/radar", label: "Radar" },
  { href: "/fornecedores", label: "Fornecedores" },
];

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
      <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
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

        {session?.user && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button variant="ghost" size="icon" className="rounded-full">
                  <Avatar className="size-8">
                    <AvatarFallback>{iniciais}</AvatarFallback>
                  </Avatar>
                </Button>
              }
            />
            <DropdownMenuContent align="end">
              <DropdownMenuLabel className="font-normal">
                <div className="flex flex-col space-y-1">
                  <p className="text-sm font-medium leading-none">{session.user.name}</p>
                  <p className="text-muted-foreground text-xs leading-none">{session.user.email}</p>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={() => signOut({ redirectTo: "/login" })}>
                Sair
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </header>
  );
}
