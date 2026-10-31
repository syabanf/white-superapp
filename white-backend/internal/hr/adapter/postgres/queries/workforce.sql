-- name: ListShifts :many
SELECT id, name, start_time::text AS start_time, end_time::text AS end_time, break_minutes, late_tolerance_minutes,
       is_overnight, is_active, sort_order
FROM hr.shifts
WHERE sqlc.arg(include_inactive)::boolean OR is_active
ORDER BY sort_order, name;

-- name: GetShift :one
SELECT id, name, start_time::text AS start_time, end_time::text AS end_time, break_minutes, late_tolerance_minutes,
       is_overnight, is_active, sort_order
FROM hr.shifts WHERE id = $1;

-- name: InsertShift :exec
INSERT INTO hr.shifts (id, name, start_time, end_time, break_minutes, late_tolerance_minutes, is_overnight, is_active, sort_order, created_by, updated_by)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(start_time)::text::time, sqlc.arg(end_time)::text::time, sqlc.arg(break_minutes),
        sqlc.arg(late_tolerance_minutes), sqlc.arg(is_overnight), sqlc.arg(is_active), sqlc.arg(sort_order), sqlc.arg(actor), sqlc.arg(actor));

-- name: UpdateShift :execrows
UPDATE hr.shifts SET name = sqlc.arg(name), start_time = sqlc.arg(start_time)::text::time, end_time = sqlc.arg(end_time)::text::time,
    break_minutes = sqlc.arg(break_minutes), late_tolerance_minutes = sqlc.arg(late_tolerance_minutes),
    is_overnight = sqlc.arg(is_overnight), is_active = sqlc.arg(is_active), sort_order = sqlc.arg(sort_order),
    updated_by = sqlc.arg(actor), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: ShiftInUse :one
SELECT (EXISTS (SELECT 1 FROM hr.employee_shifts es WHERE es.shift_id = sqlc.arg(id)::text)
    OR EXISTS (SELECT 1 FROM hr.attendance a WHERE a.shift_id = sqlc.arg(id)::text))::boolean AS used;

-- name: DeactivateShift :exec
UPDATE hr.shifts SET is_active = false, updated_at = now() WHERE id = $1;

-- name: DeleteShift :execrows
DELETE FROM hr.shifts WHERE id = $1;

-- name: ListScheduleRows :many
SELECT es.id, es.employee_id, es.day_of_week, es.shift_id, es.effective_from, es.effective_to,
       COALESCE(s.name, '')::text AS shift_name, COALESCE(s.start_time::text, '')::text AS start_time,
       COALESCE(s.end_time::text, '')::text AS end_time, COALESCE(s.late_tolerance_minutes, 0)::int AS late_tolerance_minutes,
       COALESCE(s.is_overnight, false)::boolean AS is_overnight, COALESCE(s.break_minutes, 60)::int AS break_minutes
FROM hr.employee_shifts es
LEFT JOIN hr.shifts s ON s.id = es.shift_id
WHERE es.employee_id = ANY (sqlc.arg(employee_ids)::text[])
ORDER BY es.employee_id, es.effective_from, es.day_of_week;

-- name: CloseSchedule :exec
UPDATE hr.employee_shifts SET effective_to = sqlc.arg(effective_to)
WHERE employee_id = sqlc.arg(employee_id) AND effective_from < sqlc.arg(effective_from)
  AND (effective_to IS NULL OR effective_to >= sqlc.arg(effective_from));

-- name: DeleteScheduleFrom :exec
DELETE FROM hr.employee_shifts WHERE employee_id = $1 AND effective_from >= $2;

-- name: InsertScheduleRow :exec
INSERT INTO hr.employee_shifts (id, employee_id, day_of_week, shift_id, effective_from, created_by)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListHolidays :many
SELECT * FROM hr.public_holidays
WHERE holiday_date BETWEEN sqlc.arg(date_from) AND sqlc.arg(date_to)
  AND (NOT sqlc.arg(only_active)::boolean OR status = 'aktif')
ORDER BY holiday_date, name;

