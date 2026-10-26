-- name: InsertImprovement :execrows
INSERT INTO performance.improvements (id, scope, employee_id, department_id, source, source_ref, title, description,
  owner_employee_id, due_date, status, linked_task_id, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)
ON CONFLICT (source, source_ref) WHERE source_ref IS NOT NULL DO NOTHING;

-- name: GetImprovement :one
SELECT * FROM performance.improvements WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateImprovement :exec
UPDATE performance.improvements SET title = $2, description = $3, owner_employee_id = $4, due_date = $5, status = $6,
  linked_task_id = $7, completed_at = $8, updated_by = $9, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteImprovement :exec
UPDATE performance.improvements SET deleted_at = now(), updated_by = $2, updated_at = now() WHERE id = $1;

-- name: ListImprovements :many
SELECT * FROM performance.improvements
WHERE deleted_at IS NULL
  AND (sqlc.arg(scope)::text = '' OR scope = sqlc.arg(scope)::text)
  AND (sqlc.narg(employee_ids)::text[] IS NULL OR employee_id = ANY(sqlc.narg(employee_ids)::text[]))
  AND (sqlc.narg(department_ids)::text[] IS NULL OR department_id = ANY(sqlc.narg(department_ids)::text[]))
  AND (sqlc.arg(person_id)::text = '' OR employee_id = sqlc.arg(person_id)::text OR owner_employee_id = sqlc.arg(person_id)::text)
  AND (sqlc.narg(statuses)::text[] IS NULL OR status = ANY(sqlc.narg(statuses)::text[]))
  AND (sqlc.arg(source)::text = '' OR source = sqlc.arg(source)::text)
ORDER BY array_position(ARRAY['suggested', 'open', 'in_progress', 'done', 'dropped', 'declined'], status),
  due_date NULLS LAST, created_at DESC
LIMIT sqlc.arg(row_limit)::int;

-- name: OverdueImprovements :many
SELECT * FROM performance.improvements
WHERE deleted_at IS NULL AND status IN ('open', 'in_progress') AND due_date < sqlc.arg(today)::date
ORDER BY due_date;

-- name: ListRetros :many
SELECT * FROM performance.retro_sessions
WHERE sqlc.narg(department_ids)::text[] IS NULL OR department_id = ANY(sqlc.narg(department_ids)::text[])
ORDER BY period_year DESC, period_month DESC, department_id
LIMIT 200;

-- name: GetRetro :one
SELECT * FROM performance.retro_sessions WHERE id = $1;

-- name: GetRetroFor :one
SELECT * FROM performance.retro_sessions WHERE department_id = $1 AND period_year = $2 AND period_month = $3;

-- name: InsertRetro :exec
INSERT INTO performance.retro_sessions (id, department_id, period_year, period_month, went_well, to_improve, actions, status, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9);

-- name: UpdateRetro :exec
UPDATE performance.retro_sessions SET went_well = $2, to_improve = $3, actions = $4, status = $5, closed_at = $6,
  updated_by = $7, updated_at = now()
WHERE id = $1;

-- name: ClaimSend :execrows
INSERT INTO performance.reminder_sends (kind, subject_id, day) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;
