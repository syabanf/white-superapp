# NüHabit inventory for the Sales and Content port

Repo root: /Users/syabanf/Documents/works/new-nuhabit. All Go paths below are under backend/internal/modules/ unless a full path is given.

**Main points**
- NüHabit has a full B2B sales funnel: leads, deals, quotations, invoices, approvals, workflows, scoring, public lead forms, report builder, WA inbox and dataroom.
- It has none of the agency **Content** features (6–11).
- Quotations are generated from data entered in a form. There is no template engine, no DOCX, no e-signature and no pitch decks.
- The WhatsApp "chatbot" is only a fixed out-of-hours auto-reply plus a CSAT prompt. AI is used for conversation analysis, not for replying.
- The scheduled jobs for sales and CRM still run as Next.js `setInterval` watchers, started in /Users/syabanf/Documents/works/new-nuhabit/frontend/src/instrumentation.ts. They have not been ported to Go.

---

## Shared platform pieces (backend/internal/platform)

**Layout**
- Each module exposes `module.Module{Name(), Routes() []module.Route}`, with routes written as Go 1.22 mux patterns ("METHOD /path").
- `httpx.Handle` wraps handlers and returns the envelope `{success, data, message | pagination}`.
- `module.Deps` carries DB (pgx), Auth, Events (outbox bus), Log and Now.
- Most modules use a domain/ subpackage for pure logic, plus ports and adapters wired in internal/app/adapters_*.go.

**Auth**
- `auth.Service.RequireMenuPrefix(r, iam.X...)` checks menu-prefix permissions. `RequireMenuAction(r, "create"|"update", ...)` checks per-action rights.
- Prefixes live in platform/iam/prefixes_gen.go:
  - SalesFunnel = "sales-funnel"; Accounting = "accounting"
  - Crm = "crm"; CrmSettings = "crm.settings"; CrmMembers = "crm.members"; CrmInbox = "crm.members.inbox"
  - CrmPromo = "crm.promo" / "promo"; CrmReports = "crm.reports"; CrmEngagement = "crm.engagement"
  - CrmPartners = "crm.loyalty.partners"; Dataroom = "dataroom"; Tasks = "tasks"
  - SettingsIntegrations, SettingsPaymentGateways
- Business scope (company_id / branch_id) comes from platform/scope.

**Other platform services**
- **Outbox:** platform.outbox_events, outbox_deliveries and outbox_subscriptions, with subscriber receipts.
- **Jobs:** platform/jobs. Registered with `RegisterJob(module, name, every, svc, fn)` in internal/app; locks and runs in platform.job_runs and job_settings.
- **pdfgen:** a pdfkit clone (Helvetica, A4, tables, images, watermark). Used for quotations, invoices, HRIS contracts and payslips.
- **mail:** providers are Resend, Microsoft Graph and Microsoft SMTP. Sends are logged in platform.mail_log (queued/sent/failed, retries). The job `integrations.mail-retry` runs every 1m.
- **whatsapp:** providers are Meta Cloud API, the self-hosted gateway and Fonnte.
  - Every send is logged to crm.wa_messages.
  - Owner alerts are de-duplicated in configuration.wa_notif_log (notif_type, dedup_key, message, recipients jsonb, sent_at; UNIQUE(notif_type, dedup_key)).
- **Other:** storage (private files), xlsx, ratelimit (platform.rate_limits), safehttp, ocr/extract.
- **In-app notifications:** rows in `public.notifications`, written by the reporting NotificationsSQL adapter.

---

## 1. crm module (crm/module.go, MODULES key "crm")

**Size and layout**
- 167 files, 31,905 Go lines (24,016 non-test).
- Umbrella package that mounts the subpackages advance, collectibles, engagement, inbox, loyalty, marketing, members, partners, publicforms, reporting, reports and xp. Each has routes.go, domain/ and adapters_*.go.
- Shared kit in crm/internal/kit: guards, respond, xlsx, upload, zod-style validation.
- Auth goes through `kit.Guard.Require(r, Gate)`. Gates:

| Gate | Menu prefixes |
|---|---|
| settings | crm.settings |
| engagement | crm.engagement |
| campaign | crm.promo / promo |
| segments | promo + crm |
| inbox | crm.members.inbox |
| reports | crm.reports |
| operator | loyalty + members + pos.operations |
| reader | reports + members + loyalty |
| memberRead | crm.members + crm.loyalty |
| memberLoyaltyWrite | crm.loyalty + crm.settings |
| memberReviews | crm.members.member-reviews |
| partners | crm.loyalty.partners |

- Size per subpackage (non-test lines): advance 3,598; inbox 3,276; marketing 2,912; reporting 2,899; engagement 1,942; loyalty 1,685; members 1,581; xp 1,282; publicforms 1,075; reports 916; collectibles 923; partners 725.

### Routes relevant to Sales and Content

**advance** (gate settings; sales-funnel prefix for approvals)
- `GET|POST /api/crm/approval-rules`; `PATCH|DELETE /api/crm/approval-rules/{id}`
- `GET /api/crm/approvals` (inbox, status=pending|approved|rejected|all); `POST /api/crm/approvals/{id}/decide`
- `GET|POST /api/crm/workflow-rules`; `GET|PATCH|DELETE /api/crm/workflow-rules/{id}`; `GET /api/crm/workflow-rules/{id}/runs`
- `GET|POST /api/crm/scoring-rules`; `PATCH|DELETE /api/crm/scoring-rules/{id}`; `POST /api/crm/scoring/recalculate`
- `GET|POST /api/crm/custom-fields`; `PATCH|DELETE /api/crm/custom-fields/{id}`

