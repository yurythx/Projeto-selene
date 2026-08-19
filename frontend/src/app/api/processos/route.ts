import { NextResponse } from "next/server";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { assertOrigemSegura } from "@/lib/verify-origin";
import { assertDentroDoLimite } from "@/lib/rate-limit";
import { novoProcessoSchema } from "@/lib/validation/bff-schemas";
import { criarProcesso, listarProcessos, ApiError } from "@/lib/api/client";

/**
 * GET /api/processos?etapa=&pagina=&tamanho= — usado pelo botão "Carregar
 * mais" de uma coluna do Kanban (KanbanBoard, client component): a carga
 * inicial de cada coluna já vem do Server Component (kanban/page.tsx,
 * chamando listarProcessos direto — mesmo padrão de leitura do resto do
 * app), mas uma página seguinte pedida por clique do usuário precisa de
 * uma rota de BFF de verdade, já que só Server Components chamam o
 * backend direto. Sem checagem de Origin/isFiscal — é leitura (GET),
 * mesmo padrão de GET .../documentos.
 */
export async function GET(request: Request) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "não autenticado" }, { status: 401 });
  }

  const { searchParams } = new URL(request.url);
  const etapa = Number(searchParams.get("etapa"));
  if (!Number.isInteger(etapa) || etapa <= 0) {
    return NextResponse.json({ error: "parâmetro 'etapa' precisa ser um número inteiro positivo" }, { status: 400 });
  }
  const pagina = Number(searchParams.get("pagina")) || 1;
  const tamanho = Number(searchParams.get("tamanho")) || 100;

  try {
    const resultado = await listarProcessos(accessToken, etapa, pagina, tamanho);
    return NextResponse.json(resultado);
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}

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
  if (!session.user.isFiscal) {
    return NextResponse.json({ error: "usuário não é fiscal" }, { status: 403 });
  }

  const resultado = novoProcessoSchema.safeParse(await request.json().catch(() => null));
  if (!resultado.success) {
    return NextResponse.json(
      { error: "corpo inválido", detalhes: resultado.error.flatten() },
      { status: 400 }
    );
  }

  try {
    const processo = await criarProcesso(accessToken, resultado.data.contrato_id, resultado.data.mes_referencia);
    return NextResponse.json(processo, { status: 201 });
  } catch (erro) {
    if (erro instanceof ApiError) {
      return NextResponse.json(erro.body ?? { error: "erro na API" }, { status: erro.status });
    }
    return NextResponse.json({ error: "erro inesperado" }, { status: 500 });
  }
}
