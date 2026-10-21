# Conventions (read before writing a service)

`internal/identity` is the reference. Copy its shape; do not invent a second
one.

## Layers

- **domain**: structs, string enums (`type PostStatus string`), repository
  interfaces (`Store` that embeds every repo and has `WithTx`), and pure
  rules (state machines, validation of invariants, metric formulas). No
  imports from gRPC, pgx, NATS or other services.
- **app**: one struct per area (`Users`, `Clients`, ...) built with
  `NewX(deps...)`. Methods take `ctx` first, plain Go values, and return
  domain values plus `error`. User-facing failures are `*apperr.Error`;
  anything else bubbles up as an internal error.
- **adapter/grpc**: `Register(s *grpc.Server, svc Services)`. Handlers do
  `authn.Require`/`RequireAdmin`, convert proto to Go, call app, convert
  back. No business rules here.
- **adapter/postgres**: sqlc queries in `queries/*.sql`, `store.go`
  implementing the domain ports. Translate unique violations to
  `*domain.ErrUnique`; `pgx.ErrNoRows` becomes a nil result, never an
  error.
- **adapter/providers/<name>**: an interface in the package, `real.go` and
  `mock.go`. A `Pick(creds)` function chooses per call based on the
  credentials the integrations service returned, so saving keys in the
  setup wizard switches to real without a restart.
- **adapter/clients**: thin wrappers over generated gRPC clients of other
  services. Put caches here (access checks 30 s, provider credentials 60 s).

## Authorization

- `authn.Require(ctx)` for any signed-in user; `authn.RequireAdmin(ctx)` for
  ADMIN-only RPCs.
- Client-scoped RPCs call `AccessClient.Check(ctx, userID, clientID)` and
  then check `CanManage` where the old code used `manageContext` /
  `canManage`. Keep the old permission matrix exactly (see the inventory:
  many actions allow VIEWER).
- Internal RPCs (no `google.api.http` annotation) trust their parameters;
  the gateway never exposes them.

## Errors

- Codes: `apperr.Validation`, `Unauthenticated`, `Unauthorized`, `NotFound`,
  `Conflict`, `RateLimit`, `TokenExpired`, `Permission`, `Network`,
  `Unknown`. Messages are Bahasa Indonesia; the defaults in `apperr` match
  `t.errors` from the Next.js app.
- Field errors: `validate.New()` accumulator or `apperr.Field(field, msg)`.
  Keys are the old zod top-level field names (`email`, `slug`, `username`,
  `siteUrl`, `externalId`, ...).
- Provider failures: `apperr.FromProvider(provider, code, msg, cause)`;
  Meta and Google mapping lives in `pkg/providers/meta` and
  `pkg/providers/google`.

## Data rules carried over

- Every metric day is a UTC date (`pkg/dates`). Presets end on the latest
  complete day. Comparison period = same length ending the day before.
- Deltas use `metrics.ComputeDelta`; `pct` is nil when the previous value is
  0. Lower-is-better is a presentation flag, never baked into the number.
- A day GSC or GA4 did not report stays `null` through the API. Never
  coalesce it to 0 in a daily series (sparklines may, as before).
- Demo mode: `isDemo` = every source row has no connection id. Mock adapters
  are deterministic (`pkg/providers/mockrand`, same seeds as the TS mocks).
- Every sync path records a `sync_jobs` row (`kind`, `status`,
  `started_at`, `finished_at`, `message`, `error`). Kinds and messages are
  listed in `docs/inventory/publishing-seosuite-sync.md`.
- bcrypt cost 10 for passwords and share PINs. AES-GCM ciphertext format is
  fixed (`pkg/crypto`).

## SQL

- snake_case tables and columns inside the service schema; ids are `text`.
- Postgres enums per schema for fixed value sets (`identity.user_role`).
- `00001_init.sql` creates the schema; `00002_import_prisma.sql` copies from
  `public."PrismaModel"` inside a `DO $$ ... $$` block guarded by
  `to_regclass`, `ON CONFLICT DO NOTHING`, so it is safe on a fresh database
  and on the existing one.
- sqlc: `emit_pointers_for_null_types`, timestamptz -> `time.Time`, jsonb ->
  `[]byte` (see `internal/identity/adapter/postgres/sqlc.yaml`).
- Bulk inserts: `:copyfrom` for append-only, `INSERT ... ON CONFLICT DO
  UPDATE` for upserts, batched in the store.

## Proto

- One file per service: `api/proto/white/<svc>/v1/<svc>.proto`, package
  `white.<svc>.v1`, `go_package .../gen/go/white/<svc>/v1;<svc>v1`.
- Public RPCs carry `google.api.http`. Paths: `/v1/clients/{client_id}/...`
  for client-scoped resources, `/v1/<resource>` otherwise. Verbs as custom
  methods: `POST /v1/...:sync`.
- Enums start with `_UNSPECIFIED = 0`. Optional scalars use `optional`.
  Days are `string` (`YYYY-MM-DD`); instants are `google.protobuf.Timestamp`.
- Shared types live in `white/common/v1/common.proto` (`RangeQuery`,
  `ResolvedRange`, `Delta`, module/platform enums, `DataState`).
- After editing: `./scripts/proto.sh` and commit `gen/`.

## Testing

- Pure rules: table tests in the package.
- Repositories and use cases: integration tests that call
  `postgres.TestPool(t, Migrations(), "migrations", Schema)` and skip
  without `WHITE_TEST_DATABASE_URL`. Truncate the schema at the start of
  each test. Packages of one service share that schema, so run `go test -p 1`
  (the Makefile and CI do).
- Providers: tests against `httptest.Server` for the real adapters and
  snapshot-style tests for the mocks.
- `go vet ./...`, `gofmt -l`, `go test ./...` must pass.

## Prose

Code comments, docs and commit messages in English; every user-facing
string in Bahasa Indonesia. No em dashes.
