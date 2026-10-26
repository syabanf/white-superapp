-- name: InsertFolder :exec
INSERT INTO content.asset_folders (id, client_id, parent_id, name, created_by) VALUES ($1, $2, $3, $4, $5);

-- name: GetFolder :one
SELECT * FROM content.asset_folders WHERE id = $1 AND client_id = $2;

-- name: ListFolders :many
SELECT sqlc.embed(f), (SELECT count(*) FROM content.assets a WHERE a.folder_id = f.id AND a.deleted_at IS NULL)::int AS asset_count
FROM content.asset_folders f
WHERE f.client_id = $1
ORDER BY lower(f.name);

-- name: RenameFolder :exec
UPDATE content.asset_folders SET name = $2, updated_at = now() WHERE id = $1;

-- name: DeleteEmptyFolder :execrows
DELETE FROM content.asset_folders f
WHERE f.id = $1
  AND NOT EXISTS (SELECT 1 FROM content.assets a WHERE a.folder_id = f.id AND a.deleted_at IS NULL)
  AND NOT EXISTS (SELECT 1 FROM content.asset_folders c WHERE c.parent_id = f.id);

-- name: InsertAsset :exec
INSERT INTO content.assets (id, client_id, folder_id, kind, name, description, tags, current_version, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9);

-- name: GetAsset :one
SELECT * FROM content.assets WHERE id = $1 AND client_id = $2 AND deleted_at IS NULL;

-- name: SearchAssets :many
SELECT sqlc.embed(a), sqlc.embed(v)
FROM content.assets a
JOIN content.asset_versions v ON v.asset_id = a.id AND v.version = a.current_version
WHERE a.client_id = $1 AND a.deleted_at IS NULL
  AND (sqlc.narg(kind)::text IS NULL OR a.kind::text = sqlc.narg(kind))
  AND (sqlc.narg(folder_id)::text IS NULL OR a.folder_id = sqlc.narg(folder_id))
  AND (sqlc.narg(tag)::text IS NULL OR lower(sqlc.narg(tag)) = ANY (SELECT lower(t) FROM unnest(a.tags) t))
  AND (sqlc.narg(query)::text IS NULL
       OR a.name ILIKE '%' || sqlc.narg(query) || '%'
       OR a.description ILIKE '%' || sqlc.narg(query) || '%'
       OR EXISTS (SELECT 1 FROM unnest(a.tags) t WHERE t ILIKE '%' || sqlc.narg(query) || '%'))
ORDER BY a.updated_at DESC, a.id;

-- name: UpdateAsset :exec
UPDATE content.assets SET
    folder_id = $2,
    kind = $3,
    name = $4,
    description = $5,
    tags = $6,
    current_version = $7,
    updated_by = $8,
    updated_at = now()
WHERE id = $1;

-- name: SoftDeleteAsset :exec
UPDATE content.assets SET deleted_at = now(), updated_by = $2, updated_at = now() WHERE id = $1;

-- name: InsertAssetVersion :exec
INSERT INTO content.asset_versions (id, asset_id, version, file_key, file_name, mime_type, size_bytes, note, uploaded_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: ListAssetVersions :many
SELECT * FROM content.asset_versions WHERE asset_id = $1 ORDER BY version DESC;
