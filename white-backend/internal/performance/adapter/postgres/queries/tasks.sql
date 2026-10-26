-- name: ActiveTaskSchedules :many
SELECT id, recurrence, weekly_day, monthly_day, due_date, (created_at AT TIME ZONE timezone)::date AS created_date
FROM performance.department_tasks
WHERE is_active AND (sqlc.narg(department_ids)::text[] IS NULL OR department_id = ANY(sqlc.narg(department_ids)::text[]));

-- name: ActiveTaskDepartments :many
SELECT DISTINCT department_id FROM performance.department_tasks WHERE is_active;

-- name: InsertOccurrences :exec
INSERT INTO performance.department_task_occurrences (id, task_id, occurrence_date)
SELECT gen_random_uuid()::text, (sqlc.arg(task_ids)::text[])[i], (sqlc.arg(dates)::date[])[i]
FROM generate_subscripts(sqlc.arg(task_ids)::text[], 1) AS i
ON CONFLICT (task_id, occurrence_date) DO NOTHING;

-- name: InsertTask :exec
INSERT INTO performance.department_tasks (id, department_id, assignee_employee_id, title, description, recurrence,
  weekly_day, monthly_day, due_date, priority, kpi_indicator_id, kpi_contribution, timezone, review_required,
  created_by_employee_id, created_by_user_id, created_by_name, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $18);

-- name: InsertSubtask :exec
INSERT INTO performance.department_task_subtasks (id, task_id, title, weight, sort_order) VALUES ($1, $2, $3, $4, $5);

-- name: GetTask :one
SELECT * FROM performance.department_tasks WHERE id = $1;

-- name: UpdateTask :exec
UPDATE performance.department_tasks SET title = $2, description = $3, priority = $4, assignee_employee_id = $5,
  kpi_indicator_id = $6, kpi_contribution = $7, is_active = $8, updated_by = $9, updated_at = now()
WHERE id = $1;

-- name: QueryItems :many
SELECT o.id AS occurrence_id, t.id AS task_id, t.title, COALESCE(t.description, '')::text AS description, t.department_id,
  t.assignee_employee_id, t.created_by_user_id, t.created_by_name,
  o.occurrence_date, o.status, o.progress_percent, o.started_at, o.done_at, o.done_by_name, o.review_notes, o.reviewed_by_name,
  t.recurrence, t.priority, t.review_required, t.timezone, t.weekly_day, t.monthly_day, t.due_date,
  t.kpi_indicator_id, k.code AS kpi_code, k.name AS kpi_name, t.kpi_contribution,
  COALESCE(st.total, 0)::float8 AS subtask_total, COALESCE(st.done, 0)::float8 AS subtask_done, COALESCE(st.n, 0)::int AS subtask_count,
  (SELECT count(*) FROM performance.department_task_updates u WHERE u.occurrence_id = o.id AND u.kind = 'comment')::int AS comment_count,
  (SELECT count(*) FROM performance.department_task_attachments x WHERE x.occurrence_id = o.id)::int AS attachment_count,
  lu.note AS last_note, COALESCE(lu.author_name, '')::text AS last_note_by, COALESCE(lu.created_at, 'epoch'::timestamptz)::timestamptz AS last_note_at
FROM performance.department_task_occurrences o
JOIN performance.department_tasks t ON t.id = o.task_id
LEFT JOIN performance.kpi_indicators k ON k.id = t.kpi_indicator_id
LEFT JOIN LATERAL (
  SELECT SUM(s.weight) AS total, SUM(s.weight) FILTER (WHERE oi.is_checked) AS done, count(*) AS n
  FROM performance.department_task_subtasks s
  LEFT JOIN performance.department_task_occurrence_items oi ON oi.subtask_id = s.id AND oi.occurrence_id = o.id
  WHERE s.task_id = t.id) st ON true
LEFT JOIN LATERAL (
  SELECT u.note, u.author_name, u.created_at FROM performance.department_task_updates u
  WHERE u.occurrence_id = o.id AND u.note IS NOT NULL ORDER BY u.created_at DESC LIMIT 1) lu ON true
