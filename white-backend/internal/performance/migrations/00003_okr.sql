-- +goose Up
-- Objectives and key results per quarter, at company, department or
-- employee level, with check-ins that move each key result.
CREATE TABLE performance.objectives (
    id            text PRIMARY KEY,
    -- YYYY-Qn
    period        text NOT NULL CHECK (period ~ '^\d{4}-Q[1-4]$'),
    level         text NOT NULL CHECK (level IN ('company', 'department', 'employee')),
    department_id text,
    employee_id   text,
    parent_id     text REFERENCES performance.objectives(id) ON DELETE SET NULL,
    title         text NOT NULL,
    description   text NOT NULL DEFAULT '',
    status        text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'done', 'cancelled')),
    created_by    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CHECK ((level = 'department') = (department_id IS NOT NULL)),
    CHECK ((level = 'employee') = (employee_id IS NOT NULL))
);
CREATE INDEX objectives_period_idx ON performance.objectives (period, level);

CREATE TABLE performance.key_results (
    id            text PRIMARY KEY,
    objective_id  text NOT NULL REFERENCES performance.objectives(id) ON DELETE CASCADE,
    title         text NOT NULL,
    -- number | percent | currency
    metric        text NOT NULL CHECK (metric IN ('number', 'percent', 'currency')),
    start_value   numeric(18, 2) NOT NULL DEFAULT 0,
    target_value  numeric(18, 2) NOT NULL,
    current_value numeric(18, 2) NOT NULL DEFAULT 0,
    unit          text NOT NULL DEFAULT '',
    sort_order    integer NOT NULL DEFAULT 0,
    CHECK (target_value <> start_value)
);
CREATE INDEX key_results_objective_idx ON performance.key_results (objective_id, sort_order);

CREATE TABLE performance.okr_checkins (
    id            text PRIMARY KEY,
    key_result_id text NOT NULL REFERENCES performance.key_results(id) ON DELETE CASCADE,
    value         numeric(18, 2) NOT NULL,
    -- 1 (off track) to 10 (certain)
    confidence    integer NOT NULL CHECK (confidence BETWEEN 1 AND 10),
    note          text NOT NULL DEFAULT '',
    created_by    text,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX okr_checkins_kr_idx ON performance.okr_checkins (key_result_id, created_at DESC);

-- +goose Down
DROP TABLE performance.okr_checkins;
DROP TABLE performance.key_results;
DROP TABLE performance.objectives;
