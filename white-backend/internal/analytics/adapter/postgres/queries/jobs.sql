-- name: InsertSyncJob :one
INSERT INTO analytics.sync_jobs (id, client_id, kind, status, started_at, finished_at, message, error)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateSyncJob :exec
UPDATE analytics.sync_jobs SET status = $2, finished_at = $3, message = $4, error = $5 WHERE id = $1;

-- name: LatestSyncJob :one
SELECT * FROM analytics.sync_jobs
WHERE client_id = $1 AND kind = ANY($2::text[])
ORDER BY started_at DESC
LIMIT 1;

-- name: ListSyncJobs :many
SELECT * FROM analytics.sync_jobs WHERE client_id = $1 ORDER BY started_at DESC LIMIT $2;

-- name: LatestSyncByClients :many
SELECT client_id, MAX(started_at)::timestamptz AS started_at
FROM analytics.sync_jobs
WHERE client_id = ANY($1::text[])
GROUP BY client_id;

-- name: DeleteSyncJobsBefore :execrows
DELETE FROM analytics.sync_jobs WHERE started_at < $1;

-- name: DeleteSyncJobsByClient :exec
DELETE FROM analytics.sync_jobs WHERE client_id = $1;
