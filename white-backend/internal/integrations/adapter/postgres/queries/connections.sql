-- name: GetConnection :one
SELECT * FROM integrations.connections WHERE id = $1;

-- name: ListConnections :many
SELECT * FROM integrations.connections WHERE client_id = $1 ORDER BY created_at ASC;

-- name: FindConnection :one
SELECT * FROM integrations.connections
WHERE client_id = $1 AND provider = $2
ORDER BY created_at ASC
LIMIT 1;

-- name: UpsertConnection :one
INSERT INTO integrations.connections AS c
    (id, client_id, provider, account_id, display_name, access_token_enc, refresh_token_enc, expires_at, scopes)
VALUES (@id, @client_id, @provider, @account_id, @display_name, @access_token_enc, @refresh_token_enc, @expires_at, @scopes)
ON CONFLICT (client_id, provider, account_id) DO UPDATE SET
    display_name      = EXCLUDED.display_name,
    access_token_enc  = EXCLUDED.access_token_enc,
    refresh_token_enc = CASE
        WHEN EXCLUDED.refresh_token_enc IS NULL AND @keep_refresh_token::boolean THEN c.refresh_token_enc
        ELSE EXCLUDED.refresh_token_enc
    END,
    expires_at        = EXCLUDED.expires_at,
    scopes            = EXCLUDED.scopes,
    updated_at        = now()
RETURNING *;

-- name: UpdateAccessToken :exec
UPDATE integrations.connections
SET access_token_enc = $2, expires_at = $3, updated_at = now()
WHERE id = $1;

-- name: DeleteConnection :execrows
DELETE FROM integrations.connections WHERE id = $1;

-- name: DeleteClientConnections :execrows
DELETE FROM integrations.connections WHERE client_id = $1;
