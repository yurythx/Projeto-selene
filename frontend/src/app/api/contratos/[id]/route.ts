import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { atualizarContrato, ApiError, type AtualizarContratoRequest } from "@/lib/api/client";

export async function PATCH(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isFiscal) {
    return NextResponse.json({ error: "usuário não é fiscal" }, { status: 403 });
  }

  const { id } = await params;
  const corpo = (await request.json()) as AtualizarContratoRequest;

  try {
    const contrato = await atualizarContrato(accessToken, id, corpo);
    return NextResponse.json(contrato);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
