-- +goose Up
-- Sales: CRM pipeline (ported from NüHabit salesfunnel + crm/advance +
-- crm/inbox + crm/publicforms) plus WHITE's agency automations: quotation
-- drafts, contracts with e-signature, pitch decks, client onboarding and
-- the lead chatbot. Value sets are CHECK constraints, as in NüHabit.
CREATE SCHEMA IF NOT EXISTS sales;

-- Monthly document counters: prefix "QT-2610" -> last number used.
CREATE TABLE sales.number_sequences (
    prefix     text PRIMARY KEY,
    last_value integer NOT NULL DEFAULT 0
);

-- ── pipeline ────────────────────────────────────────────────────────────
CREATE TABLE sales.pipelines (
    id         text PRIMARY KEY,
    code       text NOT NULL UNIQUE,
    name       text NOT NULL,
    is_default boolean NOT NULL DEFAULT false,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.stages (
    id                   text PRIMARY KEY,
    pipeline_id          text NOT NULL REFERENCES sales.pipelines(id) ON DELETE CASCADE,
    code                 text NOT NULL,
    name                 text NOT NULL,
    sort_order           integer NOT NULL DEFAULT 0,
    probability          integer NOT NULL DEFAULT 0 CHECK (probability BETWEEN 0 AND 100),
    is_won               boolean NOT NULL DEFAULT false,
    is_lost              boolean NOT NULL DEFAULT false,
    stuck_threshold_days integer,
    is_active            boolean NOT NULL DEFAULT true,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (pipeline_id, code),
    CHECK (NOT (is_won AND is_lost))
);

CREATE TABLE sales.lost_reasons (
    id         text PRIMARY KEY,
    code       text NOT NULL UNIQUE,
    name       text NOT NULL,
    sort_order integer NOT NULL DEFAULT 0,
    is_active  boolean NOT NULL DEFAULT true
);

-- ── accounts, contacts, leads ───────────────────────────────────────────
CREATE TABLE sales.accounts (
    id            text PRIMARY KEY,
    name          text NOT NULL,
    account_type  text NOT NULL DEFAULT 'corporate',
    industry      text,
    address       text,
    city          text,
    phone         text,
    email         text,
    website       text,
    npwp          text,
    notes         text,
    owner_user_id text,
    custom        jsonb NOT NULL DEFAULT '{}',
    created_by    text,
    updated_by    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);
CREATE INDEX accounts_name_idx ON sales.accounts (lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE sales.contacts (
    id         text PRIMARY KEY,
    account_id text REFERENCES sales.accounts(id) ON DELETE SET NULL,
    name       text NOT NULL,
    title      text,
    phone      text,
    email      text,
    is_primary boolean NOT NULL DEFAULT false,
    notes      text,
    custom     jsonb NOT NULL DEFAULT '{}',
    created_by text,
    updated_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX contacts_account_idx ON sales.contacts (account_id) WHERE deleted_at IS NULL;

CREATE TABLE sales.leads (
    id               text PRIMARY KEY,
    org_name         text NOT NULL,
    org_type         text NOT NULL DEFAULT 'corporate',
    pic_name         text NOT NULL,
    pic_title        text,
    pic_phone        text,
    pic_email        text,
    city             text,
    source           text NOT NULL DEFAULT 'lainnya',
    temperature      text NOT NULL DEFAULT 'hangat' CHECK (temperature IN ('panas', 'hangat', 'dingin')),
    status           text NOT NULL DEFAULT 'baru' CHECK (status IN ('baru', 'dihubungi', 'qualified', 'tidak-cocok')),
    notes            text,
    owner_user_id    text,
    account_id       text REFERENCES sales.accounts(id) ON DELETE SET NULL,
    contact_id       text REFERENCES sales.contacts(id) ON DELETE SET NULL,
    custom           jsonb NOT NULL DEFAULT '{}',
    score            integer NOT NULL DEFAULT 0,
    score_breakdown  jsonb NOT NULL DEFAULT '[]',
    score_updated_at timestamptz,
    utm_source       text,
    utm_medium       text,
    utm_campaign     text,
    utm_term         text,
    utm_content      text,
    landing_page     text,
    referrer         text,
    created_by       text,
    updated_by       text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    deleted_at       timestamptz
);
CREATE INDEX leads_phone_idx ON sales.leads (pic_phone) WHERE deleted_at IS NULL;
CREATE INDEX leads_email_idx ON sales.leads (lower(pic_email)) WHERE deleted_at IS NULL;
CREATE INDEX leads_owner_idx ON sales.leads (owner_user_id) WHERE deleted_at IS NULL;

-- ── deals ───────────────────────────────────────────────────────────────
CREATE TABLE sales.deals (
    id                  text PRIMARY KEY,
    lead_id             text REFERENCES sales.leads(id) ON DELETE SET NULL,
    account_id          text REFERENCES sales.accounts(id) ON DELETE SET NULL,
    title               text NOT NULL,
    service_lines       text[] NOT NULL DEFAULT '{}',
    pipeline_id         text NOT NULL REFERENCES sales.pipelines(id),
    stage_id            text NOT NULL REFERENCES sales.stages(id),
    value_estimate      numeric(18,2),
    value_final         numeric(18,2),
    owner_user_id       text,
    lost_reason_id      text REFERENCES sales.lost_reasons(id),
    expected_close_date date,
    notes               text,
    forecast_category   text NOT NULL DEFAULT 'pipeline' CHECK (forecast_category IN ('pipeline', 'best_case', 'commit', 'closed_won', 'closed_lost')),
    entered_stage_at    timestamptz NOT NULL DEFAULT now(),
    closed_at           timestamptz,
    custom              jsonb NOT NULL DEFAULT '{}',
    created_by          text,
    updated_by          text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz
);
CREATE INDEX deals_stage_idx ON sales.deals (stage_id) WHERE deleted_at IS NULL;
CREATE INDEX deals_lead_idx ON sales.deals (lead_id) WHERE deleted_at IS NULL;

CREATE TABLE sales.deal_members (
    deal_id       text NOT NULL REFERENCES sales.deals(id) ON DELETE CASCADE,
    user_id       text NOT NULL,
    role          text NOT NULL DEFAULT 'support' CHECK (role IN ('owner', 'support', 'pre_sales', 'account_manager', 'finance')),
    split_percent numeric(5,2) NOT NULL DEFAULT 0 CHECK (split_percent BETWEEN 0 AND 100),
    created_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deal_id, user_id)
);

CREATE TABLE sales.deal_stage_history (
    id         text PRIMARY KEY,
    deal_id    text NOT NULL REFERENCES sales.deals(id) ON DELETE CASCADE,
    stage_id   text NOT NULL REFERENCES sales.stages(id),
    entered_at timestamptz NOT NULL DEFAULT now(),
    moved_by   text
);
CREATE INDEX deal_stage_history_deal_idx ON sales.deal_stage_history (deal_id, entered_at);

-- ── activities / tasks ──────────────────────────────────────────────────
CREATE TABLE sales.activities (
    id               text PRIMARY KEY,
    lead_id          text REFERENCES sales.leads(id) ON DELETE CASCADE,
    deal_id          text REFERENCES sales.deals(id) ON DELETE CASCADE,
    subject_type     text CHECK (subject_type IN ('lead', 'deal', 'account', 'contact')),
    subject_id       text,
    activity_type    text NOT NULL CHECK (activity_type IN ('telepon', 'wa', 'meeting', 'catatan', 'tugas', 'email')),
    title            text NOT NULL,
    notes            text,
    priority         text NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    status           text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'done', 'cancelled')),
    due_at           timestamptz,
    done_at          timestamptz,
    reminder_at      timestamptz,
    reminder_sent_at timestamptz,
    recurrence       jsonb,
    parent_task_id   text REFERENCES sales.activities(id) ON DELETE SET NULL,
    owner_user_id    text,
    created_by       text,
    updated_by       text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    deleted_at       timestamptz
);
CREATE INDEX activities_due_reminder_idx ON sales.activities (reminder_at) WHERE reminder_sent_at IS NULL AND deleted_at IS NULL;
CREATE INDEX activities_lead_idx ON sales.activities (lead_id);
CREATE INDEX activities_deal_idx ON sales.activities (deal_id);

