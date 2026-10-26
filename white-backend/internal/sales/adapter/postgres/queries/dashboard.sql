-- Dashboard aggregates. Every deal query takes the same optional filters
-- (pipeline, owner, line, source) against deals d and their lead l; months
-- are WIB ("YYYY-MM").

-- name: DashClosedMonthly :many
SELECT to_char(d.closed_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM')::text AS month,
       count(*) FILTER (WHERE s.is_won)::int AS won_count,
       coalesce(sum(coalesce(d.value_final, d.value_estimate, 0)) FILTER (WHERE s.is_won), 0)::float8 AS won_value,
       count(*) FILTER (WHERE s.is_lost)::int AS lost_count,
       coalesce(sum(coalesce(d.value_final, d.value_estimate, 0)) FILTER (WHERE s.is_lost), 0)::float8 AS lost_value,
       coalesce(sum(extract(epoch FROM d.closed_at - d.created_at) / 86400) FILTER (WHERE s.is_won), 0)::float8 AS won_cycle_days
FROM sales.deals d
JOIN sales.stages s ON s.id = d.stage_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
WHERE d.deleted_at IS NULL AND (s.is_won OR s.is_lost)
  AND d.closed_at >= sqlc.arg('from_at')::timestamptz AND d.closed_at < sqlc.arg('to_at')::timestamptz
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
GROUP BY 1
ORDER BY 1;

-- name: DashWonByOwnerMonthly :many
SELECT to_char(d.closed_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM')::text AS month,
       coalesce(d.owner_user_id, '')::text AS owner_user_id,
       coalesce(sum(coalesce(d.value_final, d.value_estimate, 0)), 0)::float8 AS won_value
FROM sales.deals d
JOIN sales.stages s ON s.id = d.stage_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
WHERE d.deleted_at IS NULL AND s.is_won
  AND d.closed_at >= sqlc.arg('from_at')::timestamptz AND d.closed_at < sqlc.arg('to_at')::timestamptz
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
GROUP BY 1, 2
ORDER BY 1, 2;

-- name: DashOpenAsOf :one
-- Open pipeline at an instant, weighted by the stage each deal was in then.
SELECT count(*)::int AS open_count,
       coalesce(sum(x.v), 0)::float8 AS open_value,
       coalesce(sum(x.v * x.p / 100.0), 0)::float8 AS weighted_value
FROM (
    SELECT coalesce(d.value_final, d.value_estimate, 0) AS v,
           coalesce((SELECT s2.probability FROM sales.deal_stage_history h JOIN sales.stages s2 ON s2.id = h.stage_id
                     WHERE h.deal_id = d.id AND h.entered_at <= sqlc.arg('at')::timestamptz ORDER BY h.entered_at DESC LIMIT 1), s.probability)::int AS p
    FROM sales.deals d
    JOIN sales.stages s ON s.id = d.stage_id
    LEFT JOIN sales.leads l ON l.id = d.lead_id
    WHERE d.deleted_at IS NULL AND d.created_at <= sqlc.arg('at')::timestamptz AND (d.closed_at IS NULL OR d.closed_at > sqlc.arg('at')::timestamptz)
      AND (s.is_won OR s.is_lost OR d.closed_at IS NULL)
      AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
      AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
      AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
      AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
) x;

-- name: DashOpenDeals :many
SELECT d.id, d.title, coalesce(a.name, '')::text AS account_name, d.stage_id, s.name AS stage_name, s.sort_order,
       s.probability, coalesce(s.stuck_threshold_days, 0)::int AS threshold_days, coalesce(d.owner_user_id, '')::text AS owner_user_id,
       coalesce(d.value_final, d.value_estimate, 0)::float8 AS value, d.service_lines, d.entered_stage_at, d.created_at
FROM sales.deals d
JOIN sales.stages s ON s.id = d.stage_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
WHERE d.deleted_at IS NULL AND NOT s.is_won AND NOT s.is_lost
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
ORDER BY s.sort_order, d.entered_stage_at
LIMIT 5000;

-- name: DashFunnel :many
-- Deals created in the window that reached each non-lost stage of the pipeline.
WITH cohort AS (
    SELECT d.id, coalesce(d.value_final, d.value_estimate, 0) AS v,
           greatest(cs.sort_order * (NOT cs.is_lost)::int,
                    coalesce((SELECT max(s2.sort_order) FROM sales.deal_stage_history h JOIN sales.stages s2 ON s2.id = h.stage_id
                              WHERE h.deal_id = d.id AND NOT s2.is_lost), 0)) AS reached
    FROM sales.deals d
    JOIN sales.stages cs ON cs.id = d.stage_id
    LEFT JOIN sales.leads l ON l.id = d.lead_id
    WHERE d.deleted_at IS NULL AND d.pipeline_id = sqlc.arg('pipeline_id')
      AND d.created_at >= sqlc.arg('from_at')::timestamptz AND d.created_at < sqlc.arg('to_at')::timestamptz
      AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
      AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
      AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
)
SELECT s.id, s.name, s.sort_order, count(c.id)::int AS deals, coalesce(sum(c.v), 0)::float8 AS value
FROM sales.stages s
LEFT JOIN cohort c ON c.reached >= s.sort_order
WHERE s.pipeline_id = sqlc.arg('pipeline_id') AND NOT s.is_lost AND s.is_active
GROUP BY s.id, s.name, s.sort_order
ORDER BY s.sort_order;

-- name: DashStageDays :many
-- Average time spent in a stage, for stage exits inside the window.
WITH hx AS (
    SELECT h1.deal_id, h1.stage_id, h1.entered_at,
           (lead(h1.entered_at) OVER (PARTITION BY h1.deal_id ORDER BY h1.entered_at))::timestamptz AS left_at
    FROM sales.deal_stage_history h1
)
SELECT s.id, s.name, s.sort_order, count(*)::int AS exits,
       avg(extract(epoch FROM hx.left_at - hx.entered_at) / 86400)::float8 AS avg_days
FROM hx
JOIN sales.stages s ON s.id = hx.stage_id
JOIN sales.deals d ON d.id = hx.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
WHERE hx.left_at IS NOT NULL AND hx.left_at >= sqlc.arg('from_at')::timestamptz AND hx.left_at < sqlc.arg('to_at')::timestamptz
  AND d.deleted_at IS NULL AND NOT s.is_won AND NOT s.is_lost
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
GROUP BY s.id, s.name, s.sort_order
ORDER BY s.sort_order;

-- name: DashClosedBreakdown :many
-- dim: line (won, value split across lines) | industry (won) | source (won) | lost_reason (lost).
WITH fd AS (
    SELECT d.id, d.service_lines, d.lost_reason_id, d.account_id, l.source AS lead_source, s.is_won, s.is_lost,
           coalesce(d.value_final, d.value_estimate, 0) AS v
    FROM sales.deals d
    JOIN sales.stages s ON s.id = d.stage_id
    LEFT JOIN sales.leads l ON l.id = d.lead_id
    WHERE d.deleted_at IS NULL AND (s.is_won OR s.is_lost)
      AND d.closed_at >= sqlc.arg('from_at')::timestamptz AND d.closed_at < sqlc.arg('to_at')::timestamptz
      AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
      AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
      AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
      AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
)
SELECT 'line'::text AS dim, ln.line::text AS key, ''::text AS label, count(*)::int AS deals,
       coalesce(sum(fd.v / greatest(cardinality(fd.service_lines), 1)), 0)::float8 AS value
FROM fd CROSS JOIN LATERAL unnest(CASE WHEN cardinality(fd.service_lines) = 0 THEN ARRAY['lainnya']::text[] ELSE fd.service_lines END) AS ln(line)
WHERE fd.is_won
GROUP BY 2
UNION ALL
SELECT 'industry', coalesce(nullif(a.industry, ''), 'Lainnya'), '', count(*)::int, coalesce(sum(fd.v), 0)::float8
FROM fd LEFT JOIN sales.accounts a ON a.id = fd.account_id
WHERE fd.is_won
GROUP BY 2
UNION ALL
SELECT 'source', coalesce(fd.lead_source, 'lainnya'), '', count(*)::int, coalesce(sum(fd.v), 0)::float8
FROM fd
WHERE fd.is_won
GROUP BY 2
UNION ALL
SELECT 'lost_reason', coalesce(fd.lost_reason_id, ''), coalesce(lr.name, 'Tanpa alasan'), count(*)::int, coalesce(sum(fd.v), 0)::float8
FROM fd LEFT JOIN sales.lost_reasons lr ON lr.id = fd.lost_reason_id
WHERE fd.is_lost
GROUP BY 2, 3;

-- name: DashQuotes :many
-- Quotations sent in the window, by status.
SELECT q.status, count(*)::int AS quotes, coalesce(sum(q.total), 0)::float8 AS total,
       coalesce(sum(q.discount_percent), 0)::float8 AS discount_sum,
       coalesce(sum(extract(epoch FROM q.decided_at - q.sent_at) / 86400) FILTER (WHERE q.decided_at IS NOT NULL), 0)::float8 AS decide_days,
       count(*) FILTER (WHERE q.decided_at IS NOT NULL)::int AS decided
FROM sales.quotations q
JOIN sales.deals d ON d.id = q.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
WHERE q.deleted_at IS NULL AND d.deleted_at IS NULL AND q.sent_at IS NOT NULL
  AND q.sent_at >= sqlc.arg('from_at')::timestamptz AND q.sent_at < sqlc.arg('to_at')::timestamptz
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
GROUP BY q.status;

-- name: DashOwners :many
WITH fd AS (
    SELECT d.owner_user_id AS o, d.closed_at, s.is_won, s.is_lost, coalesce(d.value_final, d.value_estimate, 0) AS v
    FROM sales.deals d
    JOIN sales.stages s ON s.id = d.stage_id
    LEFT JOIN sales.leads l ON l.id = d.lead_id
    WHERE d.deleted_at IS NULL AND d.owner_user_id IS NOT NULL
      AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
      AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
      AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
      AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
),
closed AS (
    SELECT o, count(*) FILTER (WHERE is_won) AS wc, coalesce(sum(v) FILTER (WHERE is_won), 0) AS wv,
           count(*) FILTER (WHERE is_lost) AS lc
    FROM fd WHERE (is_won OR is_lost) AND closed_at >= sqlc.arg('from_at')::timestamptz AND closed_at < sqlc.arg('to_at')::timestamptz
    GROUP BY o
),
prev AS (
    SELECT o, coalesce(sum(v), 0) AS pv
    FROM fd WHERE is_won AND closed_at >= sqlc.arg('prev_from_at')::timestamptz AND closed_at < sqlc.arg('prev_to_at')::timestamptz
    GROUP BY o
),
open AS (
    SELECT o, count(*) AS oc, coalesce(sum(v), 0) AS ov FROM fd WHERE NOT is_won AND NOT is_lost GROUP BY o
),
act AS (
    SELECT a.owner_user_id AS o, count(*) AS ac, count(*) FILTER (WHERE a.status = 'done') AS ad
    FROM sales.activities a
    WHERE a.deleted_at IS NULL AND a.owner_user_id IS NOT NULL
      AND a.created_at >= sqlc.arg('from_at')::timestamptz AND a.created_at < sqlc.arg('to_at')::timestamptz
      AND (sqlc.narg('owner')::text IS NULL OR a.owner_user_id = sqlc.narg('owner'))
    GROUP BY a.owner_user_id
),
owners AS (
    SELECT o FROM closed UNION SELECT o FROM open UNION SELECT o FROM act UNION SELECT o FROM prev
)
SELECT ow.o::text AS owner_user_id,
       coalesce(open.oc, 0)::int AS open_deals, coalesce(open.ov, 0)::float8 AS pipeline_value,
       coalesce(closed.wc, 0)::int AS won_deals, coalesce(closed.wv, 0)::float8 AS won_value,
       coalesce(closed.lc, 0)::int AS lost_deals,
       coalesce(act.ac, 0)::int AS activities, coalesce(act.ad, 0)::int AS activities_done,
       coalesce(prev.pv, 0)::float8 AS previous_won_value
FROM owners ow
LEFT JOIN closed ON closed.o = ow.o
LEFT JOIN open ON open.o = ow.o
LEFT JOIN act ON act.o = ow.o
LEFT JOIN prev ON prev.o = ow.o
ORDER BY won_value DESC, pipeline_value DESC;

-- name: DashTargets :many
SELECT coalesce(user_id, '')::text AS user_id, period_month, target_value::float8 AS target_value
FROM sales.targets
WHERE period_month >= sqlc.arg('from_month')::text AND period_month <= sqlc.arg('to_month')::text;

-- name: DashLeadsMonthly :many
SELECT to_char(l.created_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM')::text AS month, l.source, count(*)::int AS leads
FROM sales.leads l
WHERE l.deleted_at IS NULL AND l.created_at >= sqlc.arg('from_at')::timestamptz AND l.created_at < sqlc.arg('to_at')::timestamptz
  AND (sqlc.narg('owner')::text IS NULL OR l.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
GROUP BY 1, 2
ORDER BY 1, 2;

-- name: DashLeadSources :many
SELECT l.source, count(*)::int AS leads,
       count(*) FILTER (WHERE l.status = 'qualified')::int AS qualified,
       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM sales.deals d WHERE d.lead_id = l.id AND d.deleted_at IS NULL))::int AS converted,
       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM sales.deals d JOIN sales.stages s ON s.id = d.stage_id
                                      WHERE d.lead_id = l.id AND d.deleted_at IS NULL AND s.is_won))::int AS won
