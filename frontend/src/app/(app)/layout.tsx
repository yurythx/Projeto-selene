import { Sidebar } from "@/components/sidebar";
import { SidebarProvider } from "@/components/sidebar-context";
import { TopBar } from "@/components/top-bar";
import { SessionErrorWatcher } from "@/components/session-error-watcher";

// Layout do grupo de rotas autenticadas — /login fica fora deste grupo e
// não recebe a Sidebar. A checagem de autenticação em si é feita no
// proxy.ts (nível de rota) e em cada Server Component (auth()/
// getAccessToken()).
//
// Duas barras com responsabilidades separadas (pedido explícito do
// usuário): a Sidebar é só o MENU (ver sidebar.tsx pro detalhe dos 3
// estados responsivos); a TopBar é persistente em qualquer largura de
// tela e carrega o tema + o usuário logado — isso morava no rodapé da
// Sidebar antes, foi movido pra cá. SidebarProvider existe só pra
// compartilhar "drawer mobile aberto" entre Sidebar e TopBar, que vivem
// em galhos irmãos desta árvore.
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <SidebarProvider>
      <div className="flex min-h-screen">
        <Sidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <TopBar />
          <SessionErrorWatcher />
          <main className="mx-auto w-full max-w-[1600px] flex-1 px-4 py-6 sm:px-6 lg:px-8">
            {children}
          </main>
        </div>
      </div>
    </SidebarProvider>
  );
}
