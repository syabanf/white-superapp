-- +goose Up
-- Finance schema: ported from the NüHabit accounting module (single
-- company, so no company_id) plus reimbursement claims and AI insights.
-- Money is numeric(18,2); ids are text (UUIDv7).
CREATE SCHEMA IF NOT EXISTS finance;

CREATE TABLE finance.account_types (
  id             text PRIMARY KEY,
  code           varchar(30) NOT NULL UNIQUE,
  name           text NOT NULL,
  normal_balance text NOT NULL CHECK (normal_balance IN ('DEBIT', 'CREDIT')),
  sort_order     int NOT NULL DEFAULT 0,
  is_active      boolean NOT NULL DEFAULT true,
  created_at     timestamptz NOT NULL DEFAULT now(),
  created_by     text,
  updated_at     timestamptz NOT NULL DEFAULT now(),
  updated_by     text
);

CREATE TABLE finance.accounts (
  id                 text PRIMARY KEY,
  code               varchar(20) NOT NULL,
  name               text NOT NULL,
  parent_id          text REFERENCES finance.accounts (id),
  account_type_id    text NOT NULL REFERENCES finance.account_types (id),
  level              int NOT NULL CHECK (level BETWEEN 1 AND 4),
  is_postable        boolean NOT NULL DEFAULT true,
  is_contra          boolean NOT NULL DEFAULT false,
  is_cash_bank       boolean NOT NULL DEFAULT false,
  cash_flow_category text CHECK (cash_flow_category IN ('OPERATING', 'INVESTING', 'FINANCING', 'NON_CASH')),
  description        text NOT NULL DEFAULT '',
  is_active          boolean NOT NULL DEFAULT true,
  created_at         timestamptz NOT NULL DEFAULT now(),
  created_by         text,
  updated_at         timestamptz NOT NULL DEFAULT now(),
  updated_by         text,
  deleted_at         timestamptz,
  deleted_by         text
);
CREATE UNIQUE INDEX accounts_code_live ON finance.accounts (code) WHERE deleted_at IS NULL;
CREATE INDEX accounts_parent ON finance.accounts (parent_id);

CREATE TABLE finance.journal_mappings (
  id          text PRIMARY KEY,
  event_code  varchar(60) NOT NULL,
  name        text NOT NULL,
  description text NOT NULL DEFAULT '',
  module      text NOT NULL,
  is_active   boolean NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL DEFAULT now(),
  created_by  text,
  updated_at  timestamptz NOT NULL DEFAULT now(),
  updated_by  text,
  deleted_at  timestamptz,
  deleted_by  text
);
CREATE UNIQUE INDEX journal_mappings_event_live ON finance.journal_mappings (event_code) WHERE deleted_at IS NULL;

CREATE TABLE finance.journal_mapping_lines (
  id            text PRIMARY KEY,
  mapping_id    text NOT NULL REFERENCES finance.journal_mappings (id) ON DELETE CASCADE,
  entry_side    text NOT NULL CHECK (entry_side IN ('DEBIT', 'CREDIT')),
  line_role     varchar(40) NOT NULL,
  account_id    text REFERENCES finance.accounts (id),
  amount_source text NOT NULL CHECK (amount_source IN ('TOTAL', 'SUBTOTAL', 'TAX', 'COGS', 'PAID', 'DISCOUNT', 'SERVICE_CHARGE')),
  sort_order    int NOT NULL DEFAULT 0,
  is_required   boolean NOT NULL DEFAULT true
);
CREATE INDEX journal_mapping_lines_mapping ON finance.journal_mapping_lines (mapping_id);

CREATE TABLE finance.fiscal_years (
  id         text PRIMARY KEY,
  code       text NOT NULL,
  name       text NOT NULL,
  start_date date NOT NULL,
  end_date   date NOT NULL,
  is_active  boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  created_by text,
  updated_at timestamptz NOT NULL DEFAULT now(),
  updated_by text,
  deleted_at timestamptz,
  deleted_by text
);
CREATE UNIQUE INDEX fiscal_years_code_live ON finance.fiscal_years (code) WHERE deleted_at IS NULL;

