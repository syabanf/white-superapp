-- name: ListNotifications :many
SELECT * FROM identity.notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: CountUnreadNotifications :one
SELECT count(*)::int FROM identity.notifications WHERE user_id = $1 AND read_at IS NULL;

-- name: MarkNotificationRead :execrows
UPDATE identity.notifications SET read_at = now() WHERE id = $1 AND user_id = $2 AND read_at IS NULL;

-- name: MarkAllNotificationsRead :execrows
UPDATE identity.notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL;

-- name: InsertNotification :copyfrom
INSERT INTO identity.notifications (id, user_id, client_id, type, title, body, href) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: DeleteReadNotificationsBefore :execrows
DELETE FROM identity.notifications WHERE read_at < $1;

-- name: DeleteNotificationsBefore :execrows
DELETE FROM identity.notifications WHERE created_at < $1;

-- name: GetAppSetting :one
SELECT value FROM identity.app_settings WHERE key = $1;

-- name: UpsertAppSetting :exec
INSERT INTO identity.app_settings (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();

-- name: InsertRefreshSession :exec
INSERT INTO identity.refresh_sessions (id, user_id, token_hash, expires_at, user_agent, ip) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetRefreshSession :one
SELECT * FROM identity.refresh_sessions WHERE token_hash = $1;

-- name: RevokeRefreshSession :execrows
UPDATE identity.refresh_sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: RevokeUserSessions :execrows
UPDATE identity.refresh_sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredRefreshSessions :execrows
DELETE FROM identity.refresh_sessions WHERE expires_at < sqlc.arg(before) OR revoked_at < sqlc.arg(before);
