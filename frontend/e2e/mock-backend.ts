/**
 * Stub do backend Go usado nos testes E2E — implementa só o subconjunto
 * de rotas exercitado pelas specs, com dados em memória.
 *
 * Por quê um stub, e não o backend real: os testes E2E validam o
 * FRONTEND (rotas, formulários, o proxy BFF, tratamento do 422 de
 * checklist etc.), não o backend — isso já é coberto pelos testes Go do
 * próprio backend. Rodar contra um Keycloak real também exigiria
 * credenciais de um usuário de verdade na instância de produção da
 * prefeitura, o que não faz sentido usar em CI. A sessão do Auth.js é
 * injetada diretamente (ver e2e/fixtures/auth.ts), pulando o login
 * interativo — esse fluxo (redirect pro Keycloak, construção da URL de
 * autorização) já foi validado manualmente contra o Keycloak real.
 *
 * Sem dependências novas — só http/node built-in.
 */
import { createServer } from "node:http";

interface Usuario {
  ID: string;
  Nome: string;
  Email: string;
  IsFiscal: boolean;
  IsAdmin: boolean;
  Matricula: string;
}

const usuarios: Usuario[] = [
  { ID: "fiscal-1", Nome: "Fiscal Teste", Email: "fiscal@example.com", IsFiscal: true, IsAdmin: false, Matricula: "001" },
  { ID: "admin-1", Nome: "Admin Teste", Email: "admin@example.com", IsFiscal: false, IsAdmin: true, Matricula: "002" },
];

const etapas = [
  { ID: 1, Nome: "Elaborar OF / Pré-Empenho", Posicao: 1 },
  { ID: 2, Nome: "Tramitação Externa", Posicao: 2 },
  { ID: 3, Nome: "Empenho", Posicao: 3 },
  { ID: 4, Nome: "Recebimento", Posicao: 4 },
  { ID: 5, Nome: "Regularidade Fiscal", Posicao: 5 },
  { ID: 6, Nome: "Contabilidade", Posicao: 6 },
];

const tiposDocumento = [
  { ID: 1, Nome: "Ordem de Fornecimento (OF)" },
  { ID: 2, Nome: "Nota Fiscal / Fatura" },
  // ExigeValidade=true testa o campo condicional "data_validade" no
  // formulário de upload (ver components/kanban/processo-dialog.tsx).
  { ID: 3, Nome: "CND Trabalhista", ExigeValidade: true },
];

interface Contrato {
  ID: string;
  NumeroContrato: string;
  PortariaNomeacao: string;
  DataAssinatura: string;
  ContratadaNome: string;
  ContratadaCNPJ: string;
  ContratadaEmail: string;
  FiscalID: string;
  Fiscal: Usuario;
  TipoObjeto: string;
  Ativo: boolean;
  DataVigenciaFim: string | null;
  CreatedAt: string;
  UpdatedAt: string;
}

interface Processo {
  ID: string;
  ContratoID: string;
  Contrato: Contrato;
  MesReferencia: string;
  EtapaAtualID: number;
  EtapaAtual: (typeof etapas)[number];
  Status: "Ativo" | "Concluido";
  CreatedAt: string;
  UpdatedAt: string;
}

interface Documento {
  ID: string;
  ProcessoPagamentoID: string;
  TipoDocumentoID: number;
  TipoDocumento: (typeof tiposDocumento)[number];
  NomeArquivo: string;
  HashArquivo: string;
  EnviadoPorID: string;
  EnviadoPor: Usuario;
  DataUpload: string;
}

let seq = 0;
const nextId = (prefix: string) => `${prefix}-${++seq}`;

// Processo cujo avancar SEMPRE responde 422 — usado pra testar o fluxo de
// checklist incompleto sem precisar de lógica condicional no client.
const processoChecklistPendenteId = "processo-checklist-pendente";

// Fábricas, não constantes — cada teste que edita/encerra um contrato ou
// avança um processo MUTA o objeto diretamente (como o backend real faz
// numa entidade). Se o seed fosse um objeto compartilhado, resetState()
// só reapontaria o array pra ele, mas a mutação em si (Ativo=false,
// ContratadaNome renomeado, etc.) continuaria — vazando estado de um
// teste pro próximo. Cada reset precisa de objetos novos.
function criarContratoSeed(): Contrato {
  return {
    // Precisa ter formato de UUID de verdade: novoProcessoSchema
    // (lib/validation/bff-schemas.ts) valida contrato_id com z.uuid(),
    // igual o backend real exige (Contrato.ID é uuid.UUID no Go).
    ID: "11111111-1111-4111-8111-111111111111",
    NumeroContrato: "1/2026",
    PortariaNomeacao: "Portaria 1/2026",
    DataAssinatura: "2026-01-01",
    ContratadaNome: "Fornecedora Seed Ltda",
    ContratadaCNPJ: "00.000.000/0001-00",
    ContratadaEmail: "contato@fornecedora-seed.example",
    FiscalID: "fiscal-1",
    Fiscal: usuarios[0],
    TipoObjeto: "SERVICO",
    Ativo: true,
    DataVigenciaFim: null,
    CreatedAt: new Date().toISOString(),
    UpdatedAt: new Date().toISOString(),
  };
}

