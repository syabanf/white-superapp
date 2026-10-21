-- name: GetLatestResearch :one
SELECT * FROM seosuite.keyword_research
WHERE client_id = $1 AND seed = $2 AND location_code = $3 AND language_code = $4
ORDER BY fetched_at DESC LIMIT 1;

-- name: InsertResearch :exec
INSERT INTO seosuite.keyword_research (id, client_id, seed, location_code, language_code, provider, results, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: DeleteOtherResearch :exec
DELETE FROM seosuite.keyword_research
WHERE client_id = $1 AND seed = $2 AND location_code = $3 AND language_code = $4 AND id <> $5;

-- name: DeleteResearchBefore :execrows
DELETE FROM seosuite.keyword_research WHERE fetched_at < $1;
