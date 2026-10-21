-- name: GetLatestJob :one
SELECT * FROM seosuite.sync_jobs WHERE client_id = $1 AND kind = $2 ORDER BY started_at DESC LIMIT 1;

-- name: InsertJob :exec
INSERT INTO seosuite.sync_jobs (id, client_id, kind, status, started_at, message) VALUES ($1, $2, $3, $4, $5, $6);

-- name: FinishJob :exec
UPDATE seosuite.sync_jobs SET status = $2, finished_at = $3, message = $4, error = $5 WHERE id = $1;

-- name: ListJobs :many
SELECT * FROM seosuite.sync_jobs WHERE client_id = $1 AND kind = ANY(sqlc.arg(kinds)::text[]) ORDER BY started_at DESC LIMIT sqlc.arg(row_limit);

-- name: DeleteJobsBefore :execrows
DELETE FROM seosuite.sync_jobs WHERE started_at < $1;

-- name: PurgeKeywords :exec
DELETE FROM seosuite.tracked_keywords WHERE client_id = $1;

-- name: PurgeRankSnapshots :exec
DELETE FROM seosuite.rank_snapshots WHERE client_id = $1;

-- name: PurgeResearch :exec
DELETE FROM seosuite.keyword_research WHERE client_id = $1;

-- name: PurgeBacklinks :exec
DELETE FROM seosuite.backlinks WHERE client_id = $1;

-- name: PurgeBacklinkSnapshots :exec
DELETE FROM seosuite.backlink_snapshots WHERE client_id = $1;

-- name: PurgeCompetitorDomains :exec
DELETE FROM seosuite.competitor_domains WHERE client_id = $1;

-- name: PurgeDomainSnapshots :exec
DELETE FROM seosuite.domain_snapshots WHERE client_id = $1;

-- name: PurgeJobs :exec
DELETE FROM seosuite.sync_jobs WHERE client_id = $1;
