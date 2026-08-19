import { CheckCircle2Icon, XCircleIcon } from "lucide-react";
import { verificarDocumento } from "@/lib/api/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const TIPO_LABEL: Record<string, string> = {
  NOTIFICACAO_DESCUMPRIMENTO: "Notificação de Descumprimento",
  ATESTO: "Atesto",
  MINUTA_ADITIVO: "Minuta de Aditivo",
};

// Página PÚBLICA (fora do grupo (app), sem sessão — ver proxy.ts) pra onde
// o QR code de um Atesto impresso aponta. Quem escaneia normalmente não
// tem login no Selene (ex: auditor do TCE), então esta tela não pode
// depender de autenticação nem do layout autenticado (Nav).
export default async function VerificarPage({
  params,
}: {
  params: Promise<{ codigo: string }>;
}) {
  const { codigo } = await params;
  const resultado = await verificarDocumento(codigo);

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="flex items-center gap-2">
            {resultado.valido ? (
              <CheckCircle2Icon className="text-primary size-6" />
            ) : (
              <XCircleIcon className="text-destructive size-6" />
            )}
            <CardTitle>
              {resultado.valido ? "Documento autêntico" : "Documento não encontrado"}
            </CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          {resultado.valido ? (
            <>
              <div>
                <p className="text-muted-foreground">Tipo</p>
                <p>{resultado.tipo ? TIPO_LABEL[resultado.tipo] : "—"}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Contrato</p>
                <p>{resultado.numero_contrato}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Contratada</p>
                <p>{resultado.contratada_nome}</p>
              </div>
              {resultado.mes_referencia && (
                <div>
                  <p className="text-muted-foreground">Mês de referência</p>
                  <p>{resultado.mes_referencia}</p>
                </div>
              )}
              <div>
                <p className="text-muted-foreground">Gerado por</p>
                <p>{resultado.gerado_por_nome}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Data de emissão</p>
                <p>{resultado.criado_em}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Código de verificação</p>
                <p className="font-mono">{resultado.codigo_verificacao}</p>
              </div>
            </>
          ) : (
            <p className="text-muted-foreground">
              O código <span className="font-mono">{codigo}</span> não corresponde a nenhum
              documento emitido pelo Selene. Se você escaneou um QR code de um documento
              impresso, confirme que o código não foi digitado incorretamente.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
