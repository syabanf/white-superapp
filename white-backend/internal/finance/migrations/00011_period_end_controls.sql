-- +goose Up
-- Bad debt needs its own expense account. A new database gets it from the seed.
-- +goose StatementBegin
INSERT INTO finance.accounts (id, code, name, parent_id, account_type_id, level, is_postable, cash_flow_category)
SELECT 'acc-8102001', '8102001', 'Beban Piutang Tak Tertagih', c.parent_id, c.account_type_id, c.level, true, 'OPERATING'
  FROM finance.accounts c
 WHERE c.code = '8101001' AND c.deleted_at IS NULL
   AND NOT EXISTS (SELECT 1 FROM finance.accounts a WHERE a.code = '8102001' AND a.deleted_at IS NULL);
-- +goose StatementEnd

-- Money received or paid now and recognized month by month.
CREATE TABLE finance.deferrals (
  id                  text PRIMARY KEY,
  kind                text NOT NULL CHECK (kind IN ('REVENUE', 'EXPENSE')),
  name                text NOT NULL,
  party_name          text NOT NULL DEFAULT '',
  reference           text NOT NULL DEFAULT '',
  deferred_account_id text NOT NULL REFERENCES finance.accounts (id),
  target_account_id   text NOT NULL REFERENCES finance.accounts (id),
  tag_id              text,
  start_date          date NOT NULL,
  months              int NOT NULL CHECK (months > 0),
  posted_months       int NOT NULL DEFAULT 0,
  amount              numeric(18,2) NOT NULL CHECK (amount > 0),
  is_active           boolean NOT NULL DEFAULT true,
  last_entry_id       text,
  last_error          text NOT NULL DEFAULT '',
  created_by          text,
  created_at          timestamptz NOT NULL DEFAULT now()
);

-- Tax paid to the state, with its proof of payment.
CREATE TABLE finance.tax_remittances (
  id               text PRIMARY KEY,
  remit_no         text NOT NULL UNIQUE,
  period           text NOT NULL,
  remit_date       date NOT NULL,
  account_id       text NOT NULL REFERENCES finance.accounts (id),
  cash_account_id  text NOT NULL REFERENCES finance.accounts (id),
  amount           numeric(18,2) NOT NULL CHECK (amount > 0),
  ntpn             text NOT NULL DEFAULT '',
  notes            text NOT NULL DEFAULT '',
  status           text NOT NULL CHECK (status IN ('POSTED', 'VOID')),
  journal_entry_id text NOT NULL,
  void_reason      text NOT NULL DEFAULT '',
  created_by       text,
  created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tax_remittances_period ON finance.tax_remittances (period);

-- One row of company-wide finance controls.
CREATE TABLE finance.control_settings (
  id                bool PRIMARY KEY DEFAULT true CHECK (id),
  separate_approver bool NOT NULL DEFAULT false,
  updated_by        text,
  updated_at        timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE finance.control_settings;
DROP TABLE finance.tax_remittances;
DROP TABLE finance.deferrals;
DELETE FROM finance.accounts WHERE id = 'acc-8102001'
  AND NOT EXISTS (SELECT 1 FROM finance.journal_entry_lines l WHERE l.account_id = 'acc-8102001');
