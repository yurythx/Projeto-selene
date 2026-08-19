-- Restringe cada TipoDocumento ao(s) tipo(s) de contrato a que ele
-- realmente se aplica — antes disso, o select de upload (e o endpoint de
-- referência que o alimenta) listava TODOS os tipos de documento pra
-- QUALQUER processo, inclusive os condicionais de SERVICO/terceirização
-- (Planilha de Medição, Boleto DAM, os 4 documentos trabalhistas do
-- Art.9º-XXXII), que não fazem sentido num contrato de CONSUMO/PERMANENTE
-- sem essas condições. Esses dois campos são a mesma fonte de verdade que
-- já existia como lista hardcoded em internal/service/checklist.go
-- (checklistCondicionalServico/checklistCondicionalTerceirizacao) — agora
-- também expressa em dados, pra alimentar tanto a validação de upload
-- quanto o filtro do select no frontend sem duplicar a regra em TS.
ALTER TABLE tipos_documento
    ADD COLUMN restrito_tipo_objeto varchar(20)
    CONSTRAINT chk_tipos_documento_restrito_tipo_objeto
    CHECK (restrito_tipo_objeto IS NULL OR restrito_tipo_objeto IN ('CONSUMO', 'PERMANENTE', 'SERVICO'));

ALTER TABLE tipos_documento
    ADD COLUMN restrito_terceirizacao boolean NOT NULL DEFAULT false;

-- Backfill dos tipos já semeados em bases existentes (Seed() só cria
-- registros novos via FirstOrCreate, nunca atualiza os já existentes —
-- mesmo padrão da migration 000003 para ExigeValidade).
UPDATE tipos_documento SET restrito_tipo_objeto = 'SERVICO'
    WHERE nome IN ('Planilha de Medição de Serviços', 'Boleto DAM');

UPDATE tipos_documento SET restrito_terceirizacao = true
    WHERE nome IN (
        'Comprovante de Pagamento de Salário',
        'Protocolo GFIP',
        'Guia GRF/GPS',
        'Relação de Trabalhadores (SEFIP)'
    );
