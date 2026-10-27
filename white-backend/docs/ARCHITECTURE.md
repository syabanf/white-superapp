# Mixist Backend: architecture

Go rewrite of the Mixist Analytics backend (`../white-analytics`, Next.js server
actions + Prisma). Eight deployable services, one Postgres instance with one
schema per service, NATS JetStream for asynchronous work, gRPC between
services and a grpc-gateway REST facade for the Next.js frontend.

```
                 ┌──────────────┐  REST/JSON + cookies
   Next.js  ───▶ │   gateway    │  :8080  (grpc-gateway, JWT check, OAuth redirects,
                 └──────┬───────┘          media bytes, PDF bytes, share cookies)
                        │ gRPC (plaintext, private network)
     ┌──────────┬───────┴────────┬──────────────┬──────────────┐
┌────▼─────┐ ┌──▼──────────┐ ┌───▼────────┐ ┌───▼──────────┐ ┌─▼──────────┐
│ identity │ │integrations │ │ analytics  │ │  publishing  │ │  seosuite  │
│  :9001   │ │   :9002     │ │   :9003    │ │    :9004     │ │   :9005    │   reports :9006 composes
└────┬─────┘ └──────┬──────┘ └─────┬──────┘ └──────┬───────┘ └─────┬──────┘
     │ identity     │ integrations │ analytics     │ publishing    │ seosuite      (Postgres schemas)
     └──────────────┴──────────────┴───────────────┴───────────────┴──────────┐
                              NATS JetStream  (stream WHITE, subjects white.>) ◀── scheduler (cron ticks)
```

## Services and ownership

| Service | Owns (Postgres schema) | Talks to |
|---|---|---|
| **identity** | users, refresh sessions, clients, memberships, notifications, app settings | publishes `client.created`, `client.deleted`, `user.deleted` |
| **integrations** | OAuth connections (encrypted tokens), integration settings (encrypted provider keys), Meta review checklist | identity.Access for the MANAGER check on OAuth; publishes `integration.updated`, `connection.deleted` |
| **analytics** | social accounts/snapshots/posts, SEO properties, GSC/GA4 daily, audits + crawls, ad accounts/campaigns/insights/demographics, reports, AI insights, its sync jobs | identity.Access, identity.Notify, integrations.GetAccessToken / GetProviderCredentials |
| **publishing** | media assets, content posts, targets, comments, activities, its sync jobs | identity.Access, identity.Notify, integrations (tokens), analytics (own social accounts) |
| **seosuite** | tracked keywords, rank snapshots, keyword research cache, backlinks, competitor domains, domain snapshots, its sync jobs | identity, integrations, analytics (SEO properties) |
| **reports** | saved reports, AI insights | identity, integrations (OpenRouter key), analytics / publishing / seosuite `ReportDataService`; renders PDFs and the public share dashboard |
| **content** | editorial plans and items, caption revisions, comments and activity, design tasks and files, brand assets with versions, content reports | identity (client access, Notify, service tokens), integrations (OpenRouter), analytics (top posts, engagement), publishing (uploads media, saves and schedules posts incl. stories, then polls their status on `erp.tick.15m`) |
| **finance** | chart of accounts, fiscal years/periods, journals and mappings, cash-bank, AP/AR invoices and payments, reimbursement claims and batches, AI insights | hr (employee, manager), identity.Notify, integrations (OpenRouter); consumes `hr.payroll.paid`, `sales.invoice.issued`, `sales.payment.recorded`; publishes `finance.reimbursement.paid` |
| **hr** | employees, departments, positions, attendance, shifts, holidays, leave, overtime, contracts, announcements, onboarding/offboarding, salaries, payroll runs, payslips, loans | identity (users, Notify); publishes `hr.payroll.paid` |
| **recruitment** | job openings, candidates, stage history, screening, psikotes bank and sessions, AI interviews, interviews and scorecards, offers, rate limits | hr (departments, interviewers), identity.Notify, integrations (OpenRouter); publishes `recruitment.candidate.hired` |
| **sales** | leads, accounts, contacts, pipelines, deals, activities, quotations, invoices, payments, targets, workflow and scoring rules, forms, contracts, pitch decks, onboarding, chat | identity (Notify, CreateClient), integrations (OpenRouter); publishes `sales.invoice.issued`, `sales.payment.recorded`, `sales.deal.won` |
| **performance** | KPI indicators, weights, targets, scorecards, review cycles, 360 feedback, department tasks, logbook, daily and monthly recaps, improvements | hr (directory, attendance summary), identity.Notify, integrations (OpenRouter) |
| **scheduler** | nothing | publishes daily/publish/retention ticks per client and the ERP ticks `white.erp.tick.15m`, `.hourly`, `.daily` |
| **gateway** | nothing | every service |

Rules that follow from ownership:

- A service reads and writes only its own schema. No cross-schema SQL, no
  foreign keys across schemas. `client_id` columns outside `identity` are
  plain text.
- Cascade deletes across services happen through events: on
  `white.client.deleted` every service purges that client's rows.
- Rows copied from the Prisma era keep their cuid ids (see each service's
  `00002_import_prisma.sql`). New rows use UUIDv7 (`pkg/ids`).

## Request flow

1. The frontend calls REST on the gateway with `Authorization: Bearer <access
   token>` (15 min, EdDSA, issued by identity) and refreshes with the opaque
   refresh token (14 days, rotated on every use).
