-- +goose Up
CREATE SCHEMA IF NOT EXISTS performance;

-- ─── KPI ─────────────────────────────────────────────────────

CREATE TABLE performance.kpi_indicators (
    id             text PRIMARY KEY,
    code           text NOT NULL UNIQUE,
    name           text NOT NULL,
    description    text NOT NULL DEFAULT '',
    unit           text NOT NULL DEFAULT '',
    direction      text NOT NULL DEFAULT 'higher_better' CHECK (direction IN ('higher_better', 'lower_better', 'boolean')),
    source_kind    text NOT NULL DEFAULT 'manual' CHECK (source_kind IN ('auto', 'manual')),
    default_target numeric(18,4),
    is_active      boolean NOT NULL DEFAULT true,
    created_by     text,
    updated_by     text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

-- Department weighting of indicators (hr department ids, plain text).
CREATE TABLE performance.kpi_department_indicators (
    department_id text NOT NULL,
    indicator_id  text NOT NULL REFERENCES performance.kpi_indicators(id) ON DELETE CASCADE,
    weight        numeric(6,2) NOT NULL CHECK (weight > 0 AND weight <= 100),
    updated_by    text,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (department_id, indicator_id)
);

CREATE TABLE performance.kpi_targets (
    id            text PRIMARY KEY,
    indicator_id  text NOT NULL REFERENCES performance.kpi_indicators(id) ON DELETE CASCADE,
    period_year   integer CHECK (period_year BETWEEN 2020 AND 2100),
    period_month  integer CHECK (period_month BETWEEN 1 AND 12),
    department_id text,
    employee_id   text,
    target        numeric(18,4) NOT NULL CHECK (target >= 0),
    created_by    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CHECK (department_id IS NULL OR employee_id IS NULL)
);
CREATE INDEX kpi_targets_indicator_idx ON performance.kpi_targets (indicator_id);

CREATE TABLE performance.kpi_snapshots (
    id            text PRIMARY KEY,
    employee_id   text NOT NULL,
    indicator_id  text NOT NULL REFERENCES performance.kpi_indicators(id) ON DELETE CASCADE,
    period_year   integer NOT NULL,
    period_month  integer NOT NULL,
    actual        numeric(18,4),
    target        numeric(18,4),
    attainment    numeric(10,4),
    sample_size   numeric(18,2) NOT NULL DEFAULT 0,
    source_detail jsonb NOT NULL DEFAULT '{}',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (employee_id, indicator_id, period_year, period_month)
);

CREATE TABLE performance.kpi_scorecards (
    id            text PRIMARY KEY,
    employee_id   text NOT NULL,
    department_id text,
    period_year   integer NOT NULL,
    period_month  integer NOT NULL,
    score         numeric(6,2),
    raw_score     numeric(8,2),
    used_weight   numeric(8,2) NOT NULL DEFAULT 0,
    breakdown     jsonb NOT NULL DEFAULT '[]',
    status        text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'final')),
    reviewed_by   text,
    reviewed_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (employee_id, period_year, period_month)
);
CREATE INDEX kpi_scorecards_period_idx ON performance.kpi_scorecards (period_year, period_month);

-- ─── Quarterly performance reviews ───────────────────────────

CREATE TABLE performance.review_cycles (
    id              text PRIMARY KEY,
    name            text NOT NULL UNIQUE,
    period_year     integer NOT NULL CHECK (period_year BETWEEN 2020 AND 2100),
    period_quarter  integer NOT NULL CHECK (period_quarter BETWEEN 1 AND 4),
    start_date      date NOT NULL,
    end_date        date NOT NULL,
    -- Deadline for self assessments and reviewer ratings (reminders).
    due_date        date NOT NULL,
    status          text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_by      text,
    created_by_name text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (period_year, period_quarter)
);

