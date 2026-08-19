ALTER TABLE documentos_emitidos DROP COLUMN IF EXISTS formato;
ALTER TABLE modelos_documento DROP CONSTRAINT IF EXISTS fk_modelos_documento_versao_ativa;
DROP TABLE IF EXISTS modelo_documento_versoes;
DROP TABLE IF EXISTS modelos_documento;