-- ── service catalog and pricing ─────────────────────────────────────────
CREATE TABLE sales.service_catalog (
    id                  text PRIMARY KEY,
    code                text NOT NULL UNIQUE,
    name                text NOT NULL,
    service_line        text NOT NULL CHECK (service_line IN ('social_media_management', 'seo', 'ads_management', 'content_production', 'website', 'branding')),
    unit                text NOT NULL DEFAULT 'paket',
    default_price       numeric(18,2) NOT NULL DEFAULT 0,
    description         text NOT NULL DEFAULT '',
    is_retainer         boolean NOT NULL DEFAULT false,
    is_default_for_line boolean NOT NULL DEFAULT false,
    is_active           boolean NOT NULL DEFAULT true,
    sort_order          integer NOT NULL DEFAULT 0,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

-- Package tiers: a tier price overrides default_price.
CREATE TABLE sales.catalog_tiers (
    catalog_id  text NOT NULL REFERENCES sales.service_catalog(id) ON DELETE CASCADE,
    tier        text NOT NULL CHECK (tier IN ('basic', 'standard', 'premium')),
    price       numeric(18,2) NOT NULL,
    description text NOT NULL DEFAULT '',
    PRIMARY KEY (catalog_id, tier)
);

-- Retainer length discounts: the highest min_months reached applies.
CREATE TABLE sales.retainer_discounts (
    min_months       integer PRIMARY KEY CHECK (min_months > 0),
    discount_percent numeric(5,2) NOT NULL CHECK (discount_percent BETWEEN 0 AND 100)
);

-- ── quotations ──────────────────────────────────────────────────────────
CREATE TABLE sales.quotations (
    id                  text PRIMARY KEY,
    deal_id             text NOT NULL REFERENCES sales.deals(id) ON DELETE CASCADE,
    quote_number        text NOT NULL UNIQUE,
    version             integer NOT NULL DEFAULT 1,
    parent_quotation_id text REFERENCES sales.quotations(id) ON DELETE SET NULL,
    status              text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'terkirim', 'diterima', 'ditolak', 'superseded')),
    use_ppn             boolean NOT NULL DEFAULT true,
    ppn_percent         numeric(5,2) NOT NULL DEFAULT 11,
    subtotal            numeric(18,2) NOT NULL DEFAULT 0,
    discount_percent    numeric(5,2) NOT NULL DEFAULT 0,
    discount_nominal    numeric(18,2) NOT NULL DEFAULT 0,
    ppn_nominal         numeric(18,2) NOT NULL DEFAULT 0,
    total               numeric(18,2) NOT NULL DEFAULT 0,
    notes               text,
    valid_until         date,
    approval_status     text NOT NULL DEFAULT 'none' CHECK (approval_status IN ('none', 'pending', 'approved', 'rejected')),
    approval_request_id text,
    generated_by        text,
    sent_at             timestamptz,
    decided_at          timestamptz,
    superseded_at       timestamptz,
    expiry_reminded_at  timestamptz,
    created_by          text,
    updated_by          text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz
);
CREATE INDEX quotations_deal_idx ON sales.quotations (deal_id) WHERE deleted_at IS NULL;

