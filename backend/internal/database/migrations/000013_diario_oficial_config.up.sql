-- Configuração da integração com a API do Diário Oficial da cidade,
-- editável em runtime (Configurações → Diário Oficial), mesmo padrão de
-- keycloak_config (migration 000012): linha única (singleton, id fixo em
-- 1) — só existe UMA integração de Diário Oficial configurada por vez.
--
-- Pedido do usuário: uma tela pra cadastrar/testar a conexão com a API, e
-- outra pra buscar novos contratos publicados (por nome, CPF e data).
-- Decisão de escopo confirmada com o usuário: a API REAL do Diário
-- Oficial da cidade ainda não está definida — esta migration/integração
-- é a estrutura GENÉRICA (URL base + chave de API + um contrato de
-- request/response razoável, documentado em
-- internal/service/diario_oficial_service.go), pronta pra apontar pra
-- API real assim que a documentação dela existir.
CREATE TABLE diario_oficial_config (
    id             smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    base_url       text NOT NULL,
    -- Chave de API NUNCA é devolvida pela API depois de salva (só
    -- escrita) — mesmo padrão do client_secret em keycloak_config.
    api_key        text NOT NULL,
    updated_by_id  uuid NOT NULL REFERENCES users (id),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
