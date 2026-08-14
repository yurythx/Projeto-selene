import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { atualizarUsuario, ApiError, type AtualizarUsuarioRequest } from "@/lib/api/client";

export async function PATCH(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const erroOrigem = assertOrigemSegura(request);
  if (erroOrigem) return erroOrigem;

  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  // Checagem redundante com o backend (que também exige IsAdmin) — defesa
  // em profundidade, não confiamos só no 403 da API.
  if (!session.user.isAdmin) {
    return NextResponse.json({ error: "usuário não é administrador" }, { status: 403 });
  }

  const { id } = await params;
  const corpo = (await request.json()) as AtualizarUsuarioRequest;

  try {
    const usuario = await atualizarUsuario(accessToken, id, corpo);
    return NextResponse.json(usuario);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
