-- name: GetPayrollSettings :one
SELECT * FROM hr.payroll_settings WHERE id = 1;

-- name: UpdatePayrollSettings :one
UPDATE hr.payroll_settings SET
    bpjs_tk_jht_employee = $1, bpjs_tk_jht_employer = $2, bpjs_tk_jp_employee = $3, bpjs_tk_jp_employer = $4,
    bpjs_tk_jkk = $5, bpjs_tk_jkm = $6, bpjs_kes_employee = $7, bpjs_kes_employer = $8, bpjs_kes_max_upah = $9,
    tapera_employee = $10, tapera_employer = $11, ptkp_tk_0 = $12, ptkp_tk_1 = $13, ptkp_tk_2 = $14, ptkp_tk_3 = $15,
    ptkp_k_0 = $16, ptkp_k_1 = $17, ptkp_k_2 = $18, ptkp_k_3 = $19, pph21_bracket_1 = $20, pph21_bracket_2 = $21,
    pph21_bracket_3 = $22, pph21_bracket_4 = $23, overtime_multiplier = $24, overtime_multiplier_holiday = $25,
    overtime_hourly_divisor = $26, late_deduction_mode = $27, late_deduction_amount = $28,
    loan_max_installment_percent = $29, loan_max_active_per_employee = $30, payroll_day = $31,
    updated_by = $32, updated_at = now()
WHERE id = 1
RETURNING *;

-- name: ListSalaries :many
SELECT sqlc.embed(s), e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name
FROM hr.employee_salary s
JOIN hr.employees e ON e.id = s.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE (sqlc.arg(employee_id)::text = '' OR s.employee_id = sqlc.arg(employee_id)::text)
  AND (NOT sqlc.arg(active_only)::boolean OR s.is_active)
ORDER BY e.full_name, s.effective_date DESC;

-- name: GetSalary :one
SELECT sqlc.embed(s), e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name
FROM hr.employee_salary s
JOIN hr.employees e ON e.id = s.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE s.id = $1;

-- name: ActiveSalary :one
SELECT * FROM hr.employee_salary WHERE employee_id = $1 AND is_active ORDER BY effective_date DESC LIMIT 1;

-- name: EndActiveSalary :exec
UPDATE hr.employee_salary SET is_active = false, end_date = $2, updated_by = $3, updated_at = now()
WHERE employee_id = $1 AND is_active;

-- name: InsertSalary :exec
INSERT INTO hr.employee_salary (id, employee_id, base_salary, fixed_allowance, variable_allowance, transport_allowance,
    meal_allowance, housing_allowance, ptkp_status, is_taxable, bpjs_tk_enrolled, bpjs_kes_enrolled, tapera_enrolled,
    effective_date, notes, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $16);

-- name: UpdateSalary :execrows
UPDATE hr.employee_salary SET base_salary = $2, fixed_allowance = $3, variable_allowance = $4, transport_allowance = $5,
    meal_allowance = $6, housing_allowance = $7, ptkp_status = $8, is_taxable = $9, bpjs_tk_enrolled = $10,
    bpjs_kes_enrolled = $11, tapera_enrolled = $12, effective_date = $13, notes = $14, updated_by = $15, updated_at = now()
WHERE id = $1;

-- name: DeactivateSalary :execrows
UPDATE hr.employee_salary SET is_active = false, end_date = $2, updated_by = $3, updated_at = now() WHERE id = $1 AND is_active;

-- name: ListPayrollRuns :many
SELECT sqlc.embed(r), COALESCE(a.full_name, '')::text AS approved_by_name
FROM hr.payroll_runs r
LEFT JOIN hr.employees a ON a.user_id = r.approved_by AND a.deleted_at IS NULL
WHERE (sqlc.arg(year)::int = 0 OR r.period_year = sqlc.arg(year)::int)
  AND (sqlc.arg(status)::text = '' OR r.status = sqlc.arg(status)::text)
ORDER BY r.period_year DESC, r.period_month DESC;

-- name: GetPayrollRun :one
SELECT sqlc.embed(r), COALESCE(a.full_name, '')::text AS approved_by_name
FROM hr.payroll_runs r
LEFT JOIN hr.employees a ON a.user_id = r.approved_by AND a.deleted_at IS NULL
WHERE r.id = $1;

