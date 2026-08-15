import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth-token";
import { listarServidores, ApiError } from "@/lib/api/client";

/**
 * Route Handler do BFF: proxy autenticado pra GET /api/v1/servidores.
 * Só exige sessão válida (nem is_fiscal, nem is_admin) — o backend já
 * não restringe esta rota, ver o comentário em UserHandler.ListarServidores.
 * Usado pelo seletor de servidor de "Nova designação" (SGF-Rondonópolis).
 */
export async function GET() {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  try {
    const servidores = await listarServidores(accessToken);
    return NextResponse.json(servidores);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
