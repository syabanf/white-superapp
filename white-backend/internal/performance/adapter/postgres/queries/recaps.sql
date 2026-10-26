-- name: GetDailyRecap :one
SELECT * FROM performance.daily_recaps WHERE employee_id = $1 AND recap_date = $2;

-- name: UpsertDailyRecap :exec
INSERT INTO performance.daily_recaps (id, employee_id, department_id, recap_date, items, generated_summary, summary_model, generated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (employee_id, recap_date) DO UPDATE SET department_id = EXCLUDED.department_id, items = EXCLUDED.items,
  generated_summary = EXCLUDED.generated_summary, summary_model = EXCLUDED.summary_model,
  generated_at = EXCLUDED.generated_at, updated_at = now();

-- name: SetDailyNotes :exec
INSERT INTO performance.daily_recaps (id, employee_id, department_id, recap_date, notes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (employee_id, recap_date) DO UPDATE SET notes = EXCLUDED.notes, updated_at = now();

-- name: ListDailyRecaps :many
SELECT * FROM performance.daily_recaps
WHERE (sqlc.arg(recap_date)::text = '' OR recap_date = NULLIF(sqlc.arg(recap_date)::text, '')::date)
  AND (sqlc.narg(employee_ids)::text[] IS NULL OR employee_id = ANY(sqlc.narg(employee_ids)::text[]))
  AND (sqlc.arg(from_date)::text = '' OR recap_date >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (sqlc.arg(to_date)::text = '' OR recap_date <= NULLIF(sqlc.arg(to_date)::text, '')::date)
ORDER BY recap_date DESC, employee_id
LIMIT sqlc.arg(row_limit)::int;

-- name: UpsertMonthlyRecap :one
INSERT INTO performance.monthly_recaps (id, scope, subject_id, department_id, period_year, period_month, metrics, accomplishments,
  narrative, narrative_model, generated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
ON CONFLICT (scope, subject_id, period_year, period_month) DO UPDATE SET department_id = EXCLUDED.department_id,
  metrics = EXCLUDED.metrics, accomplishments = EXCLUDED.accomplishments, narrative = EXCLUDED.narrative,
  narrative_model = EXCLUDED.narrative_model, generated_at = now(), updated_at = now()
RETURNING id;

-- name: GetMonthlyRecap :one
SELECT * FROM performance.monthly_recaps WHERE id = $1;

-- name: ListMonthlyRecaps :many
SELECT * FROM performance.monthly_recaps
WHERE (sqlc.narg(period_year)::int IS NULL OR (period_year = sqlc.narg(period_year)::int AND period_month = sqlc.narg(period_month)::int))
  AND (sqlc.arg(scope)::text = '' OR scope = sqlc.arg(scope)::text)
  AND (sqlc.narg(subject_ids)::text[] IS NULL OR subject_id = ANY(sqlc.narg(subject_ids)::text[]))
  AND (sqlc.arg(department_id)::text = '' OR department_id = sqlc.arg(department_id)::text)
ORDER BY period_year DESC, period_month DESC, scope DESC, subject_id
LIMIT sqlc.arg(row_limit)::int;
