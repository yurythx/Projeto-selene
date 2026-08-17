import Link from "next/link";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { listarContratos, requireApi } from "@/lib/api/client";
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
import { ContratosFiltro } from "@/components/contratos/contratos-filtro";
import { Paginacao } from "@/components/paginacao";

const TIPO_OBJETO_LABEL: Record<string, string> = {
  CONSUMO: "Consumo",
  PERMANENTE: "Permanente",
  SERVICO: "Serviço",
};

const TAMANHO_PAGINA = 20;

function formatarData(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("pt-BR", { timeZone: "UTC" });
}

export default async function ContratosPage({
  searchParams,
}: {
  searchParams: Promise<{ pagina?: string; busca?: string; tipo_objeto?: string; situacao?: string }>;
}) {
  const session = await auth();
  const accessToken = await getAccessToken();

  // proxy.ts já redireciona sessões ausentes pra /login antes de chegar
  // aqui; isto é só uma proteção defensiva caso a rota seja renderizada
  // fora do fluxo normal (ex: teste, chamada direta durante o build).
  if (!accessToken) {
    return null;
  }

  const sp = await searchParams;
  const paginaAtual = Number(sp.pagina) > 0 ? Number(sp.pagina) : 1;
  const busca = sp.busca ?? "";
  const tipoObjeto = sp.tipo_objeto ?? "";
  const situacao = sp.situacao === "ativo" || sp.situacao === "encerrado" ? sp.situacao : undefined;

  const resultado = await requireApi(
    listarContratos(accessToken, paginaAtual, TAMANHO_PAGINA, { busca, tipoObjeto, situacao })
  );
  const total = resultado.total ?? 0;
  const totalPaginas = Math.max(1, Math.ceil(total / (resultado.tamanho_pagina ?? TAMANHO_PAGINA)));

  const filtroAtivo = Boolean(busca || tipoObjeto || situacao);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Contratos</h1>
          <p className="text-muted-foreground text-sm">
            {total} contrato{total === 1 ? "" : "s"}
            {filtroAtivo ? " encontrado" : " cadastrado"}
            {total === 1 ? "" : "s"}.
          </p>
        </div>
        {session?.user?.isFiscal && <NovoContratoDialog fiscalNome={session.user.name ?? ""} />}
      </div>

      <ContratosFiltro buscaInicial={busca} tipoObjetoInicial={tipoObjeto} situacaoInicial={situacao ?? ""} />

      <div className="overflow-x-auto rounded-lg border shadow-sm">
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
                  {filtroAtivo
                    ? "Nenhum contrato encontrado com esses filtros."
                    : "Nenhum contrato cadastrado ainda."}
                </TableCell>
              </TableRow>
            )}
            {resultado.dados.map((contrato) => (
              <TableRow key={contrato.ID} className="hover:bg-accent">
                <TableCell className="font-medium">
                  <Link href={`/contratos/${contrato.ID}`} className="hover:underline">
                    {contrato.NumeroContrato}
                  </Link>
                </TableCell>
                <TableCell>{contrato.ContratadaNome}</TableCell>
                <TableCell>
                  {contrato.TipoObjeto ? TIPO_OBJETO_LABEL[contrato.TipoObjeto] : "—"}
                </TableCell>
                <TableCell>{contrato.Fiscal?.Nome ?? "—"}</TableCell>
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
      </div>

      <Paginacao paginaAtual={resultado.pagina ?? paginaAtual} totalPaginas={totalPaginas} />
    </div>
  );
}
