-- name: GetIntegrationSetting :one
SELECT * FROM integrations.integration_settings WHERE provider = $1;

-- name: ListIntegrationSettings :many
SELECT * FROM integrations.integration_settings ORDER BY provider ASC;

-- name: UpsertIntegrationSetting :one
INSERT INTO integrations.integration_settings AS s
    (provider, public_id, secret_enc, meta, verified_at, updated_by_id)
VALUES (@provider, @public_id, @secret_enc, @meta, @verified_at, @updated_by_id)
ON CONFLICT (provider) DO UPDATE SET
    public_id     = CASE WHEN @set_public_id::boolean THEN EXCLUDED.public_id ELSE s.public_id END,
    secret_enc    = CASE WHEN @set_secret::boolean THEN EXCLUDED.secret_enc ELSE s.secret_enc END,
    meta          = CASE WHEN @set_meta::boolean THEN EXCLUDED.meta ELSE s.meta END,
    verified_at   = CASE WHEN @set_verified_at::boolean THEN EXCLUDED.verified_at ELSE s.verified_at END,
    updated_by_id = COALESCE(EXCLUDED.updated_by_id, s.updated_by_id),
    updated_at    = now()
RETURNING *;

-- name: DeleteIntegrationSetting :execrows
DELETE FROM integrations.integration_settings WHERE provider = $1;
