-- ─── AI interview ────────────────────────────────────────────

-- name: InsertAiSession :exec
INSERT INTO recruitment.interview_ai_sessions (
    id, candidate_id, token, status, max_questions, invited_at, expires_at, created_by, created_by_name
) VALUES (
    @id, @candidate_id, @token, @status, @max_questions, sqlc.narg(invited_at), sqlc.narg(expires_at),
    sqlc.narg(created_by), sqlc.narg(created_by_name)
);

-- name: ListAiSessions :many
SELECT sqlc.embed(s), ''::text AS candidate_name, ''::text AS position_title
FROM recruitment.interview_ai_sessions s
WHERE s.candidate_id = $1
ORDER BY s.created_at DESC, s.id;

-- name: AiSessionByToken :one
SELECT sqlc.embed(s), c.full_name AS candidate_name, COALESCE(j.title, '')::text AS position_title
FROM recruitment.interview_ai_sessions s
JOIN recruitment.candidates c ON c.id = s.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE s.token = $1;

-- name: SetAiSessionStatus :exec
UPDATE recruitment.interview_ai_sessions SET status = @status, updated_at = now() WHERE id = @id;

-- name: StartAiSession :execrows
UPDATE recruitment.interview_ai_sessions
SET status = 'in_progress', webcam_consent = true, started_at = now(), updated_at = now()
WHERE id = @id AND status = 'sent';

-- name: CompleteAiSession :exec
UPDATE recruitment.interview_ai_sessions
SET status = 'completed', completed_at = now(), ai_summary = sqlc.narg(ai_summary), summary_model = sqlc.narg(summary_model),
    summarized_at = CASE WHEN sqlc.narg(ai_summary)::jsonb IS NULL THEN NULL ELSE now() END, updated_at = now()
WHERE id = @id;

-- name: ListTurns :many
SELECT t.*
FROM recruitment.interview_ai_turns t
JOIN recruitment.interview_ai_sessions s ON s.id = t.session_id
WHERE (sqlc.narg(session_id)::text IS NULL OR t.session_id = sqlc.narg(session_id))
  AND (sqlc.narg(candidate_id)::text IS NULL OR s.candidate_id = sqlc.narg(candidate_id))
ORDER BY t.session_id, t.turn_no;

-- name: GetTurn :one
SELECT * FROM recruitment.interview_ai_turns WHERE id = @id AND session_id = @session_id;

-- name: InsertTurn :execrows
INSERT INTO recruitment.interview_ai_turns (id, session_id, turn_no, topic, question, answer_transcript, answer_mode, asked_at, answered_at)
VALUES (@id, @session_id, @turn_no, sqlc.narg(topic), @question, sqlc.narg(answer_transcript), sqlc.narg(answer_mode),
        @asked_at, sqlc.narg(answered_at))
ON CONFLICT (session_id, turn_no) DO NOTHING;

-- name: AnswerTurn :execrows
UPDATE recruitment.interview_ai_turns
SET answer_transcript = @answer_transcript, answer_mode = @answer_mode, answered_at = now()
WHERE id = @id AND session_id = @session_id AND answered_at IS NULL;

-- ─── scheduled interviews ────────────────────────────────────

-- name: ListInterviews :many
SELECT sqlc.embed(i), c.full_name AS candidate_name, COALESCE(j.title, '')::text AS job_opening_title
FROM recruitment.interviews i
JOIN recruitment.candidates c ON c.id = i.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE (sqlc.narg(candidate_id)::text IS NULL OR i.candidate_id = sqlc.narg(candidate_id))
  AND (sqlc.narg(status)::text IS NULL OR i.status = sqlc.narg(status))
  AND (sqlc.narg(interviewer_employee_id)::text IS NULL OR i.interviewer_employee_id = sqlc.narg(interviewer_employee_id))
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR i.interview_date >= sqlc.narg(from_at))
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR i.interview_date < sqlc.narg(to_at))
ORDER BY i.interview_date, i.id;

-- name: GetInterview :one
SELECT sqlc.embed(i), c.full_name AS candidate_name, COALESCE(j.title, '')::text AS job_opening_title
FROM recruitment.interviews i
JOIN recruitment.candidates c ON c.id = i.candidate_id
LEFT JOIN recruitment.job_openings j ON j.id = c.job_opening_id
WHERE i.id = $1;

-- name: InsertInterview :exec
INSERT INTO recruitment.interviews (
    id, candidate_id, interviewer_employee_id, interviewer_user_id, interviewer_name, interview_date, duration_minutes,
    type, mode, meeting_link, location, status, scorecard, overall_score, recommendation, notes, created_by, updated_by
) VALUES (
    @id, @candidate_id, sqlc.narg(interviewer_employee_id), sqlc.narg(interviewer_user_id), @interviewer_name,
    @interview_date, @duration_minutes, @type, @mode, sqlc.narg(meeting_link), sqlc.narg(location), @status, @scorecard,
    sqlc.narg(overall_score), sqlc.narg(recommendation), sqlc.narg(notes), sqlc.narg(created_by), sqlc.narg(created_by)
);

-- name: UpdateInterview :execrows
UPDATE recruitment.interviews SET
    interviewer_employee_id = sqlc.narg(interviewer_employee_id), interviewer_user_id = sqlc.narg(interviewer_user_id),
    interviewer_name = @interviewer_name, interview_date = @interview_date, duration_minutes = @duration_minutes,
    type = @type, mode = @mode, meeting_link = sqlc.narg(meeting_link), location = sqlc.narg(location),
    status = @status, scorecard = @scorecard, overall_score = sqlc.narg(overall_score),
    recommendation = sqlc.narg(recommendation), notes = sqlc.narg(notes), cancel_reason = sqlc.narg(cancel_reason),
    updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id = @id;
