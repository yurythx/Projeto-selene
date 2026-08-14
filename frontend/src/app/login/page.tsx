import { signIn } from "@/auth";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { CredentialsLoginForm } from "@/components/login/credentials-login-form";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ callbackUrl?: string }>;
}) {
  const { callbackUrl } = await searchParams;
  const destino = callbackUrl ?? "/";

  return (
    <div className="flex min-h-svh items-center justify-center px-4">
      <div className="w-full max-w-sm space-y-8">
        <div className="space-y-1 text-center">
          <h1 className="text-2xl font-semibold">Projeto Selene</h1>
          <p className="text-muted-foreground text-sm">
            Fiscalização de contratos — acesse com sua conta institucional ou
            com um e-mail e senha cadastrados por um administrador.
          </p>
        </div>

        <form
          action={async () => {
            "use server";
            await signIn("keycloak", { redirectTo: destino });
          }}
        >
          <Button type="submit" className="w-full">
            Entrar com Keycloak
          </Button>
        </form>

        <div className="flex items-center gap-3">
          <Separator className="flex-1" />
          <span className="text-muted-foreground text-xs">ou</span>
          <Separator className="flex-1" />
        </div>

        <CredentialsLoginForm callbackUrl={destino} />
      </div>
    </div>
  );
}
