-- +goose Up
-- What settled an invoice without moving cash: a bank fee taken from the
-- transfer, tax the other party held back, or a discount.
CREATE TABLE finance.payment_adjustments (
  id         text PRIMARY KEY,
  payment_id text NOT NULL REFERENCES finance.payments (id) ON DELETE CASCADE,
  kind       text NOT NULL CHECK (kind IN ('FEE', 'DISCOUNT', 'WITHHOLDING')),
  account_id text NOT NULL REFERENCES finance.accounts (id),
  tax_code   text NOT NULL DEFAULT '',
  tax_name   text NOT NULL DEFAULT '',
  tax_rate   numeric(6,2) NOT NULL DEFAULT 0,
  report_code text NOT NULL DEFAULT '',
  tax_base   numeric(18,2) NOT NULL DEFAULT 0,
  amount     numeric(18,2) NOT NULL CHECK (amount > 0),
  memo       text NOT NULL DEFAULT '',
  sort_order int NOT NULL DEFAULT 0
);
CREATE INDEX payment_adjustments_payment ON finance.payment_adjustments (payment_id);

CREATE TABLE finance.advance_refunds (
  id               text PRIMARY KEY,
  advance_id       text NOT NULL REFERENCES finance.advances (id),
  refund_no        text NOT NULL UNIQUE,
  refund_date      date NOT NULL,
  amount           numeric(18,2) NOT NULL CHECK (amount > 0),
  cash_account_id  text NOT NULL REFERENCES finance.accounts (id),
  reason           text NOT NULL,
  status           text NOT NULL CHECK (status IN ('POSTED', 'VOID')),
  journal_entry_id text NOT NULL,
  void_reason      text NOT NULL DEFAULT '',
  created_at       timestamptz NOT NULL DEFAULT now(),
  created_by       text
);
CREATE INDEX advance_refunds_advance ON finance.advance_refunds (advance_id);

-- +goose Down
DROP TABLE finance.advance_refunds;
DROP TABLE finance.payment_adjustments;
