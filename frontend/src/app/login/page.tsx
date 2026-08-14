import { signIn } from "@/auth";
import { Button } from "@/components/ui/button";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ callbackUrl?: string }>;
}) {
  const { callbackUrl } = await searchParams;

  return (
    <div className="flex min-h-svh items-center justify-center px-4">
      <div className="w-full max-w-sm space-y-8 text-center">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold">Projeto Selene</h1>
          <p className="text-muted-foreground text-sm">
            Fiscalização de contratos — acesse com sua conta institucional.
          </p>
        </div>
        <form
          action={async () => {
            "use server";
            await signIn("keycloak", { redirectTo: callbackUrl ?? "/" });
          }}
        >
          <Button type="submit" className="w-full">
            Entrar com Keycloak
          </Button>
        </form>
      </div>
    </div>
  );
}
