import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { criarContrato, ApiError, type NovoContratoRequest } from "@/lib/api/client";

/**
 * Route Handler do BFF: proxy autenticado pro backend Go em
 * POST /api/v1/contratos.
 *
 * `fiscal_id` é sempre o usuário logado — nunca um valor vindo do corpo da
 * requisição do client. Hoje não existe como um não-admin listar outros
 * fiscais, então esta é a única opção segura pra v1 do formulário; um
 * valor de fiscal_id enviado pelo client é ignorado por completo.
 */
export async function POST(request: Request) {
  const erroOrigem = assertOrigemSegura(request);
  if (erroOrigem) return erroOrigem;

  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  if (!session.user.isFiscal) {
    return NextResponse.json({ error: "usuário não é fiscal" }, { status: 403 });
  }

  const corpo = (await request.json()) as Omit<NovoContratoRequest, "fiscal_id">;

  try {
    const contrato = await criarContrato(accessToken, {
      ...corpo,
      fiscal_id: session.user.id,
    });
    return NextResponse.json(contrato, { status: 201 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
