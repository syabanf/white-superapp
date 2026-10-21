-- +goose Up
CREATE SCHEMA IF NOT EXISTS integrations;

CREATE TYPE integrations.connection_provider AS ENUM ('META', 'GOOGLE');

-- OAuth connections. Tokens are AES-GCM ciphertext (pkg/crypto); client_id
-- is plain text because identity owns the clients table.
CREATE TABLE integrations.connections (
    id                text PRIMARY KEY,
    client_id         text NOT NULL,
    provider          integrations.connection_provider NOT NULL,
    account_id        text NOT NULL,
    display_name      text NOT NULL,
    access_token_enc  text NOT NULL,
    refresh_token_enc text,
    expires_at        timestamptz,
    scopes            text[] NOT NULL DEFAULT '{}',
    meta              jsonb,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (client_id, provider, account_id)
);
CREATE INDEX connections_client_idx ON integrations.connections (client_id);

-- Workspace-level provider credentials saved by the setup wizard.
CREATE TABLE integrations.integration_settings (
    provider      text PRIMARY KEY,
    public_id     text,
    secret_enc    text,
    meta          jsonb,
    verified_at   timestamptz,
    updated_by_id text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP SCHEMA integrations CASCADE;
