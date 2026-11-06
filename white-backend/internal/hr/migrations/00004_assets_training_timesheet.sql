-- +goose Up
-- Company assets lent to employees, training with participants, and the
-- timesheet that spreads payroll cost over Finance projects.

CREATE TABLE hr.assets (
    id             text PRIMARY KEY,
    asset_code     text NOT NULL UNIQUE,
    name           text NOT NULL,
    -- laptop | phone | vehicle | furniture | other
    category       text NOT NULL,
    serial_no      text NOT NULL DEFAULT '',
    purchase_date  date,
    purchase_value numeric(18, 2) NOT NULL DEFAULT 0 CHECK (purchase_value >= 0),
    status         text NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'assigned', 'repair', 'retired')),
    -- Set while status is assigned.
    employee_id    text REFERENCES hr.employees(id) ON DELETE SET NULL,
    notes          text NOT NULL DEFAULT '',
    created_by     text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX assets_employee_idx ON hr.assets (employee_id) WHERE employee_id IS NOT NULL;

CREATE TABLE hr.asset_assignments (
    id               text PRIMARY KEY,
    asset_id         text NOT NULL REFERENCES hr.assets(id) ON DELETE CASCADE,
    employee_id      text NOT NULL REFERENCES hr.employees(id) ON DELETE CASCADE,
    assigned_on      date NOT NULL,
    returned_on      date,
    condition_out    text NOT NULL DEFAULT '',
    condition_in     text NOT NULL DEFAULT '',
    created_by       text,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX asset_assignments_asset_idx ON hr.asset_assignments (asset_id, assigned_on DESC);

CREATE TABLE hr.trainings (
    id          text PRIMARY KEY,
    title       text NOT NULL,
    provider    text NOT NULL DEFAULT '',
    start_date  date NOT NULL,
    end_date    date NOT NULL,
    location    text NOT NULL DEFAULT '',
    cost        numeric(18, 2) NOT NULL DEFAULT 0 CHECK (cost >= 0),
    status      text NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'done', 'cancelled')),
    notes       text NOT NULL DEFAULT '',
    created_by  text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

CREATE TABLE hr.training_participants (
    training_id text NOT NULL REFERENCES hr.trainings(id) ON DELETE CASCADE,
    employee_id text NOT NULL REFERENCES hr.employees(id) ON DELETE CASCADE,
    status      text NOT NULL DEFAULT 'registered' CHECK (status IN ('registered', 'attended', 'absent')),
    score       numeric(5, 2) CHECK (score BETWEEN 0 AND 100),
    certificate text NOT NULL DEFAULT '',
    PRIMARY KEY (training_id, employee_id)
);
CREATE INDEX training_participants_employee_idx ON hr.training_participants (employee_id);

CREATE TABLE hr.timesheet_entries (
    id            text PRIMARY KEY,
    employee_id   text NOT NULL REFERENCES hr.employees(id) ON DELETE CASCADE,
    work_date     date NOT NULL,
    -- Finance project tag; empty for internal work.
    project_id    text NOT NULL DEFAULT '',
    project_name  text NOT NULL DEFAULT '',
    hours         numeric(5, 2) NOT NULL CHECK (hours > 0 AND hours <= 24),
    description   text NOT NULL DEFAULT '',
    status        text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'approved', 'rejected')),
    reject_reason text NOT NULL DEFAULT '',
    decided_by    text,
    decided_at    timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX timesheet_employee_date_idx ON hr.timesheet_entries (employee_id, work_date);
CREATE INDEX timesheet_status_idx ON hr.timesheet_entries (status, work_date);

-- +goose Down
DROP TABLE hr.timesheet_entries;
DROP TABLE hr.training_participants;
DROP TABLE hr.trainings;
DROP TABLE hr.asset_assignments;
DROP TABLE hr.assets;
