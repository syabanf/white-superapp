-- +goose Up
-- Vendors, and one document table for purchase requests, purchase orders,
-- letterhead quotations and letterhead invoices. Goods receipt, stock and QC
-- are out of scope for an agency.
CREATE TABLE finance.vendors (
  id                 text PRIMARY KEY,
  name               text NOT NULL,
  pic_name           text NOT NULL DEFAULT '',
  phone              text NOT NULL DEFAULT '',
  email              text NOT NULL DEFAULT '',
  address            text NOT NULL DEFAULT '',
  npwp               text NOT NULL DEFAULT '',
  bank_name          text NOT NULL DEFAULT '',
  bank_account_no    text NOT NULL DEFAULT '',
  bank_account_name  text NOT NULL DEFAULT '',
  payment_terms_days int NOT NULL DEFAULT 0 CHECK (payment_terms_days BETWEEN 0 AND 365),
  notes              text NOT NULL DEFAULT '',
  is_active          boolean NOT NULL DEFAULT true,
  created_at         timestamptz NOT NULL DEFAULT now(),
  created_by         text,
  updated_at         timestamptz NOT NULL DEFAULT now(),
  updated_by         text
);
CREATE UNIQUE INDEX vendors_name_idx ON finance.vendors (lower(name));

CREATE TABLE finance.documents (
  id                text PRIMARY KEY,
  kind              text NOT NULL CHECK (kind IN ('PR', 'PO', 'QUOTATION', 'INVOICE')),
  -- Who the document is with: B2B is a client, SUPPLIER a vendor.
  side              text NOT NULL CHECK (side IN ('B2B', 'SUPPLIER')),
  number            text NOT NULL UNIQUE,
  status            text NOT NULL,
  vendor_id         text REFERENCES finance.vendors (id),
  party_name        text NOT NULL DEFAULT '',
  pic_name          text NOT NULL DEFAULT '',
  phone             text NOT NULL DEFAULT '',
  email             text NOT NULL DEFAULT '',
  address           text NOT NULL DEFAULT '',
  title             text NOT NULL,
  issue_date        date NOT NULL,
  due_date          date,
  external_number   text NOT NULL DEFAULT '',
  notes             text NOT NULL DEFAULT '',
  items             jsonb NOT NULL DEFAULT '[]',
  discount_percent  numeric(5,2) NOT NULL DEFAULT 0 CHECK (discount_percent BETWEEN 0 AND 100),
  ppn_percent       numeric(5,2) NOT NULL DEFAULT 0 CHECK (ppn_percent BETWEEN 0 AND 100),
  subtotal          numeric(18,2) NOT NULL DEFAULT 0,
  discount_amount   numeric(18,2) NOT NULL DEFAULT 0,
  tax_amount        numeric(18,2) NOT NULL DEFAULT 0,
  total_amount      numeric(18,2) NOT NULL DEFAULT 0,
  account_id        text REFERENCES finance.accounts (id),
  source_id         text REFERENCES finance.documents (id),
  ledger_invoice_id text REFERENCES finance.invoices (id),
  -- Copied when a document leaves draft, so a sent paper never changes.
  letterhead        jsonb,
  reject_reason     text NOT NULL DEFAULT '',
  decided_by        text,
  decided_at        timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now(),
  created_by        text,
  updated_at        timestamptz NOT NULL DEFAULT now(),
  updated_by        text
);
CREATE INDEX documents_list_idx ON finance.documents (kind, status, issue_date DESC);
-- A source converts once.
CREATE UNIQUE INDEX documents_source_idx ON finance.documents (source_id) WHERE source_id IS NOT NULL;

-- +goose Down
DROP TABLE finance.documents;
DROP TABLE finance.vendors;