-- name: InsertHoliday :one
INSERT INTO hr.public_holidays (id, holiday_date, name, type, deducts_leave, status, note, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
RETURNING *;

-- name: UpdateHoliday :one
UPDATE hr.public_holidays SET holiday_date = $2, name = $3, type = $4, deducts_leave = $5, status = $6, note = $7,
    updated_by = $8, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteHoliday :execrows
DELETE FROM hr.public_holidays WHERE id = $1;

-- name: ListAttendance :many
SELECT sqlc.embed(a), e.full_name, e.employee_no,
       COALESCE(d.name, '')::text AS department_name, COALESCE(p.title, '')::text AS position_name,
       COALESCE(s.name, '')::text AS shift_name, COALESCE(v.full_name, '')::text AS validated_by_name,
       count(*) OVER () AS total
FROM hr.attendance a
JOIN hr.employees e ON e.id = a.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions p ON p.id = e.position_id
LEFT JOIN hr.shifts s ON s.id = a.shift_id
LEFT JOIN hr.employees v ON v.id = a.validated_by
WHERE (cardinality(sqlc.arg(employee_ids)::text[]) = 0 OR a.employee_id = ANY (sqlc.arg(employee_ids)::text[]))
  AND (sqlc.narg(date_from)::date IS NULL OR a.date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR a.date <= sqlc.narg(date_to)::date)
  AND (sqlc.arg(status)::text = '' OR a.status::text = sqlc.arg(status)::text)
  AND (NOT sqlc.arg(only_late)::boolean OR a.is_late)
  AND (NOT sqlc.arg(only_unvalidated)::boolean OR a.validated_at IS NULL)
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
ORDER BY a.date DESC, e.full_name
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: GetAttendance :one
SELECT sqlc.embed(a), e.full_name, e.employee_no,
       COALESCE(d.name, '')::text AS department_name, COALESCE(p.title, '')::text AS position_name,
       COALESCE(s.name, '')::text AS shift_name, COALESCE(v.full_name, '')::text AS validated_by_name
FROM hr.attendance a
JOIN hr.employees e ON e.id = a.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions p ON p.id = e.position_id
LEFT JOIN hr.shifts s ON s.id = a.shift_id
LEFT JOIN hr.employees v ON v.id = a.validated_by
WHERE a.id = $1;

-- name: AttendanceIDOn :one
SELECT id FROM hr.attendance WHERE employee_id = $1 AND date = $2;

-- name: InsertAttendance :exec
INSERT INTO hr.attendance (id, employee_id, date, clock_in, clock_in_location, clock_in_photo_key, status, is_late,
    late_minutes, shift_id, scheduled_start, scheduled_end, break_minutes, notes, created_by, clock_out, work_hours,
    overtime_hours)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18);

-- name: UpdateAttendance :execrows
UPDATE hr.attendance SET clock_in = $2, clock_out = $3, status = $4, is_late = $5, late_minutes = $6, work_hours = $7,
    overtime_hours = $8, shift_id = $9, scheduled_start = $10, scheduled_end = $11, break_minutes = $12, notes = $13,
    updated_at = now()
WHERE id = $1;

-- name: DeleteAttendance :execrows
DELETE FROM hr.attendance WHERE id = $1;

