-- +goose Up
-- hr schema: people, workforce (attendance, shifts, holidays, leave,
-- overtime, contracts, announcements, on/offboarding) and compensation
-- (salaries, payroll settings, runs, payslips, additions, loans). Ported
-- from NüHabit hris + payroll, single company, ids are text (UUIDv7).

CREATE SCHEMA IF NOT EXISTS hr;

CREATE TYPE hr.attendance_status AS ENUM ('present', 'late', 'absent', 'half_day', 'remote');

CREATE TABLE hr.departments (
    id               text PRIMARY KEY,
    code             text NOT NULL,
    name             text NOT NULL,
    parent_id        text REFERENCES hr.departments (id) ON DELETE SET NULL,
    head_employee_id text,
    is_active        boolean NOT NULL DEFAULT true,
    created_by       text,
    updated_by       text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    deleted_at       timestamptz
);
CREATE UNIQUE INDEX departments_code_key ON hr.departments (lower(code)) WHERE deleted_at IS NULL;

CREATE TABLE hr.positions (
    id            text PRIMARY KEY,
    title         text NOT NULL,
    department_id text REFERENCES hr.departments (id) ON DELETE SET NULL,
    level         text NOT NULL DEFAULT 'staff',
    salary_min    numeric(18, 2),
    salary_max    numeric(18, 2),
    is_active     boolean NOT NULL DEFAULT true,
    created_by    text,
    updated_by    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);

