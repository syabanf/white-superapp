-- +goose Up
-- Reimbursement account choice and receipt fingerprints, bank and credit card
-- reconciliation, recurring journals and fixed asset depreciation.
ALTER TABLE finance.accounts ADD COLUMN is_credit_card boolean NOT NULL DEFAULT false;

ALTER TABLE finance.reimbursement_claims
  ADD COLUMN preferred_cash_account_id text REFERENCES finance.accounts (id),
  ADD COLUMN preferred_cash_account_name text NOT NULL DEFAULT '';

ALTER TABLE finance.reimbursement_receipts
  ADD COLUMN content_hash text NOT NULL DEFAULT '',
  ADD COLUMN image_hash bigint;
CREATE INDEX reimbursement_receipts_hash ON finance.reimbursement_receipts (content_hash) WHERE content_hash <> '';

CREATE TABLE finance.bank_statements (
  id         text PRIMARY KEY,
  account_id text NOT NULL REFERENCES finance.accounts (id),
  filename   text NOT NULL,
  line_count int NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  created_by text
);

CREATE TABLE finance.bank_statement_lines (
  id               text PRIMARY KEY,
  statement_id     text NOT NULL REFERENCES finance.bank_statements (id) ON DELETE CASCADE,
  account_id       text NOT NULL REFERENCES finance.accounts (id),
  line_date        date NOT NULL,
  description      text NOT NULL,
  sender_name      text NOT NULL DEFAULT '',
  -- Positive is money in, negative is money out or a card charge.
  amount           numeric(18,2) NOT NULL CHECK (amount <> 0),
  -- Same bank row uploaded twice lands on the same key.
  dedupe_key       text NOT NULL,
  status           text NOT NULL DEFAULT 'UNMATCHED' CHECK (status IN ('UNMATCHED', 'SUGGESTED', 'MATCHED', 'IGNORED')),
  suggested_kind   text NOT NULL DEFAULT '' CHECK (suggested_kind IN ('', 'JOURNAL', 'INVOICE')),
  suggested_ref    text NOT NULL DEFAULT '',
  suggested_label  text NOT NULL DEFAULT '',
  suggested_score  int NOT NULL DEFAULT 0,
  suggested_reason text NOT NULL DEFAULT '',
  matched_line_id  text REFERENCES finance.journal_entry_lines (id) ON DELETE SET NULL,
  matched_entry_id text,
  note             text NOT NULL DEFAULT '',
  verified_by      text,
  verified_at      timestamptz,
  created_at       timestamptz NOT NULL DEFAULT now(),
  UNIQUE (account_id, dedupe_key)
);
CREATE INDEX bank_statement_lines_account ON finance.bank_statement_lines (account_id, status, line_date DESC);
-- One journal line settles one bank row, which stops double recording.
CREATE UNIQUE INDEX bank_statement_lines_matched ON finance.bank_statement_lines (matched_line_id) WHERE matched_line_id IS NOT NULL;

CREATE TABLE finance.recurring_journals (
  id          text PRIMARY KEY,
  name        text NOT NULL,
  description text NOT NULL DEFAULT '',
  frequency   text NOT NULL CHECK (frequency IN ('MONTHLY', 'QUARTERLY', 'YEARLY')),
  anchor_day  int NOT NULL CHECK (anchor_day BETWEEN 1 AND 31),
  next_run    date NOT NULL,
  end_date    date,
  lines       jsonb NOT NULL,
  is_active   boolean NOT NULL DEFAULT true,
  last_entry_id text,
  last_error  text NOT NULL DEFAULT '',
  created_at  timestamptz NOT NULL DEFAULT now(),
  created_by  text,
  updated_at  timestamptz NOT NULL DEFAULT now(),
  updated_by  text
);

CREATE TABLE finance.fixed_assets (
  id                 text PRIMARY KEY,
  name               text NOT NULL,
  acquired_on        date NOT NULL,
  cost               numeric(18,2) NOT NULL CHECK (cost > 0),
  salvage            numeric(18,2) NOT NULL DEFAULT 0 CHECK (salvage >= 0),
  life_months        int NOT NULL CHECK (life_months > 0),
  expense_account_id text NOT NULL REFERENCES finance.accounts (id),
  accum_account_id   text NOT NULL REFERENCES finance.accounts (id),
  depreciated_months int NOT NULL DEFAULT 0,
  is_active          boolean NOT NULL DEFAULT true,
  last_entry_id      text,
  last_error         text NOT NULL DEFAULT '',
  created_at         timestamptz NOT NULL DEFAULT now(),
  created_by         text,
  updated_at         timestamptz NOT NULL DEFAULT now(),
  updated_by         text,
  CHECK (salvage < cost)
);

-- +goose Down
DROP TABLE finance.fixed_assets;
DROP TABLE finance.recurring_journals;
DROP TABLE finance.bank_statement_lines;
DROP TABLE finance.bank_statements;
DROP INDEX finance.reimbursement_receipts_hash;
ALTER TABLE finance.reimbursement_receipts DROP COLUMN image_hash, DROP COLUMN content_hash;
ALTER TABLE finance.reimbursement_claims DROP COLUMN preferred_cash_account_name, DROP COLUMN preferred_cash_account_id;
ALTER TABLE finance.accounts DROP COLUMN is_credit_card;
