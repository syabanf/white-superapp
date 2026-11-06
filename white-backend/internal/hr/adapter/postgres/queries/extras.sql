-- Assets, training and timesheet.

-- name: ListAssets :many
SELECT sqlc.embed(a), COALESCE(e.full_name, '')::text AS employee_name, COALESCE(e.employee_no, '')::text AS employee_no
FROM hr.assets a
LEFT JOIN hr.employees e ON e.id = a.employee_id
WHERE (sqlc.arg(employee_id)::text = '' OR a.employee_id = sqlc.arg(employee_id)::text)
  AND (sqlc.arg(status)::text = '' OR a.status = sqlc.arg(status)::text)
  AND (sqlc.arg(search)::text = '' OR a.name ILIKE '%' || sqlc.arg(search)::text || '%' OR a.asset_code ILIKE '%' || sqlc.arg(search)::text || '%'
       OR a.serial_no ILIKE '%' || sqlc.arg(search)::text || '%')
ORDER BY a.asset_code;

-- name: GetAsset :one
SELECT sqlc.embed(a), COALESCE(e.full_name, '')::text AS employee_name, COALESCE(e.employee_no, '')::text AS employee_no
FROM hr.assets a
LEFT JOIN hr.employees e ON e.id = a.employee_id
WHERE a.id = $1;

-- name: UpsertAsset :exec
INSERT INTO hr.assets (id, asset_code, name, category, serial_no, purchase_date, purchase_value, status, employee_id, notes, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE SET asset_code = EXCLUDED.asset_code, name = EXCLUDED.name, category = EXCLUDED.category,
    serial_no = EXCLUDED.serial_no, purchase_date = EXCLUDED.purchase_date, purchase_value = EXCLUDED.purchase_value,
    status = EXCLUDED.status, employee_id = EXCLUDED.employee_id, notes = EXCLUDED.notes, updated_at = now();

-- name: InsertAssetAssignment :exec
INSERT INTO hr.asset_assignments (id, asset_id, employee_id, assigned_on, condition_out, created_by)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: CloseAssetAssignment :execrows
UPDATE hr.asset_assignments SET returned_on = $2, condition_in = $3
WHERE asset_id = $1 AND returned_on IS NULL;

-- name: ListAssetAssignments :many
SELECT sqlc.embed(x), e.full_name, e.employee_no
FROM hr.asset_assignments x
JOIN hr.employees e ON e.id = x.employee_id
WHERE x.asset_id = $1
ORDER BY x.assigned_on DESC, x.created_at DESC;

-- name: ListTrainings :many
SELECT sqlc.embed(t),
       (SELECT count(*) FROM hr.training_participants p WHERE p.training_id = t.id)::int AS participant_count
FROM hr.trainings t
WHERE (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status)::text)
  AND (sqlc.arg(year)::int = 0 OR extract(year FROM t.start_date)::int = sqlc.arg(year)::int)
  AND (sqlc.arg(employee_id)::text = '' OR EXISTS (SELECT 1 FROM hr.training_participants p WHERE p.training_id = t.id AND p.employee_id = sqlc.arg(employee_id)::text))
ORDER BY t.start_date DESC, t.title;

-- name: GetTraining :one
SELECT * FROM hr.trainings WHERE id = $1;

-- name: UpsertTraining :exec
INSERT INTO hr.trainings (id, title, provider, start_date, end_date, location, cost, status, notes, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, provider = EXCLUDED.provider, start_date = EXCLUDED.start_date,
    end_date = EXCLUDED.end_date, location = EXCLUDED.location, cost = EXCLUDED.cost, status = EXCLUDED.status,
    notes = EXCLUDED.notes, updated_at = now();

-- name: DeleteTraining :execrows
DELETE FROM hr.trainings WHERE id = $1;

-- name: ListTrainingParticipants :many
SELECT sqlc.embed(p), e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name
FROM hr.training_participants p
JOIN hr.employees e ON e.id = p.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE p.training_id = $1
ORDER BY e.full_name;

-- name: DeleteTrainingParticipantsExcept :exec
DELETE FROM hr.training_participants WHERE training_id = $1 AND NOT (employee_id = ANY(sqlc.arg(keep)::text[]));

-- name: AddTrainingParticipant :exec
INSERT INTO hr.training_participants (training_id, employee_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UpdateTrainingParticipant :execrows
UPDATE hr.training_participants SET status = $3, score = $4, certificate = $5
WHERE training_id = $1 AND employee_id = $2;

-- name: ListTimesheet :many
SELECT sqlc.embed(t), e.full_name, e.employee_no
FROM hr.timesheet_entries t
JOIN hr.employees e ON e.id = t.employee_id
WHERE (sqlc.arg(employee_id)::text = '' OR t.employee_id = sqlc.arg(employee_id)::text)
  AND (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status)::text)
  AND t.work_date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
ORDER BY t.work_date DESC, e.full_name, t.created_at;

-- name: GetTimesheetEntry :one
SELECT sqlc.embed(t), e.full_name, e.employee_no
FROM hr.timesheet_entries t
JOIN hr.employees e ON e.id = t.employee_id
WHERE t.id = $1;

-- name: SumTimesheetDay :one
SELECT COALESCE(sum(hours), 0)::numeric FROM hr.timesheet_entries
WHERE employee_id = $1 AND work_date = $2 AND id <> $3 AND status <> 'rejected';

-- name: UpsertTimesheetEntry :exec
INSERT INTO hr.timesheet_entries (id, employee_id, work_date, project_id, project_name, hours, description, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'draft')
ON CONFLICT (id) DO UPDATE SET work_date = EXCLUDED.work_date, project_id = EXCLUDED.project_id,
    project_name = EXCLUDED.project_name, hours = EXCLUDED.hours, description = EXCLUDED.description,
    status = 'draft', reject_reason = '', updated_at = now();

-- name: DeleteTimesheetEntry :execrows
DELETE FROM hr.timesheet_entries WHERE id = $1 AND status IN ('draft', 'rejected');

-- name: SubmitTimesheet :execrows
UPDATE hr.timesheet_entries SET status = 'submitted', updated_at = now()
WHERE employee_id = $1 AND status IN ('draft', 'rejected') AND work_date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date;

-- name: DecideTimesheetEntry :execrows
UPDATE hr.timesheet_entries SET status = $2, reject_reason = $3, decided_by = $4, decided_at = now(), updated_at = now()
WHERE id = $1 AND status = 'submitted';

-- name: TimesheetHours :many
-- Approved hours per employee and project in [from, to]; project_id '' is internal work.
SELECT employee_id, project_id, max(project_name)::text AS project_name, sum(hours)::numeric AS hours
FROM hr.timesheet_entries
WHERE status = 'approved' AND work_date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
GROUP BY employee_id, project_id;
