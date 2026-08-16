import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { gerarNotificacaoSchema } from "@/lib/validation/bff-schemas";
import { gerarNotificacao, ApiError } from "@/lib/api/client";
import { respostaDocumentoGerado } from "@/lib/documento-response";

/**
 * Proxy do PDF da Notificação de Descumprimento (Módulo 2 do roadmap) —
 * mesmo padrão de app/api/processos/[id]/relatorio/route.ts, mas via POST
 * (o motivo vai no corpo).
 */
export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
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

  const { id } = await params;

  const resultado = gerarNotificacaoSchema.safeParse(await request.json().catch(() => null));
  if (!resultado.success) {
    return NextResponse.json(
      { error: "corpo inválido", detalhes: resultado.error.flatten() },
      { status: 400 }
    );
  }

  try {
    const res = await gerarNotificacao(accessToken, id, resultado.data.motivo);
    return respostaDocumentoGerado(res, `notificacao-${id}`);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
