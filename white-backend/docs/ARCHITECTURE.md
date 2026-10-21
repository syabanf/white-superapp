# WHITE Backend: architecture

Go rewrite of the WHITE Analytics backend (`../white-analytics`, Next.js server
actions + Prisma). Six deployable services, one Postgres instance with one
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
│  :9001   │ │   :9002     │ │   :9003    │ │    :9004     │ │   :9005    │
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
| **scheduler** | nothing | publishes daily/publish/retention ticks per client |
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
