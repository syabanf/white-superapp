-- +goose Up
-- Excess cash and a credit note on a paid invoice both become an advance that
-- shares its document's journal, so one bank movement stays one bank line.
ALTER TABLE finance.payment_adjustments DROP CONSTRAINT payment_adjustments_kind_check;
ALTER TABLE finance.payment_adjustments ADD CONSTRAINT payment_adjustments_kind_check CHECK (kind IN ('FEE', 'DISCOUNT', 'WITHHOLDING', 'EXCESS'));

ALTER TABLE finance.advances
  ADD COLUMN source_payment_id text REFERENCES finance.payments (id),
  ADD COLUMN sales_payment_id text,
  ALTER COLUMN cash_account_id DROP NOT NULL;
CREATE UNIQUE INDEX advances_source_payment ON finance.advances (source_payment_id) WHERE source_payment_id IS NOT NULL;
CREATE UNIQUE INDEX advances_sales_payment ON finance.advances (sales_payment_id) WHERE sales_payment_id IS NOT NULL;

-- A credit note wholly on the paid share moves nothing against the invoice's
-- open balance, yet it still has to point at its invoice.
ALTER TABLE finance.payments DROP CONSTRAINT payments_amount_check;
ALTER TABLE finance.payments ADD CONSTRAINT payments_amount_check CHECK (amount >= 0);
ALTER TABLE finance.payment_allocations DROP CONSTRAINT payment_allocations_amount_check;
ALTER TABLE finance.payment_allocations ADD CONSTRAINT payment_allocations_amount_check CHECK (amount >= 0);

-- +goose Down
ALTER TABLE finance.payment_allocations DROP CONSTRAINT payment_allocations_amount_check;
ALTER TABLE finance.payment_allocations ADD CONSTRAINT payment_allocations_amount_check CHECK (amount > 0);
ALTER TABLE finance.payments DROP CONSTRAINT payments_amount_check;
ALTER TABLE finance.payments ADD CONSTRAINT payments_amount_check CHECK (amount > 0);
DROP INDEX finance.advances_sales_payment;
DROP INDEX finance.advances_source_payment;
ALTER TABLE finance.advances DROP COLUMN sales_payment_id, DROP COLUMN source_payment_id;
ALTER TABLE finance.advances ALTER COLUMN cash_account_id SET NOT NULL;
ALTER TABLE finance.payment_adjustments DROP CONSTRAINT payment_adjustments_kind_check;
ALTER TABLE finance.payment_adjustments ADD CONSTRAINT payment_adjustments_kind_check CHECK (kind IN ('FEE', 'DISCOUNT', 'WITHHOLDING'));
