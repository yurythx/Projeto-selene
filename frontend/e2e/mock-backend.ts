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
  MustChangePassword?: boolean;
}

let usuarios: Usuario[] = [];
// senha em texto puro por e-mail — só existe neste stub (o backend real usa
// bcrypt); mapa separado pra não vazar "senha" num objeto Usuario que a UI
// também usa pra exibição.
let senhasLocais: Record<string, string> = {};

function usuariosSeed(): Usuario[] {
  return [
    { ID: "22222222-2222-4222-8222-222222222222", Nome: "Fiscal Teste", Email: "fiscal@example.com", IsFiscal: true, IsAdmin: false, Matricula: "001" },
    { ID: "33333333-3333-4333-8333-333333333333", Nome: "Admin Teste", Email: "admin@example.com", IsFiscal: false, IsAdmin: true, Matricula: "002" },
    // Conta de login local já "pronta" (senha já trocada) — usada pelo
    // spec que exercita o fluxo de login tradicional.
    { ID: "44444444-4444-4444-8444-444444444444", Nome: "Fiscal Local", Email: "fiscal.local@example.com", IsFiscal: true, IsAdmin: false, Matricula: "", MustChangePassword: false },
    // Conta com senha temporária pendente — usada pelo spec de troca de
    // senha obrigatória.
    { ID: "55555555-5555-4555-8555-555555555555", Nome: "Novo Fiscal Local", Email: "novo.local@example.com", IsFiscal: true, IsAdmin: false, Matricula: "", MustChangePassword: true },
  ];
}

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
    FiscalID: "22222222-2222-4222-8222-222222222222",
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

// SGF-Rondonópolis (adequação às IN SCL 01/2019 e 04/2021) — ver o plano
// em .claude/plans/projeto-selene-rippling-kite.md.
interface PortariaDesignacao {
  ID: string;
  ContratoID: string;
  ServidorID: string;
  Servidor: Usuario;
  Papel: "FISCAL" | "FISCAL_SUPLENTE" | "GESTOR" | "FISCAL_SETORIAL";
  NumeroPortaria: string;
  PublicadoDiorondon: string;
  DataDesignacao: string;
  DataRevogacao: string | null;
  CriadoPorID: string;
  CreatedAt: string;
}

interface Empenho {
  ID: string;
  ContratoID: string;
  NumeroEmpenho: string;
  DataEmissao: string;
  ValorInicial: number;
  CreatedAt: string;
}

interface MovimentacaoEmpenho {
  ID: string;
  EmpenhoID: string;
  Tipo: "INICIAL" | "REFORCO" | "ANULACAO" | "FATURA_APROPRIADA";
  Valor: number;
  ProcessoPagamentoID: string | null;
  Observacao: string;
  RegistradoPorID: string;
  CreatedAt: string;
}

interface Ocorrencia {
  ID: string;
  ContratoID: string;
  ProcessoPagamentoID: string | null;
  Descricao: string;
  Estado: "REGISTRADA" | "NOTIFICADA" | "EM_TRATAMENTO" | "REGULARIZADA";
  RegistradoPorID: string;
  RegistradoPor: Usuario;
  DataNotificacaoGestor: string | null;
  DataRegularizacao: string | null;
  CreatedAt: string;
  UpdatedAt: string;
}

let contratos: Contrato[] = [];
let processos: Processo[] = [];
let vistorias: Vistoria[] = [];
let designacoes: PortariaDesignacao[] = [];
let empenhos: Empenho[] = [];
let movimentacoesEmpenho: MovimentacaoEmpenho[] = [];
let ocorrencias: Ocorrencia[] = [];
const documentosPorProcesso = new Map<string, Documento[]>();

// Liga/desliga via POST /__e2e__/forcar-401 — simula o cenário real que
// expôs a falta de requireApi (lib/api/client.ts): o backend rejeitando
// um token que o frontend achava válido (ex: reinício do backend
// invalidando sessões de login local, ver a LIMITAÇÃO CONHECIDA em
// internal/localauth/localauth.go).
let forcar401 = false;

function resetState() {
  seq = 0;
  forcar401 = false;
  const contratoSeed = criarContratoSeed();
  contratos = [contratoSeed];
  processos = [criarProcessoSeed(contratoSeed)];
  vistorias = [];
  designacoes = [];
  empenhos = [];
  movimentacoesEmpenho = [];
  ocorrencias = [];
  documentosPorProcesso.clear();
  usuarios = usuariosSeed();
  senhasLocais = {
    "fiscal.local@example.com": "senha12345",
    "novo.local@example.com": "temporaria123",
  };
}

