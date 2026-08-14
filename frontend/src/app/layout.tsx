import type { Metadata } from "next";
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

  return (
    <html
      lang="pt-BR"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">
        <Providers session={session}>{children}</Providers>
      </body>
    </html>
  );
}