CREATE TABLE sales.quotation_items (
    id           text PRIMARY KEY,
    quotation_id text NOT NULL REFERENCES sales.quotations(id) ON DELETE CASCADE,
    item_type    text NOT NULL DEFAULT 'bebas' CHECK (item_type IN ('katalog', 'bebas')),
    catalog_id   text REFERENCES sales.service_catalog(id) ON DELETE SET NULL,
    description  text NOT NULL,
    scope        text NOT NULL DEFAULT '',
    unit         text NOT NULL DEFAULT 'paket',
    qty          numeric(12,3) NOT NULL DEFAULT 1,
    unit_price   numeric(18,2) NOT NULL DEFAULT 0,
    line_total   numeric(18,2) NOT NULL DEFAULT 0,
    sort_order   integer NOT NULL DEFAULT 0
);
CREATE INDEX quotation_items_quotation_idx ON sales.quotation_items (quotation_id, sort_order);

CREATE TABLE sales.quotation_terms (
    id           text PRIMARY KEY,
    quotation_id text NOT NULL REFERENCES sales.quotations(id) ON DELETE CASCADE,
    label        text NOT NULL,
    percent      numeric(5,2) NOT NULL,
    due_date     date,
    sort_order   integer NOT NULL DEFAULT 0
);
CREATE INDEX quotation_terms_quotation_idx ON sales.quotation_terms (quotation_id, sort_order);

