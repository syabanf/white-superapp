-- name: ListTemplates :many
SELECT * FROM performance.logbook_templates
WHERE (sqlc.arg(department_id)::text = '' OR department_id = sqlc.arg(department_id)::text)
  AND (is_active OR sqlc.arg(include_inactive)::boolean)
ORDER BY created_at DESC;

-- name: ListTemplateItems :many
SELECT * FROM performance.logbook_template_items WHERE template_id = ANY(sqlc.arg(template_ids)::text[]) ORDER BY template_id, sort_order;

-- name: GetTemplate :one
SELECT * FROM performance.logbook_templates WHERE id = $1;

-- name: InsertTemplate :exec
INSERT INTO performance.logbook_templates (id, department_id, name, description, frequency, is_active, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7);

-- name: InsertTemplateItem :exec
INSERT INTO performance.logbook_template_items (id, template_id, title, description, weight, is_required, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: CountTemplateEntries :one
SELECT count(*)::int FROM performance.logbook_entries WHERE template_id = $1;

-- name: ArchiveTemplate :exec
UPDATE performance.logbook_templates SET is_active = false, updated_at = now() WHERE id = $1;

-- name: DeleteTemplate :exec
DELETE FROM performance.logbook_templates WHERE id = $1;

-- name: InsertEntry :exec
INSERT INTO performance.logbook_entries (id, template_id, department_id, employee_id, entry_date, title, notes, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: CopyTemplateItems :exec
INSERT INTO performance.logbook_entry_items (id, entry_id, template_item_id, title, description, weight, is_required, sort_order)
SELECT gen_random_uuid()::text, sqlc.arg(entry_id)::text, id, title, description, weight, is_required, sort_order
FROM performance.logbook_template_items WHERE template_id = sqlc.arg(template_id)::text
ORDER BY sort_order;

-- name: GetEntry :one
SELECT e.*, t.name AS template_name, t.frequency
FROM performance.logbook_entries e
JOIN performance.logbook_templates t ON t.id = e.template_id
WHERE e.id = $1;

-- name: ListEntryItems :many
SELECT * FROM performance.logbook_entry_items WHERE entry_id = ANY(sqlc.arg(entry_ids)::text[]) ORDER BY entry_id, sort_order;

-- name: GetEntryItem :one
SELECT * FROM performance.logbook_entry_items WHERE id = $1;

-- name: CountEntries :one
SELECT count(*)::int FROM performance.logbook_entries e
WHERE (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.narg(employee_ids)::text[] IS NULL OR e.employee_id = ANY(sqlc.narg(employee_ids)::text[]))
  AND (sqlc.arg(status)::text = '' OR e.status = sqlc.arg(status)::text)
  AND (sqlc.arg(entry_date)::text = '' OR e.entry_date = NULLIF(sqlc.arg(entry_date)::text, '')::date)
  AND (sqlc.arg(from_date)::text = '' OR e.entry_date >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (sqlc.arg(to_date)::text = '' OR e.entry_date <= NULLIF(sqlc.arg(to_date)::text, '')::date);

-- name: ListEntries :many
SELECT e.*, t.name AS template_name, t.frequency
FROM performance.logbook_entries e
JOIN performance.logbook_templates t ON t.id = e.template_id
WHERE (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.narg(employee_ids)::text[] IS NULL OR e.employee_id = ANY(sqlc.narg(employee_ids)::text[]))
  AND (sqlc.arg(status)::text = '' OR e.status = sqlc.arg(status)::text)
  AND (sqlc.arg(entry_date)::text = '' OR e.entry_date = NULLIF(sqlc.arg(entry_date)::text, '')::date)
  AND (sqlc.arg(from_date)::text = '' OR e.entry_date >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (sqlc.arg(to_date)::text = '' OR e.entry_date <= NULLIF(sqlc.arg(to_date)::text, '')::date)
ORDER BY e.entry_date DESC, e.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: UpdateEntryItem :exec
UPDATE performance.logbook_entry_items SET is_checked = $2, checked_by = $3, checked_at = $4, notes = $5, updated_at = now() WHERE id = $1;

-- name: UpdateEntry :exec
UPDATE performance.logbook_entries SET status = $2, notes = $3, submitted_by = $4, submitted_at = $5, reviewed_by = $6,
  reviewed_at = $7, review_notes = $8, updated_at = now()
WHERE id = $1;

-- name: SetEntryScore :exec
UPDATE performance.logbook_entries SET completion_percentage = $2, kpi_score = $3, updated_at = now() WHERE id = $1;

-- name: DeleteEntry :exec
DELETE FROM performance.logbook_entries WHERE id = $1;

-- name: LogbookSummaryRows :many
SELECT department_id, status, completion_percentage, kpi_score
FROM performance.logbook_entries
WHERE (sqlc.arg(department_id)::text = '' OR department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(from_date)::text = '' OR entry_date >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (sqlc.arg(to_date)::text = '' OR entry_date <= NULLIF(sqlc.arg(to_date)::text, '')::date)
ORDER BY entry_date DESC;

-- name: DailyTemplateDepartments :many
SELECT DISTINCT department_id FROM performance.logbook_templates WHERE is_active AND frequency = 'daily';
