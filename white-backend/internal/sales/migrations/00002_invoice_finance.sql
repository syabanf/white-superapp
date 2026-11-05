-- +goose Up
-- Finance revision note and the tax invoice attachment (one file per invoice).
ALTER TABLE sales.invoices
    ADD COLUMN note              text,
    ADD COLUMN faktur_pajak_key  text,
    ADD COLUMN faktur_pajak_name text;

-- +goose Down
ALTER TABLE sales.invoices
    DROP COLUMN note,
    DROP COLUMN faktur_pajak_key,
    DROP COLUMN faktur_pajak_name;
