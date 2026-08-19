import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { assertDentroDoLimite } from "@/lib/rate-limit";
import { criarUsuarioLocalSchema } from "@/lib/validation/bff-schemas";
import { criarUsuarioLocal, ApiError } from "@/lib/api/client";

/**
 * Login local (usuário/senha) — POST /api/v1/admin/users/local. Só um
 * admin cria contas locais (sem autocadastro público).
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
  // Checagem redundante com o backend (que também exige IsAdmin) — defesa
  // em profundidade, não confiamos só no 403 da API.
  if (!session.user.isAdmin) {
    return NextResponse.json({ error: "usuário não é administrador" }, { status: 403 });
  }

  const resultado = criarUsuarioLocalSchema.safeParse(await request.json().catch(() => null));
  if (!resultado.success) {
    return NextResponse.json(
      { error: "corpo inválido", detalhes: resultado.error.flatten() },
      { status: 400 }
    );
  }

  try {
    const usuario = await criarUsuarioLocal(accessToken, resultado.data);
    return NextResponse.json(usuario, { status: 201 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
