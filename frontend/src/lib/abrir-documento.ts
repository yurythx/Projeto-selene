"use client";

/**
 * Abre (PDF) ou baixa (.docx) o documento de uma resposta fetch (Módulo 2
 * do roadmap: os 3 geradores de documento fazem POST com corpo JSON,
 * então não dá pra usar um `<a href>` simples como o Relatório de
 * Pagamento — o navegador não manda corpo num clique de link).
 *
 * O Content-Type REAL da resposta decide o comportamento — desde
 * Configurações: Modelos de Documentos, o backend pode devolver um .docx
 * preenchido em vez do PDF fixo (ver GeradorDocumentosService/
 * RelatorioService): PDF abre numa aba nova via Object URL temporária;
 * .docx não tem como ser renderizado numa aba do browser, então força o
 * download (link temporário com o atributo `download`).
 */
export async function abrirOuBaixarDocumento(res: Response) {
  const contentType = res.headers.get("content-type") ?? "";
  const isDocx = contentType.includes("wordprocessingml");
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);

  if (isDocx) {
    const link = document.createElement("a");
    link.href = url;
    link.download = extrairNomeArquivo(res.headers.get("content-disposition")) ?? "documento.docx";
    document.body.appendChild(link);
    link.click();
    link.remove();
  } else {
    window.open(url, "_blank");
  }

  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

function extrairNomeArquivo(contentDisposition: string | null): string | null {
  if (!contentDisposition) return null;
  const match = /filename="?([^"]+)"?/.exec(contentDisposition);
  return match ? match[1] : null;
}
