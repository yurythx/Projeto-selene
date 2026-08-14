-- Permite "encerrar" um contrato sem apagar o registro (soft-close),
-- mantendo o histórico de processos/documentos consultável.
ALTER TABLE contratos ADD COLUMN ativo boolean NOT NULL DEFAULT true;
