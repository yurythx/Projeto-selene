DELETE FROM tipos_documento WHERE nome IN (
    'Comprovante de Pagamento de Salário',
    'Protocolo GFIP',
    'Guia GRF/GPS',
    'Relação de Trabalhadores (SEFIP)'
);

ALTER TABLE contratos DROP COLUMN IF EXISTS exige_fiscalizacao_terceirizacao;
