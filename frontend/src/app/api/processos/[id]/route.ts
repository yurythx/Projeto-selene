import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth-token";
import { buscarProcesso, ApiError } from "@/lib/api/client";

/**
 * Antes do SGF-Rondonópolis, nenhum client component precisava buscar um
 * processo isolado por ID — o drawer (ProcessoDialog) recebia o objeto já
 * pronto via prop, vindo da lista carregada server-side. Isso mudou:
 * allowed_actions/estado_fiscalizacao/acao_ou_espera só existem na
 * resposta de GET /processos/{id} (ver
 * backend/internal/service/fiscalizacao_service.go), não na listagem —
 * então o drawer agora refaz esse fetch ao abrir (ver processo-dialog.tsx).
 */
export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  const { id } = await params;

  try {
    const processo = await buscarProcesso(accessToken, id);
    return NextResponse.json(processo);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
