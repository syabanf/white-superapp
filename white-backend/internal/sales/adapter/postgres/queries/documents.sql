-- name: ListContractTemplates :many
SELECT * FROM sales.contract_templates WHERE (NOT sqlc.arg('active_only')::bool OR is_active) ORDER BY type, name;

-- name: GetContractTemplate :one
SELECT * FROM sales.contract_templates WHERE id = $1;

-- name: UpsertContractTemplate :exec
INSERT INTO sales.contract_templates (id, name, type, body, is_active, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, type = EXCLUDED.type, body = EXCLUDED.body, is_active = EXCLUDED.is_active,
    updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: DeleteContractTemplate :exec
DELETE FROM sales.contract_templates WHERE id = $1;

-- name: GetContract :one
SELECT * FROM sales.contracts WHERE id = $1;

-- name: GetContractByToken :one
SELECT * FROM sales.contracts WHERE sign_token = $1;

-- name: ListContracts :many
SELECT * FROM sales.contracts
WHERE (sqlc.narg('deal')::text IS NULL OR deal_id = sqlc.narg('deal'))
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC LIMIT sqlc.arg('lim');

-- name: UpsertContract :exec
INSERT INTO sales.contracts (id, contract_number, template_id, quotation_id, deal_id, type, title, client_name, body, status,
    start_date, end_date, sign_token, sent_at, signed_at, signer_name, signer_ip, signer_user_agent, signature_png,
    activated_at, ended_at, cancelled_at, cancel_reason, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, body = EXCLUDED.body, status = EXCLUDED.status,
    start_date = EXCLUDED.start_date, end_date = EXCLUDED.end_date, sent_at = EXCLUDED.sent_at, signed_at = EXCLUDED.signed_at,
    signer_name = EXCLUDED.signer_name, signer_ip = EXCLUDED.signer_ip, signer_user_agent = EXCLUDED.signer_user_agent,
    signature_png = EXCLUDED.signature_png, activated_at = EXCLUDED.activated_at, ended_at = EXCLUDED.ended_at,
    cancelled_at = EXCLUDED.cancelled_at, cancel_reason = EXCLUDED.cancel_reason, updated_by = EXCLUDED.updated_by,
    updated_at = EXCLUDED.updated_at;

-- name: ListCaseStudies :many
SELECT * FROM sales.case_studies WHERE (NOT sqlc.arg('active_only')::bool OR is_active) ORDER BY sort_order, title;

-- name: UpsertCaseStudy :exec
INSERT INTO sales.case_studies (id, title, client_name, industry, service_lines, challenge, solution, results, sort_order, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, client_name = EXCLUDED.client_name, industry = EXCLUDED.industry,
    service_lines = EXCLUDED.service_lines, challenge = EXCLUDED.challenge, solution = EXCLUDED.solution,
    results = EXCLUDED.results, sort_order = EXCLUDED.sort_order, is_active = EXCLUDED.is_active, updated_at = now();

-- name: DeleteCaseStudy :exec
DELETE FROM sales.case_studies WHERE id = $1;

-- name: ListPitchDecks :many
SELECT id, deal_id, lead_id, title, version, options, slides, octet_length(file)::bigint AS size_bytes, model, created_by, created_at
FROM sales.pitch_decks
WHERE (sqlc.narg('deal')::text IS NULL OR deal_id = sqlc.narg('deal'))
  AND (sqlc.narg('lead')::text IS NULL OR lead_id = sqlc.narg('lead'))
ORDER BY created_at DESC LIMIT 100;

-- name: GetPitchDeck :one
SELECT * FROM sales.pitch_decks WHERE id = $1;

-- name: InsertPitchDeck :exec
INSERT INTO sales.pitch_decks (id, deal_id, lead_id, title, version, options, slides, file, model, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: NextDeckVersion :one
SELECT (coalesce(max(version), 0) + 1)::int FROM sales.pitch_decks
WHERE (sqlc.narg('deal')::text IS NOT NULL AND deal_id = sqlc.narg('deal'))
   OR (sqlc.narg('deal')::text IS NULL AND lead_id = sqlc.narg('lead'));
