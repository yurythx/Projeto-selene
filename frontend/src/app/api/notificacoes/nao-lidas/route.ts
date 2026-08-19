import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth-token";
import { contarNotificacoesNaoLidas, ApiError } from "@/lib/api/client";

/**
 * Route Handler do BFF: proxy autenticado pro backend Go em
 * GET /api/v1/notificacoes/nao-lidas — badge de contagem da TopBar.
 */
export async function GET() {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  try {
    const resultado = await contarNotificacoesNaoLidas(accessToken);
    return NextResponse.json(resultado);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
