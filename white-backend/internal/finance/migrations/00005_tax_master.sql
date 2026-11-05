-- +goose Up
-- Tax codes and extra charges as master data. A document stores a copy of what
-- it used, so changing a rate never rewrites an issued document.
CREATE TABLE finance.tax_codes (
  id                  text PRIMARY KEY,
  code                text NOT NULL UNIQUE,
  name                text NOT NULL,
  -- VAT adds to the bill; WITHHOLDING is held back from the payment.
  kind                text NOT NULL CHECK (kind IN ('VAT', 'WITHHOLDING')),
  scope               text NOT NULL CHECK (scope IN ('SALE', 'PURCHASE', 'BOTH')),
  rate                numeric(6,2) NOT NULL CHECK (rate BETWEEN 0 AND 100),
  -- Share of the base the rate applies to: 11/12 for DPP nilai lain.
  base_numerator      int NOT NULL DEFAULT 1 CHECK (base_numerator > 0),
  base_denominator    int NOT NULL DEFAULT 1 CHECK (base_denominator > 0),
  price_included      boolean NOT NULL DEFAULT false,
  account_sale_id     text REFERENCES finance.accounts (id),
  account_purchase_id text REFERENCES finance.accounts (id),
  report_code         text NOT NULL DEFAULT '',
  valid_from          date,
  valid_to            date,
  is_default_sale     boolean NOT NULL DEFAULT false,
  is_default_purchase boolean NOT NULL DEFAULT false,
  is_active           boolean NOT NULL DEFAULT true,
  notes               text NOT NULL DEFAULT '',
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  updated_by          text,
  CHECK (base_numerator <= base_denominator),
  CHECK (kind = 'VAT' OR NOT price_included),
  CHECK (valid_to IS NULL OR valid_from IS NULL OR valid_to >= valid_from)
);

CREATE TABLE finance.charge_types (
  id                  text PRIMARY KEY,
  code                text NOT NULL UNIQUE,
  name                text NOT NULL,
  scope               text NOT NULL CHECK (scope IN ('SALE', 'PURCHASE', 'BOTH')),
  computation         text NOT NULL CHECK (computation IN ('FIXED', 'PERCENT')),
  -- Rupiah for FIXED, percent for PERCENT. Zero means typed on each document.
  value               numeric(18,2) NOT NULL DEFAULT 0 CHECK (value >= 0),
  -- A taxable charge joins the tax base; others are added after tax.
  taxable             boolean NOT NULL DEFAULT false,
  is_deduction        boolean NOT NULL DEFAULT false,
  account_sale_id     text REFERENCES finance.accounts (id),
  account_purchase_id text REFERENCES finance.accounts (id),
  is_active           boolean NOT NULL DEFAULT true,
  notes               text NOT NULL DEFAULT '',
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  updated_by          text
);

ALTER TABLE finance.documents
  DROP COLUMN ppn_percent,
  ADD COLUMN taxes              jsonb NOT NULL DEFAULT '[]',
  ADD COLUMN charges            jsonb NOT NULL DEFAULT '[]',
  ADD COLUMN charges_amount     numeric(18,2) NOT NULL DEFAULT 0,
  ADD COLUMN withholding_amount numeric(18,2) NOT NULL DEFAULT 0;

-- Extra journal lines of an invoice made from a document, and the tax register.
CREATE TABLE finance.invoice_postings (
  id          text PRIMARY KEY,
  invoice_id  text NOT NULL REFERENCES finance.invoices (id) ON DELETE CASCADE,
  sort_order  int NOT NULL DEFAULT 0,
  account_id  text NOT NULL REFERENCES finance.accounts (id),
  entry_side  text NOT NULL CHECK (entry_side IN ('DEBIT', 'CREDIT')),
  amount      numeric(18,2) NOT NULL CHECK (amount > 0),
  memo        text NOT NULL DEFAULT '',
  tax_code    text NOT NULL DEFAULT '',
  tax_name    text NOT NULL DEFAULT '',
  tax_kind    text NOT NULL DEFAULT '',
  tax_base    numeric(18,2) NOT NULL DEFAULT 0
);
CREATE INDEX invoice_postings_invoice ON finance.invoice_postings (invoice_id);

