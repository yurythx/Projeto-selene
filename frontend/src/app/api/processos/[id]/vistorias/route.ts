import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { assertDentroDoLimite } from "@/lib/rate-limit";
import { registrarVistoriaSchema } from "@/lib/validation/bff-schemas";
import { listarVistorias, registrarVistoria, ApiError } from "@/lib/api/client";

export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  const { id } = await params;

  try {
    const vistorias = await listarVistorias(accessToken, id);
    return NextResponse.json(vistorias);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}

export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const erroOrigem = assertOrigemSegura(request);
  if (erroOrigem) return erroOrigem;

  const erroLimite = await assertDentroDoLimite(request);
  if (erroLimite) return erroLimite;

  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isFiscal) {
    return NextResponse.json({ error: "usuário não é fiscal" }, { status: 403 });
  }

  const { id } = await params;

  const resultado = registrarVistoriaSchema.safeParse(await request.json().catch(() => null));
  if (!resultado.success) {
    return NextResponse.json(
      { error: "corpo inválido", detalhes: resultado.error.flatten() },
      { status: 400 }
    );
  }

  try {
    const vistoria = await registrarVistoria(accessToken, id, resultado.data);
    return NextResponse.json(vistoria, { status: 201 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
