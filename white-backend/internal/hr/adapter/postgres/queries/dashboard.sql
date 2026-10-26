-- HR dashboard aggregates. Every query takes the same employee filter:
-- department_id, employment_status and position_id (empty = any).

-- name: DashAttendanceDaily :many
SELECT a.date, COALESCE(e.department_id, '')::text AS department_id,
       count(*)::int AS records,
       (count(*) FILTER (WHERE a.status <> 'absent'))::int AS present,
       (count(*) FILTER (WHERE a.is_late))::int AS late,
       (count(*) FILTER (WHERE a.status = 'absent'))::int AS absent,
       COALESCE(sum(a.late_minutes), 0)::int AS late_minutes,
       COALESCE(sum(a.work_hours) FILTER (WHERE a.status <> 'absent'), 0)::float8 AS work_hours
FROM hr.attendance a
JOIN hr.employees e ON e.id = a.employee_id
WHERE a.date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
  AND e.deleted_at IS NULL
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
GROUP BY a.date, e.department_id
ORDER BY a.date;

-- name: DashAttendanceByEmployee :many
SELECT e.id AS employee_id, e.full_name, COALESCE(d.name, '')::text AS department_name, COALESCE(p.title, '')::text AS position_name,
       count(*)::int AS records,
       (count(*) FILTER (WHERE a.is_late))::int AS late,
       COALESCE(sum(a.late_minutes), 0)::int AS late_minutes,
       (count(*) FILTER (WHERE a.status = 'absent'))::int AS absent
FROM hr.attendance a
JOIN hr.employees e ON e.id = a.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions p ON p.id = e.position_id
WHERE a.date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
  AND e.deleted_at IS NULL
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
GROUP BY e.id, e.full_name, d.name, p.title
ORDER BY late DESC, late_minutes DESC, e.full_name
LIMIT 10;

-- name: DashClockInOffsets :many
SELECT (floor(extract(epoch FROM (a.clock_in - a.scheduled_start)) / 300) * 5)::int AS bin, count(*)::int AS records
FROM hr.attendance a
JOIN hr.employees e ON e.id = a.employee_id
WHERE a.date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
  AND a.clock_in IS NOT NULL AND a.scheduled_start IS NOT NULL
  AND e.deleted_at IS NULL
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
GROUP BY 1
ORDER BY 1;

-- name: DashLeaves :many
SELECT l.employee_id, COALESCE(e.department_id, '')::text AS department_id, l.leave_type, l.start_date, l.end_date
FROM hr.leaves l
JOIN hr.employees e ON e.id = l.employee_id
WHERE l.status = 'approved'
  AND l.end_date >= sqlc.arg(date_from)::date AND l.start_date <= sqlc.arg(date_to)::date
  AND e.deleted_at IS NULL
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text);

-- name: DashPending :many
SELECT x.kind, x.id, x.employee_id, x.full_name, x.department_name, x.type, x.date, x.amount, x.status, x.created_at
FROM (
    SELECT 'leave'::text AS kind, l.id, e.id AS employee_id, e.full_name, COALESCE(d.name, '')::text AS department_name,
           l.leave_type AS type, l.start_date AS date, l.total_days::float8 AS amount, l.status, l.created_at,
           e.department_id, e.employment_status, e.position_id
    FROM hr.leaves l
    JOIN hr.employees e ON e.id = l.employee_id
    LEFT JOIN hr.departments d ON d.id = e.department_id
    WHERE l.status IN ('pending', 'manager_approved') AND e.deleted_at IS NULL
    UNION ALL
    SELECT 'overtime'::text, o.id, e.id, e.full_name, COALESCE(d.name, '')::text,
           ''::text, o.date, o.hours::float8, o.status, o.created_at,
           e.department_id, e.employment_status, e.position_id
    FROM hr.overtime_requests o
    JOIN hr.employees e ON e.id = o.employee_id
    LEFT JOIN hr.departments d ON d.id = e.department_id
    WHERE o.status = 'pending' AND e.deleted_at IS NULL
) x
WHERE (sqlc.arg(department_id)::text = '' OR x.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR x.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR x.position_id = sqlc.arg(position_id)::text)
ORDER BY x.created_at;

-- name: DashLeaveBalances :many
SELECT COALESCE(e.department_id, '')::text AS department_id,
       COALESCE(sum(b.annual_leave_total), 0)::int AS quota,
       COALESCE(sum(b.annual_leave_used), 0)::int AS used
FROM hr.leave_balances b
JOIN hr.employees e ON e.id = b.employee_id
WHERE b.year = sqlc.arg(year)::int AND e.is_active AND e.deleted_at IS NULL
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
GROUP BY e.department_id;

