-- +goose Up
CREATE SCHEMA IF NOT EXISTS recruitment;

-- ─── Job openings ────────────────────────────────────────────

CREATE TABLE recruitment.job_openings (
    id                     text PRIMARY KEY,
    title                  text NOT NULL,
    slug                   text NOT NULL,
    department_id          text,
    department             text NOT NULL DEFAULT 'Operations',
    location               text NOT NULL DEFAULT 'Jakarta, ID',
    employment_type        text NOT NULL DEFAULT 'Full-time',
    work_mode              text NOT NULL DEFAULT 'On-site',
    headcount              integer NOT NULL DEFAULT 1 CHECK (headcount > 0),
    description            text,
    requirements           text,
    benefits               text,
    salary_min             numeric(18,2) CHECK (salary_min IS NULL OR salary_min >= 0),
    salary_max             numeric(18,2) CHECK (salary_max IS NULL OR salary_max >= 0),
    hiring_manager_user_id text,
    status                 text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'closed')),
    published_at           timestamptz,
    closing_date           date,
    created_by             text,
    updated_by             text,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    deleted_at             timestamptz
);

CREATE UNIQUE INDEX job_openings_slug_uq ON recruitment.job_openings (slug) WHERE deleted_at IS NULL;
CREATE INDEX job_openings_status_idx ON recruitment.job_openings (status) WHERE deleted_at IS NULL;

-- ─── Candidates ──────────────────────────────────────────────

