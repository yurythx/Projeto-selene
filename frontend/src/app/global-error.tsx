"use client";

// Boundary de último recurso — só dispara se o próprio RootLayout (ou o
// que está fora do grupo (app), como Providers) falhar. Precisa renderizar
// <html>/<body> própria porque substitui o layout raiz inteiro.
export default function GlobalError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="pt-BR">
      <body>
        <div style={{ display: "flex", minHeight: "100vh", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: "1rem", textAlign: "center", fontFamily: "system-ui, sans-serif" }}>
          <div>
            <h1 style={{ fontSize: "1.25rem", fontWeight: 600 }}>Algo deu errado</h1>
            <p style={{ color: "#666", marginTop: "0.25rem", fontSize: "0.875rem" }}>
              Tente recarregar a página.
            </p>
          </div>
          <button
            onClick={reset}
            style={{ padding: "0.5rem 1rem", borderRadius: "0.375rem", background: "#111", color: "#fff", border: "none", cursor: "pointer" }}
          >
            Tentar de novo
          </button>
        </div>
      </body>
    </html>
  );
}