-- ── discount approvals (crm advance) ────────────────────────────────────
CREATE TABLE sales.approval_rules (
    id                   text PRIMARY KEY,
    name                 text NOT NULL,
    object               text NOT NULL DEFAULT 'quotation' CHECK (object = 'quotation'),
    level                integer NOT NULL CHECK (level BETWEEN 1 AND 5),
    min_discount_percent numeric(5,2) NOT NULL CHECK (min_discount_percent BETWEEN 0 AND 100),
    -- 'sales.approve' (anyone holding the key) or 'ADMIN'; or a named user.
    approver_role        text,
    approver_user_id     text,
    is_active            boolean NOT NULL DEFAULT true,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    CHECK (approver_role IS NOT NULL OR approver_user_id IS NOT NULL)
);

CREATE TABLE sales.approval_requests (
    id               text PRIMARY KEY,
    object           text NOT NULL DEFAULT 'quotation',
    subject_id       text NOT NULL,
    status           text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    current_level    integer NOT NULL DEFAULT 1,
    discount_percent numeric(5,2) NOT NULL,
    amount           numeric(18,2) NOT NULL,
    note             text,
    requested_by     text,
    resolved_by      text,
    resolved_at      timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX approval_requests_subject_idx ON sales.approval_requests (subject_id);

CREATE TABLE sales.approval_steps (
    id               text PRIMARY KEY,
    request_id       text NOT NULL REFERENCES sales.approval_requests(id) ON DELETE CASCADE,
    level            integer NOT NULL,
    approver_role    text,
    approver_user_id text,
    status           text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'skipped')),
    decided_by       text,
    decided_at       timestamptz,
    comment          text,
    UNIQUE (request_id, level)
);

