"use client";

import { useState, useCallback } from "react";
import { Document, Page, pdfjs } from "react-pdf";
import { ChevronLeftIcon, ChevronRightIcon, ZoomInIcon, ZoomOutIcon, RotateCcwIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import "react-pdf/dist/Page/AnnotationLayer.css";
import "react-pdf/dist/Page/TextLayer.css";

// Worker e demais assets (cmaps, fontes padrão) servidos SAME-ORIGIN a
// partir de public/pdfjs/ (ver scripts/copy-pdfjs-assets.mjs) — nunca do
// CDN padrão do react-pdf/pdfjs-dist. A CSP deste app (ver src/proxy.ts)
// só permite 'self' pra worker-src/font-src; um Worker ou uma fonte
// carregada de um host externo seria bloqueada silenciosamente.
pdfjs.GlobalWorkerOptions.workerSrc = "/pdfjs/pdf.worker.min.mjs";

// Módulo, não estado do componente — <Document options={...}> exige
// (documentado no próprio react-pdf) que o objeto seja estável entre
// renders (comparado por igualdade referencial); definir aqui garante
// isso sem precisar de useMemo. withCredentials: true é o que faz o
// pdfjs-dist mandar o cookie de sessão (HttpOnly) na requisição pro
// proxy BFF — sem isso, a rota devolveria 401 (não autenticado) porque
// pdfjs usa seu próprio fetch interno, que por padrão não inclui
// credenciais.
const PDF_OPTIONS = {
  cMapUrl: "/pdfjs/cmaps/",
  standardFontDataUrl: "/pdfjs/standard_fonts/",
  wasmUrl: "/pdfjs/wasm/",
  withCredentials: true,
};

const ESCALA_MIN = 0.5;
const ESCALA_MAX = 3;
const ESCALA_PASSO = 0.25;
const ESCALA_PADRAO = 1.2;

/**
 * Visualizador de PDF embutido (PDF.js via react-pdf) — pedido explícito
 * do usuário como opção de otimização/melhoria da pré-visualização de
 * documentos, no lugar do <iframe>/visualizador nativo do navegador.
 * Zoom e navegação de página consistentes entre navegadores (o
 * visualizador nativo do Chrome/Firefox/Safari tem controles bem
 * diferentes entre si, alguns nem mostram um contador de página claro).
 *
 * Importado sempre via next/dynamic({ ssr: false }) por quem usa este
 * componente (ver processo-page.tsx) — pdfjs-dist depende de APIs de
 * browser (Worker, DOMMatrix, Canvas) que não existem no servidor
 * durante o SSR/RSC.
 */
export function DocumentoPdfViewer({ url, nomeArquivo }: { url: string; nomeArquivo: string }) {
  const [numeroPaginas, setNumeroPaginas] = useState<number | null>(null);
  const [paginaAtual, setPaginaAtual] = useState(1);
  const [escala, setEscala] = useState(ESCALA_PADRAO);
  const [erro, setErro] = useState<string | null>(null);

  const aoCarregar = useCallback(({ numPages }: { numPages: number }) => {
    setNumeroPaginas(numPages);
    setPaginaAtual(1);
    setErro(null);
  }, []);

  const aoFalharCarregar = useCallback((erroCarregamento: Error) => {
    setErro(erroCarregamento.message || "Não foi possível carregar o documento.");
  }, []);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="bg-muted/30 flex shrink-0 items-center justify-between gap-2 rounded-t-md border-x border-t px-3 py-2">
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="Página anterior"
            disabled={paginaAtual <= 1}
            onClick={() => setPaginaAtual((p) => Math.max(1, p - 1))}
          >
            <ChevronLeftIcon className="size-4" />
          </Button>
          <span className="text-muted-foreground min-w-[6.5rem] text-center text-sm whitespace-nowrap">
            {numeroPaginas ? `Página ${paginaAtual} de ${numeroPaginas}` : "Carregando…"}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="Próxima página"
            disabled={!numeroPaginas || paginaAtual >= numeroPaginas}
            onClick={() => setPaginaAtual((p) => (numeroPaginas ? Math.min(numeroPaginas, p + 1) : p))}
          >
            <ChevronRightIcon className="size-4" />
          </Button>
        </div>

        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="Diminuir zoom"
            disabled={escala <= ESCALA_MIN}
            onClick={() => setEscala((e) => Math.max(ESCALA_MIN, +(e - ESCALA_PASSO).toFixed(2)))}
          >
            <ZoomOutIcon className="size-4" />
          </Button>
          <span className="text-muted-foreground min-w-[3.5rem] text-center text-sm">
            {Math.round((escala / ESCALA_PADRAO) * 100)}%
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="Aumentar zoom"
            disabled={escala >= ESCALA_MAX}
            onClick={() => setEscala((e) => Math.min(ESCALA_MAX, +(e + ESCALA_PASSO).toFixed(2)))}
          >
            <ZoomInIcon className="size-4" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="Restaurar zoom"
            disabled={escala === ESCALA_PADRAO}
            onClick={() => setEscala(ESCALA_PADRAO)}
          >
            <RotateCcwIcon className="size-4" />
          </Button>
        </div>
      </div>

      <div className="bg-muted/30 min-h-0 flex-1 overflow-auto rounded-b-md border-x border-b p-4">
        {erro ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-center">
            <p className="text-destructive text-sm">{erro}</p>
            <p className="text-muted-foreground text-xs">
              Tente baixar &ldquo;{nomeArquivo}&rdquo; diretamente pra conferir.
            </p>
          </div>
        ) : (
          <Document
            file={url}
            options={PDF_OPTIONS}
            onLoadSuccess={aoCarregar}
            onLoadError={aoFalharCarregar}
            loading={<p className="text-muted-foreground p-4 text-center text-sm">Carregando documento…</p>}
            className="flex justify-center"
          >
            <Page pageNumber={paginaAtual} scale={escala} />
          </Document>
        )}
      </div>
    </div>
  );
}
