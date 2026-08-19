-- Configuração do Keycloak editável em runtime, via Configurações
-- (pedido do usuário: "já usamos hoje mas não temos no front, e se eu
-- quiser mudar ou implementar um novo, crie uma opção"). Linha única
-- (singleton, id fixo em 1) — não existe múltiplos realms Keycloak
-- configurados ao mesmo tempo neste app.
--
-- Antes desta tabela, Client ID/Secret/Issuer só existiam em variáveis de
-- ambiente (AUTH_KEYCLOAK_ID/SECRET/ISSUER no frontend,
-- KEYCLOAK_JWKS_URL/ISSUER/AUDIENCE no backend), lidas uma vez no boot do
-- processo — mudar exigia editar o ambiente e reiniciar os containers.
-- Com esta tabela, o admin edita pela UI e a mudança é aplicada em
-- runtime (ver middleware.AuthMiddlewareState.Reload no backend, e o
-- endpoint interno que o frontend consulta pra reconfigurar o provider
-- Keycloak do NextAuth sem reiniciar). As variáveis de ambiente
-- continuam servindo de valor inicial/fallback pro primeiro boot, antes
-- de qualquer admin salvar uma configuração customizada aqui.
CREATE TABLE keycloak_config (
    id             smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    client_id      varchar(255) NOT NULL,
    -- Client Secret NUNCA é devolvido pela API depois de salvo (só
    -- escrita) — ver KeycloakConfigService.Buscar, que devolve
    -- TemSegredoConfigurado em vez do valor real.
    client_secret  text NOT NULL,
    issuer_url     text NOT NULL,
    -- Audience é opcional — nem todo client do Keycloak popula "aud" da
    -- mesma forma (mesma observação já valia pra KEYCLOAK_AUDIENCE).
    audience       varchar(255),
    updated_by_id  uuid NOT NULL REFERENCES users (id),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