function criarProcessoSeed(contrato: Contrato): Processo {
  return {
    ID: processoChecklistPendenteId,
    ContratoID: contrato.ID,
    Contrato: contrato,
    MesReferencia: "01/2026",
    EtapaAtualID: 1,
    EtapaAtual: etapas[0],
    Status: "Ativo",
    CreatedAt: new Date().toISOString(),
    UpdatedAt: new Date().toISOString(),
  };
}

interface Vistoria {
  ID: string;
  ProcessoPagamentoID: string;
  FiscalID: string;
  Fiscal: Usuario;
  DataHora: string;
  Latitude: number | null;
  Longitude: number | null;
  Observacoes: string;
  Fotos: { ID: string; NomeArquivo: string }[];
  CreatedAt: string;
}

let contratos: Contrato[] = [];
let processos: Processo[] = [];
let vistorias: Vistoria[] = [];
const documentosPorProcesso = new Map<string, Documento[]>();

function resetState() {
  seq = 0;
  const contratoSeed = criarContratoSeed();
  contratos = [contratoSeed];
  processos = [criarProcessoSeed(contratoSeed)];
  vistorias = [];
  documentosPorProcesso.clear();
}

resetState();

function json(res: import("node:http").ServerResponse, status: number, body: unknown) {
  const payload = JSON.stringify(body);
  res.writeHead(status, { "Content-Type": "application/json", "Content-Length": Buffer.byteLength(payload) });
  res.end(payload);
}

function paginado<T>(dados: T[]) {
  return { total: dados.length, pagina: 1, tamanho_pagina: 100, dados };
}

const port = Number(process.env.MOCK_BACKEND_PORT ?? 4010);

