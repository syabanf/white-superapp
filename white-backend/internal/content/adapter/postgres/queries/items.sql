-- name: InsertItem :exec
INSERT INTO content.items (
    id, plan_id, client_id, planned_date, publish_at, platform, format, pillar, title, hook, caption, hashtags,
    visual_brief, copywriter_id, designer_id, copy_due, design_due, review_due, status, caption_version, created_by, updated_by,
    created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $21,
    coalesce(sqlc.narg(created_at)::timestamptz, now()), coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: GetItem :one
SELECT sqlc.embed(i), EXISTS (
    SELECT 1 FROM content.design_tasks t JOIN content.design_files f ON f.task_id = t.id AND f.version = t.current_version
    WHERE t.item_id = i.id AND t.status = 'APPROVED' AND t.deleted_at IS NULL
)::bool AS has_final_design
FROM content.items i
WHERE i.id = $1 AND i.client_id = $2 AND i.deleted_at IS NULL;

-- name: ListItems :many
SELECT sqlc.embed(i), EXISTS (
    SELECT 1 FROM content.design_tasks t JOIN content.design_files f ON f.task_id = t.id AND f.version = t.current_version
    WHERE t.item_id = i.id AND t.status = 'APPROVED' AND t.deleted_at IS NULL
)::bool AS has_final_design
FROM content.items i
WHERE i.client_id = $1 AND i.deleted_at IS NULL
  AND (sqlc.narg(plan_id)::text IS NULL OR i.plan_id = sqlc.narg(plan_id))
  AND (sqlc.narg(status)::text IS NULL OR i.status::text = sqlc.narg(status))
  AND (sqlc.narg(assignee_id)::text IS NULL OR i.copywriter_id = sqlc.narg(assignee_id) OR i.designer_id = sqlc.narg(assignee_id))
  AND (sqlc.narg(from_day)::date IS NULL OR i.planned_date BETWEEN sqlc.narg(from_day) AND sqlc.narg(to_day))
ORDER BY i.planned_date, i.created_at, i.id;

-- name: UpdateItem :exec
UPDATE content.items SET
    planned_date = $2,
    publish_at = $3,
    platform = $4,
    format = $5,
    pillar = $6,
    title = $7,
    hook = $8,
    caption = $9,
    hashtags = $10,
    visual_brief = $11,
    copywriter_id = $12,
    designer_id = $13,
    copy_due = $14,
    design_due = $15,
    review_due = $16,
    status = $17,
    revision_round = $18,
    caption_version = $19,
    copy_done_at = $20,
    design_done_at = $21,
    approved_at = $22,
    approved_by = $23,
    scheduled_by = $24,
    publishing_post_id = $25,
    published_at = $26,
    permalink = $27,
    external_ids = $28,
    publish_error = $29,
    updated_by = $30,
    updated_at = now()
WHERE id = $1;

-- name: BackdateItem :exec
UPDATE content.items SET created_at = $2, updated_at = $3 WHERE id = $1;

-- name: SoftDeleteItem :exec
UPDATE content.items SET deleted_at = now(), updated_by = $2, updated_at = now() WHERE id = $1;

-- name: SoftDeleteIdeaItems :exec
UPDATE content.items SET deleted_at = now(), updated_by = $2, updated_at = now()
WHERE plan_id = $1 AND status = 'IDEA' AND deleted_at IS NULL;

-- name: ListScheduledDue :many
SELECT * FROM content.items
WHERE status = 'SCHEDULED' AND deleted_at IS NULL AND publishing_post_id IS NOT NULL AND publish_at <= $1
ORDER BY publish_at
LIMIT $2;

-- name: GetItemByPublishingPost :one
SELECT * FROM content.items
WHERE publishing_post_id = $1 AND client_id = $2 AND deleted_at IS NULL
LIMIT 1;

-- name: ListOpenItemRefs :many
SELECT id, client_id, title, status, copywriter_id, designer_id, copy_due, design_due, review_due
FROM content.items
WHERE deleted_at IS NULL AND status IN ('IDEA', 'COPYWRITING', 'DESIGN', 'REVIEW', 'REVISION')
ORDER BY id;

-- name: InsertRevision :exec
INSERT INTO content.caption_revisions (id, item_id, version, caption, hook, hashtags, note, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: ListRevisions :many
SELECT * FROM content.caption_revisions WHERE item_id = $1 ORDER BY version DESC;

-- name: InsertItemComment :exec
INSERT INTO content.item_comments (id, item_id, user_id, body, kind, round, created_at) VALUES ($1, $2, $3, $4, $5, $6, coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: ListItemComments :many
SELECT * FROM content.item_comments WHERE item_id = $1 ORDER BY created_at, id;

-- name: InsertActivity :exec
INSERT INTO content.item_activities (id, item_id, user_id, type, from_status, to_status, note, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: ListActivities :many
SELECT * FROM content.item_activities WHERE item_id = $1 ORDER BY created_at DESC, id DESC;

-- name: MarkReminder :execrows
INSERT INTO content.reminders (item_id, kind, day) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;