CREATE TABLE performance.behavioral_standards (
    id                   text PRIMARY KEY,
    value_name           text NOT NULL,
    competency_name      text NOT NULL DEFAULT '',
    standard_description text NOT NULL DEFAULT '',
    score_1_description  text NOT NULL DEFAULT '',
    score_2_description  text NOT NULL DEFAULT '',
    score_3_description  text NOT NULL DEFAULT '',
    score_4_description  text NOT NULL DEFAULT '',
    score_5_description  text NOT NULL DEFAULT '',
    weight               numeric(6,2) NOT NULL DEFAULT 20,
    sort_order           integer NOT NULL DEFAULT 0,
    is_active            boolean NOT NULL DEFAULT true,
    created_at           timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE performance.performance_categories (
    id            text PRIMARY KEY,
    category_name text NOT NULL UNIQUE,
    min_score     numeric(6,2) NOT NULL,
    max_score     numeric(6,2) NOT NULL,
    description   text NOT NULL DEFAULT ''
);

CREATE TABLE performance.performance_reviews (
    id                      text PRIMARY KEY,
    cycle_id                text NOT NULL REFERENCES performance.review_cycles(id) ON DELETE CASCADE,
    employee_id             text NOT NULL,
    department_id           text,
    period_label            text NOT NULL,
    start_date              date NOT NULL,
    end_date                date NOT NULL,
    total_work_result_score numeric(6,2),
    total_behavioral_score  numeric(6,2),
    total_project_score     numeric(6,2),
    grand_total_score       numeric(6,2),
    category                text,
    status                  text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'final')),
    self_assessment         text,
    reviewer_id             text,
    reviewer_name           text,
    reviewer_notes          text,
    manager_id              text,
    manager_notes           text,
    employee_sign_date      date,
    reviewer_sign_date      date,
    finalized_by            text,
    finalized_at            timestamptz,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    UNIQUE (cycle_id, employee_id)
);
CREATE INDEX performance_reviews_employee_idx ON performance.performance_reviews (employee_id);

CREATE TABLE performance.behavioral_review_items (
    id                  text PRIMARY KEY,
    review_id           text NOT NULL REFERENCES performance.performance_reviews(id) ON DELETE CASCADE,
    value_name          text NOT NULL,
    competency          text NOT NULL DEFAULT '',
    behavioral_standard text NOT NULL DEFAULT '',
    score_1_description text NOT NULL DEFAULT '',
    score_2_description text NOT NULL DEFAULT '',
    score_3_description text NOT NULL DEFAULT '',
    score_4_description text NOT NULL DEFAULT '',
    score_5_description text NOT NULL DEFAULT '',
    weight              numeric(6,2) NOT NULL DEFAULT 0,
    score               integer CHECK (score BETWEEN 1 AND 5),
    weighted_score      numeric(8,2),
    notes               text,
    item_order          integer NOT NULL DEFAULT 0,
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX behavioral_review_items_review_idx ON performance.behavioral_review_items (review_id);

-- ─── 360 feedback ────────────────────────────────────────────

CREATE TABLE performance.feedback_cycles (
    id                     text PRIMARY KEY,
    name                   text NOT NULL,
    period_label           text NOT NULL DEFAULT '',
    start_date             date,
    end_date               date,
    status                 text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'closed')),
    kpi_weight             numeric(6,2),
    feedback_weight        numeric(6,2),
    is_anonymous           boolean NOT NULL DEFAULT true,
    allow_self_assessment  boolean NOT NULL DEFAULT true,
    require_manager_review boolean NOT NULL DEFAULT false,
    created_by             text,
    updated_by             text,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);

-- code maps a category to its summary score column (leadership,
-- communication, collaboration, accountability, problem_solving).
CREATE TABLE performance.feedback_categories (
    id            text PRIMARY KEY,
    code          text NOT NULL UNIQUE,
    name          text NOT NULL,
    description   text NOT NULL DEFAULT '',
    weight        numeric(6,2) NOT NULL DEFAULT 20,
    display_order integer NOT NULL DEFAULT 0,
    is_active     boolean NOT NULL DEFAULT true
);

CREATE TABLE performance.feedback_criteria (
    id            text PRIMARY KEY,
    category_id   text NOT NULL REFERENCES performance.feedback_categories(id) ON DELETE CASCADE,
    name          text NOT NULL,
    description   text NOT NULL DEFAULT '',
    display_order integer NOT NULL DEFAULT 0
);