-- Starting set. Finance reviews it; PPh 4(2) stays off until it has an account.
INSERT INTO finance.tax_codes (id, code, name, kind, scope, rate, base_numerator, base_denominator, account_sale_id, account_purchase_id, valid_from, is_default_sale, is_default_purchase, is_active, notes)
VALUES
  ('tax-ppn-dpp1112', 'PPN-11', 'PPN 12% DPP 11/12 (efektif 11%)', 'VAT', 'BOTH', 12, 11, 12,
    (SELECT id FROM finance.accounts WHERE code = '2103003' AND deleted_at IS NULL), (SELECT id FROM finance.accounts WHERE code = '1104001' AND deleted_at IS NULL),
    '2025-01-01', true, true, true, 'Barang dan jasa non-mewah, PMK 131/2024.'),
  ('tax-ppn-12', 'PPN-12', 'PPN 12% penuh', 'VAT', 'BOTH', 12, 1, 1,
    (SELECT id FROM finance.accounts WHERE code = '2103003' AND deleted_at IS NULL), (SELECT id FROM finance.accounts WHERE code = '1104001' AND deleted_at IS NULL),
    '2025-02-01', false, false, true, 'Barang mewah, PMK 131/2024.'),
  ('tax-pph23-2', 'PPH23-2', 'PPh 23 jasa 2%', 'WITHHOLDING', 'BOTH', 2, 1, 1,
    (SELECT id FROM finance.accounts WHERE code = '1104002' AND deleted_at IS NULL), (SELECT id FROM finance.accounts WHERE code = '2103002' AND deleted_at IS NULL),
    NULL, false, false, true, 'Penjualan: klien memotong, jadi pajak dibayar di muka. Pembelian: kita memotong, jadi hutang pajak.'),
  ('tax-pph42-10', 'PPH42-10', 'PPh 4(2) sewa 10%', 'WITHHOLDING', 'BOTH', 10, 1, 1,
    NULL, NULL, NULL, false, false, false, 'Aktifkan setelah Finance memastikan tarif dan mengisi akunnya.');

INSERT INTO finance.charge_types (id, code, name, scope, computation, value, taxable, is_deduction, notes)
VALUES
  ('chg-materai', 'MATERAI', 'Bea materai', 'BOTH', 'FIXED', 10000, false, false, 'Tidak masuk DPP.'),
  ('chg-ongkir', 'ONGKIR', 'Ongkos kirim', 'BOTH', 'FIXED', 0, true, false, 'Nominal diisi per dokumen.'),
  ('chg-admin', 'ADMIN', 'Biaya administrasi', 'BOTH', 'FIXED', 0, true, false, 'Nominal diisi per dokumen.'),
  ('chg-agency', 'AGENCY-FEE', 'Agency fee', 'SALE', 'PERCENT', 0, true, false, 'Persen diisi per dokumen.'),
  ('chg-reimburse', 'TITIPAN', 'Biaya titipan klien', 'SALE', 'FIXED', 0, false, false, 'Pengeluaran atas nama klien, tidak masuk DPP. Finance perlu memastikan perlakuan pajaknya.'),
  ('chg-potongan', 'POTONGAN', 'Potongan khusus', 'BOTH', 'FIXED', 0, true, true, 'Mengurangi DPP.');

-- +goose Down
DROP TABLE finance.invoice_postings;
ALTER TABLE finance.documents
  ADD COLUMN ppn_percent numeric(5,2) NOT NULL DEFAULT 0,
  DROP COLUMN taxes,
  DROP COLUMN charges,
  DROP COLUMN charges_amount,
  DROP COLUMN withholding_amount;
DROP TABLE finance.charge_types;
DROP TABLE finance.tax_codes;
