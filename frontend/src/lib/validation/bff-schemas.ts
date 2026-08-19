import "server-only";
import { z } from "zod";

/**
 * Schemas de validação dos Route Handlers do BFF (app/api/**). Route
 * Handlers são endpoints HTTP públicos implícitos — qualquer requisição
 * com um cookie de sessão válido pode chamá-los diretamente (curl,
 * Postman), pulando o formulário React e a validação Zod que ele já faz
 * no client. Os campos aqui espelham os schemas dos formulários (ver
 * components/contratos/*.tsx, components/kanban/*.tsx) porque a UX de
 * erro já foi pensada lá — a diferença é que ESTA validação roda no
 * servidor e não pode ser contornada.
 *
 * O backend Go também valida (`binding:"required"`), mas só checa
 * presença, não formato — nem todo campo tem uma regra de formato lá
 * (CNPJ, e-mail, "MM/AAAA", enum de tipo_objeto). Validar aqui também é
 * defesa em profundidade, não redundância inútil.
 */

// data_vigencia_fim ("AAAA-MM-DD") é opcional — alimenta o Radar de
// Alertas (Fase 1 do roadmap); string vazia é aceita como "não informado"
// nos dois schemas pra combinar com o <input type="date"> do form, que
// emite "" quando o campo fica em branco.
const dataOpcionalSchema = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/, "formato esperado: AAAA-MM-DD")
  .optional()
  .or(z.literal(""));

// cnpjSchema aceita o CNPJ mascarado ("12.345.678/0001-90") ou só dígitos
// — checa apenas a quantidade de dígitos (14), não os dígitos
// verificadores (módulo 11): mesmo escopo do backend, ver o comentário em
// cnpjValido (backend/internal/service/util.go) sobre por que checksum
// fica de fora por ora.
const cnpjSchema = z
  .string()
  .trim()
  .min(1, "obrigatório")
  .max(20)
  .refine((v) => v.replace(/\D/g, "").length === 14, "CNPJ inválido — informe os 14 dígitos");

export const novoContratoSchema = z
  .object({
    numero_contrato: z.string().trim().min(1, "obrigatório").max(50),
    portaria_nomeacao: z.string().trim().max(255).optional(),
    data_assinatura: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "formato esperado: AAAA-MM-DD"),
    contratada_nome: z.string().trim().min(1, "obrigatório").max(255),
    contratada_cnpj: cnpjSchema,
    contratada_email: z.string().trim().email("e-mail inválido").optional().or(z.literal("")),
    tipo_objeto: z.enum(["CONSUMO", "PERMANENTE", "SERVICO"]),
    data_vigencia_fim: dataOpcionalSchema,
    // Camada 2 (regra do SGF, não da norma): marca contrato de mão de obra
    // terceirizada, sujeito à IN SCL Nº 04/2021. Opcional, default false.
    exige_fiscalizacao_terceirizacao: z.boolean().optional(),
  })
  // Mesma regra do backend (ContratoService.Criar, service/errors.go —
  // ErrVigenciaAntesDaAssinatura): um contrato não pode vencer antes (ou
  // no mesmo dia) de ser assinado. Validado aqui também para dar o erro
  // já no formulário, sem precisar de round-trip até o backend.
  .refine(
    (v) => !v.data_vigencia_fim || v.data_vigencia_fim > v.data_assinatura,
    {
      message: "a vigência final precisa ser posterior à data de assinatura",
      path: ["data_vigencia_fim"],
    }
  );

export const atualizarContratoSchema = z.object({
  portaria_nomeacao: z.string().trim().max(255).optional(),
  contratada_nome: z.string().trim().min(1, "obrigatório").max(255).optional(),
  contratada_cnpj: cnpjSchema.optional(),
  contratada_email: z.string().trim().email("e-mail inválido").optional().or(z.literal("")),
  data_vigencia_fim: dataOpcionalSchema,
  exige_fiscalizacao_terceirizacao: z.boolean().optional(),
});

export const novoProcessoSchema = z.object({
  contrato_id: z.string().uuid("contrato_id precisa ser um UUID"),
  mes_referencia: z.string().regex(/^(0[1-9]|1[0-2])\/\d{4}$/, "formato esperado: MM/AAAA"),
});

export const atualizarUsuarioSchema = z.object({
  is_fiscal: z.boolean().optional(),
  is_admin: z.boolean().optional(),
  matricula: z.string().trim().max(50).optional(),
});

// tipo_documento_id chega como string (campo de FormData) — z.coerce
// converte antes de validar como inteiro positivo.
export const tipoDocumentoIdSchema = z.coerce.number().int().positive();

// data_validade ("AAAA-MM-DD") também chega como campo de FormData,
// sempre string — vazio/ausente é válido (nem todo tipo de documento
// exige validade, ver TipoDocumento.ExigeValidade).
export const dataValidadeSchema = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/, "formato esperado: AAAA-MM-DD")
  .optional()
  .or(z.literal(""));

// Módulo 2 do roadmap: Gerador Inteligente de Documentos Legais.

export const gerarNotificacaoSchema = z.object({
  motivo: z.string().trim().min(1, "obrigatório").max(2000),
});

export const minutaAditivoSchema = z.object({
  tipo_aditivo: z.enum(["VALOR", "PRAZO", "VALOR_E_PRAZO"]),
  justificativa: z.string().trim().min(1, "obrigatório").max(2000),
  novo_valor: z.string().trim().max(255).optional(),
  novo_prazo: z.string().trim().max(255).optional(),
});

