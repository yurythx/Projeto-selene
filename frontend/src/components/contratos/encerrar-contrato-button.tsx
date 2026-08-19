"use client";

import { useRouter } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";

// Botão de encerrar contrato (soft-close, Contrato.Ativo=false) — pede
// confirmação antes de mandar a requisição.
export function EncerrarContratoButton({ contratoId }: { contratoId: string }) {
  const router = useRouter();

  const mutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(`/api/contratos/${contratoId}/encerrar`, { method: "POST" });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Não foi possível encerrar o contrato.");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("Contrato encerrado.");
      router.refresh();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <Button
      variant="destructive"
      disabled={mutation.isPending}
      onClick={() => {
        // Ação irreversível (sem reativação automática, ver
        // ContratoService.Encerrar no backend) — confirmação simples é
        // suficiente pra v1, um AlertDialog dedicado é um upgrade de UX
        // futuro, não uma lacuna de segurança.
        if (window.confirm("Encerrar este contrato? Não é possível reverter automaticamente.")) {
          mutation.mutate();
        }
      }}
    >
      {mutation.isPending ? "Encerrando..." : "Encerrar contrato"}
    </Button>
  );
}
