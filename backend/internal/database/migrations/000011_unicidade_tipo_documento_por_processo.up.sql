-- No máximo um documento anexo de cada TipoDocumento por processo de
-- pagamento — pedido explícito do usuário: "não pode ter mais de um do
-- mesmo tipo (exemplo, dois Pré-Empenho)". Reforça em banco a mesma
-- regra que DocumentoService.Upload já valida em código (defesa em
-- profundidade — mesmo padrão de idx_processo_hash, que já existia pra
-- dedup por conteúdo idêntico; este índice é por TIPO, não por conteúdo).

-- Backfill: bases já em uso (inclusive a de testes deste projeto) podem
-- ter duplicatas do mesmo tipo acumuladas antes desta regra existir — o
-- CREATE UNIQUE INDEX abaixo falharia direto nelas. Mantém só o anexo
-- mais recente (maior data_upload) de cada (processo, tipo) e remove os
-- demais — mesma decisão que passa a valer daqui pra frente (o upload
-- novo de um tipo já existente é rejeitado, então "o mais recente já
-- enviado" é a melhor aproximação do que o fiscal realmente queria manter).
DELETE FROM documentos_anexos d
    USING documentos_anexos mais_novo
    WHERE d.processo_pagamento_id = mais_novo.processo_pagamento_id
      AND d.tipo_documento_id = mais_novo.tipo_documento_id
      AND d.data_upload < mais_novo.data_upload;

-- Segundo critério de desempate: se dois registros do mesmo (processo,
-- tipo) tiverem a mesma data_upload (improvável, mas possível), mantém o
-- de ID maior (UUID mais recente na prática, já que é gerado em
-- sequência de tempo pela aplicação) e remove os outros.
DELETE FROM documentos_anexos d
    USING documentos_anexos mais_novo
    WHERE d.processo_pagamento_id = mais_novo.processo_pagamento_id
      AND d.tipo_documento_id = mais_novo.tipo_documento_id
      AND d.data_upload = mais_novo.data_upload
      AND d.id < mais_novo.id;

CREATE UNIQUE INDEX idx_documentos_anexos_processo_tipo ON documentos_anexos (processo_pagamento_id, tipo_documento_id);