const server = createServer((req, res) => {
  const url = new URL(req.url ?? "/", `http://localhost:${port}`);
  const { pathname, searchParams } = url;
  if (process.env.MOCK_BACKEND_DEBUG) {
    console.log(`[mock-backend] ${req.method} ${pathname}${url.search}`);
  }
  const chunks: Buffer[] = [];

  req.on("data", (chunk) => chunks.push(chunk));
  req.on("end", () => {
    const rawBody = Buffer.concat(chunks);
    const bodyAsJSON = () => {
      try {
        return JSON.parse(rawBody.toString("utf-8") || "{}");
      } catch {
        return {};
      }
    };

    // Endpoint de controle exclusivo dos testes — não existe no backend
    // real, usado só pra isolar specs entre si.
    if (pathname === "/__e2e__/reset" && req.method === "POST") {
      resetState();
      return json(res, 204, null);
    }

    if (pathname === "/health") return json(res, 200, { status: "ok" });

    if (pathname === "/api/v1/kanban/etapas") return json(res, 200, etapas);
    if (pathname === "/api/v1/kanban/tipos-documento") return json(res, 200, tiposDocumento);

    // Radar de Alertas (Fase 1 do roadmap) — vazio por padrão: nenhum
    // dos specs hoje depende de badges de alerta aparecendo, só de a
    // chamada não quebrar o carregamento do Kanban.
    if (pathname === "/api/v1/radar" && req.method === "GET") {
      return json(res, 200, []);
    }

    if (pathname === "/api/v1/contratos" && req.method === "GET") {
      return json(res, 200, paginado(contratos));
    }
    if (pathname === "/api/v1/contratos" && req.method === "POST") {
      const corpo = bodyAsJSON();
      const fiscal = usuarios.find((u) => u.ID === corpo.fiscal_id) ?? usuarios[0];
      const novo: Contrato = {
        ID: nextId("contrato"),
        NumeroContrato: corpo.numero_contrato,
        PortariaNomeacao: corpo.portaria_nomeacao ?? "",
        DataAssinatura: corpo.data_assinatura,
        ContratadaNome: corpo.contratada_nome,
        ContratadaCNPJ: corpo.contratada_cnpj,
        ContratadaEmail: corpo.contratada_email ?? "",
        FiscalID: fiscal.ID,
        Fiscal: fiscal,
        TipoObjeto: corpo.tipo_objeto,
        Ativo: true,
        DataVigenciaFim: corpo.data_vigencia_fim || null,
        CreatedAt: new Date().toISOString(),
        UpdatedAt: new Date().toISOString(),
      };
      contratos = [novo, ...contratos];
      return json(res, 201, novo);
    }

    const contratoMatch = pathname.match(/^\/api\/v1\/contratos\/([^/]+)(\/encerrar)?$/);
    if (contratoMatch) {
      const [, id, encerrar] = contratoMatch;
      const contrato = contratos.find((c) => c.ID === id);
      if (!contrato) return json(res, 404, { error: "não encontrado" });

      if (encerrar && req.method === "POST") {
        contrato.Ativo = false;
        return json(res, 200, contrato);
      }
      if (!encerrar && req.method === "GET") return json(res, 200, contrato);
      if (!encerrar && req.method === "PATCH") {
        const corpo = bodyAsJSON();
        Object.assign(contrato, {
          PortariaNomeacao: corpo.portaria_nomeacao ?? contrato.PortariaNomeacao,
          ContratadaNome: corpo.contratada_nome ?? contrato.ContratadaNome,
          ContratadaCNPJ: corpo.contratada_cnpj ?? contrato.ContratadaCNPJ,
          ContratadaEmail: corpo.contratada_email ?? contrato.ContratadaEmail,
          DataVigenciaFim:
            corpo.data_vigencia_fim !== undefined
              ? corpo.data_vigencia_fim || null
              : contrato.DataVigenciaFim,
        });
        return json(res, 200, contrato);
      }
    }

    if (pathname === "/api/v1/processos" && req.method === "GET") {
      const etapaId = Number(searchParams.get("etapa"));
      return json(res, 200, paginado(processos.filter((p) => p.EtapaAtualID === etapaId)));
    }
    if (pathname === "/api/v1/processos" && req.method === "POST") {
      const corpo = bodyAsJSON();
      const contrato = contratos.find((c) => c.ID === corpo.contrato_id) ?? contratos[0];
      const novo: Processo = {
        ID: nextId("processo"),
        ContratoID: contrato.ID,
        Contrato: contrato,
        MesReferencia: corpo.mes_referencia,
        EtapaAtualID: 1,
        EtapaAtual: etapas[0],
        Status: "Ativo",
        CreatedAt: new Date().toISOString(),
        UpdatedAt: new Date().toISOString(),
      };
      processos = [...processos, novo];
      return json(res, 201, novo);
    }

    const processoAcaoMatch = pathname.match(/^\/api\/v1\/processos\/([^/]+)\/(avancar|concluir)$/);
    if (processoAcaoMatch && req.method === "POST") {
      const [, id, acao] = processoAcaoMatch;
      const processo = processos.find((p) => p.ID === id);
      if (!processo) return json(res, 404, { error: "não encontrado" });

      if (acao === "avancar") {
        if (id === processoChecklistPendenteId) {
          return json(res, 422, {
            error: "checklist incompleto",
            documentos_pendentes: ["Ordem de Fornecimento (OF)", "Pré-Empenho"],
          });
        }
        processo.EtapaAtualID += 1;
        processo.EtapaAtual = etapas[processo.EtapaAtualID - 1];
        return json(res, 200, processo);
      }

      processo.Status = "Concluido";
      return json(res, 200, processo);
    }

    const documentosMatch = pathname.match(/^\/api\/v1\/processos\/([^/]+)\/documentos$/);
    if (documentosMatch) {
      const [, id] = documentosMatch;
      if (req.method === "GET") {
        return json(res, 200, documentosPorProcesso.get(id) ?? []);
      }
      if (req.method === "POST") {
        // multipart/form-data mínimo o suficiente pro teste: não faz
        // parsing de verdade, só confirma que a chamada chegou e devolve
        // um documento fixo.
        const doc: Documento = {
          ID: nextId("doc"),
          ProcessoPagamentoID: id,
          TipoDocumentoID: 1,
          TipoDocumento: tiposDocumento[0],
          NomeArquivo: "arquivo-teste.pdf",
          HashArquivo: "hash-fake",
          EnviadoPorID: "fiscal-1",
          EnviadoPor: usuarios[0],
          DataUpload: new Date().toISOString(),
        };
        const lista = documentosPorProcesso.get(id) ?? [];
        lista.push(doc);
        documentosPorProcesso.set(id, lista);
        return json(res, 201, doc);
      }
    }

    const relatorioMatch = pathname.match(/^\/api\/v1\/processos\/([^/]+)\/relatorio$/);
    if (relatorioMatch && req.method === "GET") {
      const pdfFalso = Buffer.from("%PDF-1.4 conteúdo de teste");
      res.writeHead(200, { "Content-Type": "application/pdf", "Content-Length": pdfFalso.length });
      return res.end(pdfFalso);
    }

    // Módulo 2 do roadmap (Gerador de Documentos) — os 3 geradores só
    // precisam devolver um PDF fake pra exercitar o fluxo do frontend
    // (abrir numa aba nova); nenhum spec hoje verifica o conteúdo do PDF.
    const notificacaoMatch = pathname.match(/^\/api\/v1\/contratos\/([^/]+)\/notificacao$/);
    if (notificacaoMatch && req.method === "POST") {
      const pdfFalso = Buffer.from("%PDF-1.4 notificacao de teste");
      res.writeHead(200, { "Content-Type": "application/pdf", "Content-Length": pdfFalso.length });
      return res.end(pdfFalso);
    }
    const minutaAditivoMatch = pathname.match(/^\/api\/v1\/contratos\/([^/]+)\/minuta-aditivo$/);
    if (minutaAditivoMatch && req.method === "POST") {
      const pdfFalso = Buffer.from("%PDF-1.4 minuta de aditivo de teste");
      res.writeHead(200, { "Content-Type": "application/pdf", "Content-Length": pdfFalso.length });
      return res.end(pdfFalso);
    }
    const atestoMatch = pathname.match(/^\/api\/v1\/processos\/([^/]+)\/atesto$/);
    if (atestoMatch && req.method === "POST") {
      const pdfFalso = Buffer.from("%PDF-1.4 atesto de teste");
      res.writeHead(200, { "Content-Type": "application/pdf", "Content-Length": pdfFalso.length });
      return res.end(pdfFalso);
    }
    const verificarMatch = pathname.match(/^\/api\/v1\/verificar\/([^/]+)$/);
    if (verificarMatch && req.method === "GET") {
      return json(res, 200, { valido: false });
    }

    // Módulo 3 do roadmap (Vistorias) — em memória, mesmo espírito do
    // resto do stub: só o suficiente pra exercitar o fluxo do frontend.
    const vistoriasDoProcessoMatch = pathname.match(/^\/api\/v1\/processos\/([^/]+)\/vistorias$/);
    if (vistoriasDoProcessoMatch) {
      const [, processoId] = vistoriasDoProcessoMatch;
      if (req.method === "GET") {
        return json(res, 200, vistorias.filter((v) => v.ProcessoPagamentoID === processoId));
      }
      if (req.method === "POST") {
        const corpo = bodyAsJSON();
        const nova = {
          ID: nextId("vistoria"),
          ProcessoPagamentoID: processoId,
          FiscalID: "fiscal-1",
          Fiscal: usuarios[0],
          DataHora: new Date().toISOString(),
          Latitude: corpo.latitude ?? null,
          Longitude: corpo.longitude ?? null,
          Observacoes: corpo.observacoes ?? "",
          Fotos: [] as { ID: string; NomeArquivo: string }[],
          CreatedAt: new Date().toISOString(),
        };
        vistorias.push(nova);
        return json(res, 201, nova);
      }
    }

    const fotoVistoriaMatch = pathname.match(/^\/api\/v1\/vistorias\/([^/]+)\/fotos$/);
    if (fotoVistoriaMatch && req.method === "POST") {
      const [, vistoriaId] = fotoVistoriaMatch;
      const vistoria = vistorias.find((v) => v.ID === vistoriaId);
      if (!vistoria) return json(res, 404, { error: "não encontrado" });
      const foto = { ID: nextId("foto-vistoria"), NomeArquivo: "foto-teste.jpg" };
      vistoria.Fotos.push(foto);
      return json(res, 201, foto);
    }

    const relatorioCampoMatch = pathname.match(/^\/api\/v1\/vistorias\/([^/]+)\/relatorio$/);
    if (relatorioCampoMatch && req.method === "GET") {
      const pdfFalso = Buffer.from("%PDF-1.4 relatorio de campo de teste");
      res.writeHead(200, { "Content-Type": "application/pdf", "Content-Length": pdfFalso.length });
      return res.end(pdfFalso);
    }

    if (pathname === "/api/v1/admin/users" && req.method === "GET") {
      return json(res, 200, usuarios);
    }
    const usuarioMatch = pathname.match(/^\/api\/v1\/admin\/users\/([^/]+)$/);
    if (usuarioMatch && req.method === "PATCH") {
      const usuario = usuarios.find((u) => u.ID === usuarioMatch[1]);
      if (!usuario) return json(res, 404, { error: "não encontrado" });
      const corpo = bodyAsJSON();
      Object.assign(usuario, {
        IsFiscal: corpo.is_fiscal ?? usuario.IsFiscal,
        IsAdmin: corpo.is_admin ?? usuario.IsAdmin,
        Matricula: corpo.matricula ?? usuario.Matricula,
      });
      return json(res, 200, usuario);
    }

    json(res, 404, { error: `rota não mockada: ${req.method} ${pathname}` });
  });
});

server.listen(port, () => {
  console.log(`[mock-backend] ouvindo em http://localhost:${port}`);
});
