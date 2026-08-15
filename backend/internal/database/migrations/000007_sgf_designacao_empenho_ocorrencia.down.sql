DROP TABLE IF EXISTS ocorrencias;

ALTER TABLE processos_pagamento DROP COLUMN IF EXISTS empenho_id;

DROP TABLE IF EXISTS movimentacoes_empenho;
DROP TABLE IF EXISTS empenhos;

DROP TABLE IF EXISTS portarias_designacao;
