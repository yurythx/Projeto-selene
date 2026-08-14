import { NextResponse } from "next/server";
import { verificarDocumento } from "@/lib/api/client";

/**
 * Proxy PÚBLICO (sem checagem de sessão, de propósito — ver
 * GeradorDocumentosHandler.Verificar no backend) da verificação de
 * autenticidade de um documento emitido. Existe pra a página /verificar
 * poder ser consumida também via client-side fetch, se necessário; a
 * Server Component da página chama verificarDocumento() diretamente.
 */
export async function GET(_request: Request, { params }: { params: Promise<{ codigo: string }> }) {
  const { codigo } = await params;
  const resultado = await verificarDocumento(codigo);
  return NextResponse.json(resultado);
}