-- Approved overtime with its estimated cost at the current base salary.
-- name: DashOvertime :many
SELECT o.date, e.id AS employee_id, e.full_name, COALESCE(e.department_id, '')::text AS department_id,
       COALESCE(d.name, '')::text AS department_name, COALESCE(p.title, '')::text AS position_name,
       o.hours::float8 AS hours,
       COALESCE(o.hours * s.base_salary / NULLIF(ps.overtime_hourly_divisor, 0) * ps.overtime_multiplier, 0)::float8 AS cost
FROM hr.overtime_requests o
JOIN hr.employees e ON e.id = o.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions p ON p.id = e.position_id
LEFT JOIN hr.employee_salary s ON s.employee_id = e.id AND s.is_active
CROSS JOIN hr.payroll_settings ps
WHERE o.status = 'approved'
  AND o.date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
  AND e.deleted_at IS NULL
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text);

-- name: DashEmployees :many
SELECT e.id, e.full_name, COALESCE(e.department_id, '')::text AS department_id, COALESCE(d.name, '')::text AS department_name,
       COALESCE(e.position_id, '')::text AS position_id, COALESCE(p.title, '')::text AS position_name,
       COALESCE(p.level, '')::text AS level, e.employment_status, COALESCE(es.name, e.employment_status)::text AS status_name,
       e.gender, e.birth_date, e.join_date, e.end_date, e.probation_end_date, e.is_active,
       COALESCE((SELECT c.contract_type FROM hr.employment_contracts c
                 WHERE c.employee_id = e.id AND c.status <> 'draft'
                 ORDER BY c.start_date DESC LIMIT 1), '')::text AS contract_type
FROM hr.employees e
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions p ON p.id = e.position_id
LEFT JOIN hr.employment_statuses es ON es.code = e.employment_status
WHERE e.deleted_at IS NULL
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
ORDER BY e.full_name;

-- Active contracts ending in [date_from, date_to].
-- name: DashContractsEnding :many
SELECT c.id, c.contract_number, c.contract_type, c.end_date, e.id AS employee_id, e.full_name,
       COALESCE(d.name, '')::text AS department_name
FROM hr.employment_contracts c
JOIN hr.employees e ON e.id = c.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE c.status = 'active' AND c.end_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
  AND e.is_active AND e.deleted_at IS NULL
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
ORDER BY c.end_date;

-- name: DashPayroll :many
SELECT make_date(r.period_year, r.period_month, 1) AS month, COALESCE(e.department_id, '')::text AS department_id,
       count(*)::int AS employees,
       COALESCE(sum(pd.gross_salary), 0)::float8 AS gross,
       COALESCE(sum(pd.net_salary), 0)::float8 AS net,
       COALESCE(sum(pd.pph21_deduction), 0)::float8 AS pph21,
       COALESCE(sum(pd.bpjs_tk_jht_deduction + pd.bpjs_tk_jp_deduction + pd.bpjs_kes_deduction + pd.tapera_deduction), 0)::float8 AS bpjs_employee,
       COALESCE(sum(pd.bpjs_tk_jht_employer + pd.bpjs_tk_jp_employer + pd.bpjs_tk_jkk_employer + pd.bpjs_tk_jkm_employer
           + pd.bpjs_kes_employer + pd.tapera_employer), 0)::float8 AS bpjs_employer
FROM hr.payroll_details pd
JOIN hr.payroll_runs r ON r.id = pd.payroll_run_id
JOIN hr.employees e ON e.id = pd.employee_id
WHERE r.status IN ('approved', 'paid')
  AND make_date(r.period_year, r.period_month, 1) BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
GROUP BY 1, 2
ORDER BY 1;

-- name: DashPayrollByLevel :many
SELECT COALESCE(p.level, '')::text AS level,
       count(DISTINCT pd.employee_id)::int AS employees,
       COALESCE(avg(pd.full_base_salary), 0)::float8 AS avg_base,
       COALESCE(avg(pd.gross_salary), 0)::float8 AS avg_gross,
       COALESCE(min(pd.gross_salary), 0)::float8 AS min_gross,
       COALESCE(max(pd.gross_salary), 0)::float8 AS max_gross
FROM hr.payroll_details pd
JOIN hr.payroll_runs r ON r.id = pd.payroll_run_id
JOIN hr.employees e ON e.id = pd.employee_id
LEFT JOIN hr.positions p ON p.id = e.position_id
WHERE r.status IN ('approved', 'paid')
  AND make_date(r.period_year, r.period_month, 1) BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
GROUP BY 1
ORDER BY avg_gross DESC;

-- name: DashLoans :many
SELECT l.id, l.loan_type, l.principal_amount::float8 AS principal_amount, l.remaining_balance::float8 AS remaining_balance,
       l.monthly_installment::float8 AS monthly_installment, e.id AS employee_id, e.full_name,
       COALESCE(d.name, '')::text AS department_name
FROM hr.loans l
JOIN hr.employees e ON e.id = l.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE l.status = 'approved' AND l.is_active AND l.remaining_balance > 0 AND e.deleted_at IS NULL
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
ORDER BY l.remaining_balance DESC;
