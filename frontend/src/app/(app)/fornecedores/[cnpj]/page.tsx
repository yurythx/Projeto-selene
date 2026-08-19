import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { getAccessToken } from "@/lib/auth-token";
import { buscarFornecedor, ApiError } from "@/lib/api/client";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function formatarData(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("pt-BR", { timeZone: "UTC" });
}

function formatarDataHora(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("pt-BR");
}

const TIPO_NOTIFICACAO_LABEL: Record<string, string> = {
  NOTIFICACAO_DESCUMPRIMENTO: "Notificação de Descumprimento",
};

/** Badge do Score de Pontualidade — mesma paleta de nível do Radar de
 * Alertas (verde/âmbar/vermelho), com um quarto estado "sem dados" quando
 * ainda não há transições suficientes pra calcular. */
function ScoreBadge({ score }: { score?: number | null }) {
  if (score == null) {
    return <Badge variant="secondary">Sem dados suficientes</Badge>;
  }
  if (score >= 80) {
    return <Badge variant="success">{score.toFixed(0)}% no prazo</Badge>;
  }
  if (score >= 50) {
    return <Badge variant="warning">{score.toFixed(0)}% no prazo</Badge>;
  }
  return <Badge variant="destructive">{score.toFixed(0)}% no prazo</Badge>;
}

export default async function FornecedorDetalhePage({
  params,
}: {
  params: Promise<{ cnpj: string }>;
}) {
  const { cnpj } = await params;
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  let dossie;
  try {
    dossie = await buscarFornecedor(accessToken, cnpj);
  } catch (erro) {
    if (erro instanceof ApiError && (erro.status === 404 || erro.status === 400)) {
      notFound();
    }
    // 401 = sessão inválida, ver o comentário de requireApi em
    // lib/api/client.ts — mesmo tratamento (inclusive a rota
    // intermediária que limpa o cookie, sem ela vira loop de redirect),
    // inline porque esta página já tem seu próprio catch.
    if (erro instanceof ApiError && erro.status === 401) {
      redirect("/api/auth/sessao-invalida");
    }
    throw erro;
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{dossie.nome}</h1>
          <p className="text-muted-foreground text-sm">{dossie.cnpj_formatado}</p>
        </div>
        <div className="text-right">
          <p className="text-muted-foreground mb-1 text-xs">Score de Pontualidade</p>
          <ScoreBadge score={dossie.score_pontualidade} />
        </div>
      </div>

      <Card className="shadow-sm">
        <CardHeader>
          <CardTitle className="text-base">
            Contratos ({dossie.contratos?.length ?? 0})
          </CardTitle>
        </CardHeader>
        <CardContent className="overflow-x-auto p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Número</TableHead>
                <TableHead>Assinatura</TableHead>
                <TableHead>Situação</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(dossie.contratos ?? []).map((contrato) => (
                <TableRow key={contrato.ID} className="hover:bg-accent">
                  <TableCell className="font-medium">
                    <Link href={`/contratos/${contrato.ID}`} className="hover:underline">
                      {contrato.NumeroContrato}
                    </Link>
                  </TableCell>
                  <TableCell>{formatarData(contrato.DataAssinatura)}</TableCell>
                  <TableCell>
                    <Badge variant={contrato.Ativo ? "success" : "secondary"}>
                      {contrato.Ativo ? "Ativo" : "Encerrado"}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card className="shadow-sm">
        <CardHeader>
          <CardTitle className="text-base">
            Histórico de Notificações ({dossie.notificacoes?.length ?? 0})
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(dossie.notificacoes ?? []).length === 0 && (
            <p className="text-muted-foreground text-sm">
              Nenhuma notificação de descumprimento registrada.
            </p>
          )}
          {(dossie.notificacoes ?? []).map((notificacao) => (
            <div
              key={notificacao.ID}
              className="space-y-1 rounded-md border-l-4 border-l-rose-500 bg-muted/30 p-3 text-sm"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium">
                  {notificacao.Tipo ? TIPO_NOTIFICACAO_LABEL[notificacao.Tipo] ?? notificacao.Tipo : "—"}
                </span>
                <span className="text-muted-foreground text-xs">
                  {formatarDataHora(notificacao.CreatedAt)}
                </span>
              </div>
              <p className="text-muted-foreground">{notificacao.Motivo}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