WHERE (sqlc.arg(occurrence_id)::text = '' OR o.id = sqlc.arg(occurrence_id)::text)
  AND (sqlc.arg(occurrence_id)::text <> '' OR t.is_active)
  AND (sqlc.narg(department_ids)::text[] IS NULL OR t.department_id = ANY(sqlc.narg(department_ids)::text[]))
  AND (NOT sqlc.arg(mine_only)::boolean OR t.assignee_employee_id = sqlc.arg(mine_employee_id)::text
       OR (t.assignee_employee_id IS NULL AND t.department_id = sqlc.arg(mine_department_id)::text)
       OR t.created_by_user_id = sqlc.arg(mine_user_id)::text)
  AND (sqlc.arg(span_start)::text = ''
       OR o.occurrence_date BETWEEN sqlc.arg(span_start)::text::date AND sqlc.arg(span_end)::text::date
       OR (sqlc.arg(overdue_before)::text <> '' AND o.status IN ('pending', 'in_progress', 'rejected')
           AND o.occurrence_date < sqlc.arg(overdue_before)::text::date))
  AND (sqlc.arg(assignee_id)::text = '' OR t.assignee_employee_id = sqlc.arg(assignee_id)::text)
  AND (sqlc.arg(status)::text = '' OR o.status = sqlc.arg(status)::text)
  AND (sqlc.arg(indicator_id)::text = '' OR t.kpi_indicator_id = sqlc.arg(indicator_id)::text)
  AND (NOT sqlc.arg(kpi_only)::boolean OR t.kpi_indicator_id IS NOT NULL)
  AND (sqlc.arg(q)::text = '' OR t.title ILIKE '%' || sqlc.arg(q)::text || '%' OR t.description ILIKE '%' || sqlc.arg(q)::text || '%')
ORDER BY (o.status IN ('pending', 'in_progress', 'rejected') AND sqlc.arg(overdue_before)::text <> ''
          AND o.occurrence_date < NULLIF(sqlc.arg(overdue_before)::text, '')::date) DESC,
  o.occurrence_date, array_position(ARRAY['urgent', 'high', 'normal', 'low'], t.priority), t.created_at
LIMIT 2000;

-- name: GetOccurrence :one
SELECT o.*, t.title, t.department_id, t.assignee_employee_id, t.review_required, t.created_by_user_id, t.kpi_indicator_id
FROM performance.department_task_occurrences o
JOIN performance.department_tasks t ON t.id = o.task_id
WHERE o.id = $1;

-- name: SaveOccurrence :exec
UPDATE performance.department_task_occurrences SET status = $2, progress_percent = $3, started_at = $4, done_at = $5,
  done_by = $6, done_by_name = $7, review_notes = $8, reviewed_by = $9, reviewed_by_name = $10, reviewed_at = $11, updated_at = now()
WHERE id = $1;

-- name: CountSubtasks :one
SELECT count(*)::int FROM performance.department_task_subtasks WHERE task_id = $1;

-- name: SubtaskExists :one
SELECT EXISTS (SELECT 1 FROM performance.department_task_subtasks WHERE id = $1 AND task_id = $2);

-- name: ListSubtaskStates :many
SELECT st.id, st.title, st.weight, st.sort_order, COALESCE(oi.is_checked, false)::boolean AS is_checked, oi.checked_by_name, oi.checked_at
FROM performance.department_task_subtasks st
LEFT JOIN performance.department_task_occurrence_items oi ON oi.subtask_id = st.id AND oi.occurrence_id = sqlc.arg(occurrence_id)::text
WHERE st.task_id = sqlc.arg(task_id)::text
ORDER BY st.sort_order, st.created_at;

-- name: UpsertOccurrenceItem :exec
INSERT INTO performance.department_task_occurrence_items (occurrence_id, subtask_id, is_checked, checked_at, checked_by_name)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (occurrence_id, subtask_id) DO UPDATE SET is_checked = EXCLUDED.is_checked, checked_at = EXCLUDED.checked_at,
  checked_by_name = EXCLUDED.checked_by_name, updated_at = now();

