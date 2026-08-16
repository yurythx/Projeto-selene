import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { novaVersaoModeloDocumento, ApiError } from "@/lib/api/client";

const MAX_UPLOAD_BYTES = 20 * 1024 * 1024;

// Publica uma nova versão do arquivo, substituindo a ativa — o histórico
// (versões anteriores) nunca é apagado, ver ModeloDocumentoVersao.
export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const erroOrigem = assertOrigemSegura(request);
  if (erroOrigem) return erroOrigem;

  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isAdmin) {
    return NextResponse.json({ error: "usuário não é administrador" }, { status: 403 });
  }

  const { id } = await params;

  const formData = await request.formData();
  const arquivo = formData.get("arquivo");

  if (!(arquivo instanceof File)) {
    return NextResponse.json({ error: "campo 'arquivo' é obrigatório" }, { status: 400 });
  }
  if (arquivo.size > MAX_UPLOAD_BYTES) {
    return NextResponse.json({ error: "arquivo excede o limite de 20MB" }, { status: 413 });
  }

  try {
    const modelo = await novaVersaoModeloDocumento(accessToken, id, arquivo);
    return NextResponse.json(modelo);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
