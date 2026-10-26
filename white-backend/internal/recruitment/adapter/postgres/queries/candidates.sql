-- name: ListCandidates :many
SELECT sqlc.embed(c), COALESCE(j.title, '')::text AS job_opening_title, count(*) OVER ()::int AS total
FROM recruitment.candidates c
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE (cardinality(@statuses::text[]) = 0 OR c.status = ANY (@statuses::text[]))
  AND (sqlc.narg(job_opening_id)::text IS NULL OR c.job_opening_id = sqlc.narg(job_opening_id))
  AND (sqlc.narg(search)::text IS NULL OR c.full_name ILIKE sqlc.narg(search) OR c.email ILIKE sqlc.narg(search)
       OR c.phone ILIKE sqlc.narg(search) OR c.domicile ILIKE sqlc.narg(search))
  AND (sqlc.narg(date_from)::date IS NULL OR c.created_at >= (sqlc.narg(date_from)::date::timestamp AT TIME ZONE 'Asia/Jakarta'))
  AND (sqlc.narg(date_to)::date IS NULL OR c.created_at < ((sqlc.narg(date_to)::date + 1)::timestamp AT TIME ZONE 'Asia/Jakarta'))
  AND (sqlc.narg(source)::text IS NULL OR c.source = sqlc.narg(source))
ORDER BY CASE WHEN @sort_updated::bool THEN c.updated_at ELSE c.created_at END DESC, c.id
LIMIT @lim OFFSET @off;

-- name: GetCandidate :one
SELECT sqlc.embed(c), COALESCE(j.title, '')::text AS job_opening_title
FROM recruitment.candidates c
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE c.id = $1;

-- name: InsertCandidate :exec
INSERT INTO recruitment.candidates (
    id, full_name, email, phone, domicile, source, job_opening_id, status, notes, last_experience,
    last_education, availability, expected_salary, cv_key, cv_file_name, cv_content_type, created_by, updated_by
) VALUES (
    @id, @full_name, @email, @phone, @domicile, @source, sqlc.narg(job_opening_id), @status, sqlc.narg(notes),
    sqlc.narg(last_experience), sqlc.narg(last_education), sqlc.narg(availability), sqlc.narg(expected_salary),
    sqlc.narg(cv_key), sqlc.narg(cv_file_name), sqlc.narg(cv_content_type), sqlc.narg(created_by), sqlc.narg(created_by)
);

-- name: UpdateCandidateProfile :execrows
UPDATE recruitment.candidates SET
    full_name = @full_name, email = @email, phone = @phone, domicile = @domicile, source = @source,
    job_opening_id = sqlc.narg(job_opening_id), notes = sqlc.narg(notes), last_experience = sqlc.narg(last_experience),
    last_education = sqlc.narg(last_education), availability = sqlc.narg(availability),
    expected_salary = sqlc.narg(expected_salary), updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id = @id;

-- name: SetCandidateStatus :execrows
UPDATE recruitment.candidates SET
    status = @status,
    hired_at = CASE WHEN @status = 'hired' THEN COALESCE(hired_at, now()) ELSE hired_at END,
    updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id = @id;

-- name: MarkHiredEvent :exec
UPDATE recruitment.candidates SET hired_event_at = @at WHERE id = @id;

-- name: PendingHiredEvents :many
SELECT sqlc.embed(c), COALESCE(j.title, '')::text AS job_opening_title
FROM recruitment.candidates c
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE c.status = 'hired' AND c.hired_event_at IS NULL
ORDER BY c.hired_at
LIMIT 100;

-- name: DeleteCandidate :execrows
DELETE FROM recruitment.candidates WHERE id = $1;

-- name: SetCandidateCV :execrows
UPDATE recruitment.candidates SET
    cv_key = sqlc.narg(cv_key), cv_file_name = sqlc.narg(cv_file_name), cv_content_type = sqlc.narg(cv_content_type),
    updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id = @id;

-- name: CandidateAppliedTo :one
SELECT EXISTS (
    SELECT 1 FROM recruitment.candidates
    WHERE lower(email) = lower(@email) AND job_opening_id = @job_opening_id
)::bool;

-- name: CountCandidatesByStatus :many
SELECT status, count(*)::int AS n
FROM recruitment.candidates
WHERE sqlc.narg(job_opening_id)::text IS NULL OR job_opening_id = sqlc.narg(job_opening_id)
GROUP BY status;

-- name: CountCandidatesBySource :many
SELECT source, count(*)::int AS n FROM recruitment.candidates GROUP BY source ORDER BY n DESC, source;

