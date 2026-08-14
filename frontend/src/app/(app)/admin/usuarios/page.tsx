import { auth } from "@/auth";
import { getAccessToken } from "@/lib/auth-token";
import { listarUsuarios } from "@/lib/api/client";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { EditarUsuarioDialog } from "@/components/admin/editar-usuario-dialog";

export default async function AdminUsuariosPage() {
  const session = await auth();
  const accessToken = await getAccessToken();

  if (!accessToken) {
    return null;
  }

  // Checagem de autorização "de verdade" (não apenas a otimista do
  // proxy.ts) — só is_admin=true chega aqui; qualquer outro usuário vê a
  // mensagem abaixo, e a API (GET /admin/users) rejeitaria com 403 de
  // qualquer forma se essa checagem fosse pulada.
  if (!session?.user?.isAdmin) {
    return (
      <div className="text-muted-foreground text-sm">
        Você não tem permissão para acessar esta página.
      </div>
    );
  }

  const usuarios = await listarUsuarios(accessToken);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Administração de usuários</h1>
        <p className="text-muted-foreground text-sm">
          {usuarios.length} usuário{usuarios.length === 1 ? "" : "s"} provisionado
          {usuarios.length === 1 ? "" : "s"} (via primeiro login no Keycloak).
        </p>
      </div>

      <div className="overflow-x-auto rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nome</TableHead>
              <TableHead>E-mail</TableHead>
              <TableHead>Matrícula</TableHead>
              <TableHead>Fiscal</TableHead>
              <TableHead>Admin</TableHead>
              <TableHead className="text-right">Ações</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {usuarios.map((usuario) => (
              <TableRow key={usuario.ID}>
                <TableCell className="font-medium">{usuario.Nome}</TableCell>
                <TableCell>{usuario.Email}</TableCell>
                <TableCell>{usuario.Matricula || "—"}</TableCell>
                <TableCell>
                  <Badge variant={usuario.IsFiscal ? "default" : "secondary"}>
                    {usuario.IsFiscal ? "Sim" : "Não"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge variant={usuario.IsAdmin ? "default" : "secondary"}>
                    {usuario.IsAdmin ? "Sim" : "Não"}
                  </Badge>
                </TableCell>
                <TableCell className="text-right">
                  <EditarUsuarioDialog usuario={usuario} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
