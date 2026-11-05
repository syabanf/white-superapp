-- +goose Up
-- Manual journals stay off Hutang Usaha, so accruals need their own account. A new
-- database gets it from the seed.
-- +goose StatementBegin
INSERT INTO finance.accounts (id, code, name, parent_id, account_type_id, level, is_postable, cash_flow_category)
SELECT 'acc-2106000', '2106000', 'Biaya Yang Masih Harus Dibayar', p.parent_id, p.account_type_id, p.level, false, NULL
  FROM finance.accounts p
 WHERE p.code = '2105000' AND p.deleted_at IS NULL
   AND NOT EXISTS (SELECT 1 FROM finance.accounts a WHERE a.code = '2106000' AND a.deleted_at IS NULL);
INSERT INTO finance.accounts (id, code, name, parent_id, account_type_id, level, is_postable, cash_flow_category)
SELECT 'acc-2106001', '2106001', 'Biaya Yang Masih Harus Dibayar', g.id, c.account_type_id, c.level, true, 'OPERATING'
  FROM finance.accounts c
  JOIN finance.accounts g ON g.code = '2106000' AND g.deleted_at IS NULL
 WHERE c.code = '2105001' AND c.deleted_at IS NULL
   AND NOT EXISTS (SELECT 1 FROM finance.accounts a WHERE a.code = '2106001' AND a.deleted_at IS NULL);
-- +goose StatementEnd

-- +goose Down
DELETE FROM finance.accounts WHERE id IN ('acc-2106001', 'acc-2106000')
  AND NOT EXISTS (SELECT 1 FROM finance.journal_entry_lines l WHERE l.account_id = 'acc-2106001');
