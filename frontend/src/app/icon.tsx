import { ImageResponse } from "next/og";

// Favicon dinâmico (convenção de arquivo do App Router — Next.js gera a
// rota /icon automaticamente e injeta o <link rel="icon"> sozinho, sem
// precisar de metadata.icons nem de um .ico estático). Pedido explícito
// do usuário: favicon com a letra "S" no MESMO estilo do selo usado na
// sidebar/TopBar (ver sidebar.tsx) — gradiente diagonal âmbar
// (from-amber-300 to-amber-500, os mesmos #fcd34d/#f59e0b do Tailwind)
// sobre fundo escuro (midnight, igual ao --primary-foreground do tema),
// pra não depender de editor de imagem externo: é JSX renderizado pro
// PNG em build/request time.
//
// ACHADO DE REVISÃO: essa rota "/icon" tinha que entrar na exclusão do
// matcher de proxy.ts (mesmo grupo de "favicon.ico", "_next/static" etc)
// — sem isso, o middleware de autenticação tratava CADA pedido do ícone
// do navegador como uma rota protegida e respondia 307 pro /login (nunca
// servindo o PNG, e ainda gerando cookie de CSRF à toa em todo
// carregamento de página). Corrigido junto com este arquivo.
export const size = { width: 32, height: 32 };
export const contentType = "image/png";

export default function Icon() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          borderRadius: 7,
          background: "linear-gradient(135deg, #fcd34d 0%, #f59e0b 100%)",
          color: "#0f172a",
          fontSize: 21,
          fontWeight: 700,
          fontFamily: "system-ui, sans-serif",
        }}
      >
        S
      </div>
    ),
    { ...size }
  );
}
