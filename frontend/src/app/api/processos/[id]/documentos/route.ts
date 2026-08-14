import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { listarDocumentos, anexarDocumento, ApiError } from "@/lib/api/client";

export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  const { id } = await params;

  try {
    const documentos = await listarDocumentos(accessToken, id);
    return NextResponse.json(documentos);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}

export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
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

  const { id } = await params;

  const formData = await request.formData();
  const tipoDocumentoId = Number(formData.get("tipo_documento_id"));
  const arquivo = formData.get("arquivo");

  if (!tipoDocumentoId || !(arquivo instanceof File)) {
    return NextResponse.json(
      { error: "campos 'tipo_documento_id' e 'arquivo' são obrigatórios" },
      { status: 400 }
    );
  }

  try {
    const documento = await anexarDocumento(accessToken, id, tipoDocumentoId, arquivo);
    return NextResponse.json(documento, { status: 201 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