CREATE TABLE hr.employment_statuses (
    code       text PRIMARY KEY,
    name       text NOT NULL,
    is_active  boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO hr.employment_statuses (code, name, sort_order) VALUES
    ('permanent', 'Karyawan Tetap', 1),
    ('contract', 'Kontrak (PKWT)', 2),
    ('probation', 'Masa Percobaan', 3),
    ('intern', 'Magang', 4),
    ('resigned', 'Mengundurkan Diri', 5),
    ('terminated', 'Diberhentikan', 6);

CREATE TABLE hr.employees (
    id                         text PRIMARY KEY,
    user_id                    text,
    employee_no                text NOT NULL,
    full_name                  text NOT NULL,
    email                      text NOT NULL DEFAULT '',
    phone                      text NOT NULL DEFAULT '',
    nik                        text NOT NULL DEFAULT '',
    npwp                       text NOT NULL DEFAULT '',
    bpjs_tk_no                 text NOT NULL DEFAULT '',
    bpjs_kes_no                text NOT NULL DEFAULT '',
    birth_place                text NOT NULL DEFAULT '',
    birth_date                 date,
    gender                     text NOT NULL DEFAULT '',
    marital_status             text NOT NULL DEFAULT '',
    religion                   text NOT NULL DEFAULT '',
    address                    text NOT NULL DEFAULT '',
    department_id              text REFERENCES hr.departments (id) ON DELETE SET NULL,
    position_id                text REFERENCES hr.positions (id) ON DELETE SET NULL,
    manager_id                 text REFERENCES hr.employees (id) ON DELETE SET NULL,
    employment_status          text NOT NULL DEFAULT 'probation',
    is_active                  boolean NOT NULL DEFAULT true,
    join_date                  date NOT NULL,
    end_date                   date,
    probation_end_date         date,
    bank_name                  text NOT NULL DEFAULT '',
    bank_account_no            text NOT NULL DEFAULT '',
    bank_account_name          text NOT NULL DEFAULT '',
    emergency_contact_name     text NOT NULL DEFAULT '',
    emergency_contact_phone    text NOT NULL DEFAULT '',
    emergency_contact_relation text NOT NULL DEFAULT '',
    created_by                 text,
    updated_by                 text,
    created_at                 timestamptz NOT NULL DEFAULT now(),
    updated_at                 timestamptz NOT NULL DEFAULT now(),
    deleted_at                 timestamptz
);
CREATE UNIQUE INDEX employees_user_id_key ON hr.employees (user_id) WHERE user_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX employees_employee_no_key ON hr.employees (employee_no) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX employees_email_key ON hr.employees (lower(email)) WHERE email <> '' AND deleted_at IS NULL;
CREATE UNIQUE INDEX employees_nik_key ON hr.employees (nik) WHERE nik <> '' AND deleted_at IS NULL;
CREATE INDEX employees_department_idx ON hr.employees (department_id);
CREATE INDEX employees_manager_idx ON hr.employees (manager_id);

ALTER TABLE hr.departments
    ADD CONSTRAINT departments_head_fkey FOREIGN KEY (head_employee_id) REFERENCES hr.employees (id) ON DELETE SET NULL;

CREATE TABLE hr.employment_history (
    id                     text PRIMARY KEY,
    employee_id            text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    change_type            text NOT NULL,
    notes                  text NOT NULL DEFAULT '',
    prev_employment_status text,
    new_employment_status  text,
    prev_department_id     text,
    new_department_id      text,
    prev_position_id       text,
    new_position_id        text,
    created_by             text,
    created_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX employment_history_employee_idx ON hr.employment_history (employee_id, created_at DESC);

CREATE TABLE hr.employee_documents (
    id           text PRIMARY KEY,
    employee_id  text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    doc_type     text NOT NULL,
    name         text NOT NULL,
    storage_key  text NOT NULL,
    content_type text NOT NULL,
    size_bytes   bigint NOT NULL,
    created_by   text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);
CREATE INDEX employee_documents_employee_idx ON hr.employee_documents (employee_id);

-- ── Workforce ────────────────────────────────────────────────

CREATE TABLE hr.shifts (
    id                     text PRIMARY KEY,
    name                   text NOT NULL,
    start_time             time NOT NULL,
    end_time               time NOT NULL,
    break_minutes          integer NOT NULL DEFAULT 60,
    late_tolerance_minutes integer NOT NULL DEFAULT 10,
    is_overnight           boolean NOT NULL DEFAULT false,
    is_active              boolean NOT NULL DEFAULT true,
    sort_order             integer NOT NULL DEFAULT 0,
    created_by             text,
    updated_by             text,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);

-- Weekly pattern versions: day 1 = Monday .. 7 = Sunday, shift NULL = day off.
CREATE TABLE hr.employee_shifts (
    id             text PRIMARY KEY,
    employee_id    text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    day_of_week    integer NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    shift_id       text REFERENCES hr.shifts (id),
    effective_from date NOT NULL,
    effective_to   date,
    created_by     text,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX employee_shifts_employee_idx ON hr.employee_shifts (employee_id);

CREATE TABLE hr.public_holidays (
    id            text PRIMARY KEY,
    holiday_date  date NOT NULL,
    name          text NOT NULL,
    type          text NOT NULL DEFAULT 'nasional' CHECK (type IN ('nasional', 'cuti_bersama', 'perusahaan')),
    deducts_leave boolean NOT NULL DEFAULT false,
    status        text NOT NULL DEFAULT 'aktif' CHECK (status IN ('draft', 'aktif')),
    note          text NOT NULL DEFAULT '',
    created_by    text,
    updated_by    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX public_holidays_date_idx ON hr.public_holidays (holiday_date);

CREATE TABLE hr.attendance (
    id                  text PRIMARY KEY,
    employee_id         text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    date                date NOT NULL,
    clock_in            timestamptz,
    clock_out           timestamptz,
    clock_in_location   jsonb,
    clock_out_location  jsonb,
    clock_in_photo_key  text,
    clock_out_photo_key text,
    work_hours          numeric(5, 2) NOT NULL DEFAULT 0 CHECK (work_hours >= 0),
    break_minutes       integer NOT NULL DEFAULT 60,
    status              hr.attendance_status NOT NULL DEFAULT 'present',
    is_late             boolean NOT NULL DEFAULT false,
    late_minutes        integer NOT NULL DEFAULT 0 CHECK (late_minutes >= 0),
    overtime_hours      numeric(6, 2) NOT NULL DEFAULT 0,
    shift_id            text REFERENCES hr.shifts (id),
    scheduled_start     timestamptz,
    scheduled_end       timestamptz,
    validated_by        text,
    validated_at        timestamptz,
    validation_notes    text NOT NULL DEFAULT '',
    notes               text NOT NULL DEFAULT '',
    created_by          text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT attendance_employee_date_key UNIQUE (employee_id, date)
);
CREATE INDEX attendance_date_idx ON hr.attendance (date);

CREATE TABLE hr.leave_balances (
    id                 text PRIMARY KEY,
    employee_id        text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    year               integer NOT NULL,
    annual_leave_total integer NOT NULL DEFAULT 12,
    annual_leave_used  integer NOT NULL DEFAULT 0,
    sick_leave_used    integer NOT NULL DEFAULT 0,
    unpaid_leave_used  integer NOT NULL DEFAULT 0,
    other_leave_used   integer NOT NULL DEFAULT 0,
    updated_by         text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT leave_balances_employee_year_key UNIQUE (employee_id, year)
);

CREATE TABLE hr.leaves (
    id                  text PRIMARY KEY,
    employee_id         text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    leave_type          text NOT NULL,
    start_date          date NOT NULL,
    end_date            date NOT NULL,
    total_days          integer NOT NULL,
    reason              text NOT NULL DEFAULT '',
    status              text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'manager_approved', 'approved', 'rejected', 'cancelled')),
    manager_approved_by text REFERENCES hr.employees (id) ON DELETE SET NULL,
    manager_approved_at timestamptz,
    approved_by         text REFERENCES hr.employees (id) ON DELETE SET NULL,
    approved_at         timestamptz,
    rejected_by         text REFERENCES hr.employees (id) ON DELETE SET NULL,
    rejection_reason    text NOT NULL DEFAULT '',
    created_by          text,
    updated_by          text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT leaves_end_after_start CHECK (end_date >= start_date)
);
CREATE INDEX leaves_employee_idx ON hr.leaves (employee_id, start_date DESC);
CREATE INDEX leaves_status_idx ON hr.leaves (status);

CREATE TABLE hr.overtime_requests (
    id               text PRIMARY KEY,
    employee_id      text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    date             date NOT NULL,
    start_time       time NOT NULL,
    end_time         time NOT NULL,
    hours            numeric(5, 2) NOT NULL CHECK (hours > 0 AND hours <= 24),
    source           text NOT NULL DEFAULT 'employee' CHECK (source IN ('employee', 'company')),
    status           text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    reason           text NOT NULL DEFAULT '',
    requested_by     text REFERENCES hr.employees (id) ON DELETE SET NULL,
    decided_by       text REFERENCES hr.employees (id) ON DELETE SET NULL,
    decided_at       timestamptz,
    rejection_reason text NOT NULL DEFAULT '',
    created_by       text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX overtime_requests_active_key ON hr.overtime_requests (employee_id, date)
    WHERE status IN ('pending', 'approved');

CREATE TABLE hr.employment_contracts (
    id                  text PRIMARY KEY,
    contract_number     text NOT NULL UNIQUE,
    employee_id         text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    contract_type       text NOT NULL CHECK (contract_type IN ('pkwt', 'pkwtt')),
    status              text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'ended', 'terminated', 'converted')),
    start_date          date NOT NULL,
    end_date            date,
    probation_end_date  date,
    parent_contract_id  text REFERENCES hr.employment_contracts (id) ON DELETE SET NULL,
    sequence            integer NOT NULL DEFAULT 1,
    position_title      text NOT NULL DEFAULT '',
    department_name     text NOT NULL DEFAULT '',
    work_location       text NOT NULL DEFAULT '',
    base_salary         numeric(18, 2),
    signed_at           date,
    compensation_amount numeric(18, 2),
    terminated_reason   text NOT NULL DEFAULT '',
    notes               text NOT NULL DEFAULT '',
    created_by          text,
    updated_by          text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT contracts_pkwt_needs_end_date CHECK (contract_type <> 'pkwt' OR end_date IS NOT NULL),
    CONSTRAINT contracts_pkwt_no_probation CHECK (contract_type <> 'pkwt' OR probation_end_date IS NULL),
    CONSTRAINT contracts_end_after_start CHECK (end_date IS NULL OR end_date > start_date)
);
CREATE INDEX employment_contracts_employee_idx ON hr.employment_contracts (employee_id);
CREATE UNIQUE INDEX employment_contracts_one_active ON hr.employment_contracts (employee_id) WHERE status = 'active';

CREATE TABLE hr.announcements (
    id           text PRIMARY KEY,
    title        text NOT NULL,
    body         text NOT NULL DEFAULT '',
    tags         text[] NOT NULL DEFAULT '{}',
    status       text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
    is_pinned    boolean NOT NULL DEFAULT false,
    target_scope text NOT NULL DEFAULT 'global' CHECK (target_scope IN ('global', 'departments')),
    publish_at   timestamptz,
    expires_at   timestamptz,
    created_by   text,
    updated_by   text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);

CREATE TABLE hr.announcement_departments (
    announcement_id text NOT NULL REFERENCES hr.announcements (id) ON DELETE CASCADE,
    department_id   text NOT NULL REFERENCES hr.departments (id) ON DELETE CASCADE,
    PRIMARY KEY (announcement_id, department_id)
);

CREATE TABLE hr.announcement_reads (
    announcement_id text NOT NULL REFERENCES hr.announcements (id) ON DELETE CASCADE,
    employee_id     text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    read_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (announcement_id, employee_id)
);

CREATE TABLE hr.onboarding_tasks (
    id           text PRIMARY KEY,
    employee_id  text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    task_name    text NOT NULL,
    category     text NOT NULL DEFAULT 'general' CHECK (category IN ('admin', 'it', 'hr', 'manager', 'general')),
    priority     integer NOT NULL DEFAULT 2 CHECK (priority BETWEEN 1 AND 3),
    due_date     date,
    is_completed boolean NOT NULL DEFAULT false,
    completed_at timestamptz,
    completed_by text REFERENCES hr.employees (id) ON DELETE SET NULL,
    notes        text NOT NULL DEFAULT '',
    created_by   text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX onboarding_tasks_employee_idx ON hr.onboarding_tasks (employee_id);

CREATE TABLE hr.offboarding (
    id                   text PRIMARY KEY,
    employee_id          text NOT NULL UNIQUE REFERENCES hr.employees (id) ON DELETE CASCADE,
    resignation_type     text NOT NULL CHECK (resignation_type IN ('voluntary', 'termination', 'layoff', 'end_of_contract')),
    status               text NOT NULL DEFAULT 'submitted' CHECK (status IN ('submitted', 'notice_period', 'exit_interview', 'completed')),
    resignation_date     date NOT NULL,
    last_working_day     date NOT NULL,
    reason               text NOT NULL DEFAULT '',
    exit_interview_notes text NOT NULL DEFAULT '',
    clearance_hr         boolean NOT NULL DEFAULT false,
    clearance_it         boolean NOT NULL DEFAULT false,
    clearance_finance    boolean NOT NULL DEFAULT false,
    clearance_manager    boolean NOT NULL DEFAULT false,
    notes                text NOT NULL DEFAULT '',
    completed_at         timestamptz,
    created_by           text,
    updated_by           text,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

-- ── Compensation ─────────────────────────────────────────────

-- One row (id = 1). Rates are percentages, as NüHabit stored them.
CREATE TABLE hr.payroll_settings (
    id                           integer PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    bpjs_tk_jht_employee         numeric(6, 3) NOT NULL DEFAULT 2,
    bpjs_tk_jht_employer         numeric(6, 3) NOT NULL DEFAULT 3.7,
    bpjs_tk_jp_employee          numeric(6, 3) NOT NULL DEFAULT 1,
    bpjs_tk_jp_employer          numeric(6, 3) NOT NULL DEFAULT 2,
    bpjs_tk_jkk                  numeric(6, 3) NOT NULL DEFAULT 0.24,
    bpjs_tk_jkm                  numeric(6, 3) NOT NULL DEFAULT 0.3,
    bpjs_kes_employee            numeric(6, 3) NOT NULL DEFAULT 1,
    bpjs_kes_employer            numeric(6, 3) NOT NULL DEFAULT 4,
    bpjs_kes_max_upah            numeric(18, 2) NOT NULL DEFAULT 12000000,
    tapera_employee              numeric(6, 3) NOT NULL DEFAULT 2.5,
    tapera_employer              numeric(6, 3) NOT NULL DEFAULT 0.5,
    ptkp_tk_0                    numeric(18, 2) NOT NULL DEFAULT 54000000,
    ptkp_tk_1                    numeric(18, 2) NOT NULL DEFAULT 58500000,
    ptkp_tk_2                    numeric(18, 2) NOT NULL DEFAULT 63000000,
    ptkp_tk_3                    numeric(18, 2) NOT NULL DEFAULT 67500000,
    ptkp_k_0                     numeric(18, 2) NOT NULL DEFAULT 58500000,
    ptkp_k_1                     numeric(18, 2) NOT NULL DEFAULT 63000000,
    ptkp_k_2                     numeric(18, 2) NOT NULL DEFAULT 67500000,
    ptkp_k_3                     numeric(18, 2) NOT NULL DEFAULT 72000000,
    pph21_bracket_1              numeric(18, 2) NOT NULL DEFAULT 60000000,
    pph21_bracket_2              numeric(18, 2) NOT NULL DEFAULT 250000000,
    pph21_bracket_3              numeric(18, 2) NOT NULL DEFAULT 500000000,
    pph21_bracket_4              numeric(18, 2) NOT NULL DEFAULT 5000000000,
    overtime_multiplier          numeric(6, 2) NOT NULL DEFAULT 1.5,
    overtime_multiplier_holiday  numeric(6, 2) NOT NULL DEFAULT 2,
    overtime_hourly_divisor      numeric(6, 2) NOT NULL DEFAULT 173,
    late_deduction_mode          text NOT NULL DEFAULT 'off' CHECK (late_deduction_mode IN ('off', 'per_minute', 'flat')),
    late_deduction_amount        numeric(18, 2) NOT NULL DEFAULT 0,
    loan_max_installment_percent numeric(6, 2) NOT NULL DEFAULT 30,
    loan_max_active_per_employee integer NOT NULL DEFAULT 1,
    payroll_day                  integer NOT NULL DEFAULT 25,
    updated_by                   text,
    updated_at                   timestamptz NOT NULL DEFAULT now()
);
INSERT INTO hr.payroll_settings (id) VALUES (1);

CREATE TABLE hr.employee_salary (
    id                  text PRIMARY KEY,
    employee_id         text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    base_salary         numeric(18, 2) NOT NULL,
    fixed_allowance     numeric(18, 2) NOT NULL DEFAULT 0,
    variable_allowance  numeric(18, 2) NOT NULL DEFAULT 0,
    transport_allowance numeric(18, 2) NOT NULL DEFAULT 0,
    meal_allowance      numeric(18, 2) NOT NULL DEFAULT 0,
    housing_allowance   numeric(18, 2) NOT NULL DEFAULT 0,
    ptkp_status         text NOT NULL DEFAULT 'TK/0',
    is_taxable          boolean NOT NULL DEFAULT true,
    bpjs_tk_enrolled    boolean NOT NULL DEFAULT true,
    bpjs_kes_enrolled   boolean NOT NULL DEFAULT true,
    tapera_enrolled     boolean NOT NULL DEFAULT true,
    effective_date      date NOT NULL,
    end_date            date,
    is_active           boolean NOT NULL DEFAULT true,
    notes               text NOT NULL DEFAULT '',
    created_by          text,
    updated_by          text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX employee_salary_employee_idx ON hr.employee_salary (employee_id, effective_date DESC);
CREATE UNIQUE INDEX employee_salary_one_active ON hr.employee_salary (employee_id) WHERE is_active;

CREATE TABLE hr.payroll_runs (
    id                  text PRIMARY KEY,
    run_name            text NOT NULL,
    period_month        integer NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    period_year         integer NOT NULL,
    status              text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'calculated', 'approved', 'paid')),
    total_employees     integer NOT NULL DEFAULT 0,
    total_gross         numeric(18, 2) NOT NULL DEFAULT 0,
    total_deductions    numeric(18, 2) NOT NULL DEFAULT 0,
    total_net           numeric(18, 2) NOT NULL DEFAULT 0,
    total_bpjs_employee numeric(18, 2) NOT NULL DEFAULT 0,
    total_bpjs_employer numeric(18, 2) NOT NULL DEFAULT 0,
    total_pph21         numeric(18, 2) NOT NULL DEFAULT 0,
    needs_recalculation boolean NOT NULL DEFAULT false,
    calculated_at       timestamptz,
    calculated_by       text,
    approved_at         timestamptz,
    approved_by         text,
    paid_at             timestamptz,
    paid_by             text,
    notes               text NOT NULL DEFAULT '',
    created_by          text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payroll_runs_period_key UNIQUE (period_year, period_month)
);

CREATE TABLE hr.payroll_details (
    id                     text PRIMARY KEY,
    payroll_run_id         text NOT NULL REFERENCES hr.payroll_runs (id) ON DELETE CASCADE,
    employee_id            text NOT NULL REFERENCES hr.employees (id),
    base_salary            numeric(18, 2) NOT NULL DEFAULT 0,
    full_base_salary       numeric(18, 2) NOT NULL DEFAULT 0,
    fixed_allowance        numeric(18, 2) NOT NULL DEFAULT 0,
    variable_allowance     numeric(18, 2) NOT NULL DEFAULT 0,
    transport_allowance    numeric(18, 2) NOT NULL DEFAULT 0,
    meal_allowance         numeric(18, 2) NOT NULL DEFAULT 0,
    housing_allowance      numeric(18, 2) NOT NULL DEFAULT 0,
    overtime_pay           numeric(18, 2) NOT NULL DEFAULT 0,
    thr                    numeric(18, 2) NOT NULL DEFAULT 0,
    bonus                  numeric(18, 2) NOT NULL DEFAULT 0,
    gross_salary           numeric(18, 2) NOT NULL DEFAULT 0,
    bpjs_tk_jht_deduction  numeric(18, 2) NOT NULL DEFAULT 0,
    bpjs_tk_jp_deduction   numeric(18, 2) NOT NULL DEFAULT 0,
    bpjs_kes_deduction     numeric(18, 2) NOT NULL DEFAULT 0,
    tapera_deduction       numeric(18, 2) NOT NULL DEFAULT 0,
    pph21_deduction        numeric(18, 2) NOT NULL DEFAULT 0,
    unpaid_leave_deduction numeric(18, 2) NOT NULL DEFAULT 0,
    late_deduction         numeric(18, 2) NOT NULL DEFAULT 0,
    loan_deduction         numeric(18, 2) NOT NULL DEFAULT 0,
    loan_details           jsonb NOT NULL DEFAULT '[]',
    total_deductions       numeric(18, 2) NOT NULL DEFAULT 0,
    net_salary             numeric(18, 2) NOT NULL DEFAULT 0,
    bpjs_tk_jht_employer   numeric(18, 2) NOT NULL DEFAULT 0,
    bpjs_tk_jp_employer    numeric(18, 2) NOT NULL DEFAULT 0,
    bpjs_tk_jkk_employer   numeric(18, 2) NOT NULL DEFAULT 0,
    bpjs_tk_jkm_employer   numeric(18, 2) NOT NULL DEFAULT 0,
    bpjs_kes_employer      numeric(18, 2) NOT NULL DEFAULT 0,
    tapera_employer        numeric(18, 2) NOT NULL DEFAULT 0,
    taxable_income         numeric(18, 2) NOT NULL DEFAULT 0,
    ptkp_amount            numeric(18, 2) NOT NULL DEFAULT 0,
    pph21_annual           numeric(18, 2) NOT NULL DEFAULT 0,
    working_days           numeric(6, 2) NOT NULL DEFAULT 0,
    present_days           numeric(6, 2) NOT NULL DEFAULT 0,
    late_days              numeric(6, 2) NOT NULL DEFAULT 0,
    unpaid_leave_days      numeric(6, 2) NOT NULL DEFAULT 0,
    overtime_hours         numeric(6, 2) NOT NULL DEFAULT 0,
    prorate_factor         numeric(8, 6) NOT NULL DEFAULT 1,
    created_at             timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payroll_details_run_employee_key UNIQUE (payroll_run_id, employee_id)
);
CREATE INDEX payroll_details_employee_idx ON hr.payroll_details (employee_id);

CREATE TABLE hr.payroll_additions (
    id           text PRIMARY KEY,
    employee_id  text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    period_month integer NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    period_year  integer NOT NULL,
    amount       numeric(18, 2) NOT NULL CHECK (amount > 0),
    label        text NOT NULL,
    created_by   text,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX payroll_additions_period_idx ON hr.payroll_additions (period_year, period_month);

CREATE TABLE hr.loans (
    id                      text PRIMARY KEY,
    employee_id             text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE,
    loan_type               text NOT NULL,
    principal_amount        numeric(18, 2) NOT NULL CHECK (principal_amount > 0),
    interest_rate           numeric(6, 2) NOT NULL DEFAULT 0,
    tenor_months            integer NOT NULL CHECK (tenor_months BETWEEN 1 AND 60),
    monthly_installment     numeric(18, 2) NOT NULL,
    remaining_balance       numeric(18, 2) NOT NULL,
    paid_amount             numeric(18, 2) NOT NULL DEFAULT 0,
    first_installment_month integer,
    first_installment_year  integer,
    status                  text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'paid_off')),
    is_active               boolean NOT NULL DEFAULT true,
    purpose                 text NOT NULL DEFAULT '',
    approved_by             text REFERENCES hr.employees (id) ON DELETE SET NULL,
    approved_at             timestamptz,
    rejected_by             text REFERENCES hr.employees (id) ON DELETE SET NULL,
    rejected_at             timestamptz,
    rejection_reason        text NOT NULL DEFAULT '',
    created_by              text,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX loans_employee_idx ON hr.loans (employee_id);

-- Daily-tick reminders sent, so a redelivered tick never notifies twice.
CREATE TABLE hr.reminder_sends (
    kind    text NOT NULL,
    ref     text NOT NULL,
    day     date NOT NULL,
    sent_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, ref, day)
);

-- +goose Down
DROP TABLE IF EXISTS hr.reminder_sends CASCADE;
DROP TABLE IF EXISTS hr.loans CASCADE;
DROP TABLE IF EXISTS hr.payroll_additions CASCADE;
DROP TABLE IF EXISTS hr.payroll_details CASCADE;
DROP TABLE IF EXISTS hr.payroll_runs CASCADE;
DROP TABLE IF EXISTS hr.employee_salary CASCADE;
DROP TABLE IF EXISTS hr.payroll_settings CASCADE;
DROP TABLE IF EXISTS hr.offboarding CASCADE;
DROP TABLE IF EXISTS hr.onboarding_tasks CASCADE;
DROP TABLE IF EXISTS hr.announcement_reads CASCADE;
DROP TABLE IF EXISTS hr.announcement_departments CASCADE;
DROP TABLE IF EXISTS hr.announcements CASCADE;
DROP TABLE IF EXISTS hr.employment_contracts CASCADE;
DROP TABLE IF EXISTS hr.overtime_requests CASCADE;
DROP TABLE IF EXISTS hr.leaves CASCADE;
DROP TABLE IF EXISTS hr.leave_balances CASCADE;
DROP TABLE IF EXISTS hr.attendance CASCADE;
DROP TABLE IF EXISTS hr.public_holidays CASCADE;
DROP TABLE IF EXISTS hr.employee_shifts CASCADE;
DROP TABLE IF EXISTS hr.shifts CASCADE;
DROP TABLE IF EXISTS hr.employee_documents CASCADE;
DROP TABLE IF EXISTS hr.employment_history CASCADE;
DROP TABLE IF EXISTS hr.employees CASCADE;
DROP TABLE IF EXISTS hr.employment_statuses CASCADE;
DROP TABLE IF EXISTS hr.positions CASCADE;
DROP TABLE IF EXISTS hr.departments CASCADE;
DROP TYPE IF EXISTS hr.attendance_status;
