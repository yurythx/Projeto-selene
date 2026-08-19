import Link from "next/link";
import { FileTextIcon, KeyRoundIcon, NewspaperIcon, SearchIcon, ChevronRightIcon } from "lucide-react";
import { auth } from "@/auth";
import { Card, CardContent } from "@/components/ui/card";

const SECOES = [
  {
    href: "/configuracoes/modelos-documentos",
    titulo: "Modelos de Documentos",
    descricao: "Suba um .docx de referência por categoria — associe a um gatilho pra substituir o PDF padrão na geração real.",
    Icon: FileTextIcon,
  },
  {
    href: "/configuracoes/keycloak",
    titulo: "Keycloak / SSO",
    descricao: "Client ID, Client Secret e Issuer do login institucional — editável em runtime, sem reiniciar os containers.",
    Icon: KeyRoundIcon,
  },
  // Diário Oficial em DUAS seções separadas no hub (pedido explícito do
  // usuário) — antes era uma seção só, com a busca alcançável só por um
  // link "Ir para a busca →" dentro da tela de configuração. Mesmo par
  // de telas de antes (diario-oficial/page.tsx e .../buscar/page.tsx),
  // só a entrada no hub que virou duas, cada uma indo direto pro seu
  // destino — mesmo padrão das outras seções, que não se cruzam.
  {
    href: "/configuracoes/diario-oficial",
    titulo: "Diário Oficial — Configuração",
    descricao: "Cadastre e teste a conexão com a API do Diário Oficial da cidade (URL base + chave de API).",
    Icon: NewspaperIcon,
  },
  {
    href: "/configuracoes/diario-oficial/buscar",
    titulo: "Diário Oficial — Busca",
    descricao: "Busque novos contratos publicados no Diário Oficial por nome, CPF e data.",
    Icon: SearchIcon,
  },
];

// Hub de Configurações — ponto de entrada único (link "Configurações" da
// sidebar aponta pra cá) pras telas administrativas que não cabem em
// nenhuma das áreas de negócio (Kanban/Contratos/Radar/Fornecedores).
export default async function ConfiguracoesPage() {
  const session = await auth();

  if (!session?.user?.isAdmin) {
    return (
      <div className="text-muted-foreground text-sm">
        Você não tem permissão para acessar esta página.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Configurações</h1>
        <p className="text-muted-foreground text-sm">Ajustes administrativos do sistema.</p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {SECOES.map(({ href, titulo, descricao, Icon }) => (
          <Link key={href} href={href}>
            <Card className="hover:border-primary/50 h-full shadow-sm transition-colors">
              <CardContent className="flex items-start gap-3">
                <Icon className="text-muted-foreground mt-0.5 size-6 shrink-0" />
                <div className="min-w-0 flex-1">
                  <p className="font-medium">{titulo}</p>
                  <p className="text-muted-foreground mt-1 text-sm">{descricao}</p>
                </div>
                <ChevronRightIcon className="text-muted-foreground mt-0.5 size-4 shrink-0" />
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}
