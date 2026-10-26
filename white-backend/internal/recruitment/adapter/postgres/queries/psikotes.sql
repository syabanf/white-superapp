-- name: ListInstruments :many
SELECT sqlc.embed(i),
       (SELECT count(*) FROM recruitment.psikotes_questions q WHERE q.instrument_id = i.id AND q.is_active)::int AS active_questions
FROM recruitment.psikotes_instruments i
ORDER BY i.sort_order, i.name;

-- name: GetInstrument :one
SELECT sqlc.embed(i),
       (SELECT count(*) FROM recruitment.psikotes_questions q WHERE q.instrument_id = i.id AND q.is_active)::int AS active_questions
FROM recruitment.psikotes_instruments i
WHERE i.id = $1;

-- name: ActiveInstruments :many
SELECT sqlc.embed(i), 0::int AS active_questions
FROM recruitment.psikotes_instruments i
WHERE i.id = ANY (@ids::text[]) AND i.is_active
ORDER BY i.sort_order, i.name;

-- name: UpdateInstrument :execrows
UPDATE recruitment.psikotes_instruments SET
    name = COALESCE(sqlc.narg(name), name),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    duration_seconds = COALESCE(sqlc.narg(duration_seconds), duration_seconds),
    question_count = CASE WHEN @clear_question_count::bool THEN NULL ELSE COALESCE(sqlc.narg(question_count), question_count) END,
    shuffle = COALESCE(sqlc.narg(shuffle), shuffle),
    instructions = COALESCE(sqlc.narg(instructions), instructions),
    updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id = @id;

-- name: ListQuestions :many
SELECT * FROM recruitment.psikotes_questions WHERE instrument_id = $1 ORDER BY sort_order, created_at, id;

-- name: GetQuestion :one
SELECT * FROM recruitment.psikotes_questions WHERE id = $1;

-- name: QuestionsByIDs :many
SELECT * FROM recruitment.psikotes_questions WHERE id = ANY (@ids::text[]);

-- name: InsertQuestion :exec
INSERT INTO recruitment.psikotes_questions (id, instrument_id, body, options, answer_key, sort_order, is_active, created_by, updated_by)
VALUES (@id, @instrument_id, @body, @options, sqlc.narg(answer_key), @sort_order, @is_active, sqlc.narg(created_by), sqlc.narg(created_by))
ON CONFLICT (id) DO NOTHING;

-- name: UpdateQuestion :execrows
UPDATE recruitment.psikotes_questions SET
    body = @body, options = @options, answer_key = sqlc.narg(answer_key), sort_order = @sort_order,
    is_active = @is_active, updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id = @id;

-- name: DeleteQuestion :execrows
DELETE FROM recruitment.psikotes_questions WHERE id = $1;

-- name: DrawQuestionIDs :many
-- The active bank in order, or shuffled; limit 0 takes every question.
SELECT id FROM recruitment.psikotes_questions
WHERE instrument_id = @instrument_id AND is_active
ORDER BY CASE WHEN @shuffle::bool THEN random() ELSE 0 END, sort_order, created_at, id
LIMIT CASE WHEN @lim::int > 0 THEN @lim::int ELSE NULL END;

-- ─── sessions ────────────────────────────────────────────────

-- name: InsertPsikotesSession :exec
INSERT INTO recruitment.psikotes_sessions (id, candidate_id, token, status, invited_at, expires_at, created_by, created_by_name)
VALUES (@id, @candidate_id, @token, @status, sqlc.narg(invited_at), sqlc.narg(expires_at), sqlc.narg(created_by), sqlc.narg(created_by_name));

-- name: InsertSessionTest :exec
INSERT INTO recruitment.psikotes_session_tests (id, session_id, instrument_id, sort_order)
VALUES (@id, @session_id, @instrument_id, @sort_order);

-- name: ListPsikotesSessions :many
SELECT sqlc.embed(s), ''::text AS candidate_name, ''::text AS position_title
FROM recruitment.psikotes_sessions s
WHERE s.candidate_id = $1
ORDER BY s.created_at DESC, s.id;

-- name: PsikotesSessionByToken :one
SELECT sqlc.embed(s), c.full_name AS candidate_name, COALESCE(j.title, '')::text AS position_title
FROM recruitment.psikotes_sessions s
JOIN recruitment.candidates c ON c.id = s.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE s.token = $1;

-- name: SetPsikotesSessionStatus :exec
UPDATE recruitment.psikotes_sessions SET status = @status, updated_at = now() WHERE id = @id;

-- name: StartPsikotesSession :execrows
UPDATE recruitment.psikotes_sessions
SET status = 'in_progress', webcam_consent = @webcam_consent, started_at = COALESCE(started_at, now()), updated_at = now()
WHERE id = @id AND status IN ('draft', 'sent', 'in_progress');

