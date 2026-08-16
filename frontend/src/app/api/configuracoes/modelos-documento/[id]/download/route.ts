import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { baixarModeloDocumento, ApiError } from "@/lib/api/client";

// Proxy do .docx da versão ATIVA — o browser nunca chama o backend
// direto. Repassa Content-Type/Content-Disposition REAIS da resposta do
// backend (não hardcoded) — o nome do arquivo original já vem certo de
// lá, sem precisar reconstruir aqui. Consumida via <a href> normal (o
// cookie de sessão viaja junto, sem JS extra no client).
export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  if (!session.user.isAdmin) {
    return NextResponse.json({ error: "usuário não é administrador" }, { status: 403 });
  }

  const { id } = await params;

  try {
    const res = await baixarModeloDocumento(accessToken, id);
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
