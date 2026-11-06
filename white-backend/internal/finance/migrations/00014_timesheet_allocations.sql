-- +goose Up
-- Allocations generated from approved HR timesheets are replaced as a set
-- each time Finance recalculates a month; manual ones stay.
ALTER TABLE finance.project_cost_allocations
    ADD COLUMN source text NOT NULL DEFAULT 'MANUAL' CHECK (source IN ('MANUAL', 'TIMESHEET'));

-- +goose Down
ALTER TABLE finance.project_cost_allocations DROP COLUMN source;
