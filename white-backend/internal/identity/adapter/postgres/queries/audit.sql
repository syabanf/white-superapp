-- name: InsertAuditLog :exec
INSERT INTO identity.audit_logs (id, at, user_id, user_name, role, method, path, status, ip, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO NOTHING;

-- name: ListAuditLogs :many
SELECT *, count(*) OVER ()::int AS total
FROM identity.audit_logs
WHERE (sqlc.narg('q')::text IS NULL OR path ILIKE '%' || sqlc.narg('q') || '%' OR user_name ILIKE '%' || sqlc.narg('q') || '%')
  AND (sqlc.narg('failed')::boolean IS NULL OR (sqlc.narg('failed')::boolean AND status >= 400))
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR at < sqlc.narg('to_at'))
ORDER BY at DESC, id
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');