CREATE TABLE performance.feedback_assignments (
    id                text PRIMARY KEY,
    cycle_id          text NOT NULL REFERENCES performance.feedback_cycles(id) ON DELETE CASCADE,
    employee_id       text NOT NULL,
    reviewer_id       text NOT NULL,
    relationship_type text NOT NULL CHECK (relationship_type IN ('self', 'manager', 'peer', 'subordinate')),
    status            text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'submitted', 'approved', 'rejected')),
    due_date          date,
    started_at        timestamptz,
    submitted_at      timestamptz,
    reminder_count    integer NOT NULL DEFAULT 0,
    last_reminder_at  timestamptz,
    manager_comments  text,
    rejection_reason  text,
    approved_by       text,
    approved_at       timestamptz,
    created_by        text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (cycle_id, employee_id, reviewer_id)
);
CREATE INDEX feedback_assignments_reviewer_idx ON performance.feedback_assignments (reviewer_id, status);

CREATE TABLE performance.feedback_responses (
    id            text PRIMARY KEY,
    assignment_id text NOT NULL REFERENCES performance.feedback_assignments(id) ON DELETE CASCADE,
    criteria_id   text NOT NULL REFERENCES performance.feedback_criteria(id) ON DELETE CASCADE,
    rating        integer NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comments      text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (assignment_id, criteria_id)
);

CREATE TABLE performance.feedback_summaries (
    id                    text PRIMARY KEY,
    cycle_id              text NOT NULL REFERENCES performance.feedback_cycles(id) ON DELETE CASCADE,
    employee_id           text NOT NULL,
    leadership_score      numeric(6,2),
    communication_score   numeric(6,2),
    collaboration_score   numeric(6,2),
    accountability_score  numeric(6,2),
    problem_solving_score numeric(6,2),
    overall_360_score     numeric(6,2),
    kpi_score             numeric(6,2),
    final_score           numeric(6,2),
    final_grade           text,
    strengths             text[] NOT NULL DEFAULT '{}',
    weaknesses            text[] NOT NULL DEFAULT '{}',
    manager_comments      text,
    responses_count       integer NOT NULL DEFAULT 0,
    is_locked             boolean NOT NULL DEFAULT false,
    generated_at          timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (cycle_id, employee_id)
);

-- ─── Department tasks ────────────────────────────────────────

CREATE TABLE performance.department_tasks (
    id                     text PRIMARY KEY,
    department_id          text NOT NULL,
    assignee_employee_id   text,
    title                  text NOT NULL,
    description            text,
    recurrence             text NOT NULL CHECK (recurrence IN ('once', 'daily', 'weekly', 'monthly')),
    weekly_day             integer CHECK (weekly_day BETWEEN 1 AND 7),
    monthly_day            integer CHECK (monthly_day BETWEEN 1 AND 28),
    due_date               date,
    is_active              boolean NOT NULL DEFAULT true,
    priority               text NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    kpi_indicator_id       text REFERENCES performance.kpi_indicators(id) ON DELETE SET NULL,
    kpi_contribution       numeric(15,4),
    timezone               text NOT NULL DEFAULT 'Asia/Jakarta',
    review_required        boolean NOT NULL DEFAULT true,
    created_by_employee_id text,
    created_by_user_id     text,
    created_by_name        text NOT NULL DEFAULT '',
    updated_by             text,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX department_tasks_department_idx ON performance.department_tasks (department_id) WHERE is_active;
CREATE INDEX department_tasks_assignee_idx ON performance.department_tasks (assignee_employee_id) WHERE is_active;

CREATE TABLE performance.department_task_subtasks (
    id         text PRIMARY KEY,
    task_id    text NOT NULL REFERENCES performance.department_tasks(id) ON DELETE CASCADE,
    title      text NOT NULL,
    weight     numeric(6,2) NOT NULL,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX department_task_subtasks_task_idx ON performance.department_task_subtasks (task_id);

CREATE TABLE performance.department_task_occurrences (
    id               text PRIMARY KEY,
    task_id          text NOT NULL REFERENCES performance.department_tasks(id) ON DELETE CASCADE,
    occurrence_date  date NOT NULL,
    status           text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'done', 'approved', 'rejected')),
    progress_percent integer CHECK (progress_percent BETWEEN 0 AND 100),
    started_at       timestamptz,
    done_at          timestamptz,
    done_by          text,
    done_by_name     text,
    review_notes     text,
    reviewed_by      text,
    reviewed_by_name text,
    reviewed_at      timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (task_id, occurrence_date)
);
CREATE INDEX department_task_occurrences_date_idx ON performance.department_task_occurrences (occurrence_date, status);