**inbox** (gate inbox/settings)
- `GET /api/crm/inbox/conversations`; `GET|POST /api/crm/inbox/conversations/{id}`
  - POST actions: reply, add_note, set_status, set_complaint, mark_read, assign_me, unassign
- `GET|POST|DELETE /api/crm/inbox/templates` (quick replies)
- `GET|POST /api/crm/inbox/analytics` (stored or new AI insight)
- `GET|POST /api/crm/reviews` (Google reviews: sync / ignore / approve_reply / reject_reply / reply)
- `GET|POST /api/crm/instagram/webhook` (public, HMAC-signed)

**marketing** (gates campaign/segments)
- `GET|POST /api/crm/segments`; `POST /api/crm/segments/preview`; `GET|PATCH|DELETE /api/crm/segments/{id}`; `POST /api/crm/segments/{id}/preview`
- `GET|POST /api/crm/campaigns`; `POST /api/crm/campaigns/preview`; `PATCH /api/crm/campaigns/{id}`; `GET /api/crm/campaigns/{id}/report`
- `GET|PUT /api/crm/campaign-config`; `GET|POST|DELETE /api/crm/campaign-optouts`
- `GET|POST /api/crm/forms`; `GET` (submissions) `|PATCH|DELETE /api/crm/forms/{id}`

**publicforms** (public)
- `GET|POST /api/public/crm/forms/{slug}`
- `POST /api/public/site/trial`

**reporting** (gate reports)
- `GET|POST /api/crm/report-builder`; `GET /api/crm/report-builder/datasets`; `POST /api/crm/report-builder/run`
- `GET|PATCH|DELETE /api/crm/report-builder/{id}`; `GET /api/crm/report-builder/{id}/export` (xlsx with Data and Info sheets)
- `GET|POST /api/crm/dashboards`; `GET|PATCH|DELETE /api/crm/dashboards/{id}`
- `GET|POST /api/crm/report-schedules`; `PATCH|DELETE /api/crm/report-schedules/{id}`; `POST /api/crm/report-schedules/{id}` (send now)

**reports**
- `GET /api/crm/reports`, `/api/crm/reports/cs`, `/api/crm/reports/conversations`

**engagement**
- `GET|POST /api/crm/engagement/announcements`; `POST .../announcements/image`
- challenges, checkins, events, bookings
- `GET /api/crm/member-reviews`; `PATCH /api/crm/member-reviews/{id}`

**Loyalty-only routes (not needed for this port)**
- loyalty: /api/crm/dashboard, tiers, xp-rules, rewards, redemptions, loyalty-features, `GET|PUT /api/crm/settings`, `/members/{id}/xp-adjust`
- members: /api/crm/members[/{id}[/activity|consent|badges|password]]
- collectibles: badges, wallpapers, avatars[/upload], avatar-inventory, collectibles/idle-report
- partners: /api/crm/partners[/{id}], partners/events[/rematch]

### Tables owned (from migrations; columns abbreviated)

**WhatsApp / Instagram inbox**
- **crm.wa_conversations**
  - Columns: id, phone, customer_id, status, assigned_user_id, unread_count, last_message_at, last_message_preview
  - Added later: channel (whatsapp | instagram), external_id, display_name, category (produk | layanan | pembayaran | lainnya), priority (low | normal | high | urgent), is_complaint, awaiting_since, first_response_at, first_response_seconds, sla_response_breached, escalated_at, resolved_at, resolution_seconds, csat_asked_at, csat_score (1–5), auto_reply_sent_on (date)
  - Status: open | in_progress | waiting_customer | resolved
- **crm.wa_messages**: id, conversation_id, direction (in | out), message_type (otp | notification | chat | broadcast | system), phone, customer_id, body (null for OTP), media_type, status (queued | sent | failed | received), provider, provider_message_id, error_reason, sent_by_user_id, wa_from_me, channel, external_id
- crm.wa_reply_templates: title ≤80, body ≤2000, is_active
- crm.wa_internal_notes: conversation_id, author_user_id, body
- crm.wa_conversation_insights: conversation_id PK, summary, topic, sentiment, is_complaint, keywords jsonb, fingerprint, model, analyzed_at
- crm.google_reviews

**Settings**
- crm.crm_settings: key, value jsonb. Holds cs_auto_reply_enabled/text, cs_csat_*, business hours, gr_* review settings and default venue.

**Workflow, scoring, approvals (advance)**
- crm.crm_events: event_type, subject_type, subject_id, payload, actor_user_id
- crm.crm_scoring_rules
  - kind: field | event; field; operator (eq | neq | in | contains | not_empty | gt | lt); value; event_type; window_days; max_count; points
  - Scoring fields: source, org_type, temperature, status, city, pic_email, pic_title, account_type, industry
  - Scoring event types: task_done, wa_inbound, deal_created, quotation_sent
