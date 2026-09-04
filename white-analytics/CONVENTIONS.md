# WHITE Analytics — Conventions (READ BEFORE WRITING CODE)

Next.js 16 (App Router, RSC, Turbopack) · TypeScript strict · Tailwind v4 + shadcn/ui · Recharts 3 · TanStack Table v8 · Prisma 7 + PostgreSQL · Auth.js v5 · nuqs. UI copy is **Bahasa Indonesia**. Code identifiers/comments in English.

Design doc: `../docs/plans/2026-08-19-nextjs-analytics-dashboard-design.md`. The **exemplar** page is `src/app/(app)/clients/[slug]/page.tsx` + `src/features/overview/*` — copy its patterns.

## Next.js 16 gotchas (this is NOT the Next.js you may know)
- `params` and `searchParams` are **Promises**: `const { slug } = await props.params; const sp = await props.searchParams;`
- Use the global type helpers: `PageProps<'/clients/[slug]/social'>`, `LayoutProps<'/…'>`. Run `npx next typegen` if a new route's type is missing.
- `middleware.ts` is now `src/proxy.ts` (already exists — don't touch).
- `cookies()`/`headers()` are async. No `cacheComponents` (we render dynamically).
- Route Handlers: `src/app/api/**/route.ts` export `GET/POST`; use `RouteContext<'/api/…/[id]'>` type for params.
- Docs are in `node_modules/next/dist/docs/01-app/**` if unsure.

## Folder contract
```
src/app/(app)/…                    routes inside the authenticated shell (sidebar+topbar). Pages are async RSCs.
src/app/share/[slug]/…             public client dashboard (no shell) — Reports/Share owner only
src/app/api/…                      route handlers (auth, cron, pdf, ai, oauth callbacks)
src/features/<module>/queries.ts   server-only data access ("import 'server-only'"; wrap in React `cache`)
src/features/<module>/actions.ts   "use server" actions returning ActionResult<T> (src/lib/action-result.ts)
src/features/<module>/components/  module UI. Charts/tables are client components ("use client") receiving plain props.
src/lib/providers/<name>/          external adapters: `types.ts` (interface), `mock.ts`, `real.ts`, `index.ts` (picks by env)
src/lib/metrics/*                  ALL formulas live here (pure, tested). Never inline a formula in a component/query.
src/i18n/id.ts                     ALL UI strings. Add keys under your module's section (t.social / t.seo / t.ads / t.reports / t.settings / t.admin / t.clients / t.portfolio). Do not rename existing keys.
```

## Page pattern (copy from the exemplar)
```tsx
export default async function Page(props: PageProps<"/clients/[slug]/social">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canManage } = await requireClientAccess(slug);      // src/lib/rbac.ts (RBAC in the data layer)
  const { range, previous, compare } = getRange(sp);                  // src/lib/range-params.ts — global date range from URL
  const data = await getSomething(client.id, range, previous);        // features/<module>/queries.ts
  return (<> <PageHeader …/> <KpiGrid>…</KpiGrid> … </>);
}
```
- Global filters live in the topbar: date range (`?from&to` or `?preset=28d`) and compare (`?compare=0` disables). Every KPI/chart/table on the page MUST use the same `range`/`previous`. When `compare` is false, pass `delta={null}`.
- Deltas: always `computeDelta(current, previous)` from `@/lib/metrics`; render with `<DeltaBadge>` / `KpiTile delta=`. Set `lowerIsBetter` for CPR/CPC/CPM/position/LCP etc. **Never hard-code trend numbers.**
- Server components fetch; client components render. Pass serializable props only (Dates → ISO strings or numbers).
- Loading: add `loading.tsx` with `<Skeleton>`s per route folder. Errors: throw in queries only for programmer errors; provider errors → `ProviderError` (src/lib/action-result.ts) caught at the action boundary → `ProviderErrorBanner`.
- Mutations: server actions with Zod validation → `ActionResult`. Call `revalidatePath` after writes. Client forms use `useActionState` or `useTransition` + `toast` from `sonner`.

## Primitives (use these — do not create parallel versions)
| Import | Use |
|---|---|
| `@/components/dashboard/page-header` → `PageHeader`, `Section` | page title/eyebrow/description/actions; section wrapper |
| `@/components/dashboard/stat-strip` → `StatStrip` | secondary metrics as one hairline strip (auto-fits, last row stretches) |
| `@/components/dashboard/kpi-tile` → `KpiTile`, `KpiGrid` | stat tiles (value string already formatted, `delta`, `spark: number[]`, `hint`, `caption`, `lowerIsBetter`) |
| `@/components/dashboard/delta-badge` → `DeltaBadge` | period-over-period badge |
| `@/components/dashboard/chart-card` → `ChartCard` (client) | card w/ title/description/actions + optional `table` twin toggle |
| `@/components/dashboard/charts/time-series-chart` → `TimeSeriesChart` | daily series: `data: {date:'YYYY-MM-DD',…}[]`, `series: [{key,label,type:'line'|'area'|'bar',format?,color?}]`, `previousKey`, `stacked`, `reverseY` (positions), `yFormat` |
| `@/components/dashboard/charts/hbar-chart` → `HBarChart` | horizontal bar list for categories (`items:[{label,value,secondary?,href?,muted?}]`, `format`, `showShare`) |
| `@/components/dashboard/charts/heatmap` → `Heatmap` | sequential heatmap (`cells:[{row,col,value,count?}]`) |
| `@/components/dashboard/charts/score-ring` → `ScoreRing`, `RATING_LABEL` | 0–100 score meter (Lighthouse) |
| `@/components/dashboard/charts/chart-primitives` → `slotColor(i)`, `SeriesLegend`, `TooltipBox` | if you must build a custom Recharts chart |
| `@/components/dashboard/data-table` → `DataTable`, `ColMeta` | TanStack v8 table: sorting, search (`searchKey`), pagination, `exportName` (CSV), `onRowClick`, `meta:{align:'right'}` for numeric cols |
| `@/components/dashboard/empty-state` → `EmptyState` | empty/CTA states |
| `@/components/dashboard/status-badge` → `StatusBadge`, `ratingToKind` | good/warning/critical badges (icon + label) |
| `@/components/dashboard/banners` → `DemoBanner`, `ProviderErrorBanner` | demo-data notice; provider error with reconnect |
| `@/components/dashboard/sparkline` → `Sparkline` | tiny inline sparkline |
| `@/features/social/components/platform-icon` → `PlatformIcon`, `platformLabel` | IG/FB/TikTok marks (brand colors ONLY here) |
| `@/features/reports/components/ai-insight-card` → `AiInsightCard`, `MarkdownLite` | AI summary card + safe markdown-lite renderer |
| `@/features/clients/wizard` → `resolveStep`, `nextStep`, `wizardHref`, `sourceSectionsFor` | onboarding wizard step model (pure, tested in `tests/wizard.test.ts`) |
| `@/features/clients/components/wizard/*` → `WizardShell`, `WizardNav` | wizard chrome: numbered rail (desktop) / progress header (mobile) + footer nav that sticks to the bottom on phones. New steps go through these. |
| `@/features/guide/content` → `GUIDE_SECTIONS`, `guideSectionForPath` | in-app guide copy. **One source**: the `/panduan` page and the topbar help sheet both render it. Add a new route → map it in `guideSectionForPath` (tested in `tests/guide.test.ts`). |
| `@/lib/toast` → `toast` | **Always import toast from here, never from `sonner`.** It plays the matching sound cue and then delegates. (`Toaster` itself still comes from sonner, in the root layout.) |
| `@/lib/sound` → `playSound`, `isSoundEnabled`, `setSoundEnabled`, `subscribeSound` | Web Audio cues (`success` / `error` / `notify`), synthesised — no audio files. Only ever call after a user action. |
| `@/components/ui/*` | shadcn (button, card, tabs, dialog, select, badge, tooltip, table, input, label, switch, textarea, checkbox, progress, alert, sheet, dropdown-menu, skeleton, separator, avatar, scroll-area, toggle-group, command, popover, calendar, alert-dialog, radio-group, empty, spinner, kbd, field, input-group, sonner) |
| `@/lib/format` | `formatNumber, formatCompact, formatCurrency(v, currency, {compact}), formatPercent, formatDeltaPercent, formatDecimal, formatMs, formatDuration, formatDate, formatDateShort, formatDateLong, formatDateTime, formatDateRange, formatRelative, truncate, initials, slugify` — id-ID everywhere |
| `@/lib/dates` | `DateRange, resolveRange, previousRange, eachDay, dayKey, addDays, toISODate, parseISODate, latestCompleteDay, tickEvery` |
| `@/lib/metrics` | social: `postEngagements, postEngagementRate, avgEngagementRate, followerGrowth, postsPerWeek, contentType, postingHeatmap, topPosts` · seo: `ctr, aggregateSearch, positionBucket, positionDistribution, expectedCtr, strikingDistance, lowCtr, declining, rising, rateCwv, rateScore, healthScore, CWV_THRESHOLDS` · ads: `aggregateAds, costPerResult, adCtr, cpc, cpm, frequency, roas, isFatigued, EMPTY_AD_KPIS` · series: `fillDaily, sparkline, rollingMean, alignPrevious` · `computeDelta, isGoodChange, sumBy, mean, safeDiv, weightedMean, clamp, round` |
| `@/lib/rbac` | `requireUser, getSessionUser, requireAdmin, requireClientAccess(slug) → {client, role, canManage}, listAccessibleClients` |
| `@/lib/integrations` | `getIntegrationStatuses, saveIntegration, clearIntegration, hydrateIntegrationEnv, getSetupState, markSetup` — app-level credentials (see "Workspace setup wizard") |
| `@/lib/db` | Prisma client `db` + all enums re-exported. Types: `import type { SocialAccount } from "@/generated/prisma/client"` |
| `@/lib/crypto` | `encrypt/decrypt` (AES-GCM) for provider tokens; `randomPin` |
| `@/lib/action-result` | `ActionResult, ok(), fail(), ProviderError, isProviderError` |

## "Buat project" wizard (`/clients/new`)

Starting a project goes through the wizard, not a dialog — every "Klien baru" entry point links to `/clients/new`, and the portfolio empty state (first run) lands there too.

- Steps: `jenis → profil → sumber → akses → selesai`, defined once in `src/features/clients/wizard.ts`. The rail, the page and the tests all read that file — never hard-code a step list elsewhere.
- **Project type comes first and drives the rest.** The choice rides in the URL (`?types=SEO,ADS`) until the client is created, so going back and changing it costs nothing. It then persists as `Client.shareModules`, decides which source sections step 3 renders (`sourceSectionsFor`), and makes the website field required for SEO projects (`websiteRequiredFor`).
- **The client row is created at step 2** (profil), which is also where an SEO project auto-registers its `SeoProperty` from the website field — the URL is never typed twice. From then on the flow is resumable from its URL (`?step=…&client=<slug>`).
- Later steps reuse the *existing* settings actions/components (`ConnectionCard`, `SocialAccountsList`, `MembersSection`, `ShareSection`). Don't fork them — wizard and Settings must stay one implementation.
- `resolveStep()` guards direct URL access: no type → `jenis`; no client → `profil`; already created → skip ahead to `sumber`.
- Steps 3 and 4 are optional by design (demo data works without connections), so never block "Lanjut" on them.

## In-app guide (`/panduan` + topbar help)

Help lives in one place: `src/features/guide/content.ts`. The full page renders every section; the `?` button in the topbar opens just the section matching the current route, plus a deep link to the full page.

- Adding a page? Map it in `guideSectionForPath()` — the test asserts every known route resolves to a real section.
- **Formulas quoted in the guide must match `src/lib/metrics/*`.** If a formula changes there, update the guide in the same commit — a wrong explanation is worse than none.
- Guide copy is content, not chrome: it goes in `content.ts`, not scattered through components.

## Sound

Feedback cues are synthesised with the Web Audio API in `src/lib/sound.ts` — no audio assets, no dependency, nothing to fail to load.

- **Never call `playSound` on render, on mount, or on a timer** — only in response to something the user did. That is also what keeps us inside browser autoplay rules (the AudioContext is created lazily on that first gesture).
- Cues are deliberately quiet (peak gain 0.05) and short (< 250 ms): success rises, error falls, notify is a single blip. Don't add louder or longer sounds.
- Users can mute from the user menu; the preference lives in `localStorage` and is read through `useSyncExternalStore` (`subscribeSound`), so it syncs across tabs. Default is **on**.
- Because every toast is routed through `@/lib/toast`, a notification and its sound can never drift apart.

## Visual tone — withwhite.id (READ THIS BEFORE STYLING ANYTHING)

The product wears the agency's own visual language: a **plain white canvas**, one decisive **royal blue**, tight bold type, and **monospace micro-labels**. Structure comes from hairline borders — no background grid, no decorative rules. Data is the only loud thing on screen.

| Use | How |
|---|---|
| Page plane | plain `bg-background` (white). **Do not add background grids, ruler strips, or decorative line textures** — removed deliberately. |
| Accent | `bg-brand` / `text-brand` / `text-brand-ink` (never hard-code a blue) |
| Page title | `<PageHeader eyebrow="…">` → mono blue eyebrow + tight bold headline + short blue rule |
| Section title | `<Section index="01" title="…">` |
| Micro-label | `.label-mono` (mono, 10px, uppercase, wide tracking) — KPI labels, group labels, metadata |
| Technical framing | `.crop-marks` on sparse panels (KPI tiles, hero cards). NOT on chart cards or dense tables — it reads as noise there. |
| Buttons | pill by default (base `Button` is `rounded-full`). **Never hand-tune `h-*`, `px-*` or `text-xs` on a Button** — pick a `size`: `xs` 28px · `sm` 32px · `default` 36px · `lg` 40px, and `icon-xs`/`icon-sm`/`icon`/`icon-lg` for square ones. Height, padding, text size and icon size all travel with the size. |
| Tabs | `Tabs`/`TabsList`/`TabsTrigger` render a pill segmented control (40px track, 13px labels) that matches the button scale. Don't add padding classes to a trigger; for link tabs use `asChild` with a bare `<Link>`. |
| Selects next to buttons | `<SelectTrigger size="sm">` — no manual height. |
| Sidebar nav | hairline divider per row + blue **dot** for the active item; never a filled active block |
| Numbers | proportional figures for big values; `.tabular` only in aligned columns |
| Motion | One vocabulary, tokens in `globals.css` (`--dur-fast/--dur/--dur-slow`, `--ease-out`). `.lift` = clickable surface (2px rise + `--lift-shadow`); `.nudge` inside `group/nudge` = directional arrow; `.page-enter` on a page wrapper = staggered arrival of its direct children. **Never hand-roll a transform/duration** — reach for these. |
| Reduced motion | **Deliberately NOT honoured** — motion runs for every user (product decision 2026-08-21, requested by the WHITE team; note at the bottom of `globals.css`). This is why every movement stays small and short: 2px lift, 8px arrival, ≤260ms. Do not add large or looping motion — that is what would actually hurt. Restore the guard if motion discomfort is ever reported. |
| Loading | while the date range refetches, `body[data-nav-pending="true"]` dims `.page-enter`. Hold the frame — never swap live numbers for a skeleton. |
| Charts | deliberately **not** animated (`isAnimationActive={false}`): every filter change would replay the animation and read as lag. |
| Composition | **Few, large, focal elements.** Max ~4 primary `KpiTile`s per page — secondary metrics go in a `StatStrip` (one hairline strip), never a second row of cards. |
| Panels in a row | give them one shape (header · `flex-1` body · footer) + `min-h-*` so bottoms align. See `features/overview/components/module-cards.tsx` → `ModulePanel`. |
| Empty states | never leave a stretched card with one grey line — centre an icon + copy + the primary action (see `AiInsightCard`). |
| Missing days | GSC/GA4 series use `fillDaily(..., { missing: null })` — an unreported day is a gap, not zero. Ads keep `0` (no delivery really is zero). |
| Nominal bars | one hue (`--chart-1`) for every bar of a single series; never a rainbow by rank. |

Do not reintroduce the old monochrome near-black accent, and do not add a second brand hue.

## Charts — hard rules (from the dataviz method)
- **One y-axis per chart. Never dual-axis.** Two measures of different scale → two `TimeSeriesChart`s side by side (small multiples) or index to 100.
- Categorical hue = `slotColor(i)` in fixed order (**slot 1 is the brand blue**, 2 orange, 3 aqua, 4 yellow …). Color follows the entity, never its rank; ≥ 2 series → legend (built-in). ≤ 4 series per chart; more → "Lainnya"/facet.
- Sequential (magnitude) → the brand-hue ramp `--seq-100..700` (`Heatmap`). It is **reversed in dark mode** (brighter = more), so just use the tokens in order and both themes read correctly.
- Status colors (`--status-good/warning/critical`) only for real status (CWV rating, health) and always with icon + label (`StatusBadge`).
- Lines 2px, areas 10% wash, bars ≤ 24px with 4px rounded data-end, hairline solid grid — all built into the primitives; don't override.
- Every chart has a table twin when it carries > a handful of values: pass `table={<DataTable bare … />}` to `ChartCard`.
- Text never wears the series color. Values lead labels in tooltips (built-in).
- No pies/donuts for comparisons; use `HBarChart`. Part-to-whole ≤ 6 segments → stacked bar (`TimeSeriesChart stacked`) or `HBarChart showShare`.
- Position charts: `reverseY` (1 is best). Positions/CPR/CPC/CPM/LCP are **lower-is-better** for deltas.
- Platform brand colors (`--brand-instagram/facebook/tiktok`) are identity marks only (icons/chips) — never series colors.
- `formatCompact` glues the unit with a non-breaking space (`1,3 rb`) so axis ticks never wrap. Keep it that way.

## Data facts (seed) you can rely on
- 3 clients: `kopi-nusantara`, `adiharjo-property`, `bali-villa-escapes` (each with IG/FB/TikTok own accounts + 2–3 IG competitors, 120 days of `SocialSnapshot`, ~50–70 `SocialPost` per own account, 1 `SeoProperty` with 120 days `SeoDailyMetric` + `SeoDimensionMetric` (QUERY ~46 keys, PAGE 15, COUNTRY 5–8, DEVICE 3) + `Ga4DailyMetric`, 4 audit runs × MOBILE/DESKTOP in `SeoAudit`, 1 `SeoCrawl` with ~27 pages + issues, 1 `AdAccount` with 4 campaigns → adsets → ads and ad-level `AdDailyInsight` per day (`resultType` per campaign: link_click / purchase (+purchaseValue) / lead / messaging_conversation_started / reach), `AdDemographic` per campaign (age×gender), 2 `Report`s, 3 `SyncJob`s.
- Users: `admin@white.id / admin12345` (ADMIN), `tim@white.id / member12345` (MEMBER of kopi-nusantara as MANAGER, adiharjo-property as VIEWER). Client `kopi-nusantara` has share PIN `123456`; others no PIN.
- Awareness campaigns report `resultType: "reach"`; exclude `NON_CONVERSION_RESULT_TYPES` (exported from `features/overview/queries.ts`) when summing "hasil"/CPR.
- Ad-level rows have `adId` set; use `adId: { not: null }` when aggregating to avoid double counting if account/campaign-level rows are ever inserted.
- Dates in DB are UTC-midnight `@db.Date`; posts use full timestamps. Query with `{ gte: range.from, lte: range.to }` for dates and `{ gte: range.from, lt: addDays(range.to, 1) }` for timestamps.

## Real data sync (GSC/GA4, Meta Ads, Meta Insights)

One pattern for every source: an **auth-free write path** in `features/<module>/sync.ts` (injectable provider/token for tests), a **pure mapper** in `features/<module>/sync-map.ts` (tested in `tests/sync-map.test.ts`), and callers that choose the most real path available:

| Source | Real when | Otherwise |
|---|---|---|
| Social snapshots/posts | own account has a Meta `Connection` → `syncSocialMeta` (reach, impressions, profile views, per-post insights) | Apify public data (`syncSocialPublic`) → demo extrapolation |
| SEO daily + dimensions + GA4 | `SeoProperty.connectionId` has a usable Google token → `syncSeoProperty` | `mode: "demo"`, nothing written (seed stays) |
| Meta Ads hierarchy + insights + demographics | `AdAccount.connectionId` → `syncAdAccount` | `mode: "demo"`, touch `lastSyncedAt` only |

- Tokens come from `src/lib/connections.ts` → `getConnectionToken(connectionId)`: decrypts, refreshes Google access tokens with the stored refresh token, throws `TOKEN_EXPIRED` for expired Meta tokens (the UI says "Hubungkan ulang"). Never decrypt `Connection.accessTokenEnc` inline.
- `GET /api/cron/daily` runs `features/sync/daily.ts` (`runDailySync`) then the SEO-suite job. Every SyncJob message says `mock` when nothing real happened, so the activity feed never lies.
- DB-backed smoke: `pnpm sync:smoke` runs all three pipelines with the mock adapters on a throwaway client (idempotency asserted) — run it after touching any sync writer.

## Workspace setup wizard (`/setup`, ADMIN)

App-level credentials are no longer env-only. `src/lib/integrations.ts` stores them encrypted in `IntegrationSetting` and copies them into `process.env` on boot (`src/instrumentation.ts`) and on save — **adapters keep reading env and never touch the DB**. DB values win over deployed env vars. Provider singletons that used to be chosen at import time (`metaGraph`, `socialPublisher`, `seoData`) now go through `lazyProvider()` so they flip from mock to real without a restart.

- Registry of providers + Meta App Review checklist + DataForSEO cost model: `src/features/setup/integrations.ts` (pure, tested in `tests/setup.test.ts`). Add a provider there, then to `.env.example`.
- Credential checks (`src/lib/providers/verify.ts`) are one cheap call per provider; never log the secret.
- First run: an ADMIN with zero clients and an untouched `AppSetting{key:"setup"}` is redirected from `/` to `/setup`. "Lewati" is always allowed — demo mode is a feature, not a failure state.
- The wizard chrome is the shared `WizardShell` (now generic: `steps`, `current`, `hrefFor`) + `WizardNav` — the same one `/clients/new` uses. Do not build a third wizard skin.
- **Apify** (`src/lib/providers/apify/*`) is the "public data" provider: `publicSocial` (profile + posts for IG/TikTok/FB, real when `APIFY_TOKEN` is set, else deterministic mock) feeds `features/social/sync.ts` and competitor discovery; `apifySerpRanks` backs the rank tracker when DataForSEO is empty (`seoDataMode()` → `dataforseo | apify | mock`). Actor ids are overridable via `APIFY_ACTORS` JSON; normalisers in `apify/normalize.ts` are pure and tested — extend them, never parse actor output inline.
- Publish readiness is `reviewProgress(metaReview).publishReady` (all publish permissions APPROVED **and** app Live). Copy in the product must say "simulasi" until then.

## Platform modules — Publishing & SEO suite (added 2026-09-02)

Design: `../docs/plans/2026-09-02-platform-expansion-design.md`. These modules make the product *act*, not just report; they hang off the existing `Client`, `SocialAccount` and `SeoProperty` rows.

| Area | Where | Notes |
|---|---|---|
| Publishing (composer, calendar, approvals, media) | `src/features/publishing/*`, routes `/clients/[slug]/publish/**` | `ContentPost` (+ `PostTarget` per destination) is authored here; `SocialPost` stays synced history. Status machine and per-platform validation live in `features/publishing/lib.ts` (pure, tested) — never re-derive them in a component. |
| Scheduler | `features/publishing/scheduler.ts` → `publishDuePosts()` | Called by `GET /api/cron/publish` (every 5 min). Claims SCHEDULED rows atomically, so overlapping runs never double-publish. |
| Publisher adapter | `src/lib/providers/social-publisher/*` | Instagram/Facebook via Graph (needs App Review perms); TikTok throws `PERMISSION` until the Content Posting API is approved. Mock succeeds unless the text contains `[fail]`. |
| Media storage | `src/lib/storage.ts` + `GET /api/media/[...key]` | Local disk (`UPLOAD_DIR`, default `.uploads/`). Only ever go through `putObject`/`getObject`/`publicUrl` so S3/R2 is a one-file swap. Keys are `<clientId>/<uuid>.<ext>` — unguessable, served public because Meta fetches media by URL. |
| SEO suite (research, rank, backlinks, competitors) | `src/features/seo-suite/*`, routes `/clients/[slug]/seo/{research,rank,backlinks,competitors}` | Formulas in `src/lib/metrics/seo-suite.ts` (visibility = mean of `expectedCtr(position)`; null position = 0). Research results are cached in `KeywordResearch` and reused for 7 days. |
| Daily SEO job | `features/seo-suite/daily.ts` → `runSeoSuiteDaily()` | Called from `GET /api/cron/daily`: rank snapshots, backlink snapshot, domain snapshots, scheduled audits (`SeoProperty.auditCadence`). |
| SEO data adapter | `src/lib/providers/dataforseo/*` | Single paid provider (`DATAFORSEO_LOGIN/PASSWORD`). Mock is deterministic per input so history looks real and tests stay stable. |
| Notifications | `src/lib/notify.ts` → `notify(userIds, {...})`, `clientAudience()`, `clientApprovers()` | Bell in the topbar (`components/shell/notification-bell.tsx`, feed fetched by the app layout — no polling). Types are upper-snake tags per module (`POST_REVIEW`, `POST_PUBLISHED`, `RANK_DROP`, …). Always pass an `href` so the row is actionable. |
| Roles | `requireClientAccess` → `canManage` | ADMIN & MANAGER create/approve/schedule; VIEWER is read-only; a MANAGER may not approve their own post (ADMIN may). |

Seed: `prisma/seed.ts` calls `seedPublishing()` and `seedSeoSuite()` (own files, idempotent, also runnable standalone). Guide: each module exports a `GuideSection` from `features/<module>/guide.ts`; `features/guide/content.ts` imports and maps it in `guideSectionForPath`.

Mobile is a requirement for every page: no horizontal page scroll at 375 px, tables inside their own overflow container, sticky bottom action bar (with `env(safe-area-inset-bottom)`) wherever a form has one primary action — see `WizardNav` for the pattern.

## Providers (adapters)
- Env: `META_APP_ID/SECRET`, `GOOGLE_CLIENT_ID/SECRET`, `PAGESPEED_API_KEY` (optional), `OPENROUTER_API_KEY`, `OPENROUTER_MODEL`, `DATAFORSEO_LOGIN/PASSWORD`, `UPLOAD_DIR`, `CRON_SECRET`, `ENCRYPTION_KEY`, `NEXT_PUBLIC_APP_URL`. When credentials are missing → **mock adapter** and show `DemoBanner`.
- Real Google/Meta/PageSpeed/OpenRouter calls: plain `fetch` (no heavy SDKs). Normalize errors into `ProviderError(provider, code, message)`.
- Tokens are stored encrypted (`Connection.accessTokenEnc` via `encrypt()`); never returned to the client.
- Cron: `GET /api/cron/daily` (02:00 UTC: sync + SEO suite daily) and `GET /api/cron/publish` (every 5 min: due posts), both guarded by `Authorization: Bearer ${CRON_SECRET}` — log into `SyncJob`.

## Do / Don't
- DO run `pnpm typecheck` and `pnpm lint` before reporting. `pnpm test` must stay green (add tests to `tests/*.test.ts` for pure logic you add).
- DO keep files ≤ ~400 lines; split components.
- DO use `t.*` for every visible string; add new keys to `src/i18n/id.ts` under your module section only.
- DON'T add npm dependencies (everything needed is installed: recharts, @tanstack/react-table v8, papaparse, cheerio, @react-pdf/renderer, zod, date-fns, nuqs, sonner, next-themes, bcryptjs, lucide-react). If truly blocked, say so in your report instead.
- DON'T edit shared files: `prisma/schema.prisma`, `prisma/seed*.ts`, `src/lib/*` (except adding a NEW provider folder under `src/lib/providers/`), `src/components/{ui,dashboard,shell}/*`, `src/app/layout.tsx`, `src/app/(app)/layout.tsx`, `src/proxy.ts`, `src/auth*.ts`, `CONVENTIONS.md`. If a primitive lacks something you need, compose around it locally and mention it in your report.
- DON'T use `next/image` for external picsum/dicebear URLs (use plain `<img>` with `loading="lazy"` and an `onError` fallback or a plain background). ESLint rule `@next/next/no-img-element` may be disabled per line with a comment.
- DON'T call `Date.now()`/`new Date()` for range logic in components — ranges come from `getRange()`.
- The dev server runs at `http://localhost:3100` (already started; do not start another). Verify visually only if you have browser tools; otherwise rely on typecheck + reasoning.
