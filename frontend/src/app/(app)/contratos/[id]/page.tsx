import { notFound } from "next/navigation";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { buscarContrato, ApiError } from "@/lib/api/client";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EditarContratoDialog } from "@/components/contratos/editar-contrato-dialog";
import { EncerrarContratoButton } from "@/components/contratos/encerrar-contrato-button";
import { GerarNotificacaoDialog } from "@/components/contratos/gerar-notificacao-dialog";
import { GerarMinutaAditivoDialog } from "@/components/contratos/gerar-minuta-aditivo-dialog";

const TIPO_OBJETO_LABEL: Record<string, string> = {
  CONSUMO: "Consumo",
  PERMANENTE: "Permanente",
  SERVICO: "Serviço",
};

function formatarData(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("pt-BR", { timeZone: "UTC" });
}

export default async function ContratoDetalhePage({
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

  let contrato;
  try {
    contrato = await buscarContrato(accessToken, id);
  } catch (erro) {
    if (erro instanceof ApiError && (erro.status === 404 || erro.status === 400)) {
      notFound();
    }
    throw erro;
  }

  const isFiscal = Boolean(session?.user?.isFiscal);

  return (
    <div className="max-w-2xl space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{contrato.NumeroContrato}</h1>
          <p className="text-muted-foreground text-sm">
            {contrato.TipoObjeto ? TIPO_OBJETO_LABEL[contrato.TipoObjeto] : "—"}
          </p>
        </div>
        <Badge variant={contrato.Ativo ? "default" : "secondary"}>
          {contrato.Ativo ? "Ativo" : "Encerrado"}
        </Badge>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Dados do contrato</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p className="text-muted-foreground">Portaria de nomeação</p>
            <p>{contrato.PortariaNomeacao || "—"}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Data de assinatura</p>
            <p>{formatarData(contrato.DataAssinatura)}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Empresa contratada</p>
            <p>{contrato.ContratadaNome}</p>
          </div>
          <div>
            <p className="text-muted-foreground">CNPJ</p>
            <p>{contrato.ContratadaCNPJ}</p>
          </div>
          <div>
            <p className="text-muted-foreground">E-mail da contratada</p>
            <p>{contrato.ContratadaEmail || "—"}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Fiscal responsável</p>
            <p>{contrato.Fiscal?.Nome ?? "—"}</p>
          </div>
        </CardContent>
      </Card>

      {isFiscal && (
        <div className="flex flex-wrap gap-2">
          <EditarContratoDialog contrato={contrato} />
          {contrato.Ativo && <EncerrarContratoButton contratoId={contrato.ID!} />}
          <GerarNotificacaoDialog contratoId={contrato.ID!} />
          <GerarMinutaAditivoDialog contratoId={contrato.ID!} />
        </div>
      )}
    </div>
  );
}