- crm.crm_workflow_rules
  - object: lead | deal | account | contact | task | quotation
  - trigger_type: created | updated | stage_changed | status_changed | score_reached | inactive_days | due_soon
  - Other columns: trigger_config, conditions, actions jsonb, run_once_per_record, run_count
  - Condition ops: eq, neq, in, not_in, contains, gt, gte, lt, lte, is_empty, not_empty, changed, changed_to
  - Actions: send_wa, webhook, wait, create_task, assign_owner, update_field, notify_in_app
- crm.crm_workflow_runs: status success | partial | failed | skipped | scheduled; actions_result
- crm.crm_scheduled_actions: actions, context, run_at; status pending | done | failed | cancelled; attempts. Holds steps queued after a `wait` action.
- crm.crm_approval_rules: object = quotation; level 1–5; min_discount_percent; approver_role or approver_user_id
- crm.crm_approval_requests: object, subject_id, status (pending | approved | rejected | cancelled), current_level, discount_percent, amount
- crm.crm_approval_steps: level, approver_role/user_id, status (pending | approved | rejected | skipped), decided_by, comment
- crm.crm_custom_fields
  - object: lead | deal | account | contact
  - key matches `^[a-z][a-z0-9_]{1,39}$`
  - field_type: text | textarea | number | date | boolean | picklist | multipicklist | url | email | phone
  - options, is_required, validation, show_in_list

**Reporting**
- crm.crm_reports: dataset (lead | deal | quotation | task | account | contact), definition jsonb, is_shared
- crm.crm_dashboards
- crm.crm_dashboard_widgets: report_id, widget_type, width 1–3, config
- crm.crm_report_schedules: frequency, hour, day_of_week, day_of_month, channel (in_app | wa), recipients, last_run_at, last_status, next_run_at

**Marketing**
- crm.crm_segments: source, definition jsonb, last_count
- crm.crm_campaigns
  - Columns: message_template, segment / segment_id, promo_campaign_id, promo_mode (public | batch), voucher_prefix, daily_cap, channels ⊂ {wa, in_app}, scheduled_at, failure_reason
  - Status: draft | scheduled | sending | paused | done | cancelled | failed
- crm.crm_campaign_recipients: status pending | sent | failed | skipped; voucher_code
- crm.crm_marketing_optouts
- crm.crm_forms: slug, fields jsonb, submit_label, success_message, redirect_url, default_source, lead_temperature, notify_user_ids, notify_numbers, submission_count
- crm.crm_form_submissions: form_id, lead_id, payload, utm, ip_hash, status, reason
- crm.lead_consents: lead_id, channel (email | sms | whatsapp), granted, consent_text_version, ip_hash, source_path

### Workflow engine and approval state machine
- **Event flow:** salesfunnel publishes outbox topics `CrmEventRaised` (events: lead.created, lead.updated, deal.created, quotation.created) and `QuotationPriced`. crm/advance/subscribe.go has two subscribers:
  - `crm.workflow-on-sales-event` records crm_events, runs matching workflow rules and rescores the lead.
  - `crm.approval-on-quotation-priced` builds the approval request from crm_approval_rules (`RequiredApprovalLevels`).
- **Approval chain:** each level goes pending → approved (next level, or the request resolves to "approved": "Quotation boleh dikirim") or → rejected. The result updates crm_sales_quotations.approval_status (none | pending | approved | rejected).

### Scheduled jobs
- **None registered in Go for crm.** They still run in Next.js (/Users/syabanf/Documents/works/new-nuhabit/frontend/src/lib):

| Watcher | Interval | What it does |
|---|---|---|
| crm/cs-sla-watcher.ts | 60s | Flags SLA breach and escalation |
| crm/campaign-watcher.ts | 45s + jitter | Dispatches WA broadcasts within daily cap |
| crm/workflow-watcher.ts | 5m | `runDueScheduledActions` and `scanTimeBasedRules` (inactive_days / due_soon) |
| crm/report-schedule-watcher.ts | 15m | Sends scheduled reports by in_app or WA |
| crm/google-reviews-sync-job.ts | 15m | Syncs Google reviews |

### Notifications
- WA to members (inbox replies, campaigns, CSAT, auto-reply).
- Owner WA via OwnerNotifier (wa_notif_log de-dup).
- In-app `public.notifications` from the workflow `notify_in_app` action and report schedules.
- Form submissions notify `notify_user_ids` and `notify_numbers`.

### AI / LLM
- **Only use in crm:** crm/inbox/adapters_openai.go with the prompt in crm/inbox/domain/insights.go.
- **Provider:** OpenAI-compatible `/chat/completions`.
  - Key, base URL and model come from configuration.app_settings (openai_api_key / openai_base_url / openai_model), with env OPENAI_API_KEY as fallback.
  - Allowed models: gpt-4o-mini (default), gpt-4.1-mini, gpt-4o, gpt-5.4-mini, gpt-5.5. Temperature 0.
  - JSON mode, retried once without it on HTTP 400. Timeout 60s.
- **Prompt purpose:** analyse an Indonesian CS transcript ("Pelanggan:" / "Agen:" lines, truncated in the middle). Returns JSON `{summary, topic (≤4 words), sentiment positif|netral|negatif, is_complaint, keywords 3–8}` with no PII.
- Triggered manually from the inbox; the result is stored in wa_conversation_insights.

### File generation and external deps
- **Files:** xlsx export (report builder); image uploads (announcements, avatars).
- **External:** Instagram Graph (DMs), Google Business Profile (reviews), OpenAI, WA providers.

