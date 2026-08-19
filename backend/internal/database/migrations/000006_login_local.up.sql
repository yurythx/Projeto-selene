-- Login tradicional (usuário/senha), como alternativa ao Keycloak —
-- contas criadas só por administrador (sem autocadastro público).

-- keycloak_id deixa de ser obrigatório: um usuário local nunca teve
-- login via Keycloak, então não tem (nem nunca terá) um "sub" de token
-- OIDC. NULL é permitido em múltiplas linhas sem violar o índice único.
ALTER TABLE users ALTER COLUMN keycloak_id DROP NOT NULL;

ALTER TABLE users ADD COLUMN password_hash text;

-- true = senha temporária definida pelo admin na criação da conta;
-- o próprio usuário precisa trocá-la no primeiro login.
ALTER TABLE users ADD COLUMN must_change_password boolean NOT NULL DEFAULT false;