CREATE TABLE recruitment.candidates (
    id               text PRIMARY KEY,
    full_name        text NOT NULL,
    email            text NOT NULL,
    phone            text NOT NULL,
    domicile         text NOT NULL,
    source           text NOT NULL,
    job_opening_id   text REFERENCES recruitment.job_openings(id) ON DELETE SET NULL,
    status           text NOT NULL DEFAULT 'applied' CHECK (status IN (
                         'applied', 'screening', 'psikotes', 'interview', 'offer',
                         'hired', 'talent_pool', 'rejected', 'archived')),
    notes            text,
    last_experience  text,
    last_education   text,
    availability     text CHECK (availability IS NULL OR availability IN ('immediate', '1_week', '2_weeks', '1_month')),
    expected_salary  bigint CHECK (expected_salary IS NULL OR expected_salary >= 0),
    cv_key           text,
    cv_file_name     text,
    cv_content_type  text,
    hired_at         timestamptz,
    -- When white.recruitment.candidate.hired went out (retried while NULL).
    hired_event_at   timestamptz,
    created_by       text,
    updated_by       text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX candidates_status_idx ON recruitment.candidates (status, created_at DESC);
CREATE INDEX candidates_opening_idx ON recruitment.candidates (job_opening_id);
CREATE INDEX candidates_email_idx ON recruitment.candidates (lower(email));

CREATE TABLE recruitment.candidate_notes (
    id              text PRIMARY KEY,
    candidate_id    text NOT NULL REFERENCES recruitment.candidates(id) ON DELETE CASCADE,
    content         text NOT NULL,
    created_by      text,
    created_by_name text,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX candidate_notes_candidate_idx ON recruitment.candidate_notes (candidate_id, created_at DESC);

CREATE TABLE recruitment.candidate_activities (
    id              text PRIMARY KEY,
    candidate_id    text NOT NULL REFERENCES recruitment.candidates(id) ON DELETE CASCADE,
    activity_type   text NOT NULL,
    description     text NOT NULL,
    created_by      text,
    created_by_name text,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX candidate_activities_candidate_idx ON recruitment.candidate_activities (candidate_id, created_at DESC);

CREATE TABLE recruitment.candidate_screenings (
    candidate_id      text PRIMARY KEY REFERENCES recruitment.candidates(id) ON DELETE CASCADE,
    contacted         boolean NOT NULL DEFAULT false,
    interested        boolean,
    availability_note text,
    confirmed_salary  bigint CHECK (confirmed_salary IS NULL OR confirmed_salary >= 0),
    willing_shift     boolean,
    willing_placement boolean,
    notes             text,
    recommendation    text CHECK (recommendation IN ('lolos', 'hold', 'tidak_lolos')),
    updated_by        text,
    updated_by_name   text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE recruitment.candidate_psikotes_summary (
    candidate_id    text PRIMARY KEY REFERENCES recruitment.candidates(id) ON DELETE CASCADE,
    recommendation  text CHECK (recommendation IN ('lolos', 'hold', 'tidak_lolos')),
    notes           text,
    updated_by      text,
    updated_by_name text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE recruitment.candidate_ai_analysis (
    candidate_id text PRIMARY KEY REFERENCES recruitment.candidates(id) ON DELETE CASCADE,
    summary      text NOT NULL DEFAULT '',
    match_score  integer NOT NULL CHECK (match_score BETWEEN 0 AND 100),
    match_reason text NOT NULL DEFAULT '',
    strengths    jsonb NOT NULL DEFAULT '[]'::jsonb,
    gaps         jsonb NOT NULL DEFAULT '[]'::jsonb,
    job_context  text NOT NULL DEFAULT '',
    model        text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- ─── Psikotes ────────────────────────────────────────────────

CREATE TABLE recruitment.psikotes_instruments (
    id               text PRIMARY KEY,
    code             text NOT NULL UNIQUE,
    name             text NOT NULL,
    kind             text NOT NULL CHECK (kind IN ('mcq', 'forced_choice', 'drawing')),
    duration_seconds integer NOT NULL DEFAULT 600 CHECK (duration_seconds BETWEEN 30 AND 14400),
    question_count   integer CHECK (question_count IS NULL OR question_count BETWEEN 1 AND 200),
    shuffle          boolean NOT NULL DEFAULT false,
    instructions     text NOT NULL DEFAULT '',
    is_active        boolean NOT NULL DEFAULT true,
    sort_order       integer NOT NULL DEFAULT 0,
    updated_by       text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

-- options: MCQ [{key, text}] or forced choice {a: {text, scale}, b: {text, scale}}.
-- answer_key: MCQ {correct}; never sent to candidates.
CREATE TABLE recruitment.psikotes_questions (
    id            text PRIMARY KEY,
    instrument_id text NOT NULL REFERENCES recruitment.psikotes_instruments(id) ON DELETE CASCADE,
    body          text NOT NULL DEFAULT '',
    options       jsonb NOT NULL DEFAULT '[]'::jsonb,
    answer_key    jsonb,
    sort_order    integer NOT NULL DEFAULT 0,
    is_active     boolean NOT NULL DEFAULT true,
    created_by    text,
    updated_by    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX psikotes_questions_bank_idx ON recruitment.psikotes_questions (instrument_id, is_active, sort_order);

CREATE TABLE recruitment.psikotes_sessions (
    id              text PRIMARY KEY,
    candidate_id    text NOT NULL REFERENCES recruitment.candidates(id) ON DELETE CASCADE,
    token           text NOT NULL UNIQUE,
    status          text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'sent', 'in_progress', 'completed', 'expired')),
    webcam_consent  boolean,
    invited_at      timestamptz,
    expires_at      timestamptz,
    started_at      timestamptz,
    completed_at    timestamptz,
    created_by      text,
    created_by_name text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX psikotes_sessions_candidate_idx ON recruitment.psikotes_sessions (candidate_id, created_at DESC);

CREATE TABLE recruitment.psikotes_session_tests (
    id                      text PRIMARY KEY,
    session_id              text NOT NULL REFERENCES recruitment.psikotes_sessions(id) ON DELETE CASCADE,
    instrument_id           text NOT NULL REFERENCES recruitment.psikotes_instruments(id),
    status                  text NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'in_progress', 'selesai', 'perlu_review', 'reviewed')),
    question_ids            jsonb NOT NULL DEFAULT '[]'::jsonb,
    answers                 jsonb NOT NULL DEFAULT '{}'::jsonb,
    score                   numeric(5,2) CHECK (score IS NULL OR (score >= 0 AND score <= 100)),
    score_detail            jsonb,
    attachment_key          text,
    attachment_content_type text,
    review_notes            text,
    reviewed_by             text,
    reviewed_by_name        text,
    reviewed_at             timestamptz,
    sort_order              integer NOT NULL DEFAULT 0,
    started_at              timestamptz,
    completed_at            timestamptz,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    UNIQUE (session_id, instrument_id)
);

CREATE INDEX psikotes_session_tests_session_idx ON recruitment.psikotes_session_tests (session_id, sort_order);

-- ─── AI interview ────────────────────────────────────────────

CREATE TABLE recruitment.interview_ai_sessions (
    id              text PRIMARY KEY,
    candidate_id    text NOT NULL REFERENCES recruitment.candidates(id) ON DELETE CASCADE,
    token           text NOT NULL UNIQUE,
    status          text NOT NULL DEFAULT 'sent' CHECK (status IN ('sent', 'in_progress', 'completed', 'expired')),
    webcam_consent  boolean,
    max_questions   integer NOT NULL DEFAULT 8 CHECK (max_questions BETWEEN 3 AND 15),
    ai_summary      jsonb,
    summary_model   text,
    summarized_at   timestamptz,
    invited_at      timestamptz,
    expires_at      timestamptz,
    started_at      timestamptz,
    completed_at    timestamptz,
    created_by      text,
    created_by_name text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX interview_ai_sessions_candidate_idx ON recruitment.interview_ai_sessions (candidate_id, created_at DESC);

CREATE TABLE recruitment.interview_ai_turns (
    id                text PRIMARY KEY,
    session_id        text NOT NULL REFERENCES recruitment.interview_ai_sessions(id) ON DELETE CASCADE,
    turn_no           integer NOT NULL,
    topic             text,
    question          text NOT NULL,
    answer_transcript text,
    answer_mode       text CHECK (answer_mode IS NULL OR answer_mode IN ('voice', 'text')),
    asked_at          timestamptz NOT NULL DEFAULT now(),
    answered_at       timestamptz,
    UNIQUE (session_id, turn_no)
);

-- Proctoring flags of both portals (snapshots are not stored).
CREATE TABLE recruitment.proctor_events (
    id           text PRIMARY KEY,
    session_type text NOT NULL CHECK (session_type IN ('psikotes', 'interview')),
    session_id   text NOT NULL,
    event_type   text NOT NULL,
    meta         jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX proctor_events_session_idx ON recruitment.proctor_events (session_type, session_id, created_at);

-- ─── Scheduled interviews ────────────────────────────────────

CREATE TABLE recruitment.interviews (
    id                      text PRIMARY KEY,
    candidate_id            text NOT NULL REFERENCES recruitment.candidates(id) ON DELETE CASCADE,
    interviewer_employee_id text,
    interviewer_user_id     text,
    interviewer_name        text NOT NULL DEFAULT '',
    interview_date          timestamptz NOT NULL,
    duration_minutes        integer NOT NULL DEFAULT 60 CHECK (duration_minutes BETWEEN 15 AND 480),
    type                    text NOT NULL CHECK (type IN ('hrd', 'hiring_manager')),
    mode                    text NOT NULL DEFAULT 'offline' CHECK (mode IN ('offline', 'online')),
    meeting_link            text,
    location                text,
    status                  text NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'completed', 'cancelled')),
    scorecard               jsonb NOT NULL DEFAULT '[]'::jsonb,
    overall_score           numeric(4,2),
    recommendation          text CHECK (recommendation IN ('proceed', 'pool', 'reject')),
    notes                   text,
    cancel_reason           text,
    created_by              text,
    updated_by              text,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX interviews_candidate_idx ON recruitment.interviews (candidate_id, interview_date DESC);
CREATE INDEX interviews_date_idx ON recruitment.interviews (interview_date) WHERE status = 'scheduled';

-- ─── Offers ──────────────────────────────────────────────────

CREATE TABLE recruitment.candidate_offers (
    id              text PRIMARY KEY,
    candidate_id    text NOT NULL REFERENCES recruitment.candidates(id) ON DELETE CASCADE,
    version         integer NOT NULL DEFAULT 1,
    token           text NOT NULL UNIQUE,
    status          text NOT NULL DEFAULT 'sent' CHECK (status IN ('sent', 'negotiating', 'accepted', 'declined', 'expired')),
    position_title  text,
    base_salary     numeric(18,2) NOT NULL CHECK (base_salary >= 0),
    benefits        jsonb NOT NULL DEFAULT '[]'::jsonb,
    start_date      date,
    notes           text,
    response_note   text,
    responded_at    timestamptz,
    response_ip     text,
    response_source text CHECK (response_source IS NULL OR response_source IN ('portal', 'manual')),
    sent_at         timestamptz,
    expires_at      timestamptz,
    created_by      text,
    created_by_name text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (candidate_id, version)
);

-- ─── Plumbing ────────────────────────────────────────────────

-- Fixed-window rate limits of the public portals, shared by replicas.
CREATE TABLE recruitment.rate_limits (
    key          text PRIMARY KEY,
    window_start timestamptz NOT NULL,
    hits         integer NOT NULL
);

-- One reminder per kind, record and day.
CREATE TABLE recruitment.reminder_sends (
    kind    text NOT NULL,
    ref_id  text NOT NULL,
    day     date NOT NULL,
    sent_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, ref_id, day)
);

-- +goose Down
DROP SCHEMA IF EXISTS recruitment CASCADE;
