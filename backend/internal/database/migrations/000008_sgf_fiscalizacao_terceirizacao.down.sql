-- Limitação conhecida: se algum DocumentoAnexo já tiver sido anexado
-- referenciando um destes 4 tipos (uso normal do checklist condicional
-- da Fase 6), este DELETE falha por violação de FK
-- (documentos_anexos.tipo_documento_id não tem ON DELETE CASCADE, ver
-- 000001_initial_schema.up.sql). Rollback contra uma base já em uso
-- exigiria decidir manualmente o que fazer com esses documentos
-- (mover pra outro tipo? apagar?) — fora do escopo de um DOWN
-- automático. Cenário de baixa probabilidade (rollback raramente roda
-- contra base populada), documentado aqui em vez de tratado.
DELETE FROM tipos_documento WHERE nome IN (
    'Comprovante de Pagamento de Salário',
    'Protocolo GFIP',
    'Guia GRF/GPS',
    'Relação de Trabalhadores (SEFIP)'
);

ALTER TABLE contratos DROP COLUMN IF EXISTS exige_fiscalizacao_terceirizacao;
