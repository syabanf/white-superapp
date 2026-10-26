-- +goose Up
-- One row per recruitment candidate turned into an employee, so a
-- redelivered white.recruitment.candidate.hired never creates a duplicate.
-- The foreign key is deferred so the claim can precede the employee insert.
CREATE TABLE hr.recruitment_hires (
    candidate_id text PRIMARY KEY,
    employee_id  text NOT NULL REFERENCES hr.employees (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS hr.recruitment_hires;
