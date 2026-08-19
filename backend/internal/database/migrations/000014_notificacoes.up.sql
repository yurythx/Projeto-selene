-- Notificações in-app de prazos/vencimentos (Radar) — pedido explícito
-- do usuário: "precisamos ter os alertas/notificacoes a respeito de
-- prazos e vencimentos", com os dois canais confirmados (e-mail +
-- dentro do app). Esta tabela é o canal IN-APP; o e-mail é disparado no
-- mesmo momento em que uma linha nova é criada aqui, não persistido à
-- parte (ver NotificacaoService.GerarAlertas).
--
-- Uma linha por (usuario, alerta) — chave_alerta é uma string
-- determinística (tipo+contrato+processo+nível, ver
-- NotificacaoService.chaveAlerta) que identifica o MESMO alerta em
-- execuções sucessivas do gerador. Índice único evita duplicar a
-- notificação a cada vez que o scheduler roda enquanto o alerta
-- continuar valendo (ex: contrato ainda vencendo) — cada instância de
-- alerta (tipo+contrato+processo+nível) gera exatamente UMA notificação
-- na vida dela, mesmo que o usuário já tenha marcado como lida; se o
-- nível escalar (ex: ATENCAO → CRITICO), é uma chave_alerta diferente,
-- então gera uma notificação nova — escalar merece alertar de novo,
-- mesmo nível repetido não.
--
-- Não é uma FK direta pra "o alerta" (RadarService.ItemRadar não é
-- persistido, é computado on-the-fly) — só guardamos o suficiente pra
-- exibir e linkar (contrato_id sempre presente, processo_id só nos
-- tipos "certidao"/"processo_parado", NULL em "vigencia_contrato").
CREATE TABLE notificacoes (
    id            uuid PRIMARY KEY,
    usuario_id    uuid NOT NULL REFERENCES users (id),
    tipo          varchar(30) NOT NULL,
    nivel         varchar(20) NOT NULL,
    contrato_id   uuid NOT NULL REFERENCES contratos (id),
    processo_id   uuid REFERENCES processos_pagamento (id),
    mensagem      text NOT NULL,
    chave_alerta  text NOT NULL,
    lida          boolean NOT NULL DEFAULT false,
    criada_em     timestamptz NOT NULL DEFAULT now(),
    lida_em       timestamptz
);

CREATE UNIQUE INDEX idx_notificacoes_usuario_chave ON notificacoes (usuario_id, chave_alerta);

-- Consulta mais comum da UI: notificações de um usuário, não-lidas
-- primeiro (ver NotificacaoRepository.Listar/ContarNaoLidas).
CREATE INDEX idx_notificacoes_usuario_lida ON notificacoes (usuario_id, lida);
