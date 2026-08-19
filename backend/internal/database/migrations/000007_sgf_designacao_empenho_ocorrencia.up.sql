-- SGF-Rondonópolis, Fase 1: adequação estrita à IN SCL Nº 01/2019 (Versão
-- II) e à IN SCL Nº 04/2021 — ver o plano em
-- .claude/plans/projeto-selene-rippling-kite.md para a Matriz Normativa
-- completa (artigo por artigo) que fundamenta cada tabela abaixo.
--
-- Todas as mudanças aqui são aditivas: nenhuma tabela, coluna ou
-- comportamento existente é removida/renomeada. O único ALTER TABLE numa
-- tabela já existente adiciona uma coluna nullable.

-- 1) PortariaDesignacao — histórico auditável de designação de
-- fiscal/suplente/gestor/fiscal setorial por contrato (IN01 Art.4º-I,
-- Art.6º; IN04 Art.4º-I, Art.10). Substitui, como fonte de verdade,
-- contratos.fiscal_id — que permanece intocado como cache do último
-- registro papel=FISCAL ativo (ver comentário em portaria_designacao.go).
CREATE TABLE portarias_designacao (
    id                    uuid PRIMARY KEY,
    contrato_id           uuid NOT NULL REFERENCES contratos (id),
    servidor_id           uuid NOT NULL REFERENCES users (id),
    papel                 varchar(20) NOT NULL
                          CONSTRAINT chk_portarias_designacao_papel
                          CHECK (papel IN ('FISCAL', 'FISCAL_SUPLENTE', 'GESTOR', 'FISCAL_SETORIAL')),
    numero_portaria       varchar(50),
    publicado_diorondon   varchar(50),
    data_designacao       date NOT NULL,
    data_revogacao        date,
    criado_por_id         uuid NOT NULL REFERENCES users (id),
    created_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_portarias_designacao_contrato_id ON portarias_designacao (contrato_id);
CREATE INDEX idx_portarias_designacao_servidor_id ON portarias_designacao (servidor_id);

-- 2) Empenho / MovimentacaoEmpenho — registro PARALELO/informativo do
-- controle de saldo que a norma exige do FISCAL (IN01 Art.5º-VIII; IN04
-- Art.5º-XXII: "controlar o saldo do empenho em função do valor da
-- fatura"). NÃO é fonte de verdade orçamentária — essa continua sendo
-- exclusiva dos sistemas corporativos da prefeitura, como já documentado
-- em contrato.go. Ver o comentário em empenho.go para o motivo dessa
-- distinção.
CREATE TABLE empenhos (
    id               uuid PRIMARY KEY,
    contrato_id      uuid NOT NULL REFERENCES contratos (id),
    numero_empenho   varchar(50) NOT NULL,
    data_emissao     date NOT NULL,
    valor_inicial    bigint NOT NULL, -- centavos
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_empenhos_contrato_id ON empenhos (contrato_id);

CREATE TABLE movimentacoes_empenho (
    id                     uuid PRIMARY KEY,
    empenho_id             uuid NOT NULL REFERENCES empenhos (id),
    tipo                   varchar(20) NOT NULL
                           CONSTRAINT chk_movimentacoes_empenho_tipo
                           CHECK (tipo IN ('INICIAL', 'REFORCO', 'ANULACAO', 'FATURA_APROPRIADA')),
    valor                  bigint NOT NULL, -- centavos, sempre positivo; sinal implícito pelo tipo
    processo_pagamento_id  uuid REFERENCES processos_pagamento (id), -- só em FATURA_APROPRIADA
    observacao             text,
    registrado_por_id      uuid NOT NULL REFERENCES users (id),
    created_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_movimentacoes_empenho_empenho_id ON movimentacoes_empenho (empenho_id);
CREATE INDEX idx_movimentacoes_empenho_processo_pagamento_id ON movimentacoes_empenho (processo_pagamento_id);

-- Ligação opcional e nullable: processos existentes continuam com
-- empenho_id NULL, nenhum comportamento muda para quem não usa o campo.
ALTER TABLE processos_pagamento ADD COLUMN empenho_id uuid REFERENCES empenhos (id);

-- 3) Ocorrencia — registro de ocorrência da execução contratual (IN01
-- Art.3º-III, Art.5º-IV/IX; IN04 Art.3º-VIII, Art.5º-VIII/XVI). Ligada ao
-- contrato (sempre) e, opcionalmente, a um processo de pagamento
-- específico — a norma fala de ocorrências "relacionadas com a execução
-- do contrato", não amarradas a um ciclo mensal.
CREATE TABLE ocorrencias (
    id                       uuid PRIMARY KEY,
    contrato_id              uuid NOT NULL REFERENCES contratos (id),
    processo_pagamento_id    uuid REFERENCES processos_pagamento (id),
    descricao                text NOT NULL,
    estado                   varchar(20) NOT NULL DEFAULT 'REGISTRADA'
                             CONSTRAINT chk_ocorrencias_estado
                             CHECK (estado IN ('REGISTRADA', 'NOTIFICADA', 'EM_TRATAMENTO', 'REGULARIZADA')),
    registrado_por_id        uuid NOT NULL REFERENCES users (id),
    data_notificacao_gestor  date,
    data_regularizacao       date,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_ocorrencias_contrato_id ON ocorrencias (contrato_id);
CREATE INDEX idx_ocorrencias_processo_pagamento_id ON ocorrencias (processo_pagamento_id);
CREATE INDEX idx_ocorrencias_estado ON ocorrencias (estado);

-- 4) Backfill: gera um registro histórico de designação (papel=FISCAL)
-- para cada contrato já cadastrado que tem fiscal designado — melhor
-- aproximação disponível hoje, já que não existe data de publicação
-- retroativa capturada (usa data_assinatura). Confirmado com o usuário
-- antes desta migration (ver plano).
INSERT INTO portarias_designacao
    (id, contrato_id, servidor_id, papel, numero_portaria, data_designacao, criado_por_id, created_at)
SELECT gen_random_uuid(), id, fiscal_id, 'FISCAL', portaria_nomeacao, data_assinatura, fiscal_id, now()
FROM contratos
WHERE fiscal_id IS NOT NULL;