CREATE TABLE finance.fiscal_periods (
  id             text PRIMARY KEY,
  fiscal_year_id text NOT NULL REFERENCES finance.fiscal_years (id) ON DELETE CASCADE,
  period_no      int NOT NULL CHECK (period_no BETWEEN 1 AND 12),
  name           text NOT NULL,
  start_date     date NOT NULL,
  end_date       date NOT NULL,
  status         text NOT NULL CHECK (status IN ('OPEN', 'CLOSED')),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  UNIQUE (fiscal_year_id, period_no)
);

CREATE TABLE finance.journal_entries (
  id                    text PRIMARY KEY,
  entry_no              text NOT NULL,
  entry_date            date NOT NULL,
  description           text NOT NULL DEFAULT '',
  fiscal_period_id      text NOT NULL REFERENCES finance.fiscal_periods (id),
  status                text NOT NULL CHECK (status IN ('DRAFT', 'POSTED')),
  is_recon              boolean NOT NULL DEFAULT false,
  entry_type            text NOT NULL DEFAULT 'MANUAL' CHECK (entry_type IN ('MANUAL', 'OPENING', 'AUTO')),
  source_module         text NOT NULL DEFAULT '',
  source_event_code     text NOT NULL DEFAULT '',
  source_document_type  text NOT NULL DEFAULT '',
  source_document_id    text,
  source_fiscal_year_id text,
  posted_at             timestamptz,
  posted_by             text,
  created_at            timestamptz NOT NULL DEFAULT now(),
  created_by            text,
  updated_at            timestamptz NOT NULL DEFAULT now(),
  updated_by            text,
  deleted_at            timestamptz,
  deleted_by            text
);
CREATE UNIQUE INDEX journal_entries_no_live ON finance.journal_entries (entry_no) WHERE deleted_at IS NULL;
-- One live journal per (event, source document): auto-posting is idempotent.
CREATE UNIQUE INDEX journal_entries_source_live ON finance.journal_entries (source_event_code, source_document_id)
  WHERE deleted_at IS NULL AND source_document_id IS NOT NULL;
CREATE UNIQUE INDEX journal_entries_opening_live ON finance.journal_entries (source_fiscal_year_id)
  WHERE deleted_at IS NULL AND entry_type = 'OPENING';
CREATE INDEX journal_entries_date ON finance.journal_entries (entry_date) WHERE deleted_at IS NULL;
CREATE INDEX journal_entries_period ON finance.journal_entries (fiscal_period_id) WHERE deleted_at IS NULL;
CREATE INDEX journal_entries_doc_type ON finance.journal_entries (source_document_type) WHERE deleted_at IS NULL;

CREATE TABLE finance.journal_entry_lines (
  id         text PRIMARY KEY,
  entry_id   text NOT NULL REFERENCES finance.journal_entries (id) ON DELETE CASCADE,
  account_id text NOT NULL REFERENCES finance.accounts (id),
  entry_side text NOT NULL CHECK (entry_side IN ('DEBIT', 'CREDIT')),
  amount     numeric(18,2) NOT NULL CHECK (amount > 0),
  memo       text NOT NULL DEFAULT '',
  sort_order int NOT NULL DEFAULT 0
);
CREATE INDEX journal_entry_lines_entry ON finance.journal_entry_lines (entry_id);
CREATE INDEX journal_entry_lines_account ON finance.journal_entry_lines (account_id);

-- Gap-free document numbers per prefix (JE-202610-, OB-2026-, AP-20261026-, ...).
CREATE TABLE finance.document_sequences (
  prefix  text PRIMARY KEY,
  last_no int NOT NULL
);

-- Results of automatic posting (payroll, sales, payables, reimbursements),
-- with the request so a DRAFT or SKIPPED result can be retried.
CREATE TABLE finance.posting_logs (
  id             text PRIMARY KEY,
  event_code     text NOT NULL,
  document_type  text NOT NULL,
  document_id    text NOT NULL,
  document_label text NOT NULL DEFAULT '',
  status         text NOT NULL CHECK (status IN ('posted', 'draft', 'skipped')),
  reason         text NOT NULL DEFAULT '',
  entry_id       text,
  request        jsonb NOT NULL,
  attempts       int NOT NULL DEFAULT 1,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  UNIQUE (event_code, document_id)
);
CREATE INDEX posting_logs_status ON finance.posting_logs (status, updated_at DESC);

