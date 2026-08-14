import { Nav } from "@/components/nav";

// Layout do grupo de rotas autenticadas — /login fica fora deste grupo e
// não recebe o Nav. A checagem de autenticação em si é feita no proxy.ts
// (nível de rota) e em cada Server Component (auth()/getAccessToken()).
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <Nav />
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">{children}</main>
    </>
  );
}
