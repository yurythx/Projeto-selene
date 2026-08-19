"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import type { Usuario, AtualizarUsuarioRequest } from "@/lib/api/client";

// Dialog admin-only (ver /admin/usuarios) pra alternar is_fiscal/is_admin
// e editar a matrícula de um usuário existente.
export function EditarUsuarioDialog({ usuario }: { usuario: Usuario }) {
  const [open, setOpen] = useState(false);
  const [isFiscal, setIsFiscal] = useState(Boolean(usuario.IsFiscal));
  const [isAdmin, setIsAdmin] = useState(Boolean(usuario.IsAdmin));
  const [matricula, setMatricula] = useState(usuario.Matricula ?? "");
  const router = useRouter();

  const mutation = useMutation({
    mutationFn: async (dados: AtualizarUsuarioRequest) => {
      const res = await fetch(`/api/admin/usuarios/${usuario.ID}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(dados),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível salvar as alterações.");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("Usuário atualizado.");
      setOpen(false);
      router.refresh();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" size="sm">Editar</Button>} />
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{usuario.Nome}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="matricula">Matrícula</Label>
            <Input id="matricula" value={matricula} onChange={(e) => setMatricula(e.target.value)} />
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="is_fiscal">É fiscal de contratos</Label>
            <Switch id="is_fiscal" checked={isFiscal} onCheckedChange={setIsFiscal} />
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="is_admin">É administrador</Label>
            <Switch id="is_admin" checked={isAdmin} onCheckedChange={setIsAdmin} />
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            disabled={mutation.isPending}
            onClick={() =>
              mutation.mutate({ is_fiscal: isFiscal, is_admin: isAdmin, matricula })
            }
          >
            {mutation.isPending ? "Salvando..." : "Salvar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
