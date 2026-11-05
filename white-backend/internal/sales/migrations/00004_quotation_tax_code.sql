-- +goose Up
-- A quotation copies the finance tax code it uses; ppn_percent holds its
-- effective rate, so 12% on a base of 11/12 is stored as 11.
ALTER TABLE sales.quotations
  ADD COLUMN tax_code text NOT NULL DEFAULT '',
  ADD COLUMN tax_name text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sales.quotations DROP COLUMN tax_code, DROP COLUMN tax_name;
