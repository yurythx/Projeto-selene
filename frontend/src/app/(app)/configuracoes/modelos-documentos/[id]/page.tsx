import { notFound, redirect } from "next/navigation";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { buscarModeloDocumento, ApiError } from "@/lib/api/client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { GATILHO_LABEL } from "@/lib/modelos-documento";
import { SubstituirVersaoModeloDialog } from "@/components/configuracoes/substituir-versao-modelo-dialog";
import { EditarGatilhoModeloDialog } from "@/components/configuracoes/editar-gatilho-modelo-dialog";

function formatarDataHora(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("pt-BR");
}

export default async function ModeloDocumentoDetalhePage({
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
  if (!session?.user?.isAdmin) {
    return (
      <div className="text-muted-foreground text-sm">
        Você não tem permissão para acessar esta página.
      </div>
    );
  }

  let modelo;
  try {
    modelo = await buscarModeloDocumento(accessToken, id);
  } catch (erro) {
    if (erro instanceof ApiError && (erro.status === 404 || erro.status === 400)) {
      notFound();
    }
    // 401 = sessão inválida, ver o comentário de requireApi em
    // lib/api/client.ts — mesmo tratamento, inline porque esta página já
    // tem seu próprio catch.
    if (erro instanceof ApiError && erro.status === 401) {
      redirect("/api/auth/sessao-invalida");
    }
    throw erro;
  }

  const versoes = modelo.Versoes ?? [];

  return (
    <div className="max-w-2xl space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{modelo.Categoria}</h1>
          <div className="mt-1">
            {modelo.Gatilho ? (
              <Badge variant="info">{GATILHO_LABEL[modelo.Gatilho] ?? modelo.Gatilho}</Badge>
            ) : (
              <Badge variant="secondary">Biblioteca de referência</Badge>
            )}
          </div>
        </div>
        <EditarGatilhoModeloDialog modelo={modelo} />
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Versão ativa</CardTitle>
          <SubstituirVersaoModeloDialog modeloId={modelo.ID!} />
        </CardHeader>
        <CardContent className="space-y-2">
          {modelo.VersaoAtiva ? (
            <>
              <p className="text-sm font-medium">{modelo.VersaoAtiva.NomeArquivo}</p>
              <p className="text-muted-foreground text-xs">
                Enviado em {formatarDataHora(modelo.VersaoAtiva.CreatedAt)}
                {modelo.VersaoAtiva.EnviadoPor && ` por ${modelo.VersaoAtiva.EnviadoPor.Nome}`}
              </p>
              <Button variant="outline" size="sm" render={
                <a href={`/api/configuracoes/modelos-documento/${modelo.ID}/download`} download>
                  Baixar
                </a>
              } />
            </>
          ) : (
            <p className="text-muted-foreground text-sm">Nenhuma versão ativa.</p>
          )}
        </CardContent>
      </Card>

      <div>
        <h2 className="mb-2 text-lg font-semibold">Histórico de versões</h2>
        <div className="overflow-x-auto rounded-lg border shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Arquivo</TableHead>
                <TableHead>Enviado em</TableHead>
                <TableHead>Enviado por</TableHead>
                <TableHead className="text-right">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {versoes.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="text-muted-foreground text-center">
                    Nenhuma versão publicada ainda.
                  </TableCell>
                </TableRow>
              )}
              {versoes.map((versao) => (
                <TableRow key={versao.ID}>
                  <TableCell className="font-medium">
                    {versao.NomeArquivo}
                    {versao.ID === modelo.VersaoAtivaID && (
                      <Badge variant="success" className="ml-2">
                        Ativa
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>{formatarDataHora(versao.CreatedAt)}</TableCell>
                  <TableCell>{versao.EnviadoPor?.Nome ?? "—"}</TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" render={
                      <a
                        href={`/api/configuracoes/modelos-documento/${modelo.ID}/versoes/${versao.ID}/download`}
                        download
                      >
                        Baixar
                      </a>
                    } />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}
