import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { criarProcesso, ApiError } from "@/lib/api/client";

export async function POST(request: Request) {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isFiscal) {
    return NextResponse.json({ error: "usuário não é fiscal" }, { status: 403 });
  }

  const corpo = (await request.json()) as { contrato_id?: string; mes_referencia?: string };
  if (!corpo.contrato_id || !corpo.mes_referencia) {
    return NextResponse.json({ error: "contrato_id e mes_referencia são obrigatórios" }, { status: 400 });
  }

  try {
    const processo = await criarProcesso(accessToken, corpo.contrato_id, corpo.mes_referencia);
    return NextResponse.json(processo, { status: 201 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
