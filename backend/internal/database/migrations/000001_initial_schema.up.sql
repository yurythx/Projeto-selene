-- Schema inicial do Projeto Selene. Ordem respeita as dependências de
-- chave estrangeira: uma tabela referenciada precisa existir antes da
-- tabela que a referencia. Corresponde 1:1 aos structs em internal/models.

CREATE TABLE users (
    id             uuid PRIMARY KEY,
    keycloak_id    varchar(255) NOT NULL,
    nome           varchar(255) NOT NULL,
    email          varchar(255) NOT NULL,
    is_fiscal      boolean NOT NULL DEFAULT false,
    is_admin       boolean NOT NULL DEFAULT false,
    matricula      varchar(50),
    criado_em      timestamptz NOT NULL DEFAULT now(),
    atualizado_em  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_users_keycloak_id ON users (keycloak_id);
CREATE UNIQUE INDEX idx_users_email ON users (email);

CREATE TABLE kanban_etapas (
    id      bigint PRIMARY KEY,
    nome    varchar(100) NOT NULL,
    posicao bigint NOT NULL
);
CREATE UNIQUE INDEX idx_kanban_etapas_posicao ON kanban_etapas (posicao);

CREATE TABLE tipos_documento (
    id   bigserial PRIMARY KEY,
    nome varchar(100) NOT NULL
);
CREATE UNIQUE INDEX idx_tipos_documento_nome ON tipos_documento (nome);

CREATE TABLE contratos (
    id                 uuid PRIMARY KEY,
    numero_contrato    varchar(50) NOT NULL,
    portaria_nomeacao  varchar(255),
    data_assinatura    date NOT NULL,
    contratada_nome    varchar(255) NOT NULL,
    contratada_cnpj    varchar(18) NOT NULL,
    contratada_email   varchar(255),
    fiscal_id          uuid NOT NULL REFERENCES users (id),
    tipo_objeto        varchar(20) NOT NULL
                       CONSTRAINT chk_contratos_tipo_objeto
                       CHECK (tipo_objeto IN ('CONSUMO', 'PERMANENTE', 'SERVICO')),
    created_at         timestamptz,
    updated_at         timestamptz
);
CREATE UNIQUE INDEX idx_contratos_numero_contrato ON contratos (numero_contrato);
CREATE INDEX idx_contratos_contratada_cnpj ON contratos (contratada_cnpj);
CREATE INDEX idx_contratos_fiscal_id ON contratos (fiscal_id);

CREATE TABLE processos_pagamento (
    id              uuid PRIMARY KEY,
    contrato_id     uuid NOT NULL REFERENCES contratos (id),
    mes_referencia  varchar(7) NOT NULL,
    etapa_atual_id  bigint NOT NULL REFERENCES kanban_etapas (id),
    status          varchar(20) NOT NULL DEFAULT 'Ativo'
                    CONSTRAINT chk_processos_pagamento_status
                    CHECK (status IN ('Ativo', 'Concluido')),
    created_at      timestamptz,
    updated_at      timestamptz
);
CREATE UNIQUE INDEX idx_contrato_mes ON processos_pagamento (contrato_id, mes_referencia);
CREATE INDEX idx_processos_pagamento_etapa_atual_id ON processos_pagamento (etapa_atual_id);

CREATE TABLE documentos_anexos (
    id                     uuid PRIMARY KEY,
    processo_pagamento_id  uuid NOT NULL REFERENCES processos_pagamento (id),
    tipo_documento_id      bigint NOT NULL REFERENCES tipos_documento (id),
    nome_arquivo           varchar(255) NOT NULL,
    caminho_storage        text NOT NULL,
    hash_arquivo           char(64) NOT NULL,
    enviado_por_id         uuid NOT NULL REFERENCES users (id),
    data_upload            timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_processo_hash ON documentos_anexos (processo_pagamento_id, hash_arquivo);
CREATE INDEX idx_documentos_anexos_tipo_documento_id ON documentos_anexos (tipo_documento_id);
CREATE INDEX idx_documentos_anexos_enviado_por_id ON documentos_anexos (enviado_por_id);

CREATE TABLE kanban_logs (
    id                     uuid PRIMARY KEY,
    processo_pagamento_id  uuid NOT NULL REFERENCES processos_pagamento (id),
    usuario_id             uuid NOT NULL REFERENCES users (id),
    etapa_origem_id        bigint REFERENCES kanban_etapas (id),
    etapa_destino_id       bigint NOT NULL REFERENCES kanban_etapas (id),
    movido_em              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_kanban_logs_processo_pagamento_id ON kanban_logs (processo_pagamento_id);
CREATE INDEX idx_kanban_logs_usuario_id ON kanban_logs (usuario_id);
CREATE INDEX idx_kanban_logs_etapa_origem_id ON kanban_logs (etapa_origem_id);
CREATE INDEX idx_kanban_logs_etapa_destino_id ON kanban_logs (etapa_destino_id);
