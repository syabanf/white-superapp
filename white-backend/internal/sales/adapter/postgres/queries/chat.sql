-- name: GetConversation :one
SELECT * FROM sales.lead_conversations WHERE id = $1;

-- name: GetConversationByToken :one
SELECT * FROM sales.lead_conversations WHERE visitor_token = $1;

-- name: GetConversationByExternal :one
SELECT * FROM sales.lead_conversations WHERE channel = $1 AND external_id = $2;

-- name: ListConversations :many
SELECT * FROM sales.lead_conversations
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('channel')::text IS NULL OR channel = sqlc.narg('channel'))
  AND (sqlc.narg('assigned')::text IS NULL OR assigned_user_id = sqlc.narg('assigned'))
  AND (sqlc.narg('lead')::text IS NULL OR lead_id = sqlc.narg('lead'))
  AND (sqlc.narg('q')::text IS NULL OR display_name ILIKE '%' || sqlc.narg('q') || '%' OR phone ILIKE '%' || sqlc.narg('q') || '%'
       OR email ILIKE '%' || sqlc.narg('q') || '%' OR last_message_preview ILIKE '%' || sqlc.narg('q') || '%')
ORDER BY last_message_at DESC NULLS LAST, created_at DESC
LIMIT sqlc.arg('lim');

-- name: UpsertConversation :exec
INSERT INTO sales.lead_conversations (id, channel, external_id, visitor_token, lead_id, display_name, phone, email, status, priority,
    assigned_user_id, bot_active, handoff_reason, handoff_at, is_complaint, unread_count, last_message_at, last_message_preview,
    awaiting_since, first_response_at, first_response_seconds, sla_response_breached, resolved_at, resolution_seconds,
    page_url, utm, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
ON CONFLICT (id) DO UPDATE SET lead_id = EXCLUDED.lead_id, display_name = EXCLUDED.display_name, phone = EXCLUDED.phone,
    email = EXCLUDED.email, status = EXCLUDED.status, priority = EXCLUDED.priority, assigned_user_id = EXCLUDED.assigned_user_id,
    bot_active = EXCLUDED.bot_active, handoff_reason = EXCLUDED.handoff_reason, handoff_at = EXCLUDED.handoff_at,
    is_complaint = EXCLUDED.is_complaint, unread_count = EXCLUDED.unread_count, last_message_at = EXCLUDED.last_message_at,
    last_message_preview = EXCLUDED.last_message_preview, awaiting_since = EXCLUDED.awaiting_since,
    first_response_at = EXCLUDED.first_response_at, first_response_seconds = EXCLUDED.first_response_seconds,
    sla_response_breached = EXCLUDED.sla_response_breached, resolved_at = EXCLUDED.resolved_at,
    resolution_seconds = EXCLUDED.resolution_seconds, page_url = EXCLUDED.page_url, utm = EXCLUDED.utm,
    updated_at = EXCLUDED.updated_at;

-- name: InsertMessage :execrows
INSERT INTO sales.lead_messages (id, conversation_id, direction, sender, body, status, provider, provider_message_id,
    error_reason, sent_by_user_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT DO NOTHING;

-- name: ListMessages :many
SELECT * FROM sales.lead_messages WHERE conversation_id = $1 AND created_at > $2 ORDER BY created_at, id LIMIT $3;

-- name: RecentMessages :many
SELECT * FROM (SELECT * FROM sales.lead_messages WHERE conversation_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2) m
ORDER BY created_at, id;

-- name: InsertNote :exec
INSERT INTO sales.conversation_notes (id, conversation_id, author_user_id, body, created_at) VALUES ($1, $2, $3, $4, $5);

-- name: ListNotes :many
SELECT * FROM sales.conversation_notes WHERE conversation_id = $1 ORDER BY created_at;

-- name: ListReplyTemplates :many
SELECT * FROM sales.reply_templates WHERE (NOT sqlc.arg('active_only')::bool OR is_active) ORDER BY title;

-- name: UpsertReplyTemplate :exec
INSERT INTO sales.reply_templates (id, title, body, is_active, created_by) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, body = EXCLUDED.body, is_active = EXCLUDED.is_active, updated_at = now();

-- name: DeleteReplyTemplate :exec
DELETE FROM sales.reply_templates WHERE id = $1;

-- name: GetInsight :one
SELECT * FROM sales.conversation_insights WHERE conversation_id = $1;

-- name: UpsertInsight :exec
INSERT INTO sales.conversation_insights (conversation_id, summary, topic, sentiment, is_complaint, keywords, fingerprint, model, analyzed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (conversation_id) DO UPDATE SET summary = EXCLUDED.summary, topic = EXCLUDED.topic, sentiment = EXCLUDED.sentiment,
    is_complaint = EXCLUDED.is_complaint, keywords = EXCLUDED.keywords, fingerprint = EXCLUDED.fingerprint,
    model = EXCLUDED.model, analyzed_at = EXCLUDED.analyzed_at;

-- name: GetChatbotSettings :one
SELECT * FROM sales.chatbot_settings WHERE id = 'default';

-- name: UpsertChatbotSettings :exec
INSERT INTO sales.chatbot_settings (id, enabled, bot_name, persona, services_offered, pricing_hints, business_hours_start,
    business_hours_end, handoff_rules, greeting, min_confidence, handoff_user_ids, updated_by, updated_at)
VALUES ('default', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO UPDATE SET enabled = EXCLUDED.enabled, bot_name = EXCLUDED.bot_name, persona = EXCLUDED.persona,
    services_offered = EXCLUDED.services_offered, pricing_hints = EXCLUDED.pricing_hints,
    business_hours_start = EXCLUDED.business_hours_start, business_hours_end = EXCLUDED.business_hours_end,
    handoff_rules = EXCLUDED.handoff_rules, greeting = EXCLUDED.greeting, min_confidence = EXCLUDED.min_confidence,
    handoff_user_ids = EXCLUDED.handoff_user_ids, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: MarkSLABreaches :many
UPDATE sales.lead_conversations SET sla_response_breached = true, priority = CASE WHEN priority IN ('low', 'normal') THEN 'high' ELSE priority END
WHERE NOT sla_response_breached AND status <> 'resolved' AND awaiting_since IS NOT NULL AND awaiting_since < $1 AND NOT bot_active
RETURNING *;
