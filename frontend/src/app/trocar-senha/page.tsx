import { auth } from "@/auth";
import { TrocarSenhaForm } from "@/components/login/trocar-senha-form";

export default async function TrocarSenhaPage() {
  const session = await auth();
  const obrigatoria = Boolean(session?.user?.mustChangePassword);

  return (
    <div className="flex min-h-svh items-center justify-center px-4">
      <div className="w-full max-w-sm space-y-8">
        <div className="space-y-1 text-center">
          <h1 className="text-2xl font-semibold">Trocar senha</h1>
        </div>
        <TrocarSenhaForm obrigatoria={obrigatoria} />
      </div>
    </div>
  );
}
