-- +goose Up
-- Client billing master data and the invoice follow-up queue.
ALTER TABLE sales.accounts
    ADD COLUMN bank_name          text,
    ADD COLUMN bank_account_no    text,
    ADD COLUMN bank_account_name  text,
    ADD COLUMN billing_email      text,
    ADD COLUMN payment_terms_days integer CHECK (payment_terms_days BETWEEN 0 AND 365);

CREATE TABLE sales.invoice_followups (
    id          text PRIMARY KEY,
    invoice_id  text NOT NULL REFERENCES sales.invoices(id) ON DELETE CASCADE,
    -- Days from the due date: -3, 0, 3 or 7.
    stage       integer NOT NULL,
    phone       text NOT NULL DEFAULT '',
    message     text NOT NULL,
    status      text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'SENT', 'SKIPPED')),
    last_error  text NOT NULL DEFAULT '',
    approved_by text,
    approved_at timestamptz,
    sent_at     timestamptz,
    message_id  text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (invoice_id, stage)
);
CREATE INDEX invoice_followups_status_idx ON sales.invoice_followups (status, created_at DESC);

-- +goose Down
DROP TABLE sales.invoice_followups;
ALTER TABLE sales.accounts
    DROP COLUMN bank_name,
    DROP COLUMN bank_account_no,
    DROP COLUMN bank_account_name,
    DROP COLUMN billing_email,
    DROP COLUMN payment_terms_days;
