-- name: ListIndicators :many
SELECT * FROM performance.kpi_indicators
WHERE is_active OR sqlc.arg(include_inactive)::boolean
ORDER BY name;

-- name: GetIndicator :one
SELECT * FROM performance.kpi_indicators WHERE id = $1;

-- name: GetIndicatorByCode :one
SELECT * FROM performance.kpi_indicators WHERE code = $1;

-- name: UpsertIndicator :exec
INSERT INTO performance.kpi_indicators (id, code, name, description, unit, direction, source_kind, default_target, is_active, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
ON CONFLICT (id) DO UPDATE SET
  code = EXCLUDED.code, name = EXCLUDED.name, description = EXCLUDED.description, unit = EXCLUDED.unit,
  direction = EXCLUDED.direction, source_kind = EXCLUDED.source_kind, default_target = EXCLUDED.default_target,
  is_active = EXCLUDED.is_active, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: DeleteUnusedIndicator :execrows
-- Skips an indicator with history that the delete would cascade away
-- (snapshots, task contributions) or unlink (department tasks).
DELETE FROM performance.kpi_indicators k
WHERE k.id = sqlc.arg(id)::text
  AND NOT EXISTS (SELECT 1 FROM performance.kpi_snapshots s WHERE s.indicator_id = k.id)
  AND NOT EXISTS (SELECT 1 FROM performance.department_task_kpi_contributions c WHERE c.indicator_id = k.id)
  AND NOT EXISTS (SELECT 1 FROM performance.department_tasks t WHERE t.kpi_indicator_id = k.id);

-- name: DeactivateIndicator :exec
UPDATE performance.kpi_indicators SET is_active = false, updated_by = $2, updated_at = now() WHERE id = $1;

-- name: ListDepartmentWeights :many
SELECT di.department_id, di.indicator_id, i.code, di.weight, di.updated_by, di.updated_at
FROM performance.kpi_department_indicators di
JOIN performance.kpi_indicators i ON i.id = di.indicator_id
WHERE i.is_active
ORDER BY di.department_id, di.weight DESC;

-- name: DeleteDepartmentWeights :exec
DELETE FROM performance.kpi_department_indicators WHERE department_id = $1;

-- name: InsertDepartmentWeight :exec
INSERT INTO performance.kpi_department_indicators (department_id, indicator_id, weight, updated_by)
VALUES ($1, $2, $3, $4);

-- name: ListTargets :many
SELECT * FROM performance.kpi_targets ORDER BY created_at DESC LIMIT 500;

-- name: InsertTarget :exec
INSERT INTO performance.kpi_targets (id, indicator_id, period_year, period_month, department_id, employee_id, target, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateTarget :execrows
UPDATE performance.kpi_targets SET indicator_id = $2, period_year = $3, period_month = $4, department_id = $5, employee_id = $6, target = $7
WHERE id = $1;

-- name: DeleteTarget :execrows
DELETE FROM performance.kpi_targets WHERE id = $1;

-- name: UpsertSnapshot :exec
INSERT INTO performance.kpi_snapshots (id, employee_id, indicator_id, period_year, period_month, actual, target, attainment, sample_size, source_detail)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (employee_id, indicator_id, period_year, period_month) DO UPDATE SET
  actual = EXCLUDED.actual, target = EXCLUDED.target, attainment = EXCLUDED.attainment,
  sample_size = EXCLUDED.sample_size, source_detail = EXCLUDED.source_detail, updated_at = now();

-- name: ListSnapshots :many
SELECT s.*, i.code AS indicator_code, i.name AS indicator_name, i.source_kind
FROM performance.kpi_snapshots s
JOIN performance.kpi_indicators i ON i.id = s.indicator_id
WHERE s.employee_id = $1 AND s.period_year = $2 AND s.period_month = $3
ORDER BY i.name;

-- name: ListManualSnapshots :many
SELECT s.*, i.code AS indicator_code, i.name AS indicator_name, i.source_kind
FROM performance.kpi_snapshots s
JOIN performance.kpi_indicators i ON i.id = s.indicator_id
WHERE s.period_year = $1 AND s.period_month = $2 AND i.source_kind = 'manual';

-- name: UpsertDraftScorecard :execrows
INSERT INTO performance.kpi_scorecards (id, employee_id, department_id, period_year, period_month, score, raw_score, used_weight, breakdown, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'draft')
ON CONFLICT (employee_id, period_year, period_month) DO UPDATE SET
  department_id = EXCLUDED.department_id, score = EXCLUDED.score, raw_score = EXCLUDED.raw_score,
  used_weight = EXCLUDED.used_weight, breakdown = EXCLUDED.breakdown, updated_at = now()
WHERE performance.kpi_scorecards.status = 'draft';

-- name: GetScorecard :one
SELECT * FROM performance.kpi_scorecards WHERE id = $1;

-- name: GetScorecardFor :one
SELECT * FROM performance.kpi_scorecards WHERE employee_id = $1 AND period_year = $2 AND period_month = $3;

-- name: ListScorecards :many
SELECT * FROM performance.kpi_scorecards
WHERE period_year = $1 AND period_month = $2
  AND (sqlc.narg(employee_ids)::text[] IS NULL OR employee_id = ANY(sqlc.narg(employee_ids)::text[]))
ORDER BY score DESC NULLS LAST
LIMIT 500;

-- name: ScorecardHistory :many
SELECT * FROM performance.kpi_scorecards WHERE employee_id = $1
ORDER BY period_year DESC, period_month DESC LIMIT $2;

-- name: QuarterScorecards :many
SELECT * FROM performance.kpi_scorecards
WHERE employee_id = ANY(sqlc.arg(employee_ids)::text[]) AND period_year = sqlc.arg(period_year)
  AND period_month = ANY(sqlc.arg(months)::int[])
ORDER BY employee_id, period_month;

-- name: SetScorecardStatus :exec
UPDATE performance.kpi_scorecards SET status = $2, reviewed_by = $3, reviewed_at = $4, updated_at = now() WHERE id = $1;

-- name: FinalScorecardEmployees :many
SELECT employee_id FROM performance.kpi_scorecards
WHERE period_year = $1 AND period_month = $2 AND status = 'final';

-- name: RecentScoreAverages :many
WITH ranked AS (
  SELECT employee_id, score, status,
         ROW_NUMBER() OVER (PARTITION BY employee_id ORDER BY period_year DESC, period_month DESC) AS rn
  FROM performance.kpi_scorecards
  WHERE employee_id = ANY(sqlc.arg(employee_ids)::text[]) AND score IS NOT NULL)
SELECT employee_id, AVG(score)::float8 AS avg_score, COUNT(*)::int AS periods, BOOL_OR(status = 'final')::boolean AS has_final
FROM ranked WHERE rn <= 3 GROUP BY employee_id;

-- name: TaskCounts :many
SELECT t.department_id, t.assignee_employee_id, o.done_by, o.status, count(*)::int AS total
FROM performance.department_task_occurrences o
JOIN performance.department_tasks t ON t.id = o.task_id
WHERE o.occurrence_date BETWEEN sqlc.arg(start_date)::date AND sqlc.arg(end_date)::date
GROUP BY 1, 2, 3, 4;

-- name: LogbookStats :many
SELECT e.employee_id, count(DISTINCT e.id)::int AS entries,
       count(i.id) FILTER (WHERE i.is_checked)::int AS checked,
       count(i.id)::int AS total
FROM performance.logbook_entries e
LEFT JOIN performance.logbook_entry_items i ON i.entry_id = e.id
WHERE e.entry_date BETWEEN sqlc.arg(start_date)::date AND sqlc.arg(end_date)::date
GROUP BY e.employee_id;