-- name: GetPayrollRunByPeriod :one
SELECT * FROM hr.payroll_runs WHERE period_year = $1 AND period_month = $2;

-- name: InsertPayrollRun :exec
INSERT INTO hr.payroll_runs (id, run_name, period_month, period_year, created_by) VALUES ($1, $2, $3, $4, $5);

-- name: DeletePayrollDetails :exec
DELETE FROM hr.payroll_details WHERE payroll_run_id = $1;

-- name: InsertPayrollDetail :exec
INSERT INTO hr.payroll_details (id, payroll_run_id, employee_id, base_salary, full_base_salary, fixed_allowance,
    variable_allowance, transport_allowance, meal_allowance, housing_allowance, overtime_pay, thr, bonus, gross_salary,
    bpjs_tk_jht_deduction, bpjs_tk_jp_deduction, bpjs_kes_deduction, tapera_deduction, pph21_deduction,
    unpaid_leave_deduction, late_deduction, loan_deduction, loan_details, total_deductions, net_salary,
    bpjs_tk_jht_employer, bpjs_tk_jp_employer, bpjs_tk_jkk_employer, bpjs_tk_jkm_employer, bpjs_kes_employer,
    tapera_employer, taxable_income, ptkp_amount, pph21_annual, working_days, present_days, late_days,
    unpaid_leave_days, overtime_hours, prorate_factor)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24,
        $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40);

-- name: SetRunCalculated :exec
UPDATE hr.payroll_runs SET status = 'calculated', total_employees = $2, total_gross = $3, total_deductions = $4,
    total_net = $5, total_bpjs_employee = $6, total_bpjs_employer = $7, total_pph21 = $8, needs_recalculation = false,
    calculated_at = now(), calculated_by = $9, updated_at = now()
WHERE id = $1;

-- name: ApproveRun :execrows
UPDATE hr.payroll_runs SET status = 'approved', approved_at = now(), approved_by = $2, updated_at = now()
WHERE id = $1 AND status = 'calculated' AND NOT needs_recalculation;

-- name: MarkRunPaid :one
UPDATE hr.payroll_runs SET status = 'paid', paid_at = sqlc.arg(paid_at), paid_by = sqlc.arg(paid_by),
    notes = CASE WHEN sqlc.arg(notes)::text = '' THEN notes ELSE sqlc.arg(notes)::text END, updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'approved'
RETURNING *;

-- name: FlagRunRecalculation :exec
UPDATE hr.payroll_runs SET needs_recalculation = true, updated_at = now() WHERE id = $1 AND status = 'calculated';

-- name: DeletePayrollRun :execrows
DELETE FROM hr.payroll_runs WHERE id = $1 AND status <> 'paid';

-- name: ListPayslips :many
SELECT sqlc.embed(p), r.period_month, r.period_year, r.status AS run_status,
       e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name, COALESCE(po.title, '')::text AS position_name
FROM hr.payroll_details p
JOIN hr.payroll_runs r ON r.id = p.payroll_run_id
JOIN hr.employees e ON e.id = p.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions po ON po.id = e.position_id
WHERE (sqlc.arg(employee_id)::text = '' OR p.employee_id = sqlc.arg(employee_id)::text)
  AND (sqlc.arg(year)::int = 0 OR r.period_year = sqlc.arg(year)::int)
  AND (sqlc.arg(run_id)::text = '' OR p.payroll_run_id = sqlc.arg(run_id)::text)
  AND (NOT sqlc.arg(only_paid)::boolean OR r.status = 'paid')
ORDER BY r.period_year DESC, r.period_month DESC, e.full_name;

-- name: GetPayslip :one
SELECT sqlc.embed(p), r.period_month, r.period_year, r.status AS run_status,
       e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name, COALESCE(po.title, '')::text AS position_name
FROM hr.payroll_details p
JOIN hr.payroll_runs r ON r.id = p.payroll_run_id
JOIN hr.employees e ON e.id = p.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions po ON po.id = e.position_id
WHERE p.id = $1;

