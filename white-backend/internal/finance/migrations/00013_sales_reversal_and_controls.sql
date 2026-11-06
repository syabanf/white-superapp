-- +goose Up
-- Sales already removed these. Finance reverses them and keeps trying until it can.
ALTER TABLE finance.payments
  ADD COLUMN sales_deleted_reason text NOT NULL DEFAULT '',
  ADD COLUMN sales_reversal_error text NOT NULL DEFAULT '';
ALTER TABLE finance.invoices
  ADD COLUMN sales_cancel_reason text NOT NULL DEFAULT '',
  ADD COLUMN sales_cancel_settled boolean NOT NULL DEFAULT false,
  ADD COLUMN sales_reversal_error text NOT NULL DEFAULT '';

ALTER TABLE finance.fixed_assets
  ADD COLUMN disposed_on date,
  ADD COLUMN disposal_proceeds numeric(18,2) NOT NULL DEFAULT 0,
  ADD COLUMN disposal_entry_id text;

-- What a vendor delivered against a purchase order, before tax.
CREATE TABLE finance.goods_receipts (
  id          text PRIMARY KEY,
  receipt_no  text NOT NULL UNIQUE,
  document_id text NOT NULL,
  receipt_date date NOT NULL,
  amount      numeric(18,2) NOT NULL CHECK (amount > 0),
  notes       text NOT NULL DEFAULT '',
  status      text NOT NULL CHECK (status IN ('POSTED', 'VOID')),
  void_reason text NOT NULL DEFAULT '',
  created_by  text,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX goods_receipts_document ON finance.goods_receipts (document_id);

CREATE TABLE finance.invoice_po_links (
  invoice_id  text PRIMARY KEY REFERENCES finance.invoices (id) ON DELETE CASCADE,
  document_id text NOT NULL
);
CREATE INDEX invoice_po_links_document ON finance.invoice_po_links (document_id);

-- Payroll and reimbursement a person assigns to a project for one month.
CREATE TABLE finance.project_cost_allocations (
  id         text PRIMARY KEY,
  tag_id     text NOT NULL REFERENCES finance.project_tags (id),
  month      text NOT NULL,
  kind       text NOT NULL CHECK (kind IN ('PAYROLL', 'REIMBURSEMENT')),
  amount     numeric(18,2) NOT NULL CHECK (amount > 0),
  notes      text NOT NULL DEFAULT '',
  created_by text,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX project_cost_allocations_month ON finance.project_cost_allocations (month);

-- One overdue reminder per invoice per stage.
CREATE TABLE finance.invoice_reminders (
  invoice_id text NOT NULL REFERENCES finance.invoices (id) ON DELETE CASCADE,
  stage      int NOT NULL,
  sent_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (invoice_id, stage)
);

-- +goose Down
DROP TABLE finance.invoice_reminders;
DROP TABLE finance.project_cost_allocations;
DROP TABLE finance.invoice_po_links;
DROP TABLE finance.goods_receipts;
ALTER TABLE finance.fixed_assets DROP COLUMN disposal_entry_id, DROP COLUMN disposal_proceeds, DROP COLUMN disposed_on;
ALTER TABLE finance.invoices DROP COLUMN sales_reversal_error, DROP COLUMN sales_cancel_settled, DROP COLUMN sales_cancel_reason;
ALTER TABLE finance.payments DROP COLUMN sales_reversal_error, DROP COLUMN sales_deleted_reason;
