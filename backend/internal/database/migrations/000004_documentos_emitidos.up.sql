-- Módulo 2 do roadmap: Gerador Inteligente de Documentos Legais.
-- Registra todo PDF oficial gerado pelo próprio Selene (Notificação de
-- Descumprimento, Atesto, Minuta de Aditivo) — distinto de
-- documentos_anexos, que são uploads do fiscal.

CREATE TABLE documentos_emitidos (
    id                     uuid PRIMARY KEY,
    contrato_id            uuid NOT NULL REFERENCES contratos (id),
    processo_pagamento_id  uuid REFERENCES processos_pagamento (id),
    tipo                   varchar(30) NOT NULL
                           CONSTRAINT chk_documentos_emitidos_tipo
                           CHECK (tipo IN ('NOTIFICACAO_DESCUMPRIMENTO', 'ATESTO', 'MINUTA_ADITIVO')),
    motivo                 text,
    dados_extras           text,
    gerado_por_id          uuid NOT NULL REFERENCES users (id),
    codigo_verificacao     varchar(40) NOT NULL,
    created_at             timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_documentos_emitidos_codigo_verificacao ON documentos_emitidos (codigo_verificacao);
CREATE INDEX idx_documentos_emitidos_contrato_id ON documentos_emitidos (contrato_id);
CREATE INDEX idx_documentos_emitidos_processo_pagamento_id ON documentos_emitidos (processo_pagamento_id);
CREATE INDEX idx_documentos_emitidos_gerado_por_id ON documentos_emitidos (gerado_por_id);
