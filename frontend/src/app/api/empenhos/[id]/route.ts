import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth-token";
import { buscarEmpenho, ApiError } from "@/lib/api/client";

export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  const { id } = await params;

  try {
    const empenho = await buscarEmpenho(accessToken, id);
    return NextResponse.json(empenho);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
