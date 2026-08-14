import Link from "next/link";
import { getAccessToken } from "@/lib/auth-token";
import { listarFornecedores } from "@/lib/api/client";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";

export default async function FornecedoresPage() {
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  const fornecedores = await listarFornecedores(accessToken);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Dossiê do Fornecedor</h1>
        <p className="text-muted-foreground text-sm">
          {fornecedores.length} fornecedor{fornecedores.length === 1 ? "" : "es"} com contrato
          cadastrado.
        </p>
      </div>

      <div className="overflow-x-auto rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Fornecedor</TableHead>
              <TableHead>CNPJ</TableHead>
              <TableHead>Contratos</TableHead>
              <TableHead>Ativos</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {fornecedores.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="text-muted-foreground text-center">
                  Nenhum fornecedor cadastrado ainda.
                </TableCell>
              </TableRow>
            )}
            {fornecedores.map((fornecedor) => (
              <TableRow key={fornecedor.cnpj} className="hover:bg-accent">
                <TableCell className="font-medium">
                  <Link href={`/fornecedores/${fornecedor.cnpj}`} className="hover:underline">
                    {fornecedor.nome}
                  </Link>
                </TableCell>
                <TableCell>{fornecedor.cnpj_formatado}</TableCell>
                <TableCell>{fornecedor.qtd_contratos}</TableCell>
                <TableCell>
                  <Badge variant={fornecedor.qtd_contratos_ativos ? "default" : "secondary"}>
                    {fornecedor.qtd_contratos_ativos}
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
