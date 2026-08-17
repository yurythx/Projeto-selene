import { Sidebar } from "@/components/sidebar";
import { SidebarProvider } from "@/components/sidebar-context";
import { MobileTopBar } from "@/components/mobile-top-bar";
import { SessionErrorWatcher } from "@/components/session-error-watcher";

// Layout do grupo de rotas autenticadas — /login fica fora deste grupo e
// não recebe a Sidebar. A checagem de autenticação em si é feita no
// proxy.ts (nível de rota) e em cada Server Component (auth()/
// getAccessToken()).
//
// Responsivo em 3 estados (ver sidebar.tsx pro detalhe de cada um):
// desktop expandida, desktop colapsada (só ícone) e mobile (drawer fora
// do fluxo, aberto pelo hambúrguer de MobileTopBar). SidebarProvider
// existe só pra compartilhar "drawer mobile aberto" entre Sidebar e
// MobileTopBar, que vivem em galhos irmãos desta árvore.
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <SidebarProvider>
      <div className="flex min-h-screen">
        <Sidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <MobileTopBar />
          <SessionErrorWatcher />
          <main className="mx-auto w-full max-w-[1600px] flex-1 px-4 py-6 sm:px-6 lg:px-8">
            {children}
          </main>
        </div>
      </div>
    </SidebarProvider>
  );
}