-- ── invoices and payments ───────────────────────────────────────────────
CREATE TABLE sales.invoices (
    id               text PRIMARY KEY,
    deal_id          text NOT NULL REFERENCES sales.deals(id) ON DELETE CASCADE,
    quotation_id     text REFERENCES sales.quotations(id) ON DELETE SET NULL,
    term_id          text REFERENCES sales.quotation_terms(id) ON DELETE SET NULL,
    invoice_number   text NOT NULL UNIQUE,
    label            text NOT NULL,
    amount           numeric(18,2) NOT NULL CHECK (amount > 0),
    due_date         date,
    status           text NOT NULL DEFAULT 'diajukan' CHECK (status IN ('diajukan', 'terkirim', 'batal')),
    sent_at          timestamptz,
    issued_by        text,
    cancelled_at     timestamptz,
    cancelled_by     text,
    cancel_reason    text,
    created_by       text,
    updated_by       text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX invoices_deal_idx ON sales.invoices (deal_id);

CREATE TABLE sales.payments (
    id         text PRIMARY KEY,
    deal_id    text NOT NULL REFERENCES sales.deals(id) ON DELETE CASCADE,
    invoice_id text REFERENCES sales.invoices(id) ON DELETE SET NULL,
    amount     numeric(18,2) NOT NULL CHECK (amount > 0),
    method     text NOT NULL CHECK (method IN ('cash', 'transfer', 'qris', 'edc', 'lainnya')),
    paid_on    date NOT NULL,
    note       text,
    created_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX payments_deal_idx ON sales.payments (deal_id) WHERE deleted_at IS NULL;

-- ── targets ─────────────────────────────────────────────────────────────
CREATE TABLE sales.targets (
    id           text PRIMARY KEY,
    user_id      text,
    period_month text NOT NULL CHECK (period_month ~ '^\d{4}-\d{2}$'),
    target_value numeric(18,2) NOT NULL DEFAULT 0,
    target_deals integer,
    updated_by   text,
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX targets_user_month_idx ON sales.targets (coalesce(user_id, ''), period_month);

-- ── custom fields, events, workflow, scoring (crm advance) ──────────────
CREATE TABLE sales.custom_fields (
    id           text PRIMARY KEY,
    object       text NOT NULL CHECK (object IN ('lead', 'deal', 'account', 'contact')),
    key          text NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]{1,39}$'),
    label        text NOT NULL,
    field_type   text NOT NULL CHECK (field_type IN ('text', 'textarea', 'number', 'date', 'boolean', 'picklist', 'multipicklist', 'url', 'email', 'phone')),
    options      jsonb NOT NULL DEFAULT '[]',
    is_required  boolean NOT NULL DEFAULT false,
    show_in_list boolean NOT NULL DEFAULT false,
    sort_order   integer NOT NULL DEFAULT 0,
    is_active    boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (object, key)
);

CREATE TABLE sales.crm_events (
    id            text PRIMARY KEY,
    event_type    text NOT NULL,
    subject_type  text NOT NULL,
    subject_id    text NOT NULL,
    payload       jsonb NOT NULL DEFAULT '{}',
    actor_user_id text,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX crm_events_subject_idx ON sales.crm_events (subject_type, subject_id, event_type, created_at);

CREATE TABLE sales.workflow_rules (
    id                  text PRIMARY KEY,
    name                text NOT NULL,
    object              text NOT NULL CHECK (object IN ('lead', 'deal', 'account', 'contact', 'task', 'quotation')),
    trigger_type        text NOT NULL CHECK (trigger_type IN ('created', 'updated', 'stage_changed', 'status_changed', 'score_reached', 'inactive_days', 'due_soon')),
    trigger_config      jsonb NOT NULL DEFAULT '{}',
    conditions          jsonb NOT NULL DEFAULT '[]',
    actions             jsonb NOT NULL DEFAULT '[]',
    run_once_per_record boolean NOT NULL DEFAULT false,
    is_active           boolean NOT NULL DEFAULT true,
    run_count           integer NOT NULL DEFAULT 0,
    last_run_at         timestamptz,
    created_by          text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.workflow_runs (
    id             text PRIMARY KEY,
    rule_id        text NOT NULL REFERENCES sales.workflow_rules(id) ON DELETE CASCADE,
    event_id       text,
    subject_type   text NOT NULL,
    subject_id     text NOT NULL,
    status         text NOT NULL CHECK (status IN ('success', 'partial', 'failed', 'skipped', 'scheduled')),
    actions_result jsonb NOT NULL DEFAULT '[]',
    error          text,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX workflow_runs_rule_idx ON sales.workflow_runs (rule_id, subject_id, created_at);

CREATE TABLE sales.scheduled_actions (
    id           text PRIMARY KEY,
    rule_id      text REFERENCES sales.workflow_rules(id) ON DELETE CASCADE,
    run_id       text,
    subject_type text NOT NULL,
    subject_id   text NOT NULL,
    actions      jsonb NOT NULL,
    context      jsonb NOT NULL DEFAULT '{}',
    run_at       timestamptz NOT NULL,
    status       text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'done', 'failed', 'cancelled')),
    attempts     integer NOT NULL DEFAULT 0,
    last_error   text,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX scheduled_actions_due_idx ON sales.scheduled_actions (run_at) WHERE status = 'pending';

CREATE TABLE sales.scoring_rules (
    id          text PRIMARY KEY,
    name        text NOT NULL,
    kind        text NOT NULL CHECK (kind IN ('field', 'event')),
    field       text,
    operator    text,
    value       jsonb,
    event_type  text,
    window_days integer,
    max_count   integer NOT NULL DEFAULT 1,
    points      integer NOT NULL,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- ── public lead forms ───────────────────────────────────────────────────
CREATE TABLE sales.forms (
    id               text PRIMARY KEY,
    slug             text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,60}$'),
    name             text NOT NULL,
    description      text NOT NULL DEFAULT '',
    fields           jsonb NOT NULL DEFAULT '[]',
    submit_label     text NOT NULL DEFAULT 'Kirim',
    success_message  text NOT NULL DEFAULT 'Terima kasih, tim kami akan segera menghubungi Anda.',
    redirect_url     text,
    default_source   text NOT NULL DEFAULT 'website',
    lead_temperature text NOT NULL DEFAULT 'hangat',
    notify_user_ids  text[] NOT NULL DEFAULT '{}',
    is_active        boolean NOT NULL DEFAULT true,
    submission_count integer NOT NULL DEFAULT 0,
    created_by       text,
    updated_by       text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.form_submissions (
    id         text PRIMARY KEY,
    form_id    text NOT NULL REFERENCES sales.forms(id) ON DELETE CASCADE,
    lead_id    text REFERENCES sales.leads(id) ON DELETE SET NULL,
    payload    jsonb NOT NULL DEFAULT '{}',
    utm        jsonb NOT NULL DEFAULT '{}',
    ip_hash    text NOT NULL DEFAULT '',
    status     text NOT NULL CHECK (status IN ('accepted', 'spam', 'invalid')),
    reason     text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX form_submissions_form_idx ON sales.form_submissions (form_id, created_at);

-- ── contracts ───────────────────────────────────────────────────────────
CREATE TABLE sales.contract_templates (
    id         text PRIMARY KEY,
    name       text NOT NULL,
    type       text NOT NULL CHECK (type IN ('retainer', 'project', 'nda')),
    body       text NOT NULL,
    is_active  boolean NOT NULL DEFAULT true,
    created_by text,
    updated_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.contracts (
    id                text PRIMARY KEY,
    contract_number   text NOT NULL UNIQUE,
    template_id       text REFERENCES sales.contract_templates(id) ON DELETE SET NULL,
    quotation_id      text REFERENCES sales.quotations(id) ON DELETE SET NULL,
    deal_id           text NOT NULL REFERENCES sales.deals(id) ON DELETE CASCADE,
    type              text NOT NULL CHECK (type IN ('retainer', 'project', 'nda')),
    title             text NOT NULL,
    client_name       text NOT NULL,
    body              text NOT NULL,
    status            text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'sent', 'signed', 'active', 'ended', 'cancelled')),
    start_date        date,
    end_date          date,
    sign_token        text NOT NULL UNIQUE,
    sent_at           timestamptz,
    signed_at         timestamptz,
    signer_name       text,
    signer_ip         text,
    signer_user_agent text,
    signature_png     bytea,
    activated_at      timestamptz,
    ended_at          timestamptz,
    cancelled_at      timestamptz,
    cancel_reason     text,
    created_by        text,
    updated_by        text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX contracts_deal_idx ON sales.contracts (deal_id);

-- ── pitch decks ─────────────────────────────────────────────────────────
CREATE TABLE sales.case_studies (
    id            text PRIMARY KEY,
    title         text NOT NULL,
    client_name   text NOT NULL,
    industry      text NOT NULL DEFAULT '',
    service_lines text[] NOT NULL DEFAULT '{}',
    challenge     text NOT NULL DEFAULT '',
    solution      text NOT NULL DEFAULT '',
    results       text NOT NULL DEFAULT '',
    sort_order    integer NOT NULL DEFAULT 0,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.pitch_decks (
    id         text PRIMARY KEY,
    deal_id    text REFERENCES sales.deals(id) ON DELETE CASCADE,
    lead_id    text REFERENCES sales.leads(id) ON DELETE CASCADE,
    title      text NOT NULL,
    version    integer NOT NULL,
    options    jsonb NOT NULL DEFAULT '{}',
    slides     jsonb NOT NULL DEFAULT '[]',
    file       bytea NOT NULL,
    model      text NOT NULL,
    created_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (deal_id IS NOT NULL OR lead_id IS NOT NULL)
);
CREATE INDEX pitch_decks_subject_idx ON sales.pitch_decks (deal_id, lead_id, version);

-- ── client onboarding ───────────────────────────────────────────────────
CREATE TABLE sales.onboarding_templates (
    id         text PRIMARY KEY,
    name       text NOT NULL,
    is_default boolean NOT NULL DEFAULT false,
    items      jsonb NOT NULL DEFAULT '[]',
    updated_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.onboardings (
    id                 text PRIMARY KEY,
    deal_id            text NOT NULL UNIQUE REFERENCES sales.deals(id) ON DELETE CASCADE,
    template_id        text REFERENCES sales.onboarding_templates(id) ON DELETE SET NULL,
    client_name        text NOT NULL,
    status             text NOT NULL DEFAULT 'aktif' CHECK (status IN ('aktif', 'selesai', 'batal')),
    portal_token       text NOT NULL UNIQUE,
    brief              jsonb NOT NULL DEFAULT '{}',
    brief_submitted_at timestamptz,
    marketing_client_id text,
    progress           integer NOT NULL DEFAULT 0,
    completed_at       timestamptz,
    created_by         text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.onboarding_items (
    id            text PRIMARY KEY,
    onboarding_id text NOT NULL REFERENCES sales.onboardings(id) ON DELETE CASCADE,
    key           text NOT NULL,
    kind          text NOT NULL DEFAULT 'manual' CHECK (kind IN ('manual', 'brief', 'assets', 'contract', 'invoice_paid', 'create_client')),
    title         text NOT NULL,
    description   text NOT NULL DEFAULT '',
    owner_user_id text,
    due_date      date,
    status        text NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'done', 'skipped')),
    done_at       timestamptz,
    done_by       text,
    reminded_at   timestamptz,
    sort_order    integer NOT NULL DEFAULT 0,
    UNIQUE (onboarding_id, key)
);

CREATE TABLE sales.onboarding_assets (
    id            text PRIMARY KEY,
    onboarding_id text NOT NULL REFERENCES sales.onboardings(id) ON DELETE CASCADE,
    file_name     text NOT NULL,
    content_type  text NOT NULL,
    size_bytes    bigint NOT NULL,
    storage_key   text NOT NULL,
    uploader_ip   text,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- ── lead conversations (ported crm inbox) and the chatbot ───────────────
CREATE TABLE sales.lead_conversations (
    id                     text PRIMARY KEY,
    channel                text NOT NULL CHECK (channel IN ('web', 'whatsapp')),
    external_id            text NOT NULL,
    visitor_token          text UNIQUE,
    lead_id                text REFERENCES sales.leads(id) ON DELETE SET NULL,
    display_name           text,
    phone                  text,
    email                  text,
    status                 text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'waiting_customer', 'resolved')),
    priority               text NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    assigned_user_id       text,
    bot_active             boolean NOT NULL DEFAULT true,
    handoff_reason         text,
    handoff_at             timestamptz,
    is_complaint           boolean NOT NULL DEFAULT false,
    unread_count           integer NOT NULL DEFAULT 0,
    last_message_at        timestamptz,
    last_message_preview   text,
    awaiting_since         timestamptz,
    first_response_at      timestamptz,
    first_response_seconds integer,
    sla_response_breached  boolean NOT NULL DEFAULT false,
    resolved_at            timestamptz,
    resolution_seconds     integer,
    page_url               text,
    utm                    jsonb NOT NULL DEFAULT '{}',
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    UNIQUE (channel, external_id)
);
CREATE INDEX lead_conversations_last_idx ON sales.lead_conversations (last_message_at DESC);

CREATE TABLE sales.lead_messages (
    id                  text PRIMARY KEY,
    conversation_id     text NOT NULL REFERENCES sales.lead_conversations(id) ON DELETE CASCADE,
    direction           text NOT NULL CHECK (direction IN ('in', 'out')),
    sender              text NOT NULL CHECK (sender IN ('customer', 'bot', 'agent', 'system')),
    body                text NOT NULL,
    status              text NOT NULL DEFAULT 'sent' CHECK (status IN ('queued', 'sent', 'failed', 'received')),
    provider            text,
    provider_message_id text,
    error_reason        text,
    sent_by_user_id     text,
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX lead_messages_conversation_idx ON sales.lead_messages (conversation_id, created_at, id);
CREATE UNIQUE INDEX lead_messages_provider_idx ON sales.lead_messages (provider, provider_message_id) WHERE provider_message_id IS NOT NULL;

CREATE TABLE sales.conversation_notes (
    id              text PRIMARY KEY,
    conversation_id text NOT NULL REFERENCES sales.lead_conversations(id) ON DELETE CASCADE,
    author_user_id  text NOT NULL,
    body            text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.reply_templates (
    id         text PRIMARY KEY,
    title      text NOT NULL CHECK (char_length(title) <= 80),
    body       text NOT NULL CHECK (char_length(body) <= 2000),
    is_active  boolean NOT NULL DEFAULT true,
    created_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.conversation_insights (
    conversation_id text PRIMARY KEY REFERENCES sales.lead_conversations(id) ON DELETE CASCADE,
    summary         text NOT NULL,
    topic           text NOT NULL,
    sentiment       text NOT NULL,
    is_complaint    boolean NOT NULL DEFAULT false,
    keywords        jsonb NOT NULL DEFAULT '[]',
    fingerprint     text NOT NULL,
    model           text NOT NULL,
    analyzed_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sales.chatbot_settings (
    id                   text PRIMARY KEY,
    enabled              boolean NOT NULL DEFAULT true,
    bot_name             text NOT NULL,
    persona              text NOT NULL,
    services_offered     text NOT NULL,
    pricing_hints        text NOT NULL,
    business_hours_start integer NOT NULL DEFAULT 9 CHECK (business_hours_start BETWEEN 0 AND 23),
    business_hours_end   integer NOT NULL DEFAULT 18 CHECK (business_hours_end BETWEEN 0 AND 23),
    handoff_rules        text NOT NULL,
    greeting             text NOT NULL,
    min_confidence       numeric(3,2) NOT NULL DEFAULT 0.55,
    -- Notified when the bot hands a chat to a human (public chats have no
    -- caller to resolve the sales team from).
    handoff_user_ids     text[] NOT NULL DEFAULT '{}',
    updated_by           text,
    updated_at           timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP SCHEMA IF EXISTS sales CASCADE;
