-- Recruitment dashboard aggregates. from_ts and to_ts bound the range in
-- WIB (to_ts exclusive). Every candidate query takes the same optional
-- filters: job_opening_id, department (opening department name), source
-- and stage (the candidate's current status).

-- name: DashboardTrend :many
-- One row per week (Monday start) clipped to the range: flows inside the
-- bucket and pipeline snapshots at its end.
WITH f AS (
    SELECT c.id, c.created_at, c.hired_at, c.status
    FROM recruitment.candidates c
    LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
    WHERE (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
      AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
      AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
      AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
),
o AS (
    SELECT DISTINCT ON (x.candidate_id) x.candidate_id, x.status, x.responded_at, x.expires_at,
           min(x.sent_at) OVER (PARTITION BY x.candidate_id) AS first_sent
    FROM recruitment.candidate_offers x
    JOIN f ON f.id = x.candidate_id
    ORDER BY x.candidate_id, x.version DESC
),
sp AS MATERIALIZED (
    SELECT s.entered_at, s.left_at
    FROM recruitment.candidate_stage_spans s
    JOIN f ON f.id = s.candidate_id
    WHERE s.status = ANY (@active::text[])
),
w AS (
    SELECT ow.opened_at, ow.closed_at, ow.headcount
    FROM recruitment.opening_windows ow
    WHERE (sqlc.narg(job_opening_id)::text IS NULL OR ow.id = sqlc.narg(job_opening_id))
      AND (sqlc.narg(department)::text IS NULL OR ow.department = sqlc.narg(department))
),
b AS (
    SELECT gs::date AS bucket,
           GREATEST(gs::timestamp AT TIME ZONE 'Asia/Jakarta', @from_ts::timestamptz) AS s,
           LEAST((gs + interval '7 days')::timestamp AT TIME ZONE 'Asia/Jakarta', @to_ts::timestamptz) AS e
    FROM generate_series(date_trunc('week', sqlc.arg(from_day)::date::timestamp), sqlc.arg(to_day)::date::timestamp, interval '1 week') gs
)
SELECT b.bucket,
    (SELECT count(*) FROM f WHERE f.created_at >= b.s AND f.created_at < b.e)::int AS applications,
    (SELECT count(*) FROM f WHERE f.status = 'hired' AND f.hired_at >= b.s AND f.hired_at < b.e)::int AS hires,
    (SELECT COALESCE(sum(EXTRACT(EPOCH FROM f.hired_at - f.created_at) / 86400), 0) FROM f
        WHERE f.status = 'hired' AND f.hired_at >= b.s AND f.hired_at < b.e)::float8 AS hire_days,
    (SELECT count(*) FROM o WHERE o.first_sent >= b.s AND o.first_sent < b.e)::int AS offers_sent,
    (SELECT count(*) FROM o WHERE o.status = 'accepted' AND o.responded_at >= b.s AND o.responded_at < b.e)::int AS offers_accepted,
    (SELECT count(*) FROM o WHERE o.status = 'declined' AND o.responded_at >= b.s AND o.responded_at < b.e)::int AS offers_declined,
    (SELECT count(*) FROM o WHERE o.status = 'expired' AND o.expires_at >= b.s AND o.expires_at < b.e)::int AS offers_expired,
    (SELECT count(*) FROM o WHERE o.status = 'negotiating' AND o.first_sent >= b.s AND o.first_sent < b.e)::int AS offers_negotiating,
    (SELECT count(*) FROM sp WHERE sp.entered_at < b.e AND (sp.left_at IS NULL OR sp.left_at >= b.e))::int AS in_process,
    (SELECT count(*) FROM w WHERE w.opened_at < b.e AND (w.closed_at IS NULL OR w.closed_at >= b.e))::int AS open_positions,
    (SELECT COALESCE(sum(w.headcount), 0) FROM w WHERE w.opened_at < b.e AND (w.closed_at IS NULL OR w.closed_at >= b.e))::int AS open_headcount
FROM b
ORDER BY b.bucket;

-- name: DashboardCohort :many
-- Applications in the range grouped by source, current status and the
-- furthest funnel stage reached (0 applied .. 5 hired), with days to hire.
WITH coh AS (
    SELECT c.id, c.source, c.status, c.created_at, c.hired_at
    FROM recruitment.candidates c
    LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
    WHERE c.created_at >= @from_ts::timestamptz
      AND c.created_at < @to_ts::timestamptz
      AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
      AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
      AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
      AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
),
ranked AS (
    SELECT coh.id, coh.source, coh.status, coh.created_at, coh.hired_at,
           COALESCE(max(CASE ev.status WHEN 'applied' THEN 0 WHEN 'screening' THEN 1 WHEN 'psikotes' THEN 2
               WHEN 'interview' THEN 3 WHEN 'offer' THEN 4 WHEN 'hired' THEN 5 END), 0)::int AS max_rank
    FROM coh
    LEFT JOIN recruitment.candidate_stage_events ev ON ev.candidate_id = coh.id
    GROUP BY coh.id, coh.source, coh.status, coh.created_at, coh.hired_at
)
SELECT r.source, r.status, r.max_rank, count(*)::int AS n,
    COALESCE(sum(EXTRACT(EPOCH FROM r.hired_at - r.created_at) / 86400) FILTER (WHERE r.status = 'hired'), 0)::float8 AS hire_days
FROM ranked r
GROUP BY r.source, r.status, r.max_rank
ORDER BY r.source, r.status, r.max_rank;

-- name: DashboardSourceTrend :many
SELECT date_trunc('week', c.created_at AT TIME ZONE 'Asia/Jakarta')::date AS bucket, c.source, count(*)::int AS n
FROM recruitment.candidates c
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE c.created_at >= @from_ts::timestamptz
  AND c.created_at < @to_ts::timestamptz
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
  AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
GROUP BY 1, 2
ORDER BY 1, 2;

-- name: DashboardStageDurations :many
-- Days spent per pipeline stage by applications in the range; a stay that
-- is still open counts until now.
WITH coh AS (
    SELECT c.id
    FROM recruitment.candidates c
    LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
    WHERE c.created_at >= @from_ts::timestamptz
      AND c.created_at < @to_ts::timestamptz
      AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
      AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
      AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
      AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
),
d AS (
    SELECT sp.status, EXTRACT(EPOCH FROM COALESCE(sp.left_at, @now::timestamptz) - sp.entered_at) / 86400 AS days
    FROM recruitment.candidate_stage_spans sp
    JOIN coh ON coh.id = sp.candidate_id
    WHERE sp.status = ANY (@active::text[])
)
SELECT d.status, count(*)::int AS n,
    avg(d.days)::float8 AS avg_days,
    min(d.days)::float8 AS min_days,
    percentile_cont(0.25) WITHIN GROUP (ORDER BY d.days)::float8 AS p25_days,
    percentile_cont(0.5) WITHIN GROUP (ORDER BY d.days)::float8 AS median_days,
    percentile_cont(0.75) WITHIN GROUP (ORDER BY d.days)::float8 AS p75_days,
    max(d.days)::float8 AS max_days
FROM d
GROUP BY d.status;

-- name: DashboardOpeningStages :many
-- Applications in the range by opening and current status.
SELECT COALESCE(c.job_opening_id, '')::text AS job_opening_id, COALESCE(j.title, '')::text AS title, c.status, count(*)::int AS n
FROM recruitment.candidates c
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE c.created_at >= @from_ts::timestamptz
  AND c.created_at < @to_ts::timestamptz
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
  AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
GROUP BY 1, 2, 3
ORDER BY 2, 3;

-- name: DashboardStuck :many
-- Active candidates whose current stage started before the cutoff, longest first.
SELECT c.id, c.full_name, COALESCE(c.job_opening_id, '')::text AS job_opening_id, COALESCE(j.title, '')::text AS title,
       c.status, cur.entered_at::timestamptz AS entered_at
FROM recruitment.candidates c
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
JOIN LATERAL (SELECT max(ev.entered_at) AS entered_at FROM recruitment.candidate_stage_events ev WHERE ev.candidate_id = c.id) cur ON true
WHERE c.status = ANY (@active::text[])
  AND cur.entered_at < @before::timestamptz
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
  AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
ORDER BY cur.entered_at, c.id
LIMIT 25;

-- name: DashboardOpenings :many
-- Openings taking applications at some point of the range, with their
-- applicants (source and stage filters apply to the counts).
SELECT w.id, w.title, w.department, w.status, w.headcount, w.closing_date, w.opened_at,
    (w.closed_at IS NOT NULL)::bool AS has_closed, COALESCE(w.closed_at, w.opened_at)::timestamptz AS closed_at,
    count(c.id)::int AS applicants_total,
    count(c.id) FILTER (WHERE c.created_at >= @from_ts::timestamptz
        AND c.created_at < @to_ts::timestamptz)::int AS applicants_period,
    count(c.id) FILTER (WHERE c.status = 'applied')::int AS applied,
    count(c.id) FILTER (WHERE c.status = 'screening')::int AS screening,
    count(c.id) FILTER (WHERE c.status = 'psikotes')::int AS psikotes,
    count(c.id) FILTER (WHERE c.status = 'interview')::int AS interview,
    count(c.id) FILTER (WHERE c.status = 'offer')::int AS offer,
    count(c.id) FILTER (WHERE c.status = 'hired')::int AS hired,
    count(c.id) FILTER (WHERE c.status = 'rejected')::int AS rejected,
    count(c.id) FILTER (WHERE c.status IN ('talent_pool', 'archived'))::int AS pooled,
    COALESCE(EXTRACT(EPOCH FROM (SELECT h.hired_at FROM recruitment.candidates h WHERE h.job_opening_id = w.id AND h.status = 'hired'
        ORDER BY h.hired_at OFFSET GREATEST(w.headcount - 1, 0) LIMIT 1) - w.opened_at) / 86400, -1)::float8 AS fill_days
FROM recruitment.opening_windows w
LEFT JOIN recruitment.candidates c ON c.job_opening_id = w.id
    AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
    AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
WHERE w.opened_at < @to_ts::timestamptz
  AND (w.closed_at IS NULL OR w.closed_at >= @from_ts::timestamptz)
  AND (sqlc.narg(job_opening_id)::text IS NULL OR w.id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR w.department = sqlc.narg(department))
GROUP BY w.id, w.title, w.department, w.status, w.headcount, w.closing_date, w.opened_at, w.closed_at
ORDER BY w.opened_at DESC, w.id;

-- name: DashboardPsikotesStatus :many
SELECT s.status, count(*)::int AS n
FROM recruitment.psikotes_sessions s
JOIN recruitment.candidates c ON c.id = s.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE s.invited_at >= @from_ts::timestamptz
  AND s.invited_at < @to_ts::timestamptz
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
  AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
GROUP BY s.status;

-- name: DashboardPsikotesScores :many
-- Scored tests finished in the range in 10-point buckets (90 holds 90-100).
SELECT i.name AS instrument, LEAST(floor(t.score / 10) * 10, 90)::int AS bucket, count(*)::int AS n,
    avg(t.score)::float8 AS avg_score
FROM recruitment.psikotes_session_tests t
JOIN recruitment.psikotes_instruments i ON i.id = t.instrument_id
JOIN recruitment.psikotes_sessions s ON s.id = t.session_id
JOIN recruitment.candidates c ON c.id = s.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE t.score IS NOT NULL
  AND t.completed_at >= @from_ts::timestamptz
  AND t.completed_at < @to_ts::timestamptz
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
  AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
GROUP BY i.name, 2
ORDER BY i.name, 2;

-- name: DashboardAiInterviewStatus :many
SELECT s.status, count(*)::int AS n
FROM recruitment.interview_ai_sessions s
JOIN recruitment.candidates c ON c.id = s.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE s.invited_at >= @from_ts::timestamptz
  AND s.invited_at < @to_ts::timestamptz
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
  AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
GROUP BY s.status;

-- name: DashboardScorecards :many
-- Average scorecard rating per criterion and interview type for completed
-- interviews in the range.
SELECT iv.type, (item ->> 'criterion')::text AS criterion, count(*)::int AS n, avg((item ->> 'score')::numeric)::float8 AS avg_score
FROM recruitment.interviews iv
CROSS JOIN LATERAL jsonb_array_elements(iv.scorecard) AS item
JOIN recruitment.candidates c ON c.id = iv.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE iv.status = 'completed'
  AND iv.interview_date >= @from_ts::timestamptz
  AND iv.interview_date < @to_ts::timestamptz
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
  AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
GROUP BY iv.type, 2
ORDER BY iv.type, 2;

-- name: DashboardInterviewers :many
-- Interviews in the range per interviewer.
SELECT COALESCE(NULLIF(iv.interviewer_name, ''), '-')::text AS interviewer, COALESCE(iv.interviewer_employee_id, '')::text AS employee_id,
    count(*) FILTER (WHERE iv.status = 'scheduled')::int AS scheduled,
    count(*) FILTER (WHERE iv.status = 'completed')::int AS completed,
    count(*) FILTER (WHERE iv.status = 'cancelled')::int AS cancelled,
    count(*) FILTER (WHERE iv.recommendation = 'proceed')::int AS proceed,
    count(*) FILTER (WHERE iv.recommendation = 'pool')::int AS pool,
    count(*) FILTER (WHERE iv.recommendation = 'reject')::int AS reject,
    COALESCE(avg(iv.overall_score), -1)::float8 AS avg_score
FROM recruitment.interviews iv
JOIN recruitment.candidates c ON c.id = iv.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE iv.interview_date >= @from_ts::timestamptz
  AND iv.interview_date < @to_ts::timestamptz
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
  AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
GROUP BY 1, 2
ORDER BY count(*) DESC, 1;

-- name: DashboardOffers :many
-- The latest offer version per candidate whose first version went out in
-- the range, with the opening salary band and the expected salary.
WITH o AS (
    SELECT DISTINCT ON (x.candidate_id) x.candidate_id, x.status, x.base_salary, x.position_title, x.responded_at,
           min(x.sent_at) OVER (PARTITION BY x.candidate_id) AS first_sent,
           count(*) OVER (PARTITION BY x.candidate_id) AS versions
    FROM recruitment.candidate_offers x
    ORDER BY x.candidate_id, x.version DESC
)
SELECT c.id AS candidate_id, c.full_name, COALESCE(c.job_opening_id, '')::text AS job_opening_id,
    COALESCE(j.title, o.position_title, '')::text AS title, o.status, o.base_salary, j.salary_min, j.salary_max,
    c.expected_salary, o.first_sent::timestamptz AS sent_at, o.responded_at, o.versions::int AS versions
FROM o
JOIN recruitment.candidates c ON c.id = o.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE o.first_sent >= @from_ts::timestamptz
  AND o.first_sent < @to_ts::timestamptz
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
  AND (sqlc.narg(stage)::text IS NULL OR c.status = sqlc.narg(stage))
ORDER BY o.first_sent DESC, c.id;