// Mesma tabela De/Para do backend (ver
// internal/service/fiscalizacao_service.go) — reimplementada aqui em
// miniatura só pro stub responder de forma plausível ao drawer do Kanban;
// não precisa ser bit-a-bit idêntica (o stub já simplifica outras coisas,
// como o processoChecklistPendenteId acima).
const mapaEtapaEstado: Record<number, string> = {
  1: "A_EXECUTAR_CONFERIR",
  2: "EM_ANALISE_EXTERNA",
  3: "A_EXECUTAR_CONFERIR",
  4: "DOCUMENTAR_ATESTAR",
  5: "DOCUMENTAR_ATESTAR",
  6: "EM_ANALISE_EXTERNA",
};
const mapaEtapaAcao: Record<number, string> = {
  1: "ACAO_FISCAL",
  2: "ESPERA_EXTERNA",
  3: "ACAO_FISCAL",
  4: "ACAO_FISCAL",
  5: "ACAO_FISCAL",
  6: "ESPERA_EXTERNA",
};

function decorarProcesso(processo: Processo) {
  const ocorrenciasAbertas = ocorrencias.filter(
    (o) => o.ProcessoPagamentoID === processo.ID && o.Estado !== "REGULARIZADA"
  );

  let estado_fiscalizacao: string;
  const allowed_actions: string[] = [];

  if (processo.Status === "Concluido") {
    estado_fiscalizacao = "CONCLUIDO";
  } else if (ocorrenciasAbertas.length > 0) {
    estado_fiscalizacao = "PENDENCIA_DEVOLVIDO";
  } else {
    estado_fiscalizacao = mapaEtapaEstado[processo.EtapaAtualID] ?? "A_EXECUTAR_CONFERIR";
  }

  if (processo.Status === "Ativo") {
    allowed_actions.push("ANEXAR_DOCUMENTO", "REGISTRAR_OCORRENCIA", "REGISTRAR_MOVIMENTACAO_EMPENHO");
    if (ocorrenciasAbertas.length === 0) {
      if (processo.EtapaAtualID < 6) allowed_actions.push("AVANCAR_ETAPA");
      else allowed_actions.push("CONCLUIR_PAGAMENTO");
    }
  }

  return {
    ...processo,
    estado_fiscalizacao,
    acao_ou_espera: mapaEtapaAcao[processo.EtapaAtualID] ?? "ACAO_FISCAL",
    allowed_actions,
  };
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

    if (pathname === "/__e2e__/forcar-401" && req.method === "POST") {
      forcar401 = true;
      return json(res, 204, null);
    }

    if (pathname === "/health") return json(res, 200, { status: "ok" });

    // Aplicado depois de reset/health (endpoints de controle/infra, não
    // de domínio) e antes de qualquer rota /api/v1 — mesmo formato de
    // corpo que o middleware Go real usa (ver internal/middleware/auth.go).
    if (forcar401 && pathname.startsWith("/api/v1/") && pathname !== "/api/v1/auth/login") {
      return json(res, 401, { error: "token inválido ou expirado" });
    }

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

    const processoIdMatch = pathname.match(/^\/api\/v1\/processos\/([^/]+)$/);
    if (processoIdMatch && req.method === "GET") {
      const [, id] = processoIdMatch;
      const processo = processos.find((p) => p.ID === id);
      if (!processo) return json(res, 404, { error: "não encontrado" });
      return json(res, 200, decorarProcesso(processo));
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
          EnviadoPorID: "22222222-2222-4222-8222-222222222222",
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
          FiscalID: "22222222-2222-4222-8222-222222222222",
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

    // SGF-Rondonópolis (adequação às IN SCL 01/2019 e 04/2021) — mesmo
    // espírito do resto do stub: só o suficiente pra exercitar o fluxo do
    // frontend, sem reimplementar a lógica de negócio do backend real.
    const ocorrenciasDoProcessoMatch = pathname.match(/^\/api\/v1\/processos\/([^/]+)\/ocorrencias$/);
    if (ocorrenciasDoProcessoMatch) {
      const [, processoId] = ocorrenciasDoProcessoMatch;
      if (req.method === "GET") {
        return json(res, 200, ocorrencias.filter((o) => o.ProcessoPagamentoID === processoId));
      }
      if (req.method === "POST") {
        const processo = processos.find((p) => p.ID === processoId);
        if (!processo) return json(res, 404, { error: "não encontrado" });
        const corpo = bodyAsJSON();
        const agora = new Date().toISOString();
        const nova: Ocorrencia = {
          ID: nextId("ocorrencia"),
          ContratoID: processo.ContratoID,
          ProcessoPagamentoID: processoId,
          Descricao: corpo.descricao ?? "",
          Estado: "REGISTRADA",
          RegistradoPorID: "22222222-2222-4222-8222-222222222222",
          RegistradoPor: usuarios[0],
          DataNotificacaoGestor: null,
          DataRegularizacao: null,
          CreatedAt: agora,
          UpdatedAt: agora,
        };
        ocorrencias.push(nova);
        return json(res, 201, nova);
      }
    }

    const transicaoOcorrenciaMatch = pathname.match(
      /^\/api\/v1\/ocorrencias\/([^/]+)\/(notificar|tratar|regularizar)$/
    );
    if (transicaoOcorrenciaMatch && req.method === "POST") {
      const [, id, acao] = transicaoOcorrenciaMatch;
      const ocorrencia = ocorrencias.find((o) => o.ID === id);
      if (!ocorrencia) return json(res, 404, { error: "não encontrado" });
      const proximaAcao: Record<string, [Ocorrencia["Estado"], Ocorrencia["Estado"]]> = {
        notificar: ["REGISTRADA", "NOTIFICADA"],
        tratar: ["NOTIFICADA", "EM_TRATAMENTO"],
        regularizar: ["EM_TRATAMENTO", "REGULARIZADA"],
      };
      const [origem, destino] = proximaAcao[acao];
      if (ocorrencia.Estado !== origem) {
        return json(res, 400, { error: "transição de estado da ocorrência não permitida a partir do estado atual" });
      }
      ocorrencia.Estado = destino;
      ocorrencia.UpdatedAt = new Date().toISOString();
      if (destino === "NOTIFICADA") ocorrencia.DataNotificacaoGestor = ocorrencia.UpdatedAt;
      if (destino === "REGULARIZADA") ocorrencia.DataRegularizacao = ocorrencia.UpdatedAt;
      return json(res, 200, ocorrencia);
    }

    if (pathname === "/api/v1/servidores" && req.method === "GET") {
      // Projeção mínima (ID/Nome/Email), mesmo contrato do backend real —
      // ver UserHandler.ListarServidores.
      return json(
        res,
        200,
        usuarios.map((u) => ({ ID: u.ID, Nome: u.Nome, Email: u.Email }))
      );
    }

    const designacoesMatch = pathname.match(/^\/api\/v1\/contratos\/([^/]+)\/designacoes$/);
    if (designacoesMatch) {
      const [, contratoId] = designacoesMatch;
      if (req.method === "GET") {
        return json(res, 200, designacoes.filter((d) => d.ContratoID === contratoId));
      }
      if (req.method === "POST") {
        const corpo = bodyAsJSON();
        const servidor = usuarios.find((u) => u.ID === corpo.servidor_id) ?? usuarios[0];
        const nova: PortariaDesignacao = {
          ID: nextId("designacao"),
          ContratoID: contratoId,
          ServidorID: servidor.ID,
          Servidor: servidor,
          Papel: corpo.papel ?? "FISCAL",
          NumeroPortaria: corpo.numero_portaria ?? "",
          PublicadoDiorondon: corpo.publicado_diorondon ?? "",
          DataDesignacao: corpo.data_designacao ?? new Date().toISOString(),
          DataRevogacao: null,
          CriadoPorID: "22222222-2222-4222-8222-222222222222",
          CreatedAt: new Date().toISOString(),
        };
        designacoes.push(nova);
        return json(res, 201, nova);
      }
    }

    const empenhosDoContratoMatch = pathname.match(/^\/api\/v1\/contratos\/([^/]+)\/empenhos$/);
    if (empenhosDoContratoMatch) {
      const [, contratoId] = empenhosDoContratoMatch;
      if (req.method === "GET") {
        return json(res, 200, empenhos.filter((e) => e.ContratoID === contratoId));
      }
      if (req.method === "POST") {
        const corpo = bodyAsJSON();
        const novo: Empenho = {
          ID: nextId("empenho"),
          ContratoID: contratoId,
          NumeroEmpenho: corpo.numero_empenho ?? "",
          DataEmissao: corpo.data_emissao ?? new Date().toISOString(),
          ValorInicial: corpo.valor_inicial ?? 0,
          CreatedAt: new Date().toISOString(),
        };
        empenhos.push(novo);
        movimentacoesEmpenho.push({
          ID: nextId("movimentacao"),
          EmpenhoID: novo.ID,
          Tipo: "INICIAL",
          Valor: novo.ValorInicial,
          ProcessoPagamentoID: null,
          Observacao: "",
          RegistradoPorID: "22222222-2222-4222-8222-222222222222",
          CreatedAt: new Date().toISOString(),
        });
        return json(res, 201, novo);
      }
    }

    const empenhoIdMatch = pathname.match(/^\/api\/v1\/empenhos\/([^/]+)$/);
    if (empenhoIdMatch && req.method === "GET") {
      const [, id] = empenhoIdMatch;
      const empenho = empenhos.find((e) => e.ID === id);
      if (!empenho) return json(res, 404, { error: "não encontrado" });
      const saldo = movimentacoesEmpenho
        .filter((m) => m.EmpenhoID === id)
        .reduce((acc, m) => (m.Tipo === "INICIAL" || m.Tipo === "REFORCO" ? acc + m.Valor : acc - m.Valor), 0);
      return json(res, 200, { ...empenho, saldo });
    }

    const movimentacaoEmpenhoMatch = pathname.match(/^\/api\/v1\/empenhos\/([^/]+)\/movimentacoes$/);
    if (movimentacaoEmpenhoMatch && req.method === "POST") {
      const [, empenhoId] = movimentacaoEmpenhoMatch;
      const empenho = empenhos.find((e) => e.ID === empenhoId);
      if (!empenho) return json(res, 404, { error: "não encontrado" });
      const corpo = bodyAsJSON();
      const nova: MovimentacaoEmpenho = {
        ID: nextId("movimentacao"),
        EmpenhoID: empenhoId,
        Tipo: corpo.tipo,
        Valor: corpo.valor ?? 0,
        ProcessoPagamentoID: corpo.processo_pagamento_id ?? null,
        Observacao: corpo.observacao ?? "",
        RegistradoPorID: "22222222-2222-4222-8222-222222222222",
        CreatedAt: new Date().toISOString(),
      };
      movimentacoesEmpenho.push(nova);
      return json(res, 201, nova);
    }

    // Módulo 4 do roadmap (Dossiê do Fornecedor) — agrupa os contratos em
    // memória por CNPJ (só dígitos), mesma normalização do backend real.
    const apenasDigitos = (v: string) => v.replace(/\D/g, "");
    if (pathname === "/api/v1/fornecedores" && req.method === "GET") {
      const porCnpj = new Map<string, { cnpj: string; cnpj_formatado: string; nome: string; qtd_contratos: number; qtd_contratos_ativos: number }>();
      for (const c of contratos) {
        const chave = apenasDigitos(c.ContratadaCNPJ);
        const atual = porCnpj.get(chave) ?? { cnpj: chave, cnpj_formatado: c.ContratadaCNPJ, nome: c.ContratadaNome, qtd_contratos: 0, qtd_contratos_ativos: 0 };
        atual.qtd_contratos += 1;
        if (c.Ativo) atual.qtd_contratos_ativos += 1;
        porCnpj.set(chave, atual);
      }
      return json(res, 200, Array.from(porCnpj.values()));
    }
    const fornecedorMatch = pathname.match(/^\/api\/v1\/fornecedores\/([^/]+)$/);
    if (fornecedorMatch && req.method === "GET") {
      const alvo = apenasDigitos(decodeURIComponent(fornecedorMatch[1]));
      const contratosDoFornecedor = contratos.filter((c) => apenasDigitos(c.ContratadaCNPJ) === alvo);
      if (contratosDoFornecedor.length === 0) {
        return json(res, 404, { error: "não encontrado" });
      }
      return json(res, 200, {
        cnpj: alvo,
        cnpj_formatado: contratosDoFornecedor[0].ContratadaCNPJ,
        nome: contratosDoFornecedor[0].ContratadaNome,
        contratos: contratosDoFornecedor,
        notificacoes: [],
        score_pontualidade: null,
      });
    }

    // Login local (usuário/senha) — alternativa ao Keycloak.
    if (pathname === "/api/v1/auth/login" && req.method === "POST") {
      const corpo = bodyAsJSON();
      const usuario = usuarios.find((u) => u.Email === corpo.email);
      if (!usuario || senhasLocais[corpo.email] !== corpo.senha) {
        return json(res, 401, { error: "e-mail ou senha inválidos" });
      }
      return json(res, 200, { access_token: `fake-local-token-${usuario.ID}`, usuario });
    }

    if (pathname === "/api/v1/auth/trocar-senha" && req.method === "POST") {
      const auth = req.headers.authorization ?? "";
      const usuarioId = auth.replace("Bearer fake-local-token-", "");
      const usuario = usuarios.find((u) => u.ID === usuarioId);
      if (!usuario) return json(res, 401, { error: "não autenticado" });

      const corpo = bodyAsJSON();
      if (senhasLocais[usuario.Email] !== corpo.senha_atual) {
        return json(res, 401, { error: "e-mail ou senha inválidos" });
      }
      senhasLocais[usuario.Email] = corpo.senha_nova;
      usuario.MustChangePassword = false;
      return json(res, 204, null);
    }

    if (pathname === "/api/v1/admin/users/local" && req.method === "POST") {
      const corpo = bodyAsJSON();
      const novo: Usuario = {
        ID: nextId("local-user"),
        Nome: corpo.nome,
        Email: corpo.email,
        IsFiscal: Boolean(corpo.is_fiscal),
        IsAdmin: Boolean(corpo.is_admin),
        Matricula: "",
        MustChangePassword: true,
      };
      usuarios.push(novo);
      senhasLocais[novo.Email] = corpo.senha_temporaria;
      return json(res, 201, novo);
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
