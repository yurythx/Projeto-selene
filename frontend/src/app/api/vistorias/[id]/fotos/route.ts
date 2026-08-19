import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { assertDentroDoLimite } from "@/lib/rate-limit";
import { anexarFotoVistoria, ApiError } from "@/lib/api/client";

// Espelha maxUploadBytes em backend/internal/handler/documento_handler.go
// (mesmo limite usado pra fotos de vistoria).
const MAX_UPLOAD_BYTES = 20 * 1024 * 1024;

export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const erroOrigem = assertOrigemSegura(request);
  if (erroOrigem) return erroOrigem;

  const erroLimite = await assertDentroDoLimite(request);
  if (erroLimite) return erroLimite;

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
  const foto = formData.get("foto");

  if (!(foto instanceof File)) {
    return NextResponse.json({ error: "campo 'foto' é obrigatório" }, { status: 400 });
  }
  if (foto.size > MAX_UPLOAD_BYTES) {
    return NextResponse.json({ error: "arquivo excede o limite de 20MB" }, { status: 413 });
  }

  try {
    const fotoVistoria = await anexarFotoVistoria(accessToken, id, foto);
    return NextResponse.json(fotoVistoria, { status: 201 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
