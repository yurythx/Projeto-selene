import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth-token";
import { listarRadar, ApiError } from "@/lib/api/client";

/**
 * Route Handler do BFF: proxy autenticado pro backend Go em
 * GET /api/v1/radar. Sem gate de role — qualquer usuário autenticado pode
 * ver o radar (mesma política da rota do backend).
 */
export async function GET() {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  try {
    const itens = await listarRadar(accessToken);
    return NextResponse.json(itens);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