// Módulo 3 do roadmap: Vistorias e Registro Fotográfico com
// Geolocalização. Latitude/longitude vêm de navigator.geolocation no
// client — nullable de propósito (permissão negada/indisponível).
export const registrarVistoriaSchema = z.object({
  latitude: z.number().min(-90).max(90).nullable().optional(),
  longitude: z.number().min(-180).max(180).nullable().optional(),
  observacoes: z.string().trim().max(4000).optional(),
});

// Login local (usuário/senha) — contas criadas só por admin.
export const criarUsuarioLocalSchema = z.object({
  nome: z.string().trim().min(1, "obrigatório").max(255),
  email: z.string().trim().email("e-mail inválido"),
  senha_temporaria: z.string().min(8, "mínimo 8 caracteres"),
  is_fiscal: z.boolean().optional(),
  is_admin: z.boolean().optional(),
});

export const trocarSenhaSchema = z.object({
  senha_atual: z.string().min(1, "obrigatório"),
  senha_nova: z.string().min(8, "mínimo 8 caracteres"),
});

// SGF-Rondonópolis (adequação às IN SCL 01/2019 e 04/2021) — ver o plano
// em .claude/plans/projeto-selene-rippling-kite.md.
export const designarSchema = z.object({
  servidor_id: z.string().uuid(),
  papel: z.enum(["FISCAL", "FISCAL_SUPLENTE", "GESTOR", "FISCAL_SETORIAL"]),
  numero_portaria: z.string().trim().max(50).optional(),
  publicado_diorondon: z.string().trim().max(50).optional(),
  // "AAAA-MM-DD" — mesma convenção de data_assinatura/data_emissao (a
  // coluna portarias_designacao.data_designacao é `date`, não
  // `timestamptz`; ver o comentário em DesignarInput no backend).
  data_designacao: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "formato esperado: AAAA-MM-DD").optional(),
});

export const criarEmpenhoSchema = z.object({
  numero_empenho: z.string().trim().min(1, "obrigatório").max(50),
  data_emissao: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "formato esperado: AAAA-MM-DD"),
  // Centavos — o form converte de reais (string "R$ 1.000,00") pra
  // inteiro antes de chegar aqui, mesmo espírito de dataOpcionalSchema
  // combinando com o formato do <input> nativo.
  valor_inicial: z.number().int().positive("precisa ser maior que zero"),
});

export const registrarMovimentacaoSchema = z.object({
  tipo: z.enum(["REFORCO", "ANULACAO", "FATURA_APROPRIADA"]),
  valor: z.number().int().positive("precisa ser maior que zero"),
  processo_pagamento_id: z.string().uuid().optional(),
  observacao: z.string().trim().max(2000).optional(),
});

export const registrarOcorrenciaSchema = z.object({
  descricao: z.string().trim().min(1, "obrigatório").max(4000),
});

// Configurações: Modelos de Documentos. Upload/substituição de versão
// chegam como multipart/form-data (não JSON) nos Route Handlers — os
// campos de texto são validados individualmente (mesmo padrão de
// tipoDocumentoIdSchema/dataValidadeSchema acima), não como objeto único.
const GATILHOS_MODELO = ["NOTIFICACAO_DESCUMPRIMENTO", "MINUTA_ADITIVO", "ATESTO", "RELATORIO_PAGAMENTO"] as const;

export const categoriaModeloSchema = z.string().trim().min(1, "obrigatório").max(150);
export const gatilhoModeloSchema = z.enum(GATILHOS_MODELO).optional();

export const atualizarModeloDocumentoSchema = z.object({
  categoria: z.string().trim().min(1, "obrigatório").max(150).optional(),
  // "" ou "NENHUM" remove a associação atual — ver o comentário em
  // ModeloDocumentoHandler.Atualizar (backend).
  gatilho: z.enum([...GATILHOS_MODELO, "", "NENHUM"]).optional(),
});

// Configurações: Keycloak/SSO. client_secret vazio numa atualização
// mantém o segredo já salvo (ver o comentário em
// KeycloakConfigService.Salvar, backend) — só client_id/issuer_url são
// sempre obrigatórios.
export const atualizarKeycloakConfigSchema = z.object({
  client_id: z.string().trim().min(1, "obrigatório").max(255),
  client_secret: z.string().trim().max(500).optional(),
  issuer_url: z.string().trim().url("precisa ser uma URL válida (ex: https://keycloak.exemplo.gov.br/realms/selene)"),
  audience: z.string().trim().max(255).optional(),
});

// Configurações: Diário Oficial (estrutura genérica — ver o comentário
// de escopo em backend/internal/service/diario_oficial_service.go).
// api_key vazia numa atualização mantém a chave já salva.
export const atualizarDiarioOficialConfigSchema = z.object({
  base_url: z.string().trim().url("precisa ser uma URL válida (ex: https://diario.example.gov.br/api)"),
  api_key: z.string().trim().max(500).optional(),
});

// Busca de contratos no Diário Oficial — os 3 filtros pedidos pelo
// usuário, todos opcionais (a API externa decide o que fazer com uma
// busca vazia). Sem validação de formato rígida pra cpf/data: não
// sabemos o formato exato que a API real vai exigir ainda.
export const buscarContratosDiarioOficialSchema = z.object({
  nome: z.string().trim().max(255).optional(),
  cpf: z.string().trim().max(20).optional(),
  data: z.string().trim().max(20).optional(),
});
