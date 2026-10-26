-- name: ListOffers :many
SELECT sqlc.embed(o), ''::text AS candidate_name, ''::text AS candidate_email
FROM recruitment.candidate_offers o
WHERE o.candidate_id = $1
ORDER BY o.version DESC;

-- name: GetOffer :one
SELECT sqlc.embed(o), c.full_name AS candidate_name, c.email AS candidate_email
FROM recruitment.candidate_offers o
JOIN recruitment.candidates c ON c.id = o.candidate_id
WHERE o.id = $1;

-- name: OfferByToken :one
SELECT sqlc.embed(o), c.full_name AS candidate_name, c.email AS candidate_email
FROM recruitment.candidate_offers o
JOIN recruitment.candidates c ON c.id = o.candidate_id
WHERE o.token = $1;

-- name: ExpireOpenOffers :exec
-- A new revision replaces any offer still open.
UPDATE recruitment.candidate_offers SET status = 'expired', updated_at = now()
WHERE candidate_id = $1 AND status IN ('sent', 'negotiating');

-- name: NextOfferVersion :one
SELECT (COALESCE(MAX(version), 0) + 1)::int FROM recruitment.candidate_offers WHERE candidate_id = $1;

-- name: InsertOffer :exec
INSERT INTO recruitment.candidate_offers (
    id, candidate_id, version, token, status, position_title, base_salary, benefits, start_date, notes,
    response_note, responded_at, response_source, sent_at, expires_at, created_by, created_by_name
) VALUES (
    @id, @candidate_id, @version, @token, @status, sqlc.narg(position_title), @base_salary, @benefits, sqlc.narg(start_date),
    sqlc.narg(notes), sqlc.narg(response_note), sqlc.narg(responded_at), sqlc.narg(response_source), sqlc.narg(sent_at),
    sqlc.narg(expires_at), sqlc.narg(created_by), sqlc.narg(created_by_name)
);

-- name: RespondOffer :execrows
-- A portal response while the offer is still open.
UPDATE recruitment.candidate_offers
SET status = @status, response_note = sqlc.narg(response_note), responded_at = now(),
    response_ip = sqlc.narg(response_ip), response_source = 'portal', updated_at = now()
WHERE id = @id AND status IN ('sent', 'negotiating');

-- name: RecordManualResponse :execrows
UPDATE recruitment.candidate_offers
SET status = @status, response_note = sqlc.narg(response_note), responded_at = now(), response_source = 'manual', updated_at = now()
WHERE id = @id AND status NOT IN ('accepted', 'declined');

-- name: SetOfferStatus :exec
UPDATE recruitment.candidate_offers SET status = @status, updated_at = now() WHERE id = @id;

-- name: LatestAcceptedOffer :one
SELECT sqlc.embed(o), ''::text AS candidate_name, ''::text AS candidate_email
FROM recruitment.candidate_offers o
WHERE o.candidate_id = $1 AND o.status = 'accepted'
ORDER BY o.version DESC
LIMIT 1;

-- name: ExpiringOffers :many
SELECT sqlc.embed(o), c.full_name AS candidate_name, c.email AS candidate_email
FROM recruitment.candidate_offers o
JOIN recruitment.candidates c ON c.id = o.candidate_id
WHERE o.status IN ('sent', 'negotiating') AND o.expires_at > @now::timestamptz AND o.expires_at <= @until::timestamptz
ORDER BY o.expires_at;

-- ─── lapsed links ────────────────────────────────────────────

-- name: ExpireLapsedOffers :execrows
UPDATE recruitment.candidate_offers SET status = 'expired', updated_at = now()
WHERE status IN ('sent', 'negotiating')
  AND COALESCE(expires_at, COALESCE(sent_at, 'epoch'::timestamptz) + interval '30 days') < @now::timestamptz;

-- name: ExpireLapsedPsikotes :execrows
UPDATE recruitment.psikotes_sessions SET status = 'expired', updated_at = now()
WHERE status IN ('draft', 'sent', 'in_progress')
  AND COALESCE(expires_at, COALESCE(invited_at, 'epoch'::timestamptz) + interval '14 days') < @now::timestamptz;

-- name: ExpireLapsedAiSessions :execrows
UPDATE recruitment.interview_ai_sessions SET status = 'expired', updated_at = now()
WHERE status IN ('sent', 'in_progress')
  AND COALESCE(expires_at, COALESCE(invited_at, 'epoch'::timestamptz) + interval '14 days') < @now::timestamptz;

-- ─── plumbing ────────────────────────────────────────────────

-- name: HitRateLimit :one
-- Fixed window: the first hit opens it, it resets window seconds later.
INSERT INTO recruitment.rate_limits AS r (key, window_start, hits) VALUES (@key, @now, 1)
ON CONFLICT (key) DO UPDATE SET
    window_start = CASE WHEN r.window_start + make_interval(secs => @window_seconds::float8) <= @now THEN @now ELSE r.window_start END,
    hits = CASE WHEN r.window_start + make_interval(secs => @window_seconds::float8) <= @now THEN 1 ELSE r.hits + 1 END
RETURNING hits;

-- name: PruneRateLimits :exec
DELETE FROM recruitment.rate_limits WHERE window_start < @before;

-- name: ClaimReminder :execrows
INSERT INTO recruitment.reminder_sends (kind, ref_id, day) VALUES (@kind, @ref_id, @day)
ON CONFLICT DO NOTHING;
