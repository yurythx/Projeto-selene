import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { buscarContratosDiarioOficialSchema } from "@/lib/validation/bff-schemas";
import { buscarContratosDiarioOficial, ApiError } from "@/lib/api/client";

/**
 * Route Handler do BFF: proxy autenticado pro backend Go em
 * GET /api/v1/admin/diario-oficial/contratos?nome=&cpf=&data=. Restrito
 * a admin — mesmo padrão de GET .../processos (leitura, sem checagem de
 * Origin/rate-limit), mas com checagem de isAdmin redundante com o
 * backend porque esta é uma tela de admin, não de fiscal comum.
 */
export async function GET(request: Request) {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isAdmin) {
    return NextResponse.json({ error: "usuário não é administrador" }, { status: 403 });
  }

  const { searchParams } = new URL(request.url);
  const resultado = buscarContratosDiarioOficialSchema.safeParse({
    nome: searchParams.get("nome") ?? undefined,
    cpf: searchParams.get("cpf") ?? undefined,
    data: searchParams.get("data") ?? undefined,
  });
  if (!resultado.success) {
    return NextResponse.json(
      { error: "parâmetros inválidos", detalhes: resultado.error.flatten() },
      { status: 400 }
    );
  }

  try {
    const busca = await buscarContratosDiarioOficial(accessToken, resultado.data);
    return NextResponse.json(busca);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
