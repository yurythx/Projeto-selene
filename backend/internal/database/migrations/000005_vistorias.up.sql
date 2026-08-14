-- Módulo 3 do roadmap: Vistorias e Registro Fotográfico com
-- Geolocalização.

CREATE TABLE registros_vistoria (
    id                     uuid PRIMARY KEY,
    processo_pagamento_id  uuid NOT NULL REFERENCES processos_pagamento (id),
    fiscal_id              uuid NOT NULL REFERENCES users (id),
    data_hora              timestamptz NOT NULL DEFAULT now(),
    latitude               double precision,
    longitude              double precision,
    observacoes            text,
    created_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_registros_vistoria_processo_pagamento_id ON registros_vistoria (processo_pagamento_id);
CREATE INDEX idx_registros_vistoria_fiscal_id ON registros_vistoria (fiscal_id);

CREATE TABLE fotos_vistoria (
    id                uuid PRIMARY KEY,
    vistoria_id       uuid NOT NULL REFERENCES registros_vistoria (id),
    nome_arquivo      varchar(255) NOT NULL,
    caminho_storage   text NOT NULL,
    hash_arquivo      char(64) NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_vistoria_hash ON fotos_vistoria (vistoria_id, hash_arquivo);
