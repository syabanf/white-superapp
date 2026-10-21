-- name: ListKeywords :many
SELECT * FROM seosuite.tracked_keywords WHERE property_id = $1 ORDER BY created_at ASC, id ASC;

-- name: ListKeywordsByIDs :many
SELECT * FROM seosuite.tracked_keywords WHERE property_id = $1 AND id = ANY(sqlc.arg(ids)::text[]) ORDER BY created_at ASC, id ASC;

-- name: ListKeywordIDs :many
SELECT id FROM seosuite.tracked_keywords WHERE property_id = $1;

-- name: GetKeyword :one
SELECT * FROM seosuite.tracked_keywords WHERE id = $1;

-- name: InsertKeyword :batchexec
INSERT INTO seosuite.tracked_keywords (id, client_id, property_id, keyword, location_code, language_code, device, tags, volume, difficulty, cpc, intent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (property_id, keyword, location_code, device) DO NOTHING;

-- name: UpdateKeywordTags :exec
UPDATE seosuite.tracked_keywords SET tags = $2, updated_at = now() WHERE id = $1;

-- name: DeleteKeyword :execrows
DELETE FROM seosuite.tracked_keywords WHERE id = $1;
