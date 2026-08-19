import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth-token";
import { listarNotificacoes, ApiError } from "@/lib/api/client";

/**
 * Route Handler do BFF: proxy autenticado pro backend Go em
 * GET /api/v1/notificacoes — notificações da PRÓPRIA conta, sem
 * restrição de admin/fiscal.
 */
export async function GET() {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  try {
    const notificacoes = await listarNotificacoes(accessToken);
    return NextResponse.json(notificacoes);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