-- AP vendor bills and AR client invoices share one table (kind).
CREATE TABLE finance.invoices (
  id               text PRIMARY KEY,
  kind             text NOT NULL CHECK (kind IN ('AP', 'AR')),
  invoice_no       text NOT NULL,
  invoice_date     date NOT NULL,
  due_date         date,
  party_name       text NOT NULL,
  reference        text NOT NULL DEFAULT '',
  account_id       text REFERENCES finance.accounts (id),
  currency         text NOT NULL DEFAULT 'IDR',
  subtotal         numeric(18,2) NOT NULL DEFAULT 0,
  tax_amount       numeric(18,2) NOT NULL DEFAULT 0,
  total_amount     numeric(18,2) NOT NULL CHECK (total_amount > 0),
  status           text NOT NULL CHECK (status IN ('DRAFT', 'POSTED', 'VOID')),
  description      text NOT NULL DEFAULT '',
  sales_invoice_id text,
  client_id        text,
  deal_id          text,
  journal_entry_id text,
  posted_at        timestamptz,
  posted_by        text,
  created_at       timestamptz NOT NULL DEFAULT now(),
  created_by       text,
  updated_at       timestamptz NOT NULL DEFAULT now(),
  updated_by       text,
  deleted_at       timestamptz,
  deleted_by       text
);
CREATE UNIQUE INDEX invoices_no_live ON finance.invoices (invoice_no) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX invoices_sales_live ON finance.invoices (sales_invoice_id) WHERE deleted_at IS NULL AND sales_invoice_id IS NOT NULL;
CREATE INDEX invoices_kind_date ON finance.invoices (kind, invoice_date DESC) WHERE deleted_at IS NULL;

CREATE TABLE finance.payments (
  id               text PRIMARY KEY,
  kind             text NOT NULL CHECK (kind IN ('AP', 'AR')),
  payment_no       text NOT NULL,
  payment_date     date NOT NULL,
  amount           numeric(18,2) NOT NULL CHECK (amount > 0),
  method           text NOT NULL,
  cash_account_id  text REFERENCES finance.accounts (id),
  reference_number text NOT NULL DEFAULT '',
  notes            text NOT NULL DEFAULT '',
  status           text NOT NULL CHECK (status IN ('DRAFT', 'POSTED', 'VOID')),
  party_name       text NOT NULL DEFAULT '',
  sales_payment_id text,
  journal_entry_id text,
  void_reason      text NOT NULL DEFAULT '',
  voided_at        timestamptz,
  voided_by        text,
  posted_at        timestamptz,
  posted_by        text,
  created_at       timestamptz NOT NULL DEFAULT now(),
  created_by       text,
  updated_at       timestamptz NOT NULL DEFAULT now(),
  updated_by       text,
  deleted_at       timestamptz
);
CREATE UNIQUE INDEX payments_no_live ON finance.payments (payment_no) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX payments_sales_live ON finance.payments (sales_payment_id) WHERE deleted_at IS NULL AND sales_payment_id IS NOT NULL;

CREATE TABLE finance.payment_allocations (
  id         text PRIMARY KEY,
  payment_id text NOT NULL REFERENCES finance.payments (id) ON DELETE CASCADE,
  invoice_id text NOT NULL REFERENCES finance.invoices (id),
  amount     numeric(18,2) NOT NULL CHECK (amount > 0),
  UNIQUE (payment_id, invoice_id)
);
CREATE INDEX payment_allocations_invoice ON finance.payment_allocations (invoice_id);

-- ─── Reimbursement ─────────────────────────────────────────

CREATE TABLE finance.reimbursement_policies (
  category           text PRIMARY KEY CHECK (category IN ('TRANSPORT', 'MAKAN', 'AKOMODASI', 'OPERASIONAL_PROYEK', 'LAINNYA')),
  label              text NOT NULL,
  monthly_limit      numeric(18,2),
  per_claim_limit    numeric(18,2),
  expense_account_id text REFERENCES finance.accounts (id),
  -- Claims within the limits skip the finance step (approved by policy).
  auto_approve       boolean NOT NULL DEFAULT false,
  is_active          boolean NOT NULL DEFAULT true,
  updated_at         timestamptz NOT NULL DEFAULT now(),
  updated_by         text
);

