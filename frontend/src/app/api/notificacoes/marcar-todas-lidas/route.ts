import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { assertDentroDoLimite } from "@/lib/rate-limit";
import { marcarTodasNotificacoesLidas, ApiError } from "@/lib/api/client";

export async function POST(request: Request) {
  const erroOrigem = assertOrigemSegura(request);
  if (erroOrigem) return erroOrigem;

  const erroLimite = await assertDentroDoLimite(request);
  if (erroLimite) return erroLimite;

  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  try {
    await marcarTodasNotificacoesLidas(accessToken);
    return new NextResponse(null, { status: 204 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
