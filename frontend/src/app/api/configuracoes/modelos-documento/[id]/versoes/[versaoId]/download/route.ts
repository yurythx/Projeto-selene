import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { baixarVersaoModeloDocumento, ApiError } from "@/lib/api/client";

// Mesmo proxy de .../download, mas pra uma versão específica do
// histórico — ver o comentário lá.
export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string; versaoId: string }> }
) {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isAdmin) {
    return NextResponse.json({ error: "usuário não é administrador" }, { status: 403 });
  }

  const { id, versaoId } = await params;

  try {
    const res = await baixarVersaoModeloDocumento(accessToken, id, versaoId);
    return new NextResponse(res.body, {
      status: 200,
      headers: {
        "Content-Type":
          res.headers.get("content-type") ??
          "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        "Content-Disposition": res.headers.get("content-disposition") ?? "attachment",
      },
    });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
