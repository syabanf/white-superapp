-- name: InsertDesignTask :exec
INSERT INTO content.design_tasks (id, item_id, client_id, title, brief, refs, size_preset, width, height, designer_id, due_date, status, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13, coalesce(sqlc.narg(created_at)::timestamptz, now()), coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: GetDesignTask :one
SELECT * FROM content.design_tasks WHERE id = $1 AND client_id = $2 AND deleted_at IS NULL;

-- name: ListDesignTasks :many
SELECT * FROM content.design_tasks
WHERE client_id = $1 AND deleted_at IS NULL
  AND (sqlc.narg(item_id)::text IS NULL OR item_id = sqlc.narg(item_id))
  AND (sqlc.narg(designer_id)::text IS NULL OR designer_id = sqlc.narg(designer_id))
  AND (sqlc.narg(status)::text IS NULL OR status::text = sqlc.narg(status))
ORDER BY due_date NULLS LAST, created_at;

-- name: UpdateDesignTask :exec
UPDATE content.design_tasks SET
    title = $2,
    brief = $3,
    refs = $4,
    size_preset = $5,
    width = $6,
    height = $7,
    designer_id = $8,
    due_date = $9,
    status = $10,
    current_version = $11,
    updated_by = $12,
    updated_at = now()
WHERE id = $1;

-- name: InsertDesignFile :exec
INSERT INTO content.design_files (id, task_id, version, sort_order, file_key, file_name, mime_type, size_bytes, uploaded_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: GetDesignFile :one
SELECT f.* FROM content.design_files f
JOIN content.design_tasks t ON t.id = f.task_id
WHERE f.id = $1 AND t.client_id = $2 AND t.deleted_at IS NULL;

-- name: ListDesignFiles :many
SELECT * FROM content.design_files WHERE task_id = $1 ORDER BY version DESC, sort_order;

-- name: FinalDesignFiles :many
SELECT f.* FROM content.design_files f
JOIN content.design_tasks t ON t.id = f.task_id AND f.version = t.current_version
WHERE t.id = (
    SELECT t2.id FROM content.design_tasks t2
    WHERE t2.item_id = $1 AND t2.status = 'APPROVED' AND t2.deleted_at IS NULL AND t2.current_version > 0
    ORDER BY t2.updated_at DESC
    LIMIT 1
)
ORDER BY f.sort_order;

-- name: InsertDesignFeedback :exec
INSERT INTO content.design_feedback (id, task_id, file_id, version, user_id, decision, body, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: ListDesignFeedback :many
SELECT * FROM content.design_feedback WHERE task_id = $1 ORDER BY created_at, id;
