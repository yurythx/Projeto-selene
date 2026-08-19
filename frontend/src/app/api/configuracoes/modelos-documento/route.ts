import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { assertDentroDoLimite } from "@/lib/rate-limit";
import { categoriaModeloSchema, gatilhoModeloSchema } from "@/lib/validation/bff-schemas";
import { criarModeloDocumento, ApiError } from "@/lib/api/client";

// Espelha maxUploadBytes em backend/internal/handler/documento_handler.go
// (reaproveitado por ModeloDocumentoHandler) — checagem redundante e
// intencional, falha rápido antes de gastar uma chamada de rede pro
// backend.
const MAX_UPLOAD_BYTES = 20 * 1024 * 1024;

/**
 * Route Handler do BFF: proxy autenticado pro backend Go em
 * POST /api/v1/admin/modelos-documento. Restrito a admin — Configurações
 * — Modelos de Documentos.
 */
export async function POST(request: Request) {
  const erroOrigem = assertOrigemSegura(request);
  if (erroOrigem) return erroOrigem;

  const erroLimite = await assertDentroDoLimite(request);
  if (erroLimite) return erroLimite;

  const session = await auth();
  const accessToken = await getAccessToken();

  if (!session?.user?.id || !accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }
  // Checagem redundante com o backend (que também exige IsAdmin) — defesa
  // em profundidade, mesmo padrão de app/api/admin/usuarios/[id]/route.ts.
  if (!session.user.isAdmin) {
    return NextResponse.json({ error: "usuário não é administrador" }, { status: 403 });
  }

  const formData = await request.formData();
  const arquivo = formData.get("arquivo");

  const categoriaResultado = categoriaModeloSchema.safeParse(formData.get("categoria"));
  if (!categoriaResultado.success) {
    return NextResponse.json({ error: "campo 'categoria' é obrigatório" }, { status: 400 });
  }

  const gatilhoBruto = formData.get("gatilho");
  const gatilhoResultado = gatilhoModeloSchema.safeParse(gatilhoBruto || undefined);
  if (!gatilhoResultado.success) {
    return NextResponse.json({ error: "campo 'gatilho' inválido" }, { status: 400 });
  }

  if (!(arquivo instanceof File)) {
    return NextResponse.json({ error: "campo 'arquivo' é obrigatório" }, { status: 400 });
  }
  if (arquivo.size > MAX_UPLOAD_BYTES) {
    return NextResponse.json({ error: "arquivo excede o limite de 20MB" }, { status: 413 });
  }

  try {
    const modelo = await criarModeloDocumento(
      accessToken,
      categoriaResultado.data,
      gatilhoResultado.data,
      arquivo
    );
    return NextResponse.json(modelo, { status: 201 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