CREATE TABLE performance.department_task_occurrence_items (
    occurrence_id   text NOT NULL REFERENCES performance.department_task_occurrences(id) ON DELETE CASCADE,
    subtask_id      text NOT NULL REFERENCES performance.department_task_subtasks(id) ON DELETE CASCADE,
    is_checked      boolean NOT NULL DEFAULT false,
    checked_at      timestamptz,
    checked_by_name text,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (occurrence_id, subtask_id)
);

CREATE TABLE performance.department_task_updates (
    id                 text PRIMARY KEY,
    occurrence_id      text NOT NULL REFERENCES performance.department_task_occurrences(id) ON DELETE CASCADE,
    kind               text NOT NULL CHECK (kind IN ('progress', 'comment', 'status', 'review', 'attachment', 'created')),
    note               text,
    progress_percent   integer,
    author_employee_id text,
    author_user_id     text,
    author_name        text NOT NULL DEFAULT '',
    detail             jsonb NOT NULL DEFAULT '{}',
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX department_task_updates_occurrence_idx ON performance.department_task_updates (occurrence_id, created_at);

CREATE TABLE performance.department_task_attachments (
    id                  text PRIMARY KEY,
    occurrence_id       text NOT NULL REFERENCES performance.department_task_occurrences(id) ON DELETE CASCADE,
    name                text NOT NULL,
    mime                text NOT NULL,
    size_bytes          bigint NOT NULL,
    storage_path        text NOT NULL,
    uploaded_by_user_id text,
    uploaded_by_name    text NOT NULL DEFAULT '',
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX department_task_attachments_occurrence_idx ON performance.department_task_attachments (occurrence_id);

CREATE TABLE performance.department_task_kpi_contributions (
    occurrence_id text PRIMARY KEY REFERENCES performance.department_task_occurrences(id) ON DELETE CASCADE,
    indicator_id  text NOT NULL REFERENCES performance.kpi_indicators(id) ON DELETE CASCADE,
    department_id text NOT NULL,
    employee_id   text,
    period_year   integer NOT NULL,
    period_month  integer NOT NULL,
    amount        numeric(15,4),
    recorded_at   timestamptz NOT NULL DEFAULT now()
);

-- ─── Logbook ─────────────────────────────────────────────────

CREATE TABLE performance.logbook_templates (
    id            text PRIMARY KEY,
    department_id text NOT NULL,
    name          text NOT NULL,
    description   text,
    frequency     text NOT NULL DEFAULT 'daily' CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    is_active     boolean NOT NULL DEFAULT true,
    created_by    text,
    updated_by    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX logbook_templates_department_idx ON performance.logbook_templates (department_id);

CREATE TABLE performance.logbook_template_items (
    id          text PRIMARY KEY,
    template_id text NOT NULL REFERENCES performance.logbook_templates(id) ON DELETE CASCADE,
    title       text NOT NULL,
    description text,
    weight      numeric(8,2) NOT NULL DEFAULT 1,
    is_required boolean NOT NULL DEFAULT true,
    sort_order  integer NOT NULL DEFAULT 0
);

CREATE TABLE performance.logbook_entries (
    id                    text PRIMARY KEY,
    template_id           text NOT NULL REFERENCES performance.logbook_templates(id) ON DELETE RESTRICT,
    department_id         text NOT NULL,
    employee_id           text NOT NULL,
    entry_date            date NOT NULL,
    title                 text NOT NULL,
    status                text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'reviewed', 'rejected')),
    completion_percentage numeric(6,2) NOT NULL DEFAULT 0,
    kpi_score             numeric(6,2) NOT NULL DEFAULT 0,
    notes                 text,
    submitted_by          text,
    submitted_at          timestamptz,
    reviewed_by           text,
    reviewed_at           timestamptz,
    review_notes          text,
    created_by            text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (template_id, employee_id, entry_date)
);
CREATE INDEX logbook_entries_department_idx ON performance.logbook_entries (department_id, entry_date);
CREATE INDEX logbook_entries_employee_idx ON performance.logbook_entries (employee_id, entry_date);

CREATE TABLE performance.logbook_entry_items (
    id               text PRIMARY KEY,
    entry_id         text NOT NULL REFERENCES performance.logbook_entries(id) ON DELETE CASCADE,
    template_item_id text,
    title            text NOT NULL,
    description      text,
    weight           numeric(8,2) NOT NULL DEFAULT 1,
    is_required      boolean NOT NULL DEFAULT true,
    sort_order       integer NOT NULL DEFAULT 0,
    is_checked       boolean NOT NULL DEFAULT false,
    checked_by       text,
    checked_at       timestamptz,
    notes            text,
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX logbook_entry_items_entry_idx ON performance.logbook_entry_items (entry_id);

-- ─── Automations ─────────────────────────────────────────────

-- One row per (kind, subject, day): a tick that runs every 15 minutes
-- sends each reminder once a day.
CREATE TABLE performance.reminder_sends (
    kind       text NOT NULL,
    subject_id text NOT NULL,
    day        date NOT NULL,
    sent_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, subject_id, day)
);

CREATE TABLE performance.daily_recaps (
    id                text PRIMARY KEY,
    employee_id       text NOT NULL,
    department_id     text,
    recap_date        date NOT NULL,
    items             jsonb NOT NULL DEFAULT '{}',
    notes             text NOT NULL DEFAULT '',
    generated_summary text NOT NULL DEFAULT '',
    summary_model     text NOT NULL DEFAULT '',
    generated_at      timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (employee_id, recap_date)
);
CREATE INDEX daily_recaps_date_idx ON performance.daily_recaps (recap_date, department_id);

CREATE TABLE performance.monthly_recaps (
    id              text PRIMARY KEY,
    scope           text NOT NULL CHECK (scope IN ('employee', 'department')),
    -- The employee id or department id the recap is about.
    subject_id      text NOT NULL,
    department_id   text,
    period_year     integer NOT NULL,
    period_month    integer NOT NULL,
    metrics         jsonb NOT NULL DEFAULT '{}',
    accomplishments text[] NOT NULL DEFAULT '{}',
    narrative       text NOT NULL DEFAULT '',
    narrative_model text NOT NULL DEFAULT '',
    generated_at    timestamptz NOT NULL DEFAULT now(),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (scope, subject_id, period_year, period_month)
);

CREATE TABLE performance.improvements (
    id                text PRIMARY KEY,
    scope             text NOT NULL CHECK (scope IN ('employee', 'department')),
    employee_id       text,
    department_id     text,
    source            text NOT NULL CHECK (source IN ('review', 'feedback', 'task_reject', 'logbook_reject', 'retro', 'manual', 'ai')),
    -- Origin record (occurrence, entry, review, summary, recap, retro) so an
    -- automation never files the same item twice.
    source_ref        text,
    title             text NOT NULL,
    description       text NOT NULL DEFAULT '',
    owner_employee_id text,
    due_date          date,
    status            text NOT NULL DEFAULT 'open' CHECK (status IN ('suggested', 'open', 'in_progress', 'done', 'dropped', 'declined')),
    linked_task_id    text,
    completed_at      timestamptz,
    created_by        text,
    updated_by        text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz,
    CHECK (scope <> 'employee' OR employee_id IS NOT NULL),
    CHECK (scope <> 'department' OR department_id IS NOT NULL)
);
CREATE UNIQUE INDEX improvements_source_ref_idx ON performance.improvements (source, source_ref) WHERE source_ref IS NOT NULL;
CREATE INDEX improvements_employee_idx ON performance.improvements (employee_id) WHERE deleted_at IS NULL;
CREATE INDEX improvements_department_idx ON performance.improvements (department_id) WHERE deleted_at IS NULL;

CREATE TABLE performance.retro_sessions (
    id            text PRIMARY KEY,
    department_id text NOT NULL,
    period_year   integer NOT NULL,
    period_month  integer NOT NULL,
    went_well     text[] NOT NULL DEFAULT '{}',
    to_improve    text[] NOT NULL DEFAULT '{}',
    -- [{description, ownerEmployeeId, dueDate, improvementId}]
    actions       jsonb NOT NULL DEFAULT '[]',
    status        text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'closed')),
    created_by    text,
    updated_by    text,
    closed_at     timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (department_id, period_year, period_month)
);

-- +goose Down
DROP SCHEMA IF EXISTS performance CASCADE;
