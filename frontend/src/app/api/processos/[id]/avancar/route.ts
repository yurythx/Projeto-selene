import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { avancarProcesso, ApiError } from "@/lib/api/client";

export async function POST(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isFiscal) {
    return NextResponse.json({ error: "usuário não é fiscal" }, { status: 403 });
  }

  const { id } = await params;

  try {
    const processo = await avancarProcesso(accessToken, id);
    return NextResponse.json(processo);
  } catch (erro) {
    if (erro instanceof ApiError) {
      // 422 (checklist incompleto) chega aqui com o body original —
      // { error, documentos_pendentes } — repassado como está pro client.
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
