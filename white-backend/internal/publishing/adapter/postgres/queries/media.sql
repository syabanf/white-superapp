-- name: ListMedia :many
SELECT m.*, (SELECT count(*) FROM publishing.post_media pm WHERE pm.asset_id = m.id)::int AS usage_count
FROM publishing.media_assets m
WHERE m.client_id = $1
ORDER BY m.created_at DESC;

-- name: GetMedia :one
SELECT m.*, (SELECT count(*) FROM publishing.post_media pm WHERE pm.asset_id = m.id)::int AS usage_count
FROM publishing.media_assets m
WHERE m.id = $1;

-- name: ListMediaOwned :many
SELECT * FROM publishing.media_assets WHERE client_id = $1 AND id = ANY(sqlc.arg(ids)::text[]);

-- name: InsertMedia :one
INSERT INTO publishing.media_assets (id, client_id, kind, url, thumbnail_url, filename, mime_type, size_bytes, width, height, duration_sec, alt_text, tags, uploaded_by_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: UpdateMedia :one
UPDATE publishing.media_assets SET alt_text = $2, tags = $3 WHERE id = $1 RETURNING *;

-- name: UpdateMediaAlt :exec
UPDATE publishing.media_assets SET alt_text = $2 WHERE id = $1;

-- name: DeleteMedia :execrows
DELETE FROM publishing.media_assets WHERE id = $1;

-- name: ListMediaURLsByClient :many
SELECT url FROM publishing.media_assets WHERE client_id = $1;

-- name: DeleteMediaByClient :execrows
DELETE FROM publishing.media_assets WHERE client_id = $1;

-- name: NullMediaUploader :exec
UPDATE publishing.media_assets SET uploaded_by_id = NULL WHERE uploaded_by_id = $1;

-- name: ListPostMedia :many
SELECT m.*, (SELECT count(*) FROM publishing.post_media u WHERE u.asset_id = m.id)::int AS usage_count
FROM publishing.post_media pm
JOIN publishing.media_assets m ON m.id = pm.asset_id
WHERE pm.post_id = $1
ORDER BY pm.sort_order ASC;

-- name: ListPostMediaRows :many
SELECT post_id, asset_id, sort_order FROM publishing.post_media WHERE post_id = $1 ORDER BY sort_order ASC;

-- name: DeletePostMedia :exec
DELETE FROM publishing.post_media WHERE post_id = $1;

-- name: InsertPostMedia :copyfrom
INSERT INTO publishing.post_media (post_id, asset_id, sort_order) VALUES ($1, $2, $3);

-- name: FirstMediaForPosts :many
SELECT DISTINCT ON (pm.post_id) pm.post_id, m.url, m.thumbnail_url
FROM publishing.post_media pm
JOIN publishing.media_assets m ON m.id = pm.asset_id
WHERE pm.post_id = ANY(sqlc.arg(post_ids)::text[])
ORDER BY pm.post_id, pm.sort_order ASC;
