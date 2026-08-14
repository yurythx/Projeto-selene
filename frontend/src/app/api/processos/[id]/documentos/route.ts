import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { tipoDocumentoIdSchema } from "@/lib/validation/bff-schemas";
import { listarDocumentos, anexarDocumento, ApiError } from "@/lib/api/client";

// Espelha maxUploadBytes em backend/internal/handler/documento_handler.go
// — checagem redundante e intencional: falha rápido aqui, antes de gastar
// uma chamada de rede pro backend, sem depender só do limite de lá.
const MAX_UPLOAD_BYTES = 20 * 1024 * 1024;

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
  const arquivo = formData.get("arquivo");

  const tipoResultado = tipoDocumentoIdSchema.safeParse(formData.get("tipo_documento_id"));
  if (!tipoResultado.success) {
    return NextResponse.json(
      { error: "campo 'tipo_documento_id' precisa ser um número inteiro positivo" },
      { status: 400 }
    );
  }
  if (!(arquivo instanceof File)) {
    return NextResponse.json({ error: "campo 'arquivo' é obrigatório" }, { status: 400 });
  }
  if (arquivo.size > MAX_UPLOAD_BYTES) {
    return NextResponse.json({ error: "arquivo excede o limite de 20MB" }, { status: 413 });
  }

  try {
    const documento = await anexarDocumento(accessToken, id, tipoResultado.data, arquivo);
    return NextResponse.json(documento, { status: 201 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
