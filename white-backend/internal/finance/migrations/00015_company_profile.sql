-- +goose Up
-- The company profile: letterhead on Finance and Sales documents and the
-- bank account clients transfer to. One row; COMPANY_* env fills the gaps
-- until someone saves it.
CREATE TABLE finance.company_profile (
    id                boolean PRIMARY KEY DEFAULT true CHECK (id),
    legal_name        text NOT NULL,
    address           text NOT NULL DEFAULT '',
    email             text NOT NULL DEFAULT '',
    phone             text NOT NULL DEFAULT '',
    npwp              text NOT NULL DEFAULT '',
    bank_name         text NOT NULL DEFAULT '',
    bank_account_no   text NOT NULL DEFAULT '',
    bank_account_name text NOT NULL DEFAULT '',
    updated_by        text,
    updated_at        timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE finance.company_profile;