-- name: SubtaskProgress :one
SELECT COALESCE(SUM(st.weight), 0)::float8 AS total, COALESCE(SUM(st.weight) FILTER (WHERE oi.is_checked), 0)::float8 AS done
FROM performance.department_task_subtasks st
LEFT JOIN performance.department_task_occurrence_items oi ON oi.subtask_id = st.id AND oi.occurrence_id = sqlc.arg(occurrence_id)::text
WHERE st.task_id = sqlc.arg(task_id)::text;

-- name: InsertTaskUpdate :exec
INSERT INTO performance.department_task_updates (id, occurrence_id, kind, note, progress_percent, author_employee_id, author_user_id, author_name, detail, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: ListTaskUpdates :many
SELECT * FROM performance.department_task_updates WHERE occurrence_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: InsertAttachment :exec
INSERT INTO performance.department_task_attachments (id, occurrence_id, name, mime, size_bytes, storage_path, uploaded_by_user_id, uploaded_by_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListAttachments :many
SELECT * FROM performance.department_task_attachments WHERE occurrence_id = $1 ORDER BY created_at;

-- name: GetAttachment :one
SELECT * FROM performance.department_task_attachments WHERE id = $1;

-- name: DeleteAttachment :exec
DELETE FROM performance.department_task_attachments WHERE id = $1;

-- name: RecordContribution :exec
INSERT INTO performance.department_task_kpi_contributions (occurrence_id, indicator_id, department_id, employee_id, period_year, period_month, amount)
SELECT sqlc.arg(occurrence_id)::text, t.kpi_indicator_id, t.department_id, t.assignee_employee_id,
       sqlc.arg(period_year)::int, sqlc.arg(period_month)::int, t.kpi_contribution
FROM performance.department_tasks t WHERE t.id = sqlc.arg(task_id)::text AND t.kpi_indicator_id IS NOT NULL
ON CONFLICT (occurrence_id) DO UPDATE SET indicator_id = EXCLUDED.indicator_id, department_id = EXCLUDED.department_id,
  employee_id = EXCLUDED.employee_id, period_year = EXCLUDED.period_year, period_month = EXCLUDED.period_month,
  amount = EXCLUDED.amount, recorded_at = now();

-- name: DropContribution :exec
DELETE FROM performance.department_task_kpi_contributions WHERE occurrence_id = $1;

-- name: GetContribution :one
SELECT * FROM performance.department_task_kpi_contributions WHERE occurrence_id = $1;

-- name: ContributionTotals :many
SELECT c.indicator_id, COALESCE(sum(c.amount), 0)::float8 AS total
FROM performance.department_task_kpi_contributions c
JOIN performance.department_task_occurrences o ON o.id = c.occurrence_id
JOIN performance.department_tasks t ON t.id = o.task_id
WHERE t.department_id = sqlc.arg(department_id)::text
  AND o.occurrence_date BETWEEN sqlc.arg(start_date)::date AND sqlc.arg(end_date)::date
GROUP BY 1;

-- name: EmployeeContributions :many
SELECT k.id AS indicator_id, k.code, count(o.id)::int AS tasks_total,
       count(o.id) FILTER (WHERE o.status = 'approved')::int AS tasks_done,
       COALESCE(sum(c.amount), 0)::float8 AS contribution_total
FROM performance.department_task_occurrences o
JOIN performance.department_tasks t ON t.id = o.task_id
JOIN performance.kpi_indicators k ON k.id = t.kpi_indicator_id
LEFT JOIN performance.department_task_kpi_contributions c ON c.occurrence_id = o.id
WHERE t.assignee_employee_id = sqlc.arg(employee_id)::text
  AND o.occurrence_date BETWEEN sqlc.arg(start_date)::date AND sqlc.arg(end_date)::date
GROUP BY k.id, k.code ORDER BY k.code;

-- name: DepartmentKPIIndicators :many
SELECT k.*, di.weight AS department_weight
FROM performance.kpi_indicators k
LEFT JOIN performance.kpi_department_indicators di ON di.indicator_id = k.id AND di.department_id = sqlc.arg(department_id)::text
WHERE k.is_active AND (di.indicator_id IS NOT NULL OR k.id IN (
  SELECT kpi_indicator_id FROM performance.department_tasks
  WHERE department_id = sqlc.arg(department_id)::text AND is_active AND kpi_indicator_id IS NOT NULL))
ORDER BY di.weight DESC NULLS LAST, k.name;

-- name: IndicatorDepartments :many
SELECT indicator_id, array_agg(department_id ORDER BY department_id)::text[] AS department_ids
FROM performance.kpi_department_indicators GROUP BY indicator_id;

-- name: ReminderLines :many
SELECT t.assignee_employee_id::text AS employee_id, t.title, o.occurrence_date, t.priority
FROM performance.department_task_occurrences o
JOIN performance.department_tasks t ON t.id = o.task_id
WHERE t.is_active AND t.assignee_employee_id IS NOT NULL
  AND o.status IN ('pending', 'in_progress', 'rejected') AND o.occurrence_date <= sqlc.arg(up_to)::date
ORDER BY t.assignee_employee_id, o.occurrence_date, array_position(ARRAY['urgent', 'high', 'normal', 'low'], t.priority), t.created_at;

-- name: RecapOccurrences :many
SELECT o.id AS occurrence_id, t.id AS task_id, t.title, o.status, t.priority, o.progress_percent,
  COALESCE(st.total, 0)::float8 AS subtask_total, COALESCE(st.done, 0)::float8 AS subtask_done,
  COALESCE(o.done_at >= sqlc.arg(from_at)::timestamptz AND o.done_at < sqlc.arg(to_at)::timestamptz, false)::boolean AS completed_in_day
FROM performance.department_task_occurrences o
JOIN performance.department_tasks t ON t.id = o.task_id
LEFT JOIN LATERAL (
  SELECT SUM(s.weight) AS total, SUM(s.weight) FILTER (WHERE oi.is_checked) AS done
  FROM performance.department_task_subtasks s
  LEFT JOIN performance.department_task_occurrence_items oi ON oi.subtask_id = s.id AND oi.occurrence_id = o.id
  WHERE s.task_id = t.id) st ON true
WHERE (o.done_by = sqlc.arg(employee_id)::text AND o.done_at >= sqlc.arg(from_at)::timestamptz AND o.done_at < sqlc.arg(to_at)::timestamptz)
   OR EXISTS (SELECT 1 FROM performance.department_task_updates u
              WHERE u.occurrence_id = o.id AND u.author_employee_id = sqlc.arg(employee_id)::text
                AND u.created_at >= sqlc.arg(from_at)::timestamptz AND u.created_at < sqlc.arg(to_at)::timestamptz)
ORDER BY completed_in_day DESC, array_position(ARRAY['urgent', 'high', 'normal', 'low'], t.priority), t.title;

-- name: RecapNotes :many
SELECT u.occurrence_id, u.note::text AS note
FROM performance.department_task_updates u
WHERE u.author_employee_id = sqlc.arg(employee_id)::text AND u.note IS NOT NULL
  AND u.created_at >= sqlc.arg(from_at)::timestamptz AND u.created_at < sqlc.arg(to_at)::timestamptz
ORDER BY u.created_at;

-- name: MonthTasks :many
SELECT COALESCE(t.assignee_employee_id, o.done_by, '')::text AS employee_id, t.title, o.occurrence_date, o.status, t.priority, o.done_at,
  COALESCE(k.code, '')::text AS kpi_code, COALESCE(c.amount, 0)::float8 AS contributed
FROM performance.department_task_occurrences o
JOIN performance.department_tasks t ON t.id = o.task_id
LEFT JOIN performance.kpi_indicators k ON k.id = t.kpi_indicator_id
LEFT JOIN performance.department_task_kpi_contributions c ON c.occurrence_id = o.id
WHERE COALESCE(t.assignee_employee_id, o.done_by) = ANY(sqlc.arg(employee_ids)::text[])
  AND o.occurrence_date BETWEEN sqlc.arg(start_date)::date AND sqlc.arg(end_date)::date
ORDER BY o.occurrence_date;
