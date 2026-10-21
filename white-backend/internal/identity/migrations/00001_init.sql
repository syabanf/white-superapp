-- +goose Up
CREATE SCHEMA IF NOT EXISTS identity;

CREATE TYPE identity.user_role AS ENUM ('ADMIN', 'MEMBER');
CREATE TYPE identity.client_member_role AS ENUM ('MANAGER', 'VIEWER');

CREATE TABLE identity.users (
    id            text PRIMARY KEY,
    email         text NOT NULL UNIQUE,
    name          text NOT NULL,
    password_hash text NOT NULL,
    role          identity.user_role NOT NULL DEFAULT 'MEMBER',
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE identity.clients (
    id             text PRIMARY KEY,
    name           text NOT NULL,
    slug           text NOT NULL UNIQUE,
    description    text NOT NULL DEFAULT '',
    logo_url       text,
    website_url    text,
    industry       text,
    currency       text NOT NULL DEFAULT 'IDR',
    timezone       text NOT NULL DEFAULT 'Asia/Jakarta',
    share_enabled  boolean NOT NULL DEFAULT false,
    share_pin_hash text,
    share_modules  text[] NOT NULL DEFAULT '{SOCIAL,SEO,ADS}',
    created_by_id  text REFERENCES identity.users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX clients_created_by_idx ON identity.clients (created_by_id);

CREATE TABLE identity.client_members (
    user_id    text NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    client_id  text NOT NULL REFERENCES identity.clients(id) ON DELETE CASCADE,
    role       identity.client_member_role NOT NULL DEFAULT 'VIEWER',
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, client_id)
);
CREATE INDEX client_members_client_idx ON identity.client_members (client_id);

CREATE TABLE identity.notifications (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    client_id  text REFERENCES identity.clients(id) ON DELETE CASCADE,
    type       text NOT NULL,
    title      text NOT NULL,
    body       text NOT NULL DEFAULT '',
    href       text,
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_idx ON identity.notifications (user_id, read_at, created_at);

CREATE TABLE identity.app_settings (
    key        text PRIMARY KEY,
    value      jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Opaque refresh tokens. Only the SHA-256 of the token is stored.
CREATE TABLE identity.refresh_sessions (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    user_agent text NOT NULL DEFAULT '',
    ip         text NOT NULL DEFAULT ''
);
CREATE INDEX refresh_sessions_user_idx ON identity.refresh_sessions (user_id, expires_at);

-- +goose Down
DROP SCHEMA identity CASCADE;
