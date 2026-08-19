import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { assertDentroDoLimite } from "@/lib/rate-limit";
import { testarConexaoDiarioOficial, ApiError } from "@/lib/api/client";

/**
 * Route Handler do BFF: proxy autenticado pro backend Go em
 * POST /api/v1/admin/config/diario-oficial/testar. Não é uma mutação de
 * dados nossos, mas dispara uma requisição de rede de verdade contra um
 * serviço externo — protegido com os mesmos guards de CSRF/rate limit
 * dos handlers de escrita, por precaução.
 */
export async function POST(request: Request) {
  const erroOrigem = assertOrigemSegura(request);
  if (erroOrigem) return erroOrigem;

  const erroLimite = await assertDentroDoLimite(request);
  if (erroLimite) return erroLimite;

  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isAdmin) {
    return NextResponse.json({ error: "usuário não é administrador" }, { status: 403 });
  }

  try {
    const resultado = await testarConexaoDiarioOficial(accessToken);
    return NextResponse.json(resultado);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
