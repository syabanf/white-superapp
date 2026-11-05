-- +goose Up
-- Monthly budget per profit and loss account. A missing row means no budget.
CREATE TABLE finance.budgets (
  period_month date NOT NULL CHECK (extract(day FROM period_month) = 1),
  account_id   text NOT NULL REFERENCES finance.accounts (id),
  amount       numeric(18,2) NOT NULL CHECK (amount >= 0),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  updated_by   text,
  PRIMARY KEY (period_month, account_id)
);

-- +goose Down
DROP TABLE finance.budgets;
