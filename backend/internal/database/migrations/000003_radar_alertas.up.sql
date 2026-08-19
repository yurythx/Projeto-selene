-- Fase 1 do roadmap ("Radar de Alertas e Prazos Legais"): campos novos
-- pra calcular avisos de vigência de contrato e validade de certidão.
-- Ambos nullable de propósito — contratos/documentos já cadastrados não
-- têm esse dado, e não faz sentido travar o cadastro exigindo
-- retroativamente algo que não existia antes.

ALTER TABLE contratos ADD COLUMN data_vigencia_fim date;

ALTER TABLE tipos_documento ADD COLUMN exige_validade boolean NOT NULL DEFAULT false;

-- Certidões (CND/FGTS) têm validade própria, diferente da data de
-- upload — os nomes aqui precisam bater exatamente com
-- internal/database/seed.go (mesmo princípio do checklist.go, que já
-- resolve TipoDocumento por nome, não por ID).
UPDATE tipos_documento SET exige_validade = true
WHERE nome IN (
    'CND Trabalhista',
    'CND FGTS',
    'CND Municipal',
    'CND Estadual',
    'CND Federal',
    'CND INSS'
);

ALTER TABLE documentos_anexos ADD COLUMN data_validade date;
