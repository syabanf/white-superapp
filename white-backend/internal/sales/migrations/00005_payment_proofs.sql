-- +goose Up
-- Transfer proofs clients upload from the share portal. Finance approves a
-- proof into a payment or rejects it with a reason.
CREATE TABLE sales.payment_proofs (
    id            text PRIMARY KEY,
    invoice_id    text NOT NULL REFERENCES sales.invoices(id) ON DELETE CASCADE,
    amount        numeric(18, 2) NOT NULL CHECK (amount > 0),
    paid_on       date NOT NULL,
    payer_name    text NOT NULL,
    note          text NOT NULL DEFAULT '',
    file_key      text NOT NULL,
    file_name     text NOT NULL,
    status        text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    reject_reason text NOT NULL DEFAULT '',
    payment_id    text REFERENCES sales.payments(id) ON DELETE SET NULL,
    decided_by    text,
    decided_at    timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX payment_proofs_status_idx ON sales.payment_proofs (status, created_at DESC);
CREATE INDEX payment_proofs_invoice_idx ON sales.payment_proofs (invoice_id);

-- +goose Down
DROP TABLE sales.payment_proofs;
