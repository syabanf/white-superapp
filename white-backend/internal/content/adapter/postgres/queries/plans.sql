-- name: InsertPlan :exec
INSERT INTO content.plans (id, client_id, month, title, status, goals, pillars, tone, notes, frequency, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11, coalesce(sqlc.narg(created_at)::timestamptz, now()), coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: GetPlan :one
SELECT * FROM content.plans WHERE id = $1 AND client_id = $2 AND deleted_at IS NULL;

-- name: GetPlanByMonth :one
SELECT * FROM content.plans WHERE client_id = $1 AND month = $2 AND deleted_at IS NULL;

-- name: ListPlans :many
SELECT sqlc.embed(p),
    (SELECT count(*) FROM content.items i WHERE i.plan_id = p.id AND i.deleted_at IS NULL)::int AS item_count,
    (SELECT count(*) FROM content.items i WHERE i.plan_id = p.id AND i.deleted_at IS NULL AND i.status = 'PUBLISHED')::int AS published_count
FROM content.plans p
WHERE p.client_id = $1 AND p.deleted_at IS NULL
ORDER BY p.month DESC;

-- name: UpdatePlan :exec
UPDATE content.plans SET
    title = $2,
    status = $3,
    goals = $4,
    pillars = $5,
    tone = $6,
    notes = $7,
    frequency = $8,
    ai_model = $9,
    generated_at = $10,
    submitted_by = $11,
    submitted_at = $12,
    approved_by = $13,
    approved_at = $14,
    review_note = $15,
    updated_by = $16,
    updated_at = now()
WHERE id = $1;

-- name: SoftDeletePlan :exec
UPDATE content.plans SET deleted_at = now(), updated_by = $2, updated_at = now() WHERE id = $1;

-- name: InsertPlanComment :exec
INSERT INTO content.plan_comments (id, plan_id, user_id, body, created_at) VALUES ($1, $2, $3, $4, coalesce(sqlc.narg(created_at)::timestamptz, now()));

-- name: ListPlanComments :many
SELECT * FROM content.plan_comments WHERE plan_id = $1 ORDER BY created_at, id;
