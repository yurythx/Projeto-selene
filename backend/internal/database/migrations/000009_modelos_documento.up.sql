-- Configurações: Modelos de Documentos — arquivo .docx por categoria
-- (texto livre, digitado pelo admin), com histórico de versões auditável
-- (mesmo espírito de portarias_designacao/movimentacoes_empenho: nunca
-- overwrite silencioso). Quando a categoria tem um gatilho associado a
-- um dos 4 fluxos de geração já existentes (Ofício, Minuta de Aditivo,
-- Atesto, Relatório de Pagamento), esse fluxo passa a preencher o .docx
-- de verdade (merge fields) em vez do PDF fixo em fpdf — ver
-- internal/service/modelo_documento_render.go. Sem gatilho, a categoria
-- é só biblioteca de referência (upload/consulta/substituição).
CREATE TABLE modelos_documento (
    id                uuid PRIMARY KEY,
    categoria         varchar(150) NOT NULL,
    -- NULL = categoria livre sem geração automática associada (só
    -- biblioteca de referência). Os 4 valores não-nulos correspondem
    -- 1:1 aos fluxos de geração já existentes no sistema.
    gatilho           varchar(40)
                      CONSTRAINT chk_modelos_documento_gatilho
                      CHECK (gatilho IS NULL OR gatilho IN (
                          'NOTIFICACAO_DESCUMPRIMENTO', 'MINUTA_ADITIVO',
                          'ATESTO', 'RELATORIO_PAGAMENTO'
                      )),
    -- FK pra modelo_documento_versoes adicionada depois de criar essa
    -- tabela (referência circular: uma tabela aponta pra outra e
    -- vice-versa).
    versao_ativa_id   uuid,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
-- Categoria é texto livre, mas não pode duplicar ignorando
-- maiúscula/minúscula — "Ofício" e "ofício" são a mesma categoria.
CREATE UNIQUE INDEX idx_modelos_documento_categoria ON modelos_documento (lower(categoria));
-- No máximo uma categoria "dona" de cada gatilho — sem isso, a geração
-- real não saberia qual modelo usar se duas categorias reivindicassem o
-- mesmo gatilho.
CREATE UNIQUE INDEX idx_modelos_documento_gatilho ON modelos_documento (gatilho) WHERE gatilho IS NOT NULL;

-- Histórico de versões do arquivo — mesmo padrão de storage de
-- documentos_anexos/fotos_vistoria (hash SHA-256, caminho nunca exposto
-- pela API), mas SEM dedup por hash: aqui cada upload é semanticamente
-- "publicar uma versão nova", mesmo com conteúdo idêntico ao anterior.
CREATE TABLE modelo_documento_versoes (
    id                    uuid PRIMARY KEY,
    modelo_documento_id   uuid NOT NULL REFERENCES modelos_documento (id),
    nome_arquivo          varchar(255) NOT NULL,
    caminho_storage       text NOT NULL,
    hash_arquivo          char(64) NOT NULL,
    tamanho_bytes         bigint NOT NULL,
    enviado_por_id        uuid NOT NULL REFERENCES users (id),
    created_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_modelo_documento_versoes_modelo_id ON modelo_documento_versoes (modelo_documento_id, created_at DESC);

ALTER TABLE modelos_documento
    ADD CONSTRAINT fk_modelos_documento_versao_ativa
    FOREIGN KEY (versao_ativa_id) REFERENCES modelo_documento_versoes (id);

-- Rastreia se um DocumentoEmitido (Notificação/Minuta/Atesto/Relatório)
-- saiu do fallback fpdf (PDF) ou de um modelo .docx preenchido — sem
-- isso não dá pra saber depois qual caminho foi usado pra gerar cada
-- documento já emitido.
ALTER TABLE documentos_emitidos ADD COLUMN formato varchar(10) NOT NULL DEFAULT 'PDF'
    CONSTRAINT chk_documentos_emitidos_formato CHECK (formato IN ('PDF', 'DOCX'));
