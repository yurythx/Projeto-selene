import Link from "next/link";
import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { listarModelosDocumento, requireApi } from "@/lib/api/client";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { CriarModeloDocumentoDialog } from "@/components/configuracoes/criar-modelo-documento-dialog";
import { GATILHO_LABEL } from "@/lib/modelos-documento";

function formatarData(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("pt-BR");
}

// Configurações — Modelos de Documentos: admin sobe um .docx de
// referência por categoria (texto livre); quando a categoria tem um
// gatilho associado, o fluxo de geração correspondente passa a preencher
// esse modelo de verdade em vez do PDF fixo (ver o backend,
// GeradorDocumentosService/RelatorioService).
export default async function ModelosDocumentosPage() {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  // Checagem "de verdade" (não apenas a otimista do proxy.ts) — mesmo
  // padrão de admin/usuarios/page.tsx.
  if (!session?.user?.isAdmin) {
    return (
      <div className="text-muted-foreground text-sm">
        Você não tem permissão para acessar esta página.
      </div>
    );
  }

  const modelos = await requireApi(listarModelosDocumento(accessToken));

  return (
    <div className="space-y-6">
      <div>
        {/* Mesmo padrão de configuracoes/keycloak/page.tsx — as duas são
            filhas do hub /configuracoes (sidebar aponta pro hub, não
            direto pra cá), consistente ter a mesma navegação de volta. */}
        <Link href="/configuracoes" className="text-muted-foreground text-sm hover:underline">
          ← Configurações
        </Link>
        <div className="mt-1 flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-semibold">Modelos de Documentos</h1>
            <p className="text-muted-foreground text-sm">
              {modelos.length} categoria{modelos.length === 1 ? "" : "s"} cadastrada
              {modelos.length === 1 ? "" : "s"} — .docx de referência, substituível a
              qualquer momento.
            </p>
          </div>
          <CriarModeloDocumentoDialog />
        </div>
      </div>

      <div className="overflow-x-auto rounded-lg border shadow-sm">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Categoria</TableHead>
              <TableHead>Gatilho</TableHead>
              <TableHead>Arquivo ativo</TableHead>
              <TableHead>Enviado em</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {modelos.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="text-muted-foreground text-center">
                  Nenhum modelo cadastrado ainda.
                </TableCell>
              </TableRow>
            )}
            {modelos.map((modelo) => (
              <TableRow key={modelo.ID} className="hover:bg-accent">
                <TableCell className="font-medium">
                  <Link href={`/configuracoes/modelos-documentos/${modelo.ID}`} className="hover:underline">
                    {modelo.Categoria}
                  </Link>
                </TableCell>
                <TableCell>
                  {modelo.Gatilho ? (
                    <Badge variant="info">{GATILHO_LABEL[modelo.Gatilho] ?? modelo.Gatilho}</Badge>
                  ) : (
                    <Badge variant="secondary">Biblioteca de referência</Badge>
                  )}
                </TableCell>
                <TableCell>{modelo.VersaoAtiva?.NomeArquivo ?? "—"}</TableCell>
                <TableCell>{formatarData(modelo.VersaoAtiva?.CreatedAt)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
