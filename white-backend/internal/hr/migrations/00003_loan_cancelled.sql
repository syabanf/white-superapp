-- +goose Up
-- A pending loan can be withdrawn by the borrower or HR before approval.
ALTER TABLE hr.loans DROP CONSTRAINT loans_status_check;
ALTER TABLE hr.loans ADD CONSTRAINT loans_status_check
    CHECK (status IN ('pending', 'approved', 'rejected', 'paid_off', 'cancelled'));

-- +goose Down
UPDATE hr.loans SET status = 'rejected' WHERE status = 'cancelled';
ALTER TABLE hr.loans DROP CONSTRAINT loans_status_check;
ALTER TABLE hr.loans ADD CONSTRAINT loans_status_check
    CHECK (status IN ('pending', 'approved', 'rejected', 'paid_off'));
