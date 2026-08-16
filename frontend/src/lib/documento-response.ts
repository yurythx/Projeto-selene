import { NextResponse } from "next/server";

const DOCX_CONTENT_TYPE_MARCADOR = "wordprocessingml";

/**
 * Repassa a resposta binária de um dos geradores de documento (Notificação
 * de Descumprimento, Minuta de Aditivo, Atesto, Relatório de Pagamento) —
 * usados pelos Route Handlers em app/api/contratos/[id]/notificacao,
 * .../minuta-aditivo, app/api/processos/[id]/atesto e .../relatorio.
 *
 * O Content-Type REAL da resposta do backend decide o formato: quando um
 * modelo .docx está cadastrado em Configurações pro gatilho
 * correspondente, o backend devolve o modelo preenchido
 * (application/vnd.openxmlformats-officedocument.wordprocessingml.document);
 * sem modelo, devolve o PDF fixo original (fallback). Antes desta função
 * existir, os 4 Route Handlers fixavam "Content-Type: application/pdf" —
 * um .docx retornado pelo backend seria servido com o Content-Type
 * errado, corrompendo o arquivo do ponto de vista do browser.
 *
 * PDF abre inline (renderiza no browser); .docx força download — não tem
 * como abrir um Word inline numa aba.
 */
export function respostaDocumentoGerado(res: Response, nomeBase: string): NextResponse {
  const contentType = res.headers.get("content-type") ?? "application/pdf";
  const isDocx = contentType.includes(DOCX_CONTENT_TYPE_MARCADOR);
  const extensao = isDocx ? "docx" : "pdf";
  const disposicao = isDocx ? "attachment" : "inline";

  return new NextResponse(res.body, {
    status: 200,
    headers: {
      "Content-Type": contentType,
      "Content-Disposition": `${disposicao}; filename="${nomeBase}.${extensao}"`,
    },
  });
}