-- name: ListPayrollAdditions :many
SELECT sqlc.embed(a), e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name
FROM hr.payroll_additions a
JOIN hr.employees e ON e.id = a.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE (sqlc.arg(period_year)::int = 0 OR a.period_year = sqlc.arg(period_year)::int)
  AND (sqlc.arg(period_month)::int = 0 OR a.period_month = sqlc.arg(period_month)::int)
  AND (sqlc.arg(employee_id)::text = '' OR a.employee_id = sqlc.arg(employee_id)::text)
ORDER BY a.period_year DESC, a.period_month DESC, e.full_name;

-- name: GetPayrollAddition :one
SELECT * FROM hr.payroll_additions WHERE id = $1;

-- name: InsertPayrollAddition :exec
INSERT INTO hr.payroll_additions (id, employee_id, period_month, period_year, amount, label, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: DeletePayrollAddition :execrows
DELETE FROM hr.payroll_additions WHERE id = $1;

-- name: SumPayrollAdditions :one
SELECT COALESCE(sum(amount), 0)::text FROM hr.payroll_additions
WHERE employee_id = $1 AND period_month = $2 AND period_year = $3;

-- name: ListLoans :many
SELECT sqlc.embed(l), e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name,
       COALESCE(a.full_name, '')::text AS approved_by_name
FROM hr.loans l
JOIN hr.employees e ON e.id = l.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.employees a ON a.id = l.approved_by
WHERE (sqlc.arg(employee_id)::text = '' OR l.employee_id = sqlc.arg(employee_id)::text)
  AND (sqlc.arg(status)::text = '' OR l.status = sqlc.arg(status)::text)
ORDER BY l.created_at DESC;

-- name: GetLoan :one
SELECT sqlc.embed(l), e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name,
       COALESCE(a.full_name, '')::text AS approved_by_name
FROM hr.loans l
JOIN hr.employees e ON e.id = l.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.employees a ON a.id = l.approved_by
WHERE l.id = $1;

-- name: CountOpenLoans :one
SELECT count(*) FROM hr.loans
WHERE employee_id = $1 AND is_active
  AND (status = 'pending' AND sqlc.arg(include_pending)::boolean OR status = 'approved' AND remaining_balance > 0);

-- name: InsertLoan :exec
INSERT INTO hr.loans (id, employee_id, loan_type, principal_amount, interest_rate, tenor_months, monthly_installment,
    remaining_balance, purpose, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: ApproveLoan :execrows
UPDATE hr.loans SET status = 'approved', approved_by = $2, approved_at = $3, first_installment_month = $4,
    first_installment_year = $5, updated_at = now()
WHERE id = $1 AND status = 'pending';

-- name: RejectLoan :execrows
UPDATE hr.loans SET status = 'rejected', rejected_by = $2, rejected_at = now(), rejection_reason = $3, is_active = false,
    updated_at = now()
WHERE id = $1 AND status = 'pending';

-- name: CancelLoan :execrows
UPDATE hr.loans SET status = 'cancelled', is_active = false, updated_at = now()
WHERE id = $1 AND status = 'pending';

-- name: OpenLoans :many
SELECT * FROM hr.loans
WHERE employee_id = $1 AND status = 'approved' AND is_active AND remaining_balance > 0
ORDER BY approved_at NULLS LAST, created_at;

-- name: LockOpenLoans :many
SELECT * FROM hr.loans
WHERE employee_id = $1 AND status = 'approved' AND is_active AND remaining_balance > 0
ORDER BY approved_at NULLS LAST, created_at
FOR UPDATE;

-- name: ApplyLoanPayment :exec
UPDATE hr.loans SET remaining_balance = sqlc.arg(remaining)::numeric,
    paid_amount = paid_amount + sqlc.arg(paid)::numeric,
    is_active = CASE WHEN sqlc.arg(remaining)::numeric <= 0 THEN false ELSE is_active END,
    status = CASE WHEN sqlc.arg(remaining)::numeric <= 0 THEN 'paid_off' ELSE status END,
    updated_at = now()
WHERE id = sqlc.arg(id);
