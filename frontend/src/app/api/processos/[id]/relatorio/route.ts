import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth-token";
import { baixarRelatorio, ApiError } from "@/lib/api/client";
import { respostaDocumentoGerado } from "@/lib/documento-response";

// Proxy do PDF do Relatório de Pagamento — o browser nunca chama o backend
// direto, então esta rota existe só pra repassar o binário com o header de
// autorização anexado server-side. Consumida via <a href> normal (o
// cookie de sessão viaja junto, sem JS extra no client).
export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  const { id } = await params;

  try {
    const res = await baixarRelatorio(accessToken, id);
    return respostaDocumentoGerado(res, `relatorio-${id}`);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
