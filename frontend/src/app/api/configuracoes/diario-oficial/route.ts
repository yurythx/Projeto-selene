import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { assertDentroDoLimite } from "@/lib/rate-limit";
import { atualizarDiarioOficialConfigSchema } from "@/lib/validation/bff-schemas";
import {
  buscarConfiguracaoDiarioOficial,
  atualizarConfiguracaoDiarioOficial,
  ApiError,
} from "@/lib/api/client";

/**
 * Route Handler do BFF: proxy autenticado pro backend Go em
 * GET/PUT /api/v1/admin/config/diario-oficial. Restrito a admin —
 * Configurações → Diário Oficial. ESTRUTURA GENÉRICA (decisão de escopo
 * confirmada com o usuário) — ver o comentário no topo de
 * backend/internal/service/diario_oficial_service.go.
 */
export async function GET() {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isAdmin) {
    return NextResponse.json({ error: "usuário não é administrador" }, { status: 403 });
  }

  try {
    const cfg = await buscarConfiguracaoDiarioOficial(accessToken);
    return NextResponse.json(cfg);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}

export async function PUT(request: Request) {
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

  const resultado = atualizarDiarioOficialConfigSchema.safeParse(await request.json().catch(() => null));
  if (!resultado.success) {
    return NextResponse.json(
      { error: "corpo inválido", detalhes: resultado.error.flatten() },
      { status: 400 }
    );
  }

  try {
    const cfg = await atualizarConfiguracaoDiarioOficial(accessToken, resultado.data);
    return NextResponse.json(cfg);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
