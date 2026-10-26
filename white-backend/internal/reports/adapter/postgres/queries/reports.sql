-- name: ListReports :many
SELECT * FROM reports.reports WHERE client_id = $1 ORDER BY created_at DESC;

-- name: GetReport :one
SELECT * FROM reports.reports WHERE id = $1;

-- name: InsertReport :one
INSERT INTO reports.reports (id, client_id, title, date_from, date_to, compare, modules, language, ai_summary, created_by_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: DeleteReport :execrows
DELETE FROM reports.reports WHERE id = $1 AND client_id = $2;

-- name: DeleteClientReports :execrows
DELETE FROM reports.reports WHERE client_id = $1;

-- name: ClearReportCreator :execrows
UPDATE reports.reports SET created_by_id = NULL WHERE created_by_id = $1;

-- name: InsertInsight :one
INSERT INTO reports.ai_insights (id, client_id, module, date_from, date_to, language, model, content, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: LatestInsight :one
SELECT * FROM reports.ai_insights
WHERE client_id = $1 AND module = $2 AND date_from = $3 AND date_to = $4
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteClientInsights :execrows
DELETE FROM reports.ai_insights WHERE client_id = $1;
