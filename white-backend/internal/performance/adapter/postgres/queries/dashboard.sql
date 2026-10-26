-- name: DashTaskDays :many
-- Occurrences per day, department and assignee with status counts,
-- on-time finishes (done or approved on or before the day, WIB) and the
-- review turnaround of approved and rejected work.
SELECT o.occurrence_date AS day, t.department_id, coalesce(t.assignee_employee_id, '')::text AS employee_id,
       count(*) FILTER (WHERE o.status = 'pending')::int AS pending,
       count(*) FILTER (WHERE o.status = 'in_progress')::int AS in_progress,
       count(*) FILTER (WHERE o.status = 'done')::int AS done,
       count(*) FILTER (WHERE o.status = 'approved')::int AS approved,
       count(*) FILTER (WHERE o.status = 'rejected')::int AS rejected,
       count(*) FILTER (WHERE o.status IN ('done', 'approved') AND o.done_at IS NOT NULL
         AND (o.done_at AT TIME ZONE 'Asia/Jakarta')::date <= o.occurrence_date)::int AS on_time,
       count(*) FILTER (WHERE o.reviewed_at IS NOT NULL AND o.done_at IS NOT NULL AND o.status IN ('approved', 'rejected'))::int AS reviewed,
       coalesce(sum(extract(epoch FROM o.reviewed_at - o.done_at) / 3600)
         FILTER (WHERE o.reviewed_at IS NOT NULL AND o.done_at IS NOT NULL AND o.status IN ('approved', 'rejected')), 0)::float8 AS turnaround_hours,
       count(*) FILTER (WHERE o.status IN ('approved', 'rejected') AND o.reviewed_at - o.done_at < interval '4 hours')::int AS turn_lt4h,
       count(*) FILTER (WHERE o.status IN ('approved', 'rejected') AND o.reviewed_at - o.done_at >= interval '4 hours'
         AND o.reviewed_at - o.done_at < interval '24 hours')::int AS turn_4_24h,
       count(*) FILTER (WHERE o.status IN ('approved', 'rejected') AND o.reviewed_at - o.done_at >= interval '24 hours'
         AND o.reviewed_at - o.done_at < interval '72 hours')::int AS turn_1_3d,
       count(*) FILTER (WHERE o.status IN ('approved', 'rejected') AND o.reviewed_at - o.done_at >= interval '72 hours')::int AS turn_gt3d
FROM performance.department_task_occurrences o
JOIN performance.department_tasks t ON t.id = o.task_id
WHERE o.occurrence_date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
  AND (sqlc.arg(all_employees)::boolean OR t.assignee_employee_id = ANY(sqlc.arg(employee_ids)::text[]))
  AND (sqlc.arg(department_id)::text = '' OR t.department_id = sqlc.arg(department_id)::text)
GROUP BY 1, 2, 3
ORDER BY 1;

-- name: DashLogbookDays :many
SELECT e.entry_date AS day, e.department_id, e.employee_id,
       count(*) FILTER (WHERE e.status = 'draft')::int AS draft,
       count(*) FILTER (WHERE e.status = 'submitted')::int AS submitted,
       count(*) FILTER (WHERE e.status = 'reviewed')::int AS reviewed,
       count(*) FILTER (WHERE e.status = 'rejected')::int AS rejected,
       coalesce(sum(e.kpi_score) FILTER (WHERE e.status <> 'draft'), 0)::float8 AS score_sum
FROM performance.logbook_entries e
WHERE e.entry_date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
  AND (sqlc.arg(all_employees)::boolean OR e.employee_id = ANY(sqlc.arg(employee_ids)::text[]))
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
GROUP BY 1, 2, 3
ORDER BY 1;

-- name: DashRecapDays :many
SELECT r.recap_date AS day, r.employee_id, count(*)::int AS recaps
FROM performance.daily_recaps r
WHERE r.recap_date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
  AND (sqlc.arg(all_employees)::boolean OR r.employee_id = ANY(sqlc.arg(employee_ids)::text[]))
  AND (sqlc.arg(department_id)::text = '' OR r.department_id = sqlc.arg(department_id)::text)
GROUP BY 1, 2
ORDER BY 1;

-- name: DashScorecards :many
SELECT s.employee_id, s.department_id, s.period_year, s.period_month, s.score, s.status
FROM performance.kpi_scorecards s
WHERE make_date(s.period_year, s.period_month, 1) BETWEEN date_trunc('month', sqlc.arg(from_date)::date)::date AND sqlc.arg(to_date)::date
  AND (sqlc.arg(all_employees)::boolean OR s.employee_id = ANY(sqlc.arg(employee_ids)::text[]))
  AND (sqlc.arg(department_id)::text = '' OR s.department_id = sqlc.arg(department_id)::text)
ORDER BY s.period_year, s.period_month;

-- name: DashIndicatorAttainment :many
SELECT i.id AS indicator_id, i.code, i.name, i.unit, i.direction,
       coalesce(avg(s.actual), 0)::float8 AS actual, count(s.actual)::int AS actuals,
       coalesce(avg(s.target), 0)::float8 AS target, count(s.target)::int AS targets,
       coalesce(avg(s.attainment), 0)::float8 AS attainment, count(s.attainment)::int AS samples
FROM performance.kpi_snapshots s
JOIN performance.kpi_indicators i ON i.id = s.indicator_id
WHERE make_date(s.period_year, s.period_month, 1) BETWEEN date_trunc('month', sqlc.arg(from_date)::date)::date AND sqlc.arg(to_date)::date
  AND (sqlc.arg(all_employees)::boolean OR s.employee_id = ANY(sqlc.arg(employee_ids)::text[]))
GROUP BY i.id, i.code, i.name, i.unit, i.direction
ORDER BY i.name;

-- name: DashImprovements :many
SELECT m.id, m.scope, m.employee_id, m.department_id, m.owner_employee_id, m.source, m.status, m.title,
       m.due_date, m.created_at, m.completed_at, m.updated_at
FROM performance.improvements m
WHERE m.deleted_at IS NULL
  AND m.created_at < (sqlc.arg(to_date)::date + 1)::timestamp AT TIME ZONE 'Asia/Jakarta'
  AND (sqlc.arg(all_employees)::boolean OR m.employee_id = ANY(sqlc.arg(employee_ids)::text[])
       OR m.owner_employee_id = ANY(sqlc.arg(employee_ids)::text[]))
  AND (sqlc.arg(department_id)::text = '' OR m.department_id = sqlc.arg(department_id)::text);

-- name: DashFeedbackAssignments :many
SELECT a.relationship_type, a.status, count(*)::int AS total
FROM performance.feedback_assignments a
WHERE a.cycle_id = sqlc.arg(cycle_id)
  AND (sqlc.arg(all_employees)::boolean OR a.employee_id = ANY(sqlc.arg(employee_ids)::text[]))
GROUP BY 1, 2;