---

## 2. salesfunnel module (salesfunnel/, MODULES key "sales-funnel")

**Size:** 39 files, 10,719 lines (8,981 non-test). Largest files: quotations.go 733, accounts.go 677, leads.go 663, deals.go 573, insights.go 513, pdf.go 478, activities.go 471, billing.go 398.

**Layout**
- Flat handler package: http.go (route table), access.go (auth and scope), one file per resource.
- domain/ holds pure rules: quotations totals, forecast, tasks, timeline, custom fields, lead import, documents, realization.
- pgrow/ is a dynamic row helper. ports.go defines the Directory, Catalog, Stock, Members, CRM, Receivables and Gateway interfaces.
- Described as a port of frontend/src/app/api/sales-funnel/** and lib/sales-funnel/**.

### Auth
- `requireUser` requires menu prefix "sales-funnel". `requireFinance` requires "accounting"; invoice viewers may also use "sales-funnel".
- Only super_admin and sales roles may own leads or deals (SalesFunnelRoles).
- Scope is fail-closed: non-super_admin users need a company_id (`salesScope`).
- Rate limit is a fixed one-minute window stored in platform.rate_limits; used for WA cooldown and imports.

### Routes (all under /api/sales-funnel)

**Leads**
- `GET|POST /leads`; `POST /leads/import` (xlsx/csv); `GET /leads/by-phone`
- `GET|PATCH|DELETE /leads/{id}`; `POST|DELETE /leads/{id}/link-customer`

**Deals**
- `GET|POST /deals`; `PATCH|DELETE /deals/{id}`
- `GET|POST|DELETE /deals/{id}/members`
- `POST /deals/{id}/send-wa`
- `GET|POST /deals/{id}/quotations`; `GET|POST /deals/{id}/invoices`; `GET|POST /deals/{id}/payments`; `DELETE /deals/{id}/payments/{paymentId}`

**Quotations**
- `PATCH|DELETE /quotations/{id}`
- `POST /quotations/{id}/revise`, `/quotations/{id}/realize`, `/quotations/{id}/send-wa`
- `GET /quotations/{id}/pdf`

**Invoices**
- `PATCH|DELETE /invoices/{id}`; `GET /invoices/{id}/pdf`

**Activities, accounts, contacts, pipelines**
- `GET|POST /activities`; `PATCH|DELETE /activities/{id}` (tasks)
- `GET|POST /accounts`; `GET|PATCH|DELETE /accounts/{id}`
- `GET|POST /contacts`; `GET|PATCH|DELETE /contacts/{id}`
- `GET|POST /pipelines`; `PATCH /pipelines/{id}`; `GET|POST /stages`; `PATCH /stages/{id}`

**Lookups and templates**
- `GET /owners`, `/lost-reasons`, `/products`, `/customers`, `/raw-materials`, `/custom-fields`
- `GET|PUT /recipes`
- `GET|POST /wa-templates`; `PATCH|DELETE /wa-templates/{id}`

**Analytics**
- `GET /forecast`; `GET|PUT /targets`; `GET /reports`; `GET /timeline`

### Tables owned (crm schema)

**Pipeline**
- **crm_sales_stages**: code, name, sort_order, is_won, is_lost, stuck_threshold_days, is_active, probability (0–100), pipeline_id
  - Seed stages: prospek-baru, dihubungi, proposal, nego-survey, menang (won), kalah (lost)
- crm_sales_lost_reasons seed: harga, kompetitor, jadwal, batal-acara, tidak-respon, lainnya
- crm_pipelines: code, name, is_default, sort_order
- crm_sales_deal_stage_history: deal_id, stage_id, entered_at

**Leads and deals**
- **crm_sales_leads**
  - Columns: company_id, branch_id, org_name, org_type, pic_name, pic_title, pic_phone, pic_email, city, source, temperature, status, notes, owner_user_id, customer_id, account_id, contact_id, custom jsonb, score, score_breakdown, score_updated_at, utm_source/medium/campaign/term/content, landing_page, referrer, deleted_at
  - org_type: corporate | sekolah | komunitas | travel-agent | pemerintah | perorangan | lainnya
  - source: wa | instagram | referral | google | pameran | canvassing | website | lainnya
  - temperature: panas | hangat | dingin
  - status: baru | dihubungi | qualified | tidak-cocok
- **crm_sales_deals**
  - Columns: lead_id, title, event_type, event_date, is_event_date_fixed, pax_estimate, stage_id, pipeline_id, value_estimate, value_final, owner_user_id, lost_reason_id, entered_stage_at, closed_at, forecast_category, custom
  - event_type: gathering | field-trip | ulang-tahun | buyout-venue | lainnya
  - forecast_category: pipeline | best_case | commit | closed_won | closed_lost
- crm_deal_members: role owner | support | pre_sales | account_manager | finance; split_percent
- crm_accounts: name, account_type, industry, address, city, phone, email, website, npwp, owner, custom
- crm_contacts: account_id, name, title, phone, email, is_primary, customer_id, custom

**Activities / tasks**
- **crm_sales_activities**
  - Columns: lead_id, deal_id, subject_type, subject_id, title, notes, due_at, done_at, reminder_at, reminder_channels, reminder_sent_at, in_app_notified_at, parent_task_id, recurrence jsonb, owner_user_id
  - activity_type: telepon | wa | meeting | catatan | tugas | email
  - subject_type: lead | deal | account | contact | member
  - priority: low | normal | high | urgent
  - status: open | in_progress | done | cancelled
  - reminder_channels: wa | in_app | email (default wa, in_app); recurrence freqs daily | weekly | monthly

**Quotations**
- **crm_sales_quotations**
  - Columns: quote_number, status, use_ppn, ppn_persen, subtotal, discount_percent, discount_nominal, ppn_nominal, total, notes, valid_until, stock_deducted_at, bom_status (terpotong | tidak-terpotong), realized_by, version, parent_quotation_id, superseded_at, approval_status (none | pending | approved | rejected), approval_request_id, expiry_reminded_at
  - quote_number format: `QT-YYMM-####` from sequence crm_sales_quotation_number_seq
  - status: draft | terkirim | diterima | ditolak | superseded
- crm_sales_quotation_items: item_type produk | bebas; product_id; description; qty; unit_price; line_total; sort_order
- crm_sales_quotation_terms: label, percent, due_date (payment termin)

**Billing, targets, templates**
- crm_sales_invoices: quotation_id, term_id, invoice_number, label, amount, due_date, status (diajukan | draft | terkirim | batal), sent_at, faktur_pajak_url
- crm_sales_deal_payments: amount, method (cash | transfer | qris | edc | lainnya), paid_on, note
- crm_sales_targets: user_id, period_month, target_value, target_deals, pipeline_id
- crm_sales_wa_templates: name, body. Placeholders: {pic} {instansi} {acara} {tanggal_acara} {venue}

### Quotation state machine
1. **Create** (`createQuotation`): status draft. Totals are computed with discount on the subtotal and PPN on the discounted base. Deal value_estimate is updated, then the module emits `quotation.created` and publishes `QuotationPriced`, which triggers the approval flow.
2. **Release gate:** send and realize are blocked unless approval_status is none or approved.
3. **send-wa:** the WA text is built by `domain.BuildQuotationWaMessage`. It enforces a WA cooldown and logs a 'wa' activity. draft → terkirim.
4. **revise:** copies items and terms into version+1 with the same parent_quotation_id, status draft. The old version becomes superseded.
5. **realize:** locks stock, deducts raw materials by product recipes (BOM), sets stock_deducted_at and freezes the quotation. Returns 409 with shortages unless `force`.
6. **Accept/reject:** diterima / ditolak are set via PATCH.
7. **Invoices:** EPIC-025 "Opsi B": sales submit an invoice ("diajukan"); finance issues it (terkirim, which creates a receivable via the Receivables port into accounting) or cancels it (batal is not allowed once paid > 0).

### PDF generation (pdf.go)
- `quotationPDF` and `invoicePDF` use pdfgen, A4, margin 48, accent #00281a.
- Layout: centred letterhead from company/branch, a 4-cell info grid, item table, totals rows, notes and a signature block.
- File name: `{number}-{org}.pdf`.
- Layout is hard-coded in Go; there are no templates.

### Scheduled jobs (still in Next.js)
- lib/sales-funnel/followup-reminder-watcher.ts, every 5m: task reminders by WA (phone from hris.employees) and in_app. Email is deferred. At-most-once by claiming reminder_sent_at first.
- lib/sales-funnel/quotation-expiry-watcher.ts, hourly: when valid_until ≤ 3 days, sends WA and in-app once (expiry_reminded_at).

### Other
- **Notifications:** WA to the lead PIC (deal send-wa with template, quotation send-wa); owner and in-app reminders as above.
- **AI:** none.
- **External deps:** WA gateway (platform/whatsapp). Ports to inventory (stock, recipes), POS members, accounting receivables and configuration branches.
- **insights.go** (in salesfunnel) holds report and forecast SQL; it does not use an LLM.

---

## 3. integrations module (integrations/, MODULES key "integrations")

**Size:** 38 files, 8,586 lines (5,905 non-test): settings 1,525; webhooks 918; gobiz 2,231; domain 1,111.

**Layout:** `integrations.New(subs...)` mounts the settings, webhooks and gobiz subpackages. internal/appsettings reads configuration.app_settings and masks secrets.

### Routes
**settings** (auth SettingsIntegrations or SettingsPaymentGateways)
- `GET|PUT /api/settings/integrations`: AI providers. Keys deepseek_api_key/model/base_url and openai_api_key/model/base_url.
- `GET|PUT|DELETE /api/settings/google-business`; `GET|PUT|DELETE /api/settings/instagram`
- `GET|PUT /api/settings/payment-gateways`; `POST /api/settings/payment-gateways/{provider}/test`
- `GET|PATCH /api/settings/wa-gateway` (status, pairing QR); `GET|POST /api/settings/wa-gateway/messages` (log of crm.wa_messages, resend a failed message)
- `GET|PUT /api/settings/wa-notifications`; `POST /api/settings/wa-notifications/test` (owner alerts, Daily Flash Report)
- `GET|PUT /api/settings/email`; `POST /api/settings/email/test`; `GET /api/settings/email/log`; `POST /api/settings/email/log/{id}/resend`

**webhooks** (public; each checks a constant-time token or HMAC)
- `GET|POST /api/payments/xendit/webhook`; `GET|POST /api/payments/midtrans/webhook`
- `POST /api/wa/inbound` (WA gateway inbound; brute-force limiter on bad tokens)
- `POST /api/integrations/telegram/webhook/{secret}`
- `POST /api/integrations/loyalty-events/{partner}`

**gobiz:** `GET|PUT /api/settings/gobiz`; `POST /api/settings/gobiz/actions`; `POST /api/integrations/gobiz/webhook/{token}`; `GET /api/public/gofood-image/{file}`

### WA inbound flow
`/api/wa/inbound` → `Inbox.Record` (conversation and message rows) → `Inbox.OnInbound` (crm/inbox/gateway.go) → `AutoReply`.
- OnInbound applies the CS rules: awaiting_since, CSAT score parsing, business hours (default 10–22 WIB).
- If cs_auto_reply_enabled is on and the message arrives outside hours, it sends cs_auto_reply_text once per WIB day (guarded by auto_reply_sent_on).
- The adapter is in /Users/syabanf/Documents/works/new-nuhabit/backend/internal/app/adapters_integrations.go:340-352.

### Other
- **Tables:** configuration.app_settings (key, value text), configuration.payment_gateways, configuration.wa_notif_log, configuration.telegram_subscribers, platform.mail_log.
- **Jobs:** `integrations.mail-retry` every 1m. The WA notification watcher is still in Next.js (lib/wa/notifications-watcher.ts).
- **AI:** settings storage only (DeepSeek and OpenAI keys).

---

## 4. configuration module (only what the above depend on)

- **Size:** 58 files, 10,180 lines. Subpackages settings, access, apitokens, accounts, master, branches, kit.
- **Used by Sales/CRM:**
  - configuration.companies, branches and holdings (venue and scope; branch name on PDFs)
  - configuration.app_settings (OpenAI/DeepSeek keys, WA gateway config, mail config, brand)
  - `GET|PUT /api/settings/company-profile` (letterhead data)
  - `GET|PUT /api/settings/sales-target`
  - IAM roles and menus (`/api/settings/iam/roles|menus`), which generate the menu prefixes above
  - users (`/api/users`, `/api/admin/users`)
- settings/tts.go also calls OpenAI TTS; not relevant here.

## 5. desktop / insights (mostly not relevant)
- **desktop** (3,320 lines): `/api/desktop/{inbox,overview,preferences,search,status,stream,wallpapers}`. The overview shows POS sales and CRM pulse. No sales-funnel logic.
- **insights** (5,727 lines): the "Do" assistant at `/api/ai/assistant[/actions|/attachment]` (OpenAI chat with tools) plus recruitment and executive dashboards.
  - Read tools: cari_karyawan, absensi_hari_ini, stok_menipis, penjualan_periode, status_kandidat.
  - Write proposals: usulkan_pengumuman_draft, usulkan_catatan_kandidat.
  - No CRM, sales or content tools.

## 6. services/wa-gateway
- **Size:** Node ≥20 ESM, 497 lines total: src/server.js 162, src/wa.js 335. Dependencies: @whiskeysockets/baileys 7.0.0-rc13, pino, qrcode-terminal.
- **Run:** PM2 app "wa-gateway-bcd" (ecosystem.config.cjs), with `--env-file=.env`, no watch.
- **HTTP** (port WA_GATEWAY_PORT, default 3471; header x-gateway-token = WA_GATEWAY_TOKEN):
  - `GET /health`
  - `GET /qr` (pairing)
  - `POST /send` (text; random 1.5–3.5s gap between sends)
- **Inbound:** `messages.upsert` events go to a forward queue that POSTs to APP_INBOUND_URL (default http://127.0.0.1:3459/api/wa/inbound) with the token.
- **Other env:** WA_SESSION_DIR, WA_GATEWAY_EXTRA_HOSTS, WA_LOG_LEVEL, WA_DEBUG_UPSERT.
- One session equals one WA number. Text only, with no AI.

## 7. dataroom module (dataroom/, MODULES key "dataroom")

**Size:** 23 files, 6,765 lines (5,074 non-test). Flat service and handler with domain/ (drive.go, files.go).

**Auth:** prefix "dataroom" plus `RequireMenuAction` create/update. Per-node permissions, and department access via folder_departments.

### Routes
**Dashboard (/api/dataroom)**
- `GET /departments`, `/people`, `/tags`, `/tree`, `/search`, `/activity`
- Nodes:
  - `GET|POST /nodes`; `GET|PATCH|DELETE /nodes/{id}` (delete = trash)
  - `POST /nodes/{id}/restore`; `DELETE /nodes/{id}/purge`; `POST /nodes/{id}/copy`
  - `PUT|DELETE /nodes/{id}/star`; `PUT|DELETE /nodes/{id}/lock`
  - `GET /nodes/{id}/download`; `GET /nodes/{id}/text` (extracted text)
  - `GET|POST /nodes/{id}/versions`; `GET /nodes/{id}/versions/{versionId}/download`; `POST .../versions/{versionId}/restore`
  - `GET|POST /nodes/{id}/comments`; `DELETE /comments/{id}`
  - `GET /nodes/{id}/activity`; `GET|PUT /nodes/{id}/permissions`; `GET|PUT /nodes/{id}/access`
- Uploads: `POST /upload`; chunked via `POST /uploads`, `GET|DELETE /uploads/{id}`, `PUT /uploads/{id}/chunk`, `POST /uploads/{id}/complete`
- Shares: `GET|POST /shares`; `GET /shares/{id}` (logs); `DELETE /shares/{id}` (revoke)

**Public (/api/share)**
- `GET /api/share/{token}`, `/api/share/{token}/list`
- `POST /api/share/{token}/request-code`, `/api/share/{token}/verify`
- `GET /api/share/{token}/files/{nodeId}` (watermarked)

### Tables
- **dataroom.nodes**: parent_id, kind (folder | file), name, mime, size_bytes, storage_path, description, tags[], trashed_at / trashed_root / trashed_by, version, search_text, search_vector (tsvector), locked_by / locked_at, restricted
- node_versions; stars; recent_opens
- permissions: subject_type user | role; role viewer | commenter | editor | owner
- comments (with mentions uuid[]); activity_logs
- upload_sessions
- **shares**: token, node_id, access_type (public | email), allowed_emails[], pin_hash, watermark, allow_download, expires_at, revoked_at, view_count
- share_sessions, share_email_codes, share_access_logs, folder_departments

### Other
- **Job:** `dataroom.housekeeping` hourly (expired upload sessions and similar).
- **Mail:** share access codes sent via platform mail (mail.go).
- **Files:** watermark stamped on images and PDFs (files.go:548); MIME allowlist includes pptx and docx; quota and max size from env (DATAROOM_QUOTA_BYTES, DATAROOM_MAX_FILE_BYTES, DATAROOM_MAX_CHUNKED_MB, default 2 GB).
- **AI:** none.
- **Links to sales:** none. Quotations and invoices are never stored in the dataroom; PDFs are rendered on the fly.

## Related modules outside the requested list
- **Promo** is storedvalue/promo and storedvalue/giftcards: `/api/promo/{campaigns[/{id}[/codes|/redemptions]], codes/{id}, offers[/{id}], catalog, gym-packages, gift-cards..., gift-card-config}`. Tables promo.promo_campaigns, promo_codes, promo_redemptions, offer_rules, offer_rule_items, offer_usages.
- **Tasks** (features/tasks) is payroll/depttasks*.go: `/api/hris/dept-tasks[/items|/kpi|/kpi/contributions|/meta|/monitor|/{id}|/occurrences/{id}[/attachments[/{attId}]]]`. Tables hris.department_tasks, *_occurrences, *_subtasks, *_updates, *_attachments, *_occurrence_items, *_kpi_contributions. Jobs `dept-tasks.recurrence` (1h) and `dept-tasks.reminders` (15m). These are internal team tasks, separate from the sales-funnel tasks in crm_sales_activities.
- **site** (CMS): site.content (key, value jsonb), site.articles (category training | events | apparel | news; body_md; status draft | published), site.events (form_slug ties events to CRM public forms).
- **hris** contracts: hris/contract_document.go builds the PKWT/PKWTT employment contract PDF with pdfgen and stores an uploaded signed scan. This is HR only.

---

## Feature-by-feature verdict

1. **Quotation generation automation: partly present (manual entry).**
   - Files: salesfunnel/quotations.go, domain/quotations.go, pdf.go, crm/advance/approvals.go, frontend/src/features/sales-funnel/quotations/components/quotation-builder-dialog.tsx.
   - Sales enter line items (catalog product or free text), discount, PPN and payment terms in a builder. The server computes totals, numbers the quote, runs multi-level discount approval, versions revisions and renders a fixed-layout PDF.
   - It can send a WA summary and realise stock by BOM. There is no AI or template-driven generation, and no automatic quote from the lead.

2. **Contract generation automation (client contracts): not present.** The only contract generator is the HR employment contract PDF (hris/contract_document.go: hard-coded PKWT layout plus signed-scan upload).

3. **Chatbot auto-replying to client leads: not present as a chatbot.**
   - The WA gateway, `/api/wa/inbound` and crm/inbox/gateway.go only send a fixed out-of-hours auto-reply (cs_auto_reply_text, once per WIB day) and a CSAT question/score. Everything else is replied to by human agents in the inbox (with quick-reply templates).
   - The LLM is used only to summarise and classify conversations after the fact.
   - Inbound WA is not auto-converted to a sales lead. `/leads/by-phone` links the two manually, and a workflow `send_wa` action can message on CRM events.
   - On the web, public lead forms (crm/publicforms, /public/[slug]) create 'baru' leads and notify staff. There is no web chat widget.

4. **Pitch deck generation automation: not present.** pptx appears only in the dataroom MIME allowlist.

5. **Client onboarding system: not present.**
   - Closing a deal is a stage move (is_won) with a stage history row.
   - `link-customer` / `Members.UpsertFromPic` turns the PIC into a POS/CRM member.
   - Invoices and payments follow, and workflow rules can create tasks on stage_changed.
   - There is no onboarding checklist or portal for clients. hris/onboarding is for employees.

6. **Editorial plan / content drafts: not present.** The closest things are site.articles (draft/published markdown CMS) and crm campaigns' message_template.

7. **Content report creation: not present for social content.** The CRM report builder (datasets lead | deal | quotation | task | account | contact, xlsx export, scheduled in_app/WA delivery) and campaign reports are the nearest reusable pattern.

8. **Content editing / approval workflow: not present.** The reusable patterns are the crm approval engine (multi-level, role or user approver; currently object = 'quotation' only), Google review replies that wait for approval, and dataroom comments, versions and locks.

9. **Design creation (templates, canvas, AI images): not present.** The only canvas is the POS floor-plan canvas.

10. **Asset update / asset library: partial, through the dataroom only.** Folders and files with versions, tags, full-text search, stars, comments, permissions, watermarked share links and chunked uploads. There are no DAM features such as image renditions or brand kits.

11. **Automated feed and story posting: not present.** Instagram is used only for Messaging (DM inbox and webhook). There is no Graph media_publish and no scheduler.

12. **Document templating engine (DOCX/PDF, e-signature): not present.**
    - PDFs are hand-coded pdfgen layouts (quotations, invoices, HR contracts, payslips).
    - The only templating is `{placeholder}` substitution in WA templates (`domain.RenderWaTemplate`: pic, instansi, acara, tanggal_acara, venue).
    - No DOCX, and no e-signature beyond uploading a signed scan in HRIS.

13. **Dataroom: present and full-featured.** See section 7. It is not linked to sales records.

---

## Frontend (frontend/src)

### Data fetching and shared components
- Client components use TanStack Query (`useQuery` / `useMutation`) with toast via sonner.
- Each feature has api.ts (fetch to `/api/...`, served by the Go backend behind the Next proxy, src/proxy.ts), queries.ts/mutations.ts, query-keys.ts, types.ts and components/.
- CRM responses are parsed by features/crm/http.ts `parseCrmResponse` (`{success, data, error}`).
- Route files are thin wrappers (e.g. `<SalesFunnelPipelinePage/>`). Settings pages gate on the server with lib/auth/require-user and lib/iam/prefixes.
- **Shared UI:** components/ui (shadcn style: button, input, dialog, badge, select, label, confirm-dialog, switch, textarea, table, card, scroll-region, checkbox, stat-card, page-header), plus components/design-system, layout, shared.

### Feature sizes
| Feature | Files | Lines |
|---|---|---|
| crm | 159 | 20,611 |
| sales-funnel | 72 | 10,615 |
| promo | 32 | 4,410 |
| dataroom | 18 | 3,138 |
| tasks | 22 | 2,507 |
| site-forms | 12 | 636 |
| integration | 2 | 191 |

### Page routes (app/dashboard/(dashboard)/...)

**sales-funnel (11 pages)**
- /sales-funnel/leads: lead list, filters, import dialog, score badge
- /sales-funnel/leads/[id]: lead detail with timeline, deals, member loyalty card
- /sales-funnel/pipeline: kanban of deals. Deal detail dialog contains the quotation section and builder, realize dialog, invoice and payment sections, deal team, close-deal dialog
- /sales-funnel/accounts and /accounts/[id]: account list and detail
- /sales-funnel/contacts: contacts
- /sales-funnel/tasks: task list and calendar
- /sales-funnel/approvals: discount approval inbox (features/crm/advance approvals-inbox-page)
- /sales-funnel/forecast: forecast vs targets (crm/advance forecast-page)
- /sales-funnel/reports: funnel reports
- /sales-funnel/settings: stages, pipelines, WA templates, lost reasons

**crm (31 pages)**
- /crm: loyalty dashboard
- /crm/inbox: WA/IG CS inbox (conversation list, chat panel, member context, complaint panel, AI insight card)
- /crm/reviews: Google reviews
- /crm/member-reviews: in-app member reviews
- /crm/campaigns: WA/in-app broadcast campaigns
- /crm/marketing/segments, /crm/marketing/forms (public lead form builder), /crm/marketing/attribution (UTM/source attribution)
- /crm/reports, /crm/reports/builder, /crm/reports/schedules, /crm/dashboards
- /crm/settings: CRM and CS settings (auto-reply, CSAT, hours)
- /crm/settings/workflows, /crm/settings/scoring, /crm/settings/approvals, /crm/settings/custom-fields
- /crm/members and /crm/members/[id]
- Loyalty and engagement: /crm/rewards, /crm/xp-rules, /crm/badges, /crm/avatars, /crm/wallpapers, /crm/partners, /crm/announcements, /crm/challenges, /crm/events, /crm/checkins

**promo (4 pages):** /promo (campaigns and codes, gift cards), /promo/bxgy, /promo/bundling, /promo/volume (offer rules)

**dataroom (2 pages):** /dashboard/dataroom (drive browser with share, permissions, access, preview dialogs, upload queue) and public /share/[token] (features/dataroom/public/share-page)

**tasks (3 pages):** /tasks (my tasks, quick capture), /tasks/team (team board, monitor), /tasks/kpi

**Integrations / settings (5 pages)**
- /settings/integrations (features/configuration/integrations: AI keys)
- /settings/integrations/email
- /settings/instagram
- /settings/wa-gateway (QR pairing, message log, resend)
- /settings/wa-notifications (os-desktop wa-notif-settings)
- features/integration/components/integration-settings-page.tsx is a 191-line wrapper.

**site-forms / public lead capture (3 pages)**
- (public)/public and (public)/public/[slug]: CRM public form renderer
- (site)/events/[slug]: site event page with embedded form
- features/site-forms supplies public-crm-form.tsx, trial-form.tsx, consent.ts, track.ts (UTM), phone.ts and branches.ts.

**Total: about 59 pages** across these features.
