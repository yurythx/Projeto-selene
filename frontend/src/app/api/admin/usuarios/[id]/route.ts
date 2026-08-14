import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { atualizarUsuarioSchema } from "@/lib/validation/bff-schemas";
import { atualizarUsuario, ApiError } from "@/lib/api/client";

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

  const resultado = atualizarUsuarioSchema.safeParse(await request.json().catch(() => null));
  if (!resultado.success) {
    return NextResponse.json(
      { error: "corpo inválido", detalhes: resultado.error.flatten() },
      { status: 400 }
    );
  }

  try {
    const usuario = await atualizarUsuario(accessToken, id, resultado.data);
    return NextResponse.json(usuario);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
