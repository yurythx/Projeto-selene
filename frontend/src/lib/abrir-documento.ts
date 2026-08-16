"use client";

/**
 * Abre o PDF de uma resposta fetch (Módulo 2 do roadmap: os 3 geradores de
 * documento fazem POST com corpo JSON, então não dá pra usar um simples
 * `<a href>` como o Relatório de Pagamento — o navegador não manda corpo
 * num clique de link). Cria uma Object URL temporária e abre numa aba
 * nova; revoga a URL depois de um tempo, dando margem pro navegador
 * terminar de carregar o PDF antes dela ser invalidada.
 */
export async function abrirPDFDeResposta(res: Response) {
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  window.open(url, "_blank");
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}
