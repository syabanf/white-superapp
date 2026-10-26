-- name: UpsertReport :one
INSERT INTO content.reports (id, client_id, month, data, summary, ai_model, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
ON CONFLICT (client_id, month) DO UPDATE SET
    data = EXCLUDED.data,
    summary = EXCLUDED.summary,
    ai_model = EXCLUDED.ai_model,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING *;

-- name: GetReport :one
SELECT * FROM content.reports WHERE id = $1 AND client_id = $2;

-- name: ListReports :many
SELECT * FROM content.reports WHERE client_id = $1 ORDER BY month DESC;
