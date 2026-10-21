-- +goose Up
-- One-time copy from the Prisma-era tables in `public`, when they exist.
-- Rows keep their ids so connection_id references in other services stay valid.
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public."Connection"') IS NOT NULL THEN
        INSERT INTO integrations.connections (id, client_id, provider, account_id, display_name, access_token_enc,
                                              refresh_token_enc, expires_at, scopes, meta, created_at, updated_at)
        SELECT id, "clientId", provider::text::integrations.connection_provider, "accountId", "displayName", "accessTokenEnc",
               "refreshTokenEnc", "expiresAt", scopes::text[], meta, "createdAt", "updatedAt"
        FROM public."Connection"
        ON CONFLICT (id) DO NOTHING;
    END IF;

    IF to_regclass('public."IntegrationSetting"') IS NOT NULL THEN
        INSERT INTO integrations.integration_settings (provider, public_id, secret_enc, meta, verified_at, updated_by_id, created_at, updated_at)
        SELECT provider, "publicId", "secretEnc", meta, "verifiedAt", "updatedById", "createdAt", "updatedAt"
        FROM public."IntegrationSetting"
        ON CONFLICT (provider) DO NOTHING;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
