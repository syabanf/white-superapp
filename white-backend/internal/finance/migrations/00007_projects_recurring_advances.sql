-- +goose Up
-- A tag groups transactions into one project of one client. Tagging sits on the
-- journal, because that is what the profit report reads.
CREATE TABLE finance.project_tags (
  id          text PRIMARY KEY,
  name        text NOT NULL,
  client_name text NOT NULL DEFAULT '',
  -- Set on the tag Finance creates for a Sales deal.
  deal_id     text UNIQUE,
  is_active   boolean NOT NULL DEFAULT true,
  notes       text NOT NULL DEFAULT '',
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  updated_by  text
);

ALTER TABLE finance.journal_entries ADD COLUMN tag_id text REFERENCES finance.project_tags (id);
ALTER TABLE finance.invoices ADD COLUMN tag_id text REFERENCES finance.project_tags (id);
ALTER TABLE finance.recurring_journals ADD COLUMN tag_id text REFERENCES finance.project_tags (id);
CREATE INDEX journal_entries_tag ON finance.journal_entries (tag_id) WHERE tag_id IS NOT NULL;

CREATE TABLE finance.recurring_invoices (
  id            text PRIMARY KEY,
  kind          text NOT NULL CHECK (kind IN ('AP', 'AR')),
  name          text NOT NULL,
  party_name    text NOT NULL,
  client_id     text NOT NULL DEFAULT '',
  account_id    text REFERENCES finance.accounts (id),
  tag_id        text REFERENCES finance.project_tags (id),
  description   text NOT NULL DEFAULT '',
  subtotal      numeric(18,2) NOT NULL CHECK (subtotal > 0),
  tax_amount    numeric(18,2) NOT NULL DEFAULT 0,
  tax_ids       jsonb NOT NULL DEFAULT '[]',
  due_days      int NOT NULL DEFAULT 30 CHECK (due_days BETWEEN 0 AND 365),
  frequency     text NOT NULL CHECK (frequency IN ('MONTHLY', 'QUARTERLY', 'YEARLY')),
  anchor_day    int NOT NULL CHECK (anchor_day BETWEEN 1 AND 31),
  next_run      date NOT NULL,
  end_date      date,
  auto_post     boolean NOT NULL DEFAULT true,
  is_active     boolean NOT NULL DEFAULT true,
  last_invoice_id text,
  last_error    text NOT NULL DEFAULT '',
  created_at    timestamptz NOT NULL DEFAULT now(),
  created_by    text,
  updated_at    timestamptz NOT NULL DEFAULT now(),
  updated_by    text
);

-- Money paid or received before the invoice. What is used comes from the
-- applications whose payment is still POSTED, so a void frees it again.
CREATE TABLE finance.advances (
  id                 text PRIMARY KEY,
  kind               text NOT NULL CHECK (kind IN ('AP', 'AR')),
  advance_no         text NOT NULL UNIQUE,
  advance_date       date NOT NULL,
  party_name         text NOT NULL,
  amount             numeric(18,2) NOT NULL CHECK (amount > 0),
  cash_account_id    text NOT NULL REFERENCES finance.accounts (id),
  advance_account_id text NOT NULL REFERENCES finance.accounts (id),
  reference          text NOT NULL DEFAULT '',
  notes              text NOT NULL DEFAULT '',
  status             text NOT NULL CHECK (status IN ('POSTED', 'VOID')),
  journal_entry_id   text,
  void_reason        text NOT NULL DEFAULT '',
  created_at         timestamptz NOT NULL DEFAULT now(),
  created_by         text
);

CREATE TABLE finance.advance_applications (
  payment_id text PRIMARY KEY REFERENCES finance.payments (id),
  advance_id text NOT NULL REFERENCES finance.advances (id)
);
CREATE INDEX advance_applications_advance ON finance.advance_applications (advance_id);

-- +goose StatementBegin
-- Vendor advances need an asset account. A new database gets it from the seed.
INSERT INTO finance.accounts (id, code, name, parent_id, account_type_id, level, is_postable, cash_flow_category)
SELECT 'acc-1106000', '1106000', 'Uang Muka', p.parent_id, p.account_type_id, p.level, false, NULL
  FROM finance.accounts p
 WHERE p.code = '1105000' AND p.deleted_at IS NULL
   AND NOT EXISTS (SELECT 1 FROM finance.accounts a WHERE a.code = '1106000' AND a.deleted_at IS NULL);
INSERT INTO finance.accounts (id, code, name, parent_id, account_type_id, level, is_postable, cash_flow_category)
SELECT 'acc-1106001', '1106001', 'Uang Muka Pembelian', g.id, c.account_type_id, c.level, true, 'OPERATING'
  FROM finance.accounts c
  JOIN finance.accounts g ON g.code = '1106000' AND g.deleted_at IS NULL
 WHERE c.code = '1105001' AND c.deleted_at IS NULL
   AND NOT EXISTS (SELECT 1 FROM finance.accounts a WHERE a.code = '1106001' AND a.deleted_at IS NULL);
-- +goose StatementEnd

-- +goose Down
DROP TABLE finance.advance_applications;
DROP TABLE finance.advances;
DROP TABLE finance.recurring_invoices;
ALTER TABLE finance.recurring_journals DROP COLUMN tag_id;
ALTER TABLE finance.invoices DROP COLUMN tag_id;
ALTER TABLE finance.journal_entries DROP COLUMN tag_id;
DROP TABLE finance.project_tags;
