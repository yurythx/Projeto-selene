import { notFound, redirect } from "next/navigation";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import {
  buscarProcesso,
  listarDocumentos,
  listarTiposDocumento,
  listarRadar,
  requireApi,
  ApiError,
} from "@/lib/api/client";
import { itensDoProcesso } from "@/lib/radar";
import { ProcessoPage } from "@/components/kanban/processo-page";

export default async function ProcessoDetalhePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  let processo;
  try {
    processo = await buscarProcesso(accessToken, id);
  } catch (erro) {
    if (erro instanceof ApiError && (erro.status === 404 || erro.status === 400)) {
      notFound();
    }
    // 401 = sessão inválida, mesmo tratamento de requireApi (ver o
    // comentário em lib/api/client.ts) — inline porque esta página já
    // tem seu próprio catch pro 404 acima.
    if (erro instanceof ApiError && erro.status === 401) {
      redirect("/api/auth/sessao-invalida");
    }
    throw erro;
  }

  const [documentos, tiposDocumento, radarItens] = await requireApi(
    Promise.all([
      listarDocumentos(accessToken, id),
      listarTiposDocumento(accessToken),
      listarRadar(accessToken),
    ])
  );

  return (
    <ProcessoPage
      processoInicial={processo}
      documentosIniciais={documentos}
      tiposDocumento={tiposDocumento}
      alertasRadar={itensDoProcesso(radarItens, processo)}
      isFiscal={Boolean(session?.user?.isFiscal)}
    />
  );
}
