import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { assertDentroDoLimite } from "@/lib/rate-limit";
import { trocarSenhaSchema } from "@/lib/validation/bff-schemas";
import { trocarSenha, ApiError } from "@/lib/api/client";

/**
 * Troca a senha da PRÓPRIA conta autenticada — POST
 * /api/v1/auth/trocar-senha. Qualquer usuário logado (local ou Keycloak;
 * contas Keycloak recebem 400 do backend, a senha delas não é gerenciada
 * pelo Selene).
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

  const resultado = trocarSenhaSchema.safeParse(await request.json().catch(() => null));
  if (!resultado.success) {
    return NextResponse.json(
      { error: "corpo inválido", detalhes: resultado.error.flatten() },
      { status: 400 }
    );
  }

  try {
    await trocarSenha(accessToken, resultado.data.senha_atual, resultado.data.senha_nova);
    return new NextResponse(null, { status: 204 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
