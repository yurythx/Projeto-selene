import type { Metadata } from "next";
import { headers } from "next/headers";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { auth } from "@/auth";
import { Providers } from "@/components/providers";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Projeto Selene",
  description: "Fiscalização de contratos — compliance documental e Kanban.",
};

export default async function RootLayout({ children }: LayoutProps<"/">) {
  // Passa a sessão já resolvida server-side pro SessionProvider — sem
  // isso, o SessionProvider nasce com loading=true e só fica pronto
  // depois de um fetch client-side próprio a GET /api/auth/session; nesse
  // intervalo, useSession().update() vira um no-op SILENCIOSO (a própria
  // lib checa "if (loading) return" antes de sequer montar a
  // requisição) — descoberto porque TrocarSenhaForm chamava update() logo
  // após o mount, e a claim mustChangePassword nunca era atualizada na
  // sessão. session={session} elimina essa corrida.
  const session = await auth();

  // next-themes injeta um <script> inline no <head> que aplica o tema
  // salvo ANTES do primeiro paint (evita flash de tema errado). A CSP
  // deste app não tem 'unsafe-inline' em script-src (ver o comentário em
  // proxy.ts sobre o nonce por requisição) — sem passar esse nonce pro
  // script do next-themes, o browser bloqueia a execução dele, e o tema
  // salvo só seria aplicado depois da hidratação (flash visível).
  const nonce = (await headers()).get("x-nonce") ?? undefined;

  return (
    <html
      lang="pt-BR"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
      // next-themes decide a classe "dark"/"light" no client (localStorage
      // ou prefers-color-scheme) depois da hidratação — o HTML enviado pelo
      // server nunca pode saber qual é de antemão. Sem isso, o React avisa
      // (incorretamente) de um mismatch de hidratação no atributo class.
      suppressHydrationWarning
    >
      <body className="min-h-full flex flex-col">
        <Providers session={session} themeNonce={nonce}>
          {children}
        </Providers>
      </body>
    </html>
  );
}
