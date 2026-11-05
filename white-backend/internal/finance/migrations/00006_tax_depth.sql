-- +goose Up
-- Progressive rates, cash basis VAT, party tax profiles, fiscal positions and a
-- signed tax ledger that the register and the exports read.
ALTER TABLE finance.tax_codes
  ADD COLUMN brackets              jsonb NOT NULL DEFAULT '[]',
  ADD COLUMN cash_basis            boolean NOT NULL DEFAULT false,
  ADD COLUMN transition_account_id text REFERENCES finance.accounts (id);

ALTER TABLE finance.invoice_postings
  ADD COLUMN tax_id           text NOT NULL DEFAULT '',
  ADD COLUMN tax_rate         numeric(6,2) NOT NULL DEFAULT 0,
  ADD COLUMN report_code      text NOT NULL DEFAULT '',
  ADD COLUMN cash_basis       boolean NOT NULL DEFAULT false,
  ADD COLUMN final_account_id text REFERENCES finance.accounts (id);

CREATE TABLE finance.fiscal_positions (
  id         text PRIMARY KEY,
  name       text NOT NULL UNIQUE,
  -- [{"fromTaxId": "...", "toTaxId": ""}]; an empty target removes the tax.
  rules      jsonb NOT NULL DEFAULT '[]',
  is_active  boolean NOT NULL DEFAULT true,
  notes      text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now(),
  updated_by text
);

CREATE TABLE finance.party_tax_profiles (
  id                 text PRIMARY KEY,
  side               text NOT NULL CHECK (side IN ('B2B', 'SUPPLIER')),
  name               text NOT NULL,
  npwp               text NOT NULL DEFAULT '',
  address            text NOT NULL DEFAULT '',
  is_pkp             boolean NOT NULL DEFAULT true,
  fiscal_position_id text REFERENCES finance.fiscal_positions (id),
  notes              text NOT NULL DEFAULT '',
  updated_at         timestamptz NOT NULL DEFAULT now(),
  updated_by         text
);
CREATE UNIQUE INDEX party_tax_profiles_name ON finance.party_tax_profiles (side, lower(name));

CREATE TABLE finance.tax_moves (
  id          text PRIMARY KEY,
  kind        text NOT NULL CHECK (kind IN ('AP', 'AR')),
  -- INVOICE, CREDIT_NOTE, CASH or CASH_VOID.
  source_type text NOT NULL,
  source_id   text NOT NULL,
  invoice_id  text NOT NULL REFERENCES finance.invoices (id),
  tax_code    text NOT NULL DEFAULT '',
  tax_name    text NOT NULL DEFAULT '',
  tax_kind    text NOT NULL,
  tax_rate    numeric(6,2) NOT NULL DEFAULT 0,
  report_code text NOT NULL DEFAULT '',
  move_date   date NOT NULL,
  -- Signed: a credit note is negative.
  base        numeric(18,2) NOT NULL,
  amount      numeric(18,2) NOT NULL,
  UNIQUE (source_type, source_id, tax_code)
);
CREATE INDEX tax_moves_date ON finance.tax_moves (move_date);

UPDATE finance.tax_codes SET report_code = '04' WHERE code = 'PPN-11';
UPDATE finance.tax_codes SET report_code = '01' WHERE code = 'PPN-12';

INSERT INTO finance.fiscal_positions (id, name, rules, notes) VALUES
  ('fp-non-pkp', 'Vendor non-PKP', '[{"fromTaxId": "tax-ppn-dpp1112", "toTaxId": ""}, {"fromTaxId": "tax-ppn-12", "toTaxId": ""}]',
   'Vendor yang bukan PKP tidak memungut PPN, jadi kode PPN dilepas dari dokumen.');

-- +goose Down
UPDATE finance.tax_codes SET report_code = '' WHERE code IN ('PPN-11', 'PPN-12');
DROP TABLE finance.tax_moves;
DROP TABLE finance.party_tax_profiles;
DROP TABLE finance.fiscal_positions;
ALTER TABLE finance.invoice_postings
  DROP COLUMN tax_id, DROP COLUMN tax_rate, DROP COLUMN report_code, DROP COLUMN cash_basis, DROP COLUMN final_account_id;
ALTER TABLE finance.tax_codes DROP COLUMN brackets, DROP COLUMN cash_basis, DROP COLUMN transition_account_id;
