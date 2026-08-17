import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { baixarDocumentoAnexo, excluirDocumento, ApiError } from "@/lib/api/client";

/**
 * Proxy do conteúdo de um documento anexo — consumido via <iframe>/<img>
 * na pré-visualização embutida da página do processo (ver
 * components/kanban/processo-page.tsx). Repassa o Content-Type e o
 * Content-Disposition ("inline", ver o handler Go) tal qual vieram do
 * backend, sem reescrever — diferente de respostaDocumentoGerado (que é
 * específica dos 4 geradores de documento, com a lógica PDF-vs-docx).
 */
export async function GET(_request: Request, { params }: { params: Promise<{ id: string; docId: string }> }) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  const { id, docId } = await params;

  try {
    const res = await baixarDocumentoAnexo(accessToken, id, docId);
    return new NextResponse(res.body, {
      status: 200,
      headers: {
        "Content-Type": res.headers.get("content-type") ?? "application/octet-stream",
        "Content-Disposition": res.headers.get("content-disposition") ?? "inline",
      },
    });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}

export async function DELETE(request: Request, { params }: { params: Promise<{ id: string; docId: string }> }) {
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

  const { id, docId } = await params;

  try {
    await excluirDocumento(accessToken, id, docId);
    return new NextResponse(null, { status: 204 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
