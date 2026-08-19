import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth-token";
import { baixarRelatorioCampo, ApiError } from "@/lib/api/client";

/** Proxy do PDF do Relatório de Campo (Módulo 3 do roadmap). */
export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  const { id } = await params;

  try {
    const res = await baixarRelatorioCampo(accessToken, id);
    return new NextResponse(res.body, {
      status: 200,
      headers: {
        "Content-Type": "application/pdf",
        "Content-Disposition": `inline; filename="relatorio-campo-${id}.pdf"`,
      },
    });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
