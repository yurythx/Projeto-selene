ALTER TABLE users DROP COLUMN must_change_password;
ALTER TABLE users DROP COLUMN password_hash;
ALTER TABLE users ALTER COLUMN keycloak_id SET NOT NULL;
