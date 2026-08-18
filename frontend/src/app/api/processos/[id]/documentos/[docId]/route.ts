import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { baixarDocumentoAnexo, excluirDocumento, ApiError } from "@/lib/api/client";

/**
 * Proxy do conteúdo de um documento anexo — o link "Visualizar" na
 * página do processo (ver components/kanban/processo-page.tsx) aponta
 * direto pra cá com target="_blank", abrindo numa aba nova (pedido
 * explícito do usuário: "quero o mais rápido, por mais que precise abrir
 * outra aba" — o visualizador nativo do navegador é mais rápido de
 * exibir do que um visualizador embutido em JS). Repassa o Content-Type
 * e o Content-Disposition ("inline", ver o handler Go) tal qual vieram
 * do backend, sem reescrever — diferente de respostaDocumentoGerado (que
 * é específica dos 4 geradores de documento, com a lógica PDF-vs-docx).
 *
 * Também repassa o cache condicional de ponta a ponta (otimização pedida
 * pelo usuário): o If-None-Match que o navegador manda automaticamente
 * (porque o backend respondeu ETag + Cache-Control "immutable" da
 * primeira vez) é encaminhado pro backend Go; se ele responder 304, este
 * proxy também responde 304 sem corpo, em vez de sempre buscar e
 * retransmitir o arquivo inteiro de novo a cada abertura da
 * pré-visualização.
 */
export async function GET(request: Request, { params }: { params: Promise<{ id: string; docId: string }> }) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  const { id, docId } = await params;

  try {
    const ifNoneMatch = request.headers.get("if-none-match");
    const res = await baixarDocumentoAnexo(accessToken, id, docId, ifNoneMatch);

    const headers: Record<string, string> = {};
    const cacheControl = res.headers.get("cache-control");
    const etag = res.headers.get("etag");
    if (cacheControl) headers["Cache-Control"] = cacheControl;
    if (etag) headers["ETag"] = etag;

    if (res.status === 304) {
      return new NextResponse(null, { status: 304, headers });
    }

    headers["Content-Type"] = res.headers.get("content-type") ?? "application/octet-stream";
    headers["Content-Disposition"] = res.headers.get("content-disposition") ?? "inline";
    return new NextResponse(res.body, { status: 200, headers });
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