FROM sales.leads l
WHERE l.deleted_at IS NULL AND l.created_at >= sqlc.arg('from_at')::timestamptz AND l.created_at < sqlc.arg('to_at')::timestamptz
  AND (sqlc.narg('owner')::text IS NULL OR l.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
GROUP BY l.source
ORDER BY leads DESC;

-- name: DashLeadScores :many
SELECT least(greatest(l.score, 0) / 20, 4)::int AS bucket, count(*)::int AS leads
FROM sales.leads l
WHERE l.deleted_at IS NULL AND l.created_at >= sqlc.arg('from_at')::timestamptz AND l.created_at < sqlc.arg('to_at')::timestamptz
  AND (sqlc.narg('owner')::text IS NULL OR l.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
GROUP BY 1
ORDER BY 1;

-- name: DashConversations :many
SELECT to_char(c.created_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM')::text AS month, c.channel,
       count(*)::int AS conversations,
       count(c.first_response_seconds)::int AS responded,
       coalesce(sum(c.first_response_seconds), 0)::float8 AS response_seconds,
       count(*) FILTER (WHERE c.sla_response_breached)::int AS breaches
FROM sales.lead_conversations c
WHERE c.created_at >= sqlc.arg('from_at')::timestamptz AND c.created_at < sqlc.arg('to_at')::timestamptz
  AND (sqlc.narg('owner')::text IS NULL OR c.assigned_user_id = sqlc.narg('owner'))
GROUP BY 1, 2
ORDER BY 1, 2;

-- name: DashFormSubmissions :many
SELECT f.id, f.name, count(*) FILTER (WHERE fs.status = 'accepted')::int AS accepted,
       count(*) FILTER (WHERE fs.status <> 'accepted')::int AS rejected
FROM sales.form_submissions fs
JOIN sales.forms f ON f.id = fs.form_id
WHERE fs.created_at >= sqlc.arg('from_at')::timestamptz AND fs.created_at < sqlc.arg('to_at')::timestamptz
GROUP BY f.id, f.name
ORDER BY accepted DESC;

-- name: DashBillingMonthly :many
-- kind: invoiced (issued or submitted, by issue date) | collected (payments by
-- paid_on; n and pay_days cover payments linked to an invoice).
SELECT 'invoiced'::text AS kind, to_char(coalesce(i.sent_at, i.created_at) AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM')::text AS month,
       count(*)::int AS n, coalesce(sum(i.amount), 0)::float8 AS amount, 0::float8 AS pay_days
FROM sales.invoices i
JOIN sales.deals d ON d.id = i.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
WHERE i.status <> 'batal' AND d.deleted_at IS NULL
  AND coalesce(i.sent_at, i.created_at) >= sqlc.arg('from_at')::timestamptz AND coalesce(i.sent_at, i.created_at) < sqlc.arg('to_at')::timestamptz
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
GROUP BY 2
UNION ALL
SELECT 'collected', to_char(p.paid_on, 'YYYY-MM'), count(i.id)::int, coalesce(sum(p.amount), 0)::float8,
       coalesce(sum(p.paid_on - (coalesce(i.sent_at, i.created_at) AT TIME ZONE 'Asia/Jakarta')::date) FILTER (WHERE i.id IS NOT NULL), 0)::float8
FROM sales.payments p
JOIN sales.deals d ON d.id = p.deal_id
LEFT JOIN sales.invoices i ON i.id = p.invoice_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
WHERE p.deleted_at IS NULL AND d.deleted_at IS NULL
  AND p.paid_on >= sqlc.arg('from_day')::date AND p.paid_on < sqlc.arg('to_day')::date
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
GROUP BY 2;

-- name: DashReceivables :many
-- Issued invoices with what is still unpaid on them as of a day.
SELECT i.id, i.invoice_number, d.id AS deal_id, d.title AS deal_title, coalesce(a.name, '')::text AS account_name,
       i.amount::float8 AS amount, i.due_date,
       coalesce((SELECT sum(p.amount) FROM sales.payments p WHERE p.invoice_id = i.id AND p.deleted_at IS NULL AND p.paid_on <= sqlc.arg('as_of')::date), 0)::float8 AS paid
FROM sales.invoices i
JOIN sales.deals d ON d.id = i.deal_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
WHERE i.status = 'terkirim' AND d.deleted_at IS NULL
  AND (coalesce(i.sent_at, i.created_at) AT TIME ZONE 'Asia/Jakarta')::date <= sqlc.arg('as_of')::date
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
ORDER BY i.due_date NULLS LAST
LIMIT 5000;

-- name: DashTermProgress :many
-- Won deals whose accepted quotation has payment terms.
SELECT d.id AS deal_id, d.title AS deal_title, coalesce(a.name, '')::text AS account_name, q.total::float8 AS contract_value,
       (SELECT count(*) FROM sales.quotation_terms t WHERE t.quotation_id = q.id)::int AS terms,
       (SELECT count(DISTINCT i.term_id) FROM sales.invoices i WHERE i.quotation_id = q.id AND i.status = 'terkirim')::int AS terms_invoiced,
       coalesce((SELECT sum(i.amount) FROM sales.invoices i WHERE i.quotation_id = q.id AND i.status = 'terkirim'), 0)::float8 AS invoiced,
       coalesce((SELECT sum(p.amount) FROM sales.payments p WHERE p.deal_id = d.id AND p.deleted_at IS NULL), 0)::float8 AS paid
FROM sales.deals d
JOIN sales.stages s ON s.id = d.stage_id
JOIN sales.quotations q ON q.deal_id = d.id AND q.status = 'diterima' AND q.deleted_at IS NULL
LEFT JOIN sales.accounts a ON a.id = d.account_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
WHERE d.deleted_at IS NULL AND s.is_won
  AND EXISTS (SELECT 1 FROM sales.quotation_terms t WHERE t.quotation_id = q.id)
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
ORDER BY q.total - coalesce((SELECT sum(p.amount) FROM sales.payments p WHERE p.deal_id = d.id AND p.deleted_at IS NULL), 0) DESC
LIMIT 50;
