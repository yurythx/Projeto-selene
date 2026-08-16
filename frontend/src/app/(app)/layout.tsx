import { Sidebar } from "@/components/sidebar";
import { SessionErrorWatcher } from "@/components/session-error-watcher";

// Layout do grupo de rotas autenticadas — /login fica fora deste grupo e
// não recebe a Sidebar. A checagem de autenticação em si é feita no
// proxy.ts (nível de rota) e em cada Server Component (auth()/
// getAccessToken()). Sidebar fixa à esquerda (estilo Monday.com,
// substituiu a nav horizontal) + área de conteúdo rolável à direita.
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <SessionErrorWatcher />
        <main className="mx-auto w-full max-w-[1600px] flex-1 px-4 py-6 sm:px-6 lg:px-8">
          {children}
        </main>
      </div>
    </div>
  );
}