-- name: InsertAttendanceRow :exec
INSERT INTO hr.attendance (id, employee_id, date, clock_in, clock_out, status, is_late, late_minutes, work_hours,
    overtime_hours, shift_id, scheduled_start, scheduled_end, validated_by, validated_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (employee_id, date) DO NOTHING;

-- name: ClockOut :execrows
UPDATE hr.attendance SET clock_out = $2, clock_out_location = $3, clock_out_photo_key = $4, work_hours = $5,
    notes = $6, overtime_hours = $7, updated_at = now()
WHERE id = $1 AND clock_out IS NULL;

-- name: ValidateAttendance :execrows
UPDATE hr.attendance SET status = $2, validated_by = $3, validated_at = now(), validation_notes = $4, updated_at = now()
WHERE id = $1;

-- name: AttendanceSummary :many
SELECT a.employee_id,
       count(*) FILTER (WHERE a.status <> 'absent')::int AS present_days,
       count(*) FILTER (WHERE a.is_late)::int AS late_days,
       count(*) FILTER (WHERE a.status = 'absent')::int AS absent_days,
       COALESCE(sum(a.late_minutes), 0)::int AS late_minutes
FROM hr.attendance a
WHERE a.employee_id = ANY (sqlc.arg(employee_ids)::text[]) AND a.date BETWEEN sqlc.arg(date_from) AND sqlc.arg(date_to)
GROUP BY a.employee_id;

-- name: ApprovedOvertimeHours :many
SELECT employee_id, COALESCE(sum(hours), 0)::float8 AS hours
FROM hr.overtime_requests
WHERE status = 'approved' AND employee_id = ANY (sqlc.arg(employee_ids)::text[]) AND date BETWEEN sqlc.arg(date_from) AND sqlc.arg(date_to)
GROUP BY employee_id;

-- name: GetLeaveBalance :one
SELECT * FROM hr.leave_balances WHERE employee_id = $1 AND year = $2;

-- name: InsertLeaveBalance :one
INSERT INTO hr.leave_balances (id, employee_id, year, annual_leave_total, annual_leave_used, updated_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (employee_id, year) DO UPDATE SET employee_id = EXCLUDED.employee_id
RETURNING *;

-- name: SetLeaveBalance :one
UPDATE hr.leave_balances SET annual_leave_total = $3, annual_leave_used = $4, updated_by = $5, updated_at = now()
WHERE employee_id = $1 AND year = $2
RETURNING *;

-- name: AddLeaveUsage :exec
UPDATE hr.leave_balances SET
    annual_leave_used = annual_leave_used + sqlc.arg(annual)::int,
    sick_leave_used = sick_leave_used + sqlc.arg(sick)::int,
    unpaid_leave_used = unpaid_leave_used + sqlc.arg(unpaid)::int,
    other_leave_used = other_leave_used + sqlc.arg(other)::int,
    updated_at = now()
WHERE employee_id = sqlc.arg(employee_id) AND year = sqlc.arg(year);

-- name: PendingLeaveDays :one
SELECT COALESCE(sum(total_days), 0)::int FROM hr.leaves
WHERE employee_id = $1 AND status IN ('pending', 'manager_approved') AND leave_type = 'annual'
  AND extract(year FROM start_date)::int = sqlc.arg(year)::int;

-- name: ListLeaves :many
SELECT sqlc.embed(l), e.full_name, e.employee_no, e.manager_id AS employee_manager_id,
       COALESCE(d.name, '')::text AS department_name,
       COALESCE(ma.full_name, '')::text AS manager_approved_by_name,
       COALESCE(ap.full_name, '')::text AS approved_by_name,
       count(*) OVER () AS total
FROM hr.leaves l
JOIN hr.employees e ON e.id = l.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.employees ma ON ma.id = l.manager_approved_by
LEFT JOIN hr.employees ap ON ap.id = l.approved_by
WHERE (cardinality(sqlc.arg(employee_ids)::text[]) = 0 OR l.employee_id = ANY (sqlc.arg(employee_ids)::text[]))
  AND (cardinality(sqlc.arg(statuses)::text[]) = 0 OR l.status = ANY (sqlc.arg(statuses)::text[]))
  AND (sqlc.arg(leave_type)::text = '' OR l.leave_type = sqlc.arg(leave_type)::text)
  AND (sqlc.narg(date_from)::date IS NULL OR l.end_date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR l.start_date <= sqlc.narg(date_to)::date)
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
ORDER BY l.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: GetLeave :one
SELECT sqlc.embed(l), e.full_name, e.employee_no, e.manager_id AS employee_manager_id,
       COALESCE(d.name, '')::text AS department_name,
       COALESCE(ma.full_name, '')::text AS manager_approved_by_name,
       COALESCE(ap.full_name, '')::text AS approved_by_name
FROM hr.leaves l
JOIN hr.employees e ON e.id = l.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.employees ma ON ma.id = l.manager_approved_by
LEFT JOIN hr.employees ap ON ap.id = l.approved_by
WHERE l.id = $1;

-- name: InsertLeave :exec
INSERT INTO hr.leaves (id, employee_id, leave_type, start_date, end_date, total_days, reason, status,
    manager_approved_by, manager_approved_at, approved_by, approved_at, created_by, updated_by, created_at)
VALUES (sqlc.arg(id), sqlc.arg(employee_id), sqlc.arg(leave_type), sqlc.arg(start_date), sqlc.arg(end_date),
        sqlc.arg(total_days), sqlc.arg(reason), sqlc.arg(status), sqlc.narg(manager_approved_by), sqlc.narg(manager_approved_at),
        sqlc.narg(approved_by), sqlc.narg(approved_at), sqlc.narg(created_by), sqlc.narg(created_by),
        COALESCE(sqlc.narg(created_at)::timestamptz, now()));

-- name: OverlappingLeave :one
SELECT EXISTS (SELECT 1 FROM hr.leaves WHERE employee_id = $1 AND status IN ('pending', 'manager_approved', 'approved')
    AND start_date <= sqlc.arg(end_date) AND end_date >= sqlc.arg(start_date));

-- name: SetLeaveStatus :execrows
UPDATE hr.leaves SET status = sqlc.arg(status),
    manager_approved_by = COALESCE(sqlc.narg(manager_approved_by), manager_approved_by),
    manager_approved_at = CASE WHEN sqlc.narg(manager_approved_by)::text IS NULL THEN manager_approved_at ELSE now() END,
    approved_by = COALESCE(sqlc.narg(approved_by), approved_by),
    approved_at = CASE WHEN sqlc.narg(approved_by)::text IS NULL THEN approved_at ELSE now() END,
    rejected_by = COALESCE(sqlc.narg(rejected_by), rejected_by),
    rejection_reason = sqlc.arg(rejection_reason), updated_by = sqlc.arg(actor), updated_at = now()
WHERE id = sqlc.arg(id) AND status = sqlc.arg(from_status);

-- name: ApprovedLeavesOn :many
SELECT employee_id, leave_type FROM hr.leaves
WHERE status = 'approved' AND start_date <= sqlc.arg(day) AND end_date >= sqlc.arg(day);

-- name: ListOvertime :many
SELECT sqlc.embed(o), o.start_time::text AS start_hhmm, o.end_time::text AS end_hhmm,
       e.full_name, e.employee_no, e.manager_id AS employee_manager_id,
       COALESCE(d.name, '')::text AS department_name,
       COALESCE(rq.full_name, '')::text AS requested_by_name,
       COALESCE(dc.full_name, '')::text AS decided_by_name
FROM hr.overtime_requests o
JOIN hr.employees e ON e.id = o.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.employees rq ON rq.id = o.requested_by
LEFT JOIN hr.employees dc ON dc.id = o.decided_by
WHERE (cardinality(sqlc.arg(employee_ids)::text[]) = 0 OR o.employee_id = ANY (sqlc.arg(employee_ids)::text[]))
  AND (sqlc.arg(source)::text = '' OR o.source = sqlc.arg(source)::text)
  AND (sqlc.arg(status)::text = '' OR o.status = sqlc.arg(status)::text)
  AND (sqlc.narg(date_from)::date IS NULL OR o.date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR o.date <= sqlc.narg(date_to)::date)
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
ORDER BY o.date DESC, o.created_at DESC
LIMIT 500;

-- name: GetOvertime :one
SELECT sqlc.embed(o), o.start_time::text AS start_hhmm, o.end_time::text AS end_hhmm,
       e.full_name, e.employee_no, e.manager_id AS employee_manager_id,
       COALESCE(d.name, '')::text AS department_name,
       COALESCE(rq.full_name, '')::text AS requested_by_name,
       COALESCE(dc.full_name, '')::text AS decided_by_name
FROM hr.overtime_requests o
JOIN hr.employees e ON e.id = o.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.employees rq ON rq.id = o.requested_by
LEFT JOIN hr.employees dc ON dc.id = o.decided_by
WHERE o.id = $1;

-- name: InsertOvertime :exec
INSERT INTO hr.overtime_requests (id, employee_id, date, start_time, end_time, hours, source, status, reason, requested_by,
    decided_by, decided_at, created_by, created_at)
VALUES (sqlc.arg(id), sqlc.arg(employee_id), sqlc.arg(date), sqlc.arg(start_time)::text::time, sqlc.arg(end_time)::text::time,
        sqlc.arg(hours), sqlc.arg(source), sqlc.arg(status), sqlc.arg(reason), sqlc.narg(requested_by),
        sqlc.narg(decided_by), sqlc.narg(decided_at), sqlc.narg(created_by), COALESCE(sqlc.narg(created_at)::timestamptz, now()));

-- name: UpdateOvertime :execrows
UPDATE hr.overtime_requests SET date = sqlc.arg(date), start_time = sqlc.arg(start_time)::text::time,
    end_time = sqlc.arg(end_time)::text::time, hours = sqlc.arg(hours), reason = sqlc.arg(reason), updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: DeleteOvertime :execrows
DELETE FROM hr.overtime_requests WHERE id = $1 AND status <> 'approved';

-- name: DecideOvertime :execrows
UPDATE hr.overtime_requests SET status = $2, decided_by = $3, decided_at = now(), rejection_reason = $4, updated_at = now()
WHERE id = $1 AND status = 'pending';

-- name: OvertimeForPeriod :many
SELECT employee_id, date, hours FROM hr.overtime_requests
WHERE status = 'approved' AND employee_id = $1 AND date BETWEEN sqlc.arg(date_from) AND sqlc.arg(date_to);