CREATE TABLE finance.reimbursement_batches (
  id               text PRIMARY KEY,
  batch_no         text NOT NULL UNIQUE,
  paid_at          date NOT NULL,
  cash_account_id  text REFERENCES finance.accounts (id),
  total_amount     numeric(18,2) NOT NULL,
  claim_count      int NOT NULL,
  journal_entry_id text,
  journal_status   text NOT NULL DEFAULT '',
  journal_reason   text NOT NULL DEFAULT '',
  notes            text NOT NULL DEFAULT '',
  created_at       timestamptz NOT NULL DEFAULT now(),
  created_by       text
);

CREATE TABLE finance.reimbursement_claims (
  id                  text PRIMARY KEY,
  claim_no            text NOT NULL UNIQUE,
  employee_id         text NOT NULL,
  employee_user_id    text,
  employee_name       text NOT NULL,
  department_name     text NOT NULL DEFAULT '',
  manager_employee_id text,
  manager_user_id     text,
  category            text NOT NULL CHECK (category IN ('TRANSPORT', 'MAKAN', 'AKOMODASI', 'OPERASIONAL_PROYEK', 'LAINNYA')),
  transaction_date    date NOT NULL,
  amount              numeric(18,2) NOT NULL CHECK (amount > 0),
  description         text NOT NULL,
  client_id           text,
  project             text NOT NULL DEFAULT '',
  status              text NOT NULL CHECK (status IN ('DRAFT', 'SUBMITTED', 'APPROVED_MANAGER', 'APPROVED_FINANCE', 'PAID', 'REJECTED', 'CANCELLED')),
  over_limit          boolean NOT NULL DEFAULT false,
  limit_note          text NOT NULL DEFAULT '',
  submitted_at        timestamptz,
  manager_approved_at timestamptz,
  manager_approved_by text,
  finance_approved_at timestamptz,
  finance_approved_by text,
  rejected_at         timestamptz,
  rejected_by         text,
  rejected_stage      text NOT NULL DEFAULT '',
  reject_reason       text NOT NULL DEFAULT '',
  cancelled_at        timestamptz,
  paid_at             timestamptz,
  paid_by             text,
  payment_batch_id    text REFERENCES finance.reimbursement_batches (id),
  last_reminded_at    timestamptz,
  created_at          timestamptz NOT NULL DEFAULT now(),
  created_by          text,
  updated_at          timestamptz NOT NULL DEFAULT now(),
  updated_by          text
);
CREATE INDEX reimbursement_claims_employee ON finance.reimbursement_claims (employee_id, transaction_date DESC);
CREATE INDEX reimbursement_claims_user ON finance.reimbursement_claims (employee_user_id);
CREATE INDEX reimbursement_claims_manager ON finance.reimbursement_claims (manager_user_id, status);
CREATE INDEX reimbursement_claims_status ON finance.reimbursement_claims (status, submitted_at);

CREATE TABLE finance.reimbursement_receipts (
  id           text PRIMARY KEY,
  claim_id     text NOT NULL REFERENCES finance.reimbursement_claims (id) ON DELETE CASCADE,
  storage_key  text NOT NULL,
  filename     text NOT NULL,
  content_type text NOT NULL,
  size_bytes   bigint NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  created_by   text
);
CREATE INDEX reimbursement_receipts_claim ON finance.reimbursement_receipts (claim_id);

CREATE TABLE finance.reimbursement_events (
  id         text PRIMARY KEY,
  claim_id   text NOT NULL REFERENCES finance.reimbursement_claims (id) ON DELETE CASCADE,
  action     text NOT NULL,
  actor_id   text,
  actor_name text NOT NULL DEFAULT '',
  note       text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX reimbursement_events_claim ON finance.reimbursement_events (claim_id, created_at);

-- ─── Analysis ──────────────────────────────────────────────

CREATE TABLE finance.financial_insights (
  id         text PRIMARY KEY,
  date_from  date NOT NULL,
  date_to    date NOT NULL,
  language   text NOT NULL,
  model      text NOT NULL,
  content    text NOT NULL,
  metrics    jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  created_by text
);

-- +goose Down
DROP SCHEMA IF EXISTS finance CASCADE;