-- name: SummaryCounts :one
SELECT
    (SELECT count(*) FROM recruitment.job_openings jo WHERE jo.deleted_at IS NULL AND jo.status = 'published')::int AS open_jobs,
    (SELECT count(*) FROM recruitment.candidates c1 WHERE c1.status = ANY (@active::text[]))::int AS active_candidates,
    (SELECT count(*) FROM recruitment.candidates c2 WHERE c2.created_at >= @since30::timestamptz)::int AS applied_last_30,
    (SELECT count(*) FROM recruitment.candidates c3 WHERE c3.status = 'hired' AND c3.hired_at >= @since90::timestamptz)::int AS hired_last_90,
    (SELECT count(*) FROM recruitment.interviews iv WHERE iv.status = 'scheduled' AND iv.interview_date >= @now::timestamptz
        AND iv.interview_date < @next7::timestamptz)::int AS interviews_next_7,
    (SELECT count(*) FROM recruitment.candidate_offers o WHERE o.status IN ('sent', 'negotiating'))::int AS open_offers,
    COALESCE((SELECT avg(EXTRACT(EPOCH FROM c4.hired_at - c4.created_at) / 86400)
       FROM recruitment.candidates c4 WHERE c4.status = 'hired' AND c4.hired_at >= @since90::timestamptz), -1)::float8 AS avg_days_to_hire;

-- ─── notes and activities ────────────────────────────────────

-- name: ListNotes :many
SELECT * FROM recruitment.candidate_notes WHERE candidate_id = $1 ORDER BY created_at DESC, id;

-- name: InsertNote :exec
INSERT INTO recruitment.candidate_notes (id, candidate_id, content, created_by, created_by_name, created_at)
VALUES (@id, @candidate_id, @content, sqlc.narg(created_by), sqlc.narg(created_by_name), @created_at);

-- name: ListActivities :many
SELECT * FROM recruitment.candidate_activities WHERE candidate_id = $1 ORDER BY created_at DESC, id DESC;

-- name: InsertActivity :exec
INSERT INTO recruitment.candidate_activities (id, candidate_id, activity_type, description, created_by, created_by_name, created_at)
VALUES (@id, @candidate_id, @activity_type, @description, sqlc.narg(created_by), sqlc.narg(created_by_name), @created_at);

-- ─── screening, psikotes summary, AI analysis ────────────────

-- name: GetScreening :one
SELECT * FROM recruitment.candidate_screenings WHERE candidate_id = $1;

-- name: UpsertScreening :exec
INSERT INTO recruitment.candidate_screenings (
    candidate_id, contacted, interested, availability_note, confirmed_salary, willing_shift, willing_placement,
    notes, recommendation, updated_by, updated_by_name
) VALUES (
    @candidate_id, @contacted, sqlc.narg(interested), sqlc.narg(availability_note), sqlc.narg(confirmed_salary),
    sqlc.narg(willing_shift), sqlc.narg(willing_placement), sqlc.narg(notes), sqlc.narg(recommendation),
    sqlc.narg(updated_by), sqlc.narg(updated_by_name)
)
ON CONFLICT (candidate_id) DO UPDATE SET
    contacted = EXCLUDED.contacted, interested = EXCLUDED.interested, availability_note = EXCLUDED.availability_note,
    confirmed_salary = EXCLUDED.confirmed_salary, willing_shift = EXCLUDED.willing_shift,
    willing_placement = EXCLUDED.willing_placement, notes = EXCLUDED.notes, recommendation = EXCLUDED.recommendation,
    updated_by = EXCLUDED.updated_by, updated_by_name = EXCLUDED.updated_by_name, updated_at = now();

-- name: GetPsikotesSummary :one
SELECT * FROM recruitment.candidate_psikotes_summary WHERE candidate_id = $1;

-- name: UpsertPsikotesSummary :exec
INSERT INTO recruitment.candidate_psikotes_summary (candidate_id, recommendation, notes, updated_by, updated_by_name)
VALUES (@candidate_id, sqlc.narg(recommendation), sqlc.narg(notes), sqlc.narg(updated_by), sqlc.narg(updated_by_name))
ON CONFLICT (candidate_id) DO UPDATE SET
    recommendation = EXCLUDED.recommendation, notes = EXCLUDED.notes,
    updated_by = EXCLUDED.updated_by, updated_by_name = EXCLUDED.updated_by_name, updated_at = now();

-- name: GetAnalysis :one
SELECT * FROM recruitment.candidate_ai_analysis WHERE candidate_id = $1;

-- name: UpsertAnalysis :exec
INSERT INTO recruitment.candidate_ai_analysis (candidate_id, summary, match_score, match_reason, strengths, gaps, job_context, model)
VALUES (@candidate_id, @summary, @match_score, @match_reason, @strengths, @gaps, @job_context, @model)
ON CONFLICT (candidate_id) DO UPDATE SET
    summary = EXCLUDED.summary, match_score = EXCLUDED.match_score, match_reason = EXCLUDED.match_reason,
    strengths = EXCLUDED.strengths, gaps = EXCLUDED.gaps, job_context = EXCLUDED.job_context,
    model = EXCLUDED.model, updated_at = now();