2. grpc-gateway forwards `authorization`, `cookie`, `x-forwarded-for` and
   `user-agent` as gRPC metadata. Each service's interceptor verifies the JWT
   with the public key and puts the `authn.Principal` in the context. Use
   cases call `authn.Require` / `authn.RequireAdmin`; client-scoped use cases
   call identity `AccessService.CheckClientAccess` (cached 30 s).
3. Errors travel as gRPC status + `ErrorInfo{reason: <apperr.Code>}` (+
   `BadRequest` field violations). The gateway writes
   `{ok:false, error, code, fieldErrors}`: the same shape the Next.js
   `ActionResult` had.

## Browser routes on the gateway

Outside grpc-gateway, because they need cookies, redirects or raw bytes:

| Route | Purpose |
|---|---|
| `GET /api/connections/{google,meta}/start?client=<slug>` and `/callback` | OAuth connect flow (nonce cookie, 10 min) |
| `POST /api/share/{slug}/unlock` | verify the share PIN, set `share_<clientId>` (12 h) |
| `GET /api/share/{slug}/context`, `/dashboard?preset=`, `/pdf?preset=` | public share page data and PDF (401 PIN_REQUIRED, 404 DISABLED / NOT_FOUND) |
| `GET /api/reports/{id}/pdf` | report PDF |
| `POST /api/clients/{id}/media/upload`, `POST /api/clients/{id}/ads/import` | multipart uploads |
| `GET /api/media/{key}` | public media bytes with ETag/304 (Meta fetches post media by URL) |
| `POST /internal/session`, `POST /internal/users/sync` | Next.js server only (`X-Internal-Secret`): trade a session user id for an access token, mirror admin user edits into identity |
| `POST /api/finance/reimbursements/{id}/receipts`, `/api/performance/tasks/occurrences/{id}/attachments`, `/api/hr/employees/{id}/documents` | ERP multipart uploads (form field `file`, 10 MB) |
| `POST /api/content/clients/{id}/design-tasks/{task}/files`, `/api/content/clients/{id}/assets`, `/assets/{asset}/versions` | content multipart uploads (25 MB per file) |
| `POST /api/recruitment/public/jobs/{slug}/apply`, `/api/recruitment/public/psikotes/{token}/tests/{test}/upload`, `/api/recruitment/candidates/{id}/cv` | recruitment multipart uploads (field `file`, 10 MB); the first two need no login |
| `GET /api/finance/receipts/{id}`, `/api/sales/{quotations,invoices,contracts}/{id}/pdf`, `/api/sales/pitch-decks/{id}/file`, `/api/sales/onboarding/assets/{id}`, `/api/public/sales/contracts/{token}/pdf` | files from internal RPCs, under the caller's token (the public contract PDF by token) |

Any REST response whose message has `bytes data` and `string content_type`
(payslips, exports, recap PDFs) leaves the gateway as raw bytes with that
content type and an inline `Content-Disposition`, so a plain link downloads
it (`internal/gateway/files.go`).

Browser navigations cannot send headers, so these routes also accept the
access token from a `white_access` cookie. The gateway stamps every call it
forwards with `x-white-edge`; RPCs that trust internal callers reject
anonymous calls carrying that marker.

## Asynchronous work

- Stream `WHITE`, subjects `white.>`, file storage, 7 day retention. Durable
  pull consumers per service, explicit ack, NAK with 30 s delay, max 5
  deliveries.
- The scheduler publishes `white.sync.daily.analytics` and
  `white.sync.daily.seosuite` once per client at 02:00 UTC (planned with
  `planDailyRun`: least-recently-synced first), `white.publish.tick` every
  5 minutes and `white.retention.tick` daily.
- Each service's `worker` consumes its subjects. Every binary can run as
  `serve` (gRPC + worker) or `worker` (consumer only) so sync load scales
  separately from request load.

## Secrets

- `ENCRYPTION_KEY` (AES-256-GCM, format compatible with the Next.js
  `lib/crypto.ts`) is held by **integrations only**. Other services obtain
  decrypted tokens per call via `ConnectionService.GetAccessToken` and
  provider keys via `SettingsService.GetProviderCredentials`, cached for 60 s
  and busted by `white.integration.updated`.
- `JWT_PRIVATE_KEY` is held by identity. Every other service needs only
  `JWT_PUBLIC_KEY` (or `AUTH_SECRET` in development, from which both are
  derived).

## Layout

```
api/proto/white/<svc>/v1/*.proto   contracts (grpc-gateway + openapi annotations)
gen/go, gen/openapi                generated (committed)
cmd/<svc>/main.go                  app.Main(<svc>.Spec())
internal/<svc>/
  service.go                       Spec(): migrations, Register, Worker
  domain/                          entities, enums, repository ports, pure rules
  app/                             use cases (one struct per bounded area)
  adapter/grpc/                    proto <-> domain, auth checks
  adapter/postgres/                sqlc.yaml, queries/*.sql, sqlc/ (generated), store.go
  adapter/providers/<name>/        external APIs: real + mock behind one interface
  adapter/clients/                 gRPC clients to other services (+ caches)
  migrations/*.sql                 goose, embedded
pkg/                               shared: apperr, authn, grpcx, postgres, events, dates, metrics, ...
deploy/                            docker-compose.yml, Dockerfile
docs/inventory/*.md                behaviour inventory of the Next.js code being ported
```

See `CONVENTIONS.md` for the rules each layer follows and
`internal/identity` for the reference implementation.
