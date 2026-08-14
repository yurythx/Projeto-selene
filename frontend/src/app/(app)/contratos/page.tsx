import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { listarContratos } from "@/lib/api/client";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { NovoContratoDialog } from "@/components/contratos/novo-contrato-dialog";

const TIPO_OBJETO_LABEL: Record<string, string> = {
  CONSUMO: "Consumo",
  PERMANENTE: "Permanente",
  SERVICO: "Serviço",
};

function formatarData(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("pt-BR", { timeZone: "UTC" });
}

export default async function ContratosPage() {
  const session = await auth();
  const accessToken = await getAccessToken();

  // proxy.ts já redireciona sessões ausentes pra /login antes de chegar
  // aqui; isto é só uma proteção defensiva caso a rota seja renderizada
  // fora do fluxo normal (ex: teste, chamada direta durante o build).
  if (!accessToken) {
    return null;
  }

  const resultado = await listarContratos(accessToken);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Contratos</h1>
          <p className="text-muted-foreground text-sm">
            {resultado.total} contrato{resultado.total === 1 ? "" : "s"} cadastrado
            {resultado.total === 1 ? "" : "s"}.
          </p>
        </div>
        {session?.user?.isFiscal && <NovoContratoDialog fiscalNome={session.user.name ?? ""} />}
      </div>

      <div className="overflow-x-auto rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Número</TableHead>
              <TableHead>Contratada</TableHead>
              <TableHead>Tipo</TableHead>
              <TableHead>Fiscal</TableHead>
              <TableHead>Assinatura</TableHead>
              <TableHead>Situação</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {resultado.dados.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="text-muted-foreground text-center">
                  Nenhum contrato cadastrado ainda.
                </TableCell>
              </TableRow>
            )}
            {resultado.dados.map((contrato) => (
              <TableRow key={contrato.ID}>
                <TableCell className="font-medium">{contrato.NumeroContrato}</TableCell>
                <TableCell>{contrato.ContratadaNome}</TableCell>
                <TableCell>
                  {contrato.TipoObjeto ? TIPO_OBJETO_LABEL[contrato.TipoObjeto] : "—"}
                </TableCell>
                <TableCell>{contrato.Fiscal?.Nome ?? "—"}</TableCell>
                <TableCell>{formatarData(contrato.DataAssinatura)}</TableCell>
                <TableCell>
                  <Badge variant={contrato.Ativo ? "default" : "secondary"}>
                    {contrato.Ativo ? "Ativo" : "Encerrado"}
                  </Badge>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
