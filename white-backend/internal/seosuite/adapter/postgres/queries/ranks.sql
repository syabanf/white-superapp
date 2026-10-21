-- name: ListRankSnapshots :many
SELECT * FROM seosuite.rank_snapshots
WHERE keyword_id = ANY(sqlc.arg(ids)::text[]) AND date >= sqlc.arg(from_date) AND date <= sqlc.arg(to_date)
ORDER BY date ASC;

-- name: ListRankPositionsOn :many
SELECT keyword_id, position FROM seosuite.rank_snapshots WHERE keyword_id = ANY(sqlc.arg(ids)::text[]) AND date = sqlc.arg(date);

-- name: UpsertRankSnapshot :batchexec
INSERT INTO seosuite.rank_snapshots (id, client_id, keyword_id, date, position, url, serp_features, top_results)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (keyword_id, date) DO UPDATE
SET position = EXCLUDED.position, url = EXCLUDED.url, serp_features = EXCLUDED.serp_features, top_results = EXCLUDED.top_results;

-- name: DeleteRankSnapshotsBefore :execrows
DELETE FROM seosuite.rank_snapshots WHERE date < $1;