-- name: CompletePsikotesSession :execrows
UPDATE recruitment.psikotes_sessions SET status = 'completed', completed_at = now(), updated_at = now()
WHERE id = @id AND status = 'in_progress';

-- name: ListSessionTests :many
SELECT sqlc.embed(t), i.code AS instrument_code, i.name AS instrument_name, i.kind AS instrument_kind,
       i.duration_seconds, i.question_count, i.shuffle, i.instructions, s.candidate_id
FROM recruitment.psikotes_session_tests t
JOIN recruitment.psikotes_instruments i ON i.id = t.instrument_id
JOIN recruitment.psikotes_sessions s ON s.id = t.session_id
WHERE (sqlc.narg(session_id)::text IS NULL OR t.session_id = sqlc.narg(session_id))
  AND (sqlc.narg(candidate_id)::text IS NULL OR s.candidate_id = sqlc.narg(candidate_id))
ORDER BY t.session_id, t.sort_order, t.id;

-- name: GetSessionTest :one
SELECT sqlc.embed(t), i.code AS instrument_code, i.name AS instrument_name, i.kind AS instrument_kind,
       i.duration_seconds, i.question_count, i.shuffle, i.instructions, s.candidate_id
FROM recruitment.psikotes_session_tests t
JOIN recruitment.psikotes_instruments i ON i.id = t.instrument_id
JOIN recruitment.psikotes_sessions s ON s.id = t.session_id
WHERE t.id = @id AND (sqlc.narg(session_id)::text IS NULL OR t.session_id = sqlc.narg(session_id));

-- name: BeginTest :execrows
UPDATE recruitment.psikotes_session_tests
SET status = 'in_progress', question_ids = @question_ids, started_at = now(), updated_at = now()
WHERE id = @id AND status = 'pending';

-- name: MergeAnswers :one
UPDATE recruitment.psikotes_session_tests
SET answers = answers || @answers::jsonb, updated_at = now()
WHERE id = @id AND status = 'in_progress'
RETURNING (SELECT count(*) FROM jsonb_object_keys(answers || @answers::jsonb))::int AS saved;

-- name: ScoreTest :execrows
UPDATE recruitment.psikotes_session_tests
SET status = 'selesai', answers = @answers, score = sqlc.narg(score), score_detail = @score_detail,
    completed_at = now(), updated_at = now()
WHERE id = @id AND status = 'in_progress';

-- name: FinishDrawingTest :execrows
UPDATE recruitment.psikotes_session_tests
SET status = 'perlu_review', completed_at = now(), updated_at = now()
WHERE id = @id AND status = 'in_progress';

-- name: SetTestAttachment :execrows
UPDATE recruitment.psikotes_session_tests
SET attachment_key = @attachment_key, attachment_content_type = @attachment_content_type, updated_at = now()
WHERE id = @id AND status = 'in_progress';

-- name: UnfinishedTestNames :many
SELECT i.name
FROM recruitment.psikotes_session_tests t
JOIN recruitment.psikotes_instruments i ON i.id = t.instrument_id
WHERE t.session_id = $1 AND t.status NOT IN ('selesai', 'perlu_review', 'reviewed')
ORDER BY t.sort_order;

-- name: ReviewTest :execrows
UPDATE recruitment.psikotes_session_tests
SET status = 'reviewed', review_notes = @review_notes, reviewed_by = sqlc.narg(reviewed_by),
    reviewed_by_name = sqlc.narg(reviewed_by_name), reviewed_at = now(), updated_at = now()
WHERE id = @id;

-- ─── proctoring ──────────────────────────────────────────────

-- name: InsertProctorEvent :exec
INSERT INTO recruitment.proctor_events (id, session_type, session_id, event_type, meta, created_at)
VALUES (@id, @session_type, @session_id, @event_type, @meta, @created_at);

-- name: ListProctorEvents :many
SELECT * FROM recruitment.proctor_events
WHERE session_type = @session_type AND session_id = @session_id
ORDER BY created_at, id;

-- name: ProctorCounts :many
SELECT session_id, event_type, count(*)::int AS n
FROM recruitment.proctor_events
WHERE session_type = @session_type AND session_id = ANY (@session_ids::text[])
GROUP BY session_id, event_type;

-- name: SessionExists :one
SELECT CASE
    WHEN @session_type::text = 'psikotes' THEN EXISTS (SELECT 1 FROM recruitment.psikotes_sessions ps WHERE ps.id = @id::text)
    ELSE EXISTS (SELECT 1 FROM recruitment.interview_ai_sessions ai WHERE ai.id = @id::text)
END::bool;
