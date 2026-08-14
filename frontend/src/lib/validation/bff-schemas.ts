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

export const novoContratoSchema = z.object({
  numero_contrato: z.string().trim().min(1, "obrigatório").max(50),
  portaria_nomeacao: z.string().trim().max(255).optional(),
  data_assinatura: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "formato esperado: AAAA-MM-DD"),
  contratada_nome: z.string().trim().min(1, "obrigatório").max(255),
  contratada_cnpj: z.string().trim().min(1, "obrigatório").max(20),
  contratada_email: z.string().trim().email("e-mail inválido").optional().or(z.literal("")),
  tipo_objeto: z.enum(["CONSUMO", "PERMANENTE", "SERVICO"]),
});

export const atualizarContratoSchema = z.object({
  portaria_nomeacao: z.string().trim().max(255).optional(),
  contratada_nome: z.string().trim().min(1, "obrigatório").max(255).optional(),
  contratada_cnpj: z.string().trim().min(1, "obrigatório").max(20).optional(),
  contratada_email: z.string().trim().email("e-mail inválido").optional().or(z.literal("")),
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
