"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { SearchIcon } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { FornecedorResumo } from "@/lib/api/client";

function normalizar(texto: string) {
  return texto.trim().toLowerCase();
}

/**
 * Busca client-side, não server-side como em Contratos
 * (ver contratos-filtro.tsx) — decisão deliberada: FornecedorService.Listar
 * já agrega TODOS os fornecedores em memória a partir de
 * ContratoRepository.ListTodos (ver o comentário no backend sobre o
 * conjunto ser pequeno o bastante pra uma prefeitura), então não há
 * paginação nem uma query SQL própria pra filtrar — a lista inteira já
 * chega pro cliente de qualquer forma, e filtrar em memória aqui evita
 * uma segunda ida ao backend sem ganhar nada em troca.
 */
export function FornecedoresLista({ fornecedores }: { fornecedores: FornecedorResumo[] }) {
  const [busca, setBusca] = useState("");

  const filtrados = useMemo(() => {
    const alvo = normalizar(busca);
    if (!alvo) return fornecedores;
    return fornecedores.filter(
      (f) =>
        normalizar(f.nome ?? "").includes(alvo) ||
        normalizar(f.cnpj_formatado ?? "").includes(alvo) ||
        normalizar(f.cnpj ?? "").includes(alvo)
    );
  }, [fornecedores, busca]);

  return (
    <div className="space-y-3">
      <div className="relative max-w-sm">
        <SearchIcon className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
        <Input
          placeholder="Buscar por nome ou CNPJ..."
          className="pl-8"
          value={busca}
          onChange={(e) => setBusca(e.target.value)}
        />
      </div>

      <div className="overflow-x-auto rounded-lg border shadow-sm">
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
            {filtrados.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="text-muted-foreground text-center">
                  {busca
                    ? "Nenhum fornecedor encontrado com esse filtro."
                    : "Nenhum fornecedor cadastrado ainda."}
                </TableCell>
              </TableRow>
            )}
            {filtrados.map((fornecedor) => (
              <TableRow key={fornecedor.cnpj} className="hover:bg-accent">
                <TableCell className="font-medium">
                  <Link href={`/fornecedores/${fornecedor.cnpj}`} className="hover:underline">
                    {fornecedor.nome}
                  </Link>
                </TableCell>
                <TableCell>{fornecedor.cnpj_formatado}</TableCell>
                <TableCell>{fornecedor.qtd_contratos}</TableCell>
                <TableCell>
                  <Badge variant={fornecedor.qtd_contratos_ativos ? "success" : "secondary"}>
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
