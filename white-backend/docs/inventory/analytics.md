## Behavioural inventory: white-analytics backend (for the Go port)

Paths below are relative to `/Users/syabanf/Documents/works/WHITE-Superapps-main/white-analytics/`.

**Conventions that apply everywhere:**
- **Dates.** Every "day" is a UTC-midnight `Date`, stored as Prisma `@db.Date`.
- **Caching.** Query functions are wrapped in React `cache()`, which only de-duplicates within one request.
- **Auth on queries.** Query functions do no auth checks. The pages that call them do: `requireClientAccess(slug)`, then `getRange(searchParams)`.
- **Action result shape.** Server actions return `ActionResult`:
  - success: `{ok:true,data}`
  - failure: `{ok:false,error:string,code?:"RATE_LIMIT"|"TOKEN_EXPIRED"|"PERMISSION"|"NOT_FOUND"|"NETWORK"|"UNKNOWN"|"VALIDATION"|"UNAUTHORIZED",fieldErrors?}`
  - Defined in `src/lib/action-result.ts`.
- **Error messages** (`t.errors`, `src/i18n/id.ts:523`):
  - RATE_LIMIT "Batas permintaan API tercapai. Coba lagi beberapa menit."
  - TOKEN_EXPIRED "Koneksi akun kedaluwarsa. Hubungkan ulang akun."
  - PERMISSION "Izin tidak mencukupi. Periksa akses akun yang dihubungkan."
  - NOT_FOUND "Data tidak ditemukan."
  - NETWORK "Gangguan jaringan. Periksa koneksi dan coba lagi."
  - UNKNOWN "Terjadi kesalahan tak terduga."
  - VALIDATION "Periksa kembali data yang dimasukkan."
  - UNAUTHORIZED "Anda tidak punya akses."
- **`ProviderError`** has `code`, `provider` and `requiresReconnect`; `requiresReconnect` defaults to true for TOKEN_EXPIRED and PERMISSION.

### RBAC (`src/lib/rbac.ts`), used by everything
- **`getSessionUser()`** (:20)
  - Reads the auth() JWT, then re-reads `User` by id.
  - Returns null if the user is missing or `!isActive`.
  - Returns `{id,email,name,role}`.
- **`requireUser()`** (:33): redirects to `/login` when there is no user.
- **`requireClientAccess(slug)`** (:60)
  - Calls requireUser, then `client.findUnique({slug})`; missing client → redirect `/clients?missing=1`.
  - ADMIN → `{client, role:"ADMIN", canManage:true}`.
  - Otherwise looks up `clientMember` by `userId_clientId`; none → redirect `/?unauthorized=1`.
  - `canManage = membership.role==="MANAGER"`. Member roles are MANAGER and VIEWER.
- **Failure is a redirect (HTTP 307/303), not a JSON error.** This applies to pages, to actions that use requireClientAccess or requireUser, and to the PDF route.

---

### src/lib/dates.ts
- **`PresetKey`**: `"7d"|"14d"|"28d"|"30d"|"90d"|"thisMonth"|"lastMonth"|"12m"`. `DEFAULT_PRESET="28d"` (:21).
- **Basic helpers**
  - `utcDate(y,m0,d)=Date.UTC`.
  - `todayUtc(now)` = UTC midnight of now.
  - `addDays` works via UTC components, so it rolls over months.
  - `startOfMonthUtc`; `endOfMonthUtc = utcDate(y, m+1, 0)`.
- **`dayCount(r)`** = `Math.round((to-from)/86_400_000)+1` (inclusive).
- **`latestCompleteDay(now)`** = `todayUtc(now) - 1 day`. Every preset ends here.
- **`rangeForPreset`** (:55), with `end = yesterday`:
  - 7d: `[end-6, end]`
  - 14d: `[end-13, end]`
  - 28d: `[end-27, end]`
  - 30d: `[end-29, end]`
  - 90d: `[end-89, end]`
  - thisMonth: from = 1st of the current month; to = `end < startOfMonth ? today : end`. So on the 1st of a month the range is today→today.
  - lastMonth: 1st → last day of the previous month.
  - 12m: from = (end with year−1) + 1 day; to = end.
- **`previousRange(r)`** (:83): `n = dayCount(r)`, `to = r.from - 1`, `from = to - (n-1)`. Always the same length, immediately before.
- **`toISODate`** = `toISOString().slice(0,10)`.
- **`parseISODate(s)`**
  - Strict regex `^\d{4}-\d{2}-\d{2}$`, then `utcDate(y, m-1, d)`.
  - Does NOT reject overflow: 2026-02-31 becomes Mar 3. It only rejects NaN.
- **`resolveRange(sp, now)`** (:124)
  - Every param takes the first array element when repeated.
  - `compare`: absent → true; otherwise true unless the value is exactly `"0"` or `"false"`.
  - If `from` and `to` both parse and `from <= to`, the range is `{from,to}`, `preset:null`, previous = `previousRange`.
  - Otherwise uses `preset` if it is a valid key, else `"28d"`.
  - Returns `{range, previous, preset, compare}`.
  - **Timezone:** there is no timezone handling anywhere. Everything is UTC, and "today" is the UTC date.
- **Other helpers**
  - `eachDay` gives inclusive days.
  - `dayKey = toISODate`.
  - `tickEvery(n)`: ≤14→1, ≤31→3, ≤62→7, ≤120→14, else 30. UI only.

### src/lib/range-params.ts
- `getRange(searchParams)` (:7) = `resolveRange(searchParams)` with `now = new Date()`. Nothing else.

### src/lib/format.ts (rules that reach API/PDF output)
- **Locale:** `id-ID`. Thousands separator ".", decimal ",".
- **`formatNumber(v, digits=0)`**
  - null or NaN → "–" (en dash).
  - min and max fraction digits are both `digits`.
- **`formatCompact(v, digits=1)`**
  - ≥1e9 → "x M"; ≥1e6 → "x jt"; ≥1e3 → "x rb".
  - Uses `toFixed(digits)`, trims trailing zeros, then swaps "." for ",".
  - The unit is joined with a NBSP (U+00A0).
  - Below 1000: `formatNumber(abs, isInteger?0:digits)`.
  - Negatives get a "-" prefix.
- **`formatCurrency(v, cur="IDR", {compact})`**
  - compact → `"Rp " + formatCompact(v)`; non-IDR currencies use the currency code instead of "Rp".
  - Otherwise `Intl` currency style, 0 decimals for IDR and 2 for others, with NBSP replaced by a normal space.
- **`formatPercent(v, digits=2)`** = `formatNumber(v, digits)+"%"`. The input is already 0–100.
- **`formatDeltaPercent(v, digits=1)`**
  - Sign is "+" or "−" (U+2212); nothing for 0.
  - "–" when null, NaN or non-finite.
- **`formatDeltaNumber(v, digits=0)`**: same sign rules.
- **Date formatting** (all `timeZone:"UTC"`):
  - `formatDate` "19 Agu 2026"
  - `formatDateShort` "19 Agu"
  - `formatDateLong` "19 Agustus 2026"
- **`formatDateTime(d, tz="Asia/Jakarta")`** → day, short month, year, 2-digit hour:minute (e.g. "19 Agu 2026, 14.05"). This is the only place Jakarta time is used.
- **`formatDateRange(from,to)`**: same year → `"<short> – <full>"`, otherwise `"<full> – <full>"`.
- **`truncate(text, max=60)`**: if `len > max`, returns `slice(0,max-1).trimEnd()+"…"`.
- **`slugify`**: lowercase, NFKD, strip U+0300–U+036F, drop `[^a-z0-9\s-]`, trim, whitespace → "-", collapse repeated "-".

### src/lib/metrics/delta.ts
- **`computeDelta(current, previous)`** (:15)
  - `prev = previous ?? 0`.
  - `abs = current - prev`.
  - `pct = prev===0 ? null : abs/|prev|*100`.
  - `direction = |abs|<1e-9 ? "flat" : abs>0 ? "up" : "down"`.
  - Result type: `Delta {abs:number, pct:number|null, direction}`.
  - A previous value of 0 or null always gives `pct:null`; the UI and PDF then hide the badge.
- **`isGoodChange(delta, lowerIsBetter=false)`** (:26): flat → null; otherwise up = good, inverted when lowerIsBetter.
- **Where lowerIsBetter is used:** PDF position and CPR; report sections rank avgPosition. CPC, position and LCP are also lower-is-better by convention.
- **Helpers**
  - `sumBy` treats null as 0.
  - `mean([])=0`.
  - `safeDiv(n, 0)=0`.
  - `weightedMean(pairs)` = Σv·w/Σw, 0 when Σw=0.
  - `clamp`.
  - `round(v, d=2)=Math.round(v*10^d)/10^d`.

### src/lib/metrics/ads.ts
- **`AdKpis`** fields: spend, impressions, reach, clicks, linkClicks, results, cpr, ctr, cpc, cpm, frequency, `roas:number|null`.
- **Formulas**
  - `cpr = safeDiv(spend, results)`
  - `ctr = safeDiv(linkClicks, impressions)*100` (link clicks, not results)
  - `cpc = safeDiv(spend, linkClicks)`
  - `cpm = safeDiv(spend, impressions)*1000`
  - `frequency = safeDiv(impressions, reach)`
  - `roas = purchaseValue==null || spend===0 ? null : purchaseValue/spend`
- **`aggregateAds(rows)`** (:70)
  - Sums every field. Reach is summed across days (an approximation).
  - roas is non-null only if some row has a non-null purchaseValue.
- **`FREQUENCY_FATIGUE_THRESHOLD = 3.0`**; `isFatigued = freq > 3`.

### src/lib/metrics/seo.ts
- **`ctr(c, i) = safeDiv(c, i)*100`**.
- **`aggregateSearch(rows)`** = `{clicks:Σ, impressions:Σ, ctr:ctr(Σc,Σi), position:weightedMean([position, impressions])}`. CTR is recomputed, never averaged.
- **`positionBucket`**: ≤3 "1-3", ≤10 "4-10", ≤20 "11-20", else "21+".
- **`positionDistribution`**: per bucket → `{queries:count, clicks, impressions}`.
- **`EXPECTED_CTR_BY_POSITION`** = {1:28, 2:15, 3:11, 4:8, 5:7, 6:5, 7:4, 8:3, 9:3, 10:2.5}.
  - `expectedCtr(p)`: `p=max(1, round(p))`; ≤10 → table; ≤20 → 1.5; else 1.
- **`DEFAULT_OPPORTUNITY_RULE`** = `{minImpressions:100, strikingMin:4, strikingMax:15, lowCtrFraction:0.6, decliningDrop:0.3, risingGrowth:0.3}`.
- **Opportunity lists** (each sorted by potentialClicks descending):
  - **strikingDistance:** `impressions>=100 && 4<=position<=15`; `expectedCtr = expectedCtr(3) = 11`; `potentialClicks = max(0, round(impr*11/100 - clicks))`.
  - **lowCtr:** `impr>=100 && position<=10 && ctr < expectedCtr(pos)*0.6`; `potential = max(0, round(impr*exp/100 - clicks))`.
  - **declining:** `prevClicks>0 && impr>=100 && (prev-clicks)/prev >= 0.3`; `potential = max(0, prev-clicks)`.
  - **rising:** `prevClicks>0 && impr>=100 && (clicks-prev)/prev >= 0.3`; `potential = clicks-prev`.
- **`CWV_THRESHOLDS`** (good / poor):
  - lcpMs 2500 / 4000
  - inpMs 200 / 500
  - cls 0.1 / 0.25
  - fcpMs 1800 / 3000
  - ttfbMs 800 / 1800
  - tbtMs 200 / 600
  - `rateCwv`: value ≤ good → good; ≤ poor → needs-improvement; else poor; null → null.
- **`rateScore`**: ≥90 good, ≥50 needs-improvement, else poor.
- **`healthScore(scores, {errors, warnings})`** (:233)
  - null scores → null.
  - `base = perf*0.3 + seo*0.4 + a11y*0.15 + bp*0.15`.
  - `penalty = clamp(errors*2, 0, 20) + clamp(warnings*0.5, 0, 10)`.
  - Result: `round(clamp(base-penalty, 0, 100))`.

### src/lib/metrics/social.ts
- **`postEngagements`** = likes + comments + shares + saves (missing values count as 0).
- **`postEngagementRate(p, followers)`** = `safeDiv(eng, followers)*100`.
- **`postEngagementRateByReach`**: null if reach is falsy.
- **`avgEngagementRate(posts, followers)`**: 0 if there are no posts or followers ≤0; otherwise the mean of per-post ER.
- **`followerGrowth(start, end)`** = `{abs:end-start, pct: start===0 ? null : abs/start*100}`.
- **`postsPerWeek(n, days)`** = days≤0 ? 0 : n/days*7.
- **`contentType`**: productType REELS → "Reels"; STORY → "Story"; mediaType VIDEO → "Video"; CAROUSEL_ALBUM → "Carousel"; else "Foto".
- **`postingHeatmap(posts, tz="Asia/Jakarta")`**
  - 7×24 cells; Monday = 0; hour "24" maps to 0.
  - Each cell is `{day, hour, posts, avgEngagement = Σeng/n or 0}`.
- **`topPosts(posts, n=5)`**: sorted by engagements descending.

### src/lib/metrics/series.ts
- **`fillDaily(range, rows, getDate, fields, {missing=0|null})`**
  - Sums rows per day key.
  - Each day in range gets the summed value, or `missing` when there is no row.
  - **SEO uses `missing:null`.** Days GSC did not report stay null; they must never become 0.
- **`sparkline(points, field, n=14)`**: last 14 points; any non-number (including null) becomes **0**. So SEO clicks sparks DO turn null into 0. Keep this if you want parity, or flag it.
- **`alignPrevious(cur, prev, field, out=field+"Prev")`**: pairs by index; null when the previous series has no point at that index.
- **`rollingMean`**; **`weekKey`** = Monday-start ISO date.

### src/lib/metrics/seo-suite.ts and publishing.ts
These are not re-exported from the metrics index.
- **`visibility`** = mean of expectedCtr(p), counting null or >100 as 0.
- **`avgPosition`**: mean over positions ≤100, null when there are none. `countTop(positions, n)`.
- **`rankBucket`**: "1-3", "4-10", "11-20", "21-50", "51-100"; "unranked" when null, >100 or <1.
- **`classifyRankChange(prev, cur)`**
  - Positions >100 count as null.
  - Both null → none; prev null → new; cur null → lost.
  - `delta = prev - cur`: >0 up, <0 down, else flat.
- **`isRankDropAlert`**: lost, or down with `cur-prev > 5`, or `prev<=10 && cur>10`.
- **`positionToScore`** = `101 - clamp(p, 1, 100)`, 0 when unranked.
- **`shareOfVoice(lists, n=8, "Lainnya")`**: weight = expectedCtr(position), share = weight/total*100.
- **`difficultyBand`**: <30 easy, <60 medium, <80 hard, else very_hard.
- **`RESEARCH_MAX_AGE_DAYS = 7`**.
- **`keywordGap`**: matches keywords lowercased.
  - Competitor-only keywords → missing.
  - `mine.position > comp` → weak, else strong.
  - Every keyword both rank for → shared.
- **Backlinks**
  - `TOXIC_SPAM_SCORE = 60`.
  - Views: "new" = firstSeen ≤ 30 days ago and not lost; "toxic" = not lost and spam ≥ 60.
  - `dofollowShare` = df/(df+nf)*100, 0 when the total is 0.
- **Domains**
  - `normalizeDomain` strips `sc-domain:`, scheme, credentials, path, port and `www.`.
  - `MAX_COMPETITOR_DOMAINS = 5`.
- **publishing.ts**
  - `bestPostingTimes`: score = avgEng·ln(1+posts); minPosts 2; n 3; defaults are Mon 11, Wed 19, Fri 12.
  - `postStatusCounts`.
  - `failureRate` = FAILED/(PUBLISHED+FAILED)*100, null when that denominator is 0.

---

### src/features/overview/queries.ts
- **`DataQuality`** = `{state, lastSyncedAt:ISO|null, dataFrom:ISO|null, dataTo:ISO|null}`.
- **`quality(hasSource, isDemo, dates, latestSync)`** (:76). Rules are checked in this order:
  1. `!hasSource` → `"not_connected"`
  2. `latestSync?.status==="FAILED"` → `"failed"`
  3. `isDemo` → `"demo"`
  4. `dates.length===0` → `"empty"`
  5. `lastSyncedAt && Date.now()-lastSyncedAt > 48*60*60*1000` → `"stale"`
  6. otherwise `"healthy"`
  - **Inputs:** `lastSyncedAt` is the latest SyncJob's `startedAt`, not finishedAt. With no sync job, there is no stale check, so the result is healthy.
  - `dataFrom` / `dataTo` are the minimum and maximum of the current-range dates.
- **Demo / mock meaning:** `isDemo = every source row has connectionId == null`. That covers seed data, CSV import and Apify-only data.
  - When there are no source rows at all, `isDemo:true` and the state is `not_connected`.
  - Pages hide deltas when `isDemo` is true (`compare && !isDemo`). The PDF only checks `compare`.
- **`NON_CONVERSION_RESULT_TYPES`** = {"reach", "impressions", "video_view", "thruplay"} (:99).

- **`getOverviewSocial(clientId, range, previous)`** (:111) → `OverviewSocial`
  - **Return shape:** `{hasSource, hasData, isDemo, quality, followers, followersDelta, followersSpark:number[], posts, postsDelta, engagementRate, engagementRateDelta, reach, reachDelta, accounts:[{platform, username, followers}]}`.
  - **Accounts:** `socialAccount where {clientId, isCompetitor:false}`. If there are none: zero deltas (`computeDelta(0,0)` = `{abs:0, pct:null, flat}`), `hasSource:false`, `isDemo:true`.
  - **Queries, run in parallel:**
    - Snapshots: `socialSnapshot where socialAccountId IN ids AND date ∈ [previous.from-1, range.to]`, ordered by date asc. Selects followers and reach.
    - Posts: `socialPost publishedAt ∈ [range.from, range.to+1)`.
    - Previous posts: same filter over `[previous.from, previous.to+1)`.
    - Latest sync: `syncJob kind="SOCIAL_SNAPSHOT"`, latest by startedAt.
  - **followersAt(d):** per account, the last snapshot with `date ≤ d`, summed across accounts. An account with no snapshot yet contributes nothing.
    - `followersNow = followersAt(range.to)`
    - `start = followersAt(range.from-1)`
    - `prevStart = followersAt(previous.from-1)`
    - `followersDelta = computeDelta(now-start, start-prevStart)`. This compares growth with previous growth, not totals.
  - **followersSpark:** walks snapshots in order, carrying the per-account value forward. For each snapshot date ≥ range.from it records the running sum keyed by date, then takes the last 14 values. Days without a snapshot are absent, not filled.
  - **Reach:** Σ `snapshot.reach` (null as 0) over `[range.from, range.to]`; prevReach over `[previous.from, previous.to]`.
  - **Engagement rate:**
    - `followersByAcc` = the account's last snapshot in the fetched window (≤ range.to).
    - `erFor(rows)` = Σ over accounts of `avgEngagementRate(accountPosts, followersByAcc)*accountPosts.length`, divided by rows.length. Effectively a mean weighted by post count; 0 when there are no posts.
    - The previous ER uses the same `followersByAcc` (current followers).
  - **Other fields:**
    - `hasData` = current-range snapshot dates + post publishedAt dates, non-empty.
    - `quality(true, isDemo, thoseDates, latestJob)`.
    - `postsDelta = computeDelta(posts.length, prevPosts.length)`.

- **`getOverviewSeo(clientId, range, previous)`** (:252) → `OverviewSeo`
  - **Return shape:** `{hasSource, hasData, isDemo, quality, clicks, clicksDelta, clicksSpark, impressions, impressionsDelta, ctr, ctrDelta, position, positionDelta, health:number|null, daily:[{date, clicks:number|null, impressions:number|null}]}`.
  - **Properties:** `seoProperty where clientId`. If none: zeros, `health:null`, `daily:[]`.
  - **Queries, run in parallel:**
    - `seoDailyMetric` for the current range (date asc) and for the previous range.
    - `seoAudit where strategy="MOBILE"`, latest by runAt.
    - `seoCrawl where status="SUCCESS"`, latest by startedAt, including issue severities.
    - `syncJob kind IN ("SEO_GSC", "SEO_GA4")`, latest.
  - **KPIs:** `aggregateSearch` for current and previous; every delta is `computeDelta(cur, prev)`. Position is lower-is-better, applied downstream.
  - **daily:** `fillDaily(..., ["clicks","impressions"], {missing:null})`, so null stays null.
  - **clicksSpark:** `sparkline(daily, "clicks")`. Nulls become 0 here.
  - **health:** `latestAudit ? healthScore(audit, {errors, warnings} counted from the latest successful crawl's issues) : null`.
  - **hasData** = current rows OR previous rows present. **quality** uses current-row dates only.

- **`getOverviewAds(clientId, range, previous)`** (:337) → `OverviewAds`
  - **Return shape:** `{hasSource, hasData, isDemo, quality, kpis:AdKpis, prev:AdKpis, spendDelta, resultsDelta, cprDelta, spendSpark, daily:[{date, spend, results}]}`.
  - **Rows:** `adDailyInsight where adAccountId IN ids AND adId != null AND date ∈ range` (date asc), and the same for the previous range. Campaign-level rows (adId null) are excluded.
  - **Latest sync:** `syncJob kind IN ("ADS_INSIGHTS", "ADS_CSV_IMPORT")`.
  - **`withConv`:** all KPIs come from all rows, but `results`, `cpr` and `roas` are replaced by the values from conversion rows only (resultType not in NON_CONVERSION; a null resultType counts as conversion).
  - **daily:** `fillDaily(range, rows, ["spend","results"], missing 0)`. If any conversion rows exist, non-conversion rows have results set to 0 first.
  - **Deltas:** spend, results and cpr via computeDelta.

- **`getRecentSyncJobs(clientId, take=6)`** (:419): `syncJob where clientId`, startedAt desc, take 6 → `{id, kind, status, startedAt, message, error}`.

---

### src/features/social/queries.ts
- **`toPostRow(p, followers, username?)`**
  - `captionShort = truncate(caption, 90)`.
  - `type = contentType`.
  - `publishedAt` as an ISO string.
  - `engagements`.
  - `er = round(postEngagementRate(p, followers), 2)`.
- **`getSocialAccounts(clientId)`** (:167)
  - Accounts where clientId, ordered by `[isCompetitor asc, platform asc, createdAt asc]`.
  - Latest snapshot per account via `distinct socialAccountId`, ordered `[socialAccountId asc, date desc]`.
  - Returns `{id, platform, username, displayName, avatarUrl, isCompetitor, followers (0 if none)}[]`.
- **`getSocialDashboard(clientId, platform, range, previous)`** (:192) → `SocialDashboard`
  - **Account:** `socialAccount.findFirst({clientId, platform, isCompetitor:false})`. If none: `{isDemo:true, account:null, kpis:null, all arrays []}`.
  - **Queries:**
    - Snapshots for the account in `[previous.from-1, range.to]`, asc (followers, reach, impressions).
    - Posts in range, publishedAt desc.
    - Previous posts.
  - **Followers and growth:**
    - followersAt is a scan with break on `date > d`; 0 before the first snapshot.
    - `growth = followerGrowth(start, now)`; `prevGrowthAbs = start - prevStart`.
  - **Series:**
    - `dailyFollowerSeries(range, snaps)` (lib.ts:45): followers forward-filled from the last snapshot before range.from (null if none); reach and impressions per day, 0 if missing.
    - growthSeries = `alignPrevious(cur, prev, "followers")` → `{date, followers|null, followersPrev|null}`.
    - reachSeries = `{date, reach, impressions}`.
    - followersSpark = last 14 non-null followers values.
  - **KPIs** (`SocialKpis`):
    - followers = now; `followersDelta = computeDelta(growth.abs, prevGrowthAbs)`; growthAbs; growthPct.
    - `er = avgEngagementRate(posts, now)`; `prevEr = avgEngagementRate(prevPosts, start || now)`; erDelta.
    - reach and impressions: Σ over snapshots (null as 0) in range vs previous; deltas.
    - posts and postsDelta.
    - `postsPerWeek(posts.length, dayCount(range))`.
  - **Other output:**
    - `isDemo = account.connectionId == null`.
    - `erByType` = erByContentType (avg ER per type, sorted desc, with count).
    - `heatmap = postingHeatmap(posts)` (Asia/Jakarta).
    - `topPosts = topPosts(posts, 6)`.
    - `posts` = all posts in range.
- **`getCompetitorDashboard(clientId, range, previous)`** (:326) → `{hasOwn, rows:CompetitorRow[], chart:{data, series}, topPosts}`
  - **Accounts:** all INSTAGRAM accounts for the client, createdAt asc. If none: empty.
  - **Per account:** followers, start, prevStart; growth = followerGrowth; `growthDelta = computeDelta(growth.abs, start-prevStart)`; `avgEr` over posts in range against current followers; postsPerWeek; mediaCount from the last snapshot ≤ range.to.
  - **Row order:** own row first, then competitors sorted by followers desc.
  - **Chart:**
    - Uses the first 4 rows (own + top 3), key `a{i}`, label `@username`.
    - Values = `indexTo100(dailyFollowerSeries.followers)` rounded to 1 decimal.
    - indexTo100 bases on the first value >0; null or ≤0 → null; if there is no base, all values are null.
  - **topPosts:** top 6 competitor-only posts by engagements, ER against that account's followers, with username.

### src/features/social/lib.ts
- `PLATFORMS = ["INSTAGRAM","FACEBOOK","TIKTOK"]`.
- `parsePlatform`: uppercases; FACEBOOK and TIKTOK accepted, otherwise INSTAGRAM.
- `normalizeIgUsername`: trim, strip leading "@", lowercase; length 2–30; `^[a-z0-9._]+$`; must not start or end with "."; invalid → null.

### src/features/social/actions.ts ("use server")
- **`clientAccess(clientId)`** (:22)
  - Uses getSessionUser; returns null when there is no user, no client, or no membership.
  - ADMIN → canManage true; otherwise `canManage = role==="MANAGER"`.
  - Does not redirect.
- **`revalidateSocial(slug)`**: revalidates `/clients/{slug}/social` and `/clients/{slug}/social/competitors`.
- **`addCompetitor({clientId, username})`** (:48)
  - **Validation:** zod `{clientId: string.min(1), username: string.min(1).max(64)}` → fail(VALIDATION). Then normalizeIgUsername, null → fail("Username tidak valid. Gunakan 2–30 karakter huruf, angka, titik, atau garis bawah.", VALIDATION).
  - **Auth:** `!access?.canManage` → fail(UNAUTHORIZED).
  - **Duplicate check:** `socialAccount {clientId, INSTAGRAM, username, isCompetitor:true}` exists → fail("Kompetitor dengan username ini sudah ada.", VALIDATION).
  - **Profile fetch:**
    - If Apify is configured: `publicSocial.profile` + `publicSocial.posts(..., 20)`.
    - Otherwise `metaGraph.discoverBusiness("", ownIG.externalId ?? "", username)`, which is a deterministic mock without credentials.
  - **Creates the account:** `{externalId:"ig_bd_"+username, username, displayName, biography, avatarUrl, isCompetitor:true}`.
  - **Synthetic snapshots:** createMany(skipDuplicates) of 120 days ending at latestCompleteDay.
    - followers = `buildFollowerSeries("bd:"+username, 120, followersCount)`.
    - `mediaCount = max(0, mediaCount - round((119-i)*0.3))`.
  - **Posts:** createMany(skipDuplicates) of the first 20 media items (likes and comments only).
  - **Returns:** ok `{accountId, username}`.
  - **Errors:** ProviderError → `fail(t.errors[code], code)`; anything else → fail(UNKNOWN).
  - No limit on the number of competitors.
- **`removeCompetitor(accountId)`** (:155)
  - Empty or non-string id → VALIDATION.
  - Account missing or not a competitor → fail(NOT_FOUND, "NOT_FOUND").
  - `!canManage` → UNAUTHORIZED.
  - Deletes the account; snapshots and posts cascade. Revalidates. Returns ok(null).
- **`syncSocial({clientId, platform})`** (:182)
  - **Validation:** zod `platform` enum PLATFORMS.
  - **Auth:** any access (VIEWER allowed); none → UNAUTHORIZED.
  - **Accounts:** own accounts for the platform; none → fail("Tidak ada akun untuk disinkronkan.", NOT_FOUND).
  - **Step 1, Meta:** if platform≠TIKTOK and some account has a connectionId, run syncSocialMeta.
    - If `updated>0 || failed>0`: create a SyncJob SOCIAL_SNAPSHOT.
      - status FAILED if updated===0, else SUCCESS.
      - message `"{n} akun diperbarui, {p} post diperbarui ({platform lowercase} · Meta)"`.
      - error = `"@u: err; ..."` sliced to 1000 chars.
    - Revalidate. If updated==0, fail(first error); else ok `{updated}`.
    - If every account was skipped, falls through to step 2.
  - **Step 2, Apify:** if configured, syncSocialPublic with includeCompetitors:true.
    - SyncJob status FAILED only if `failed.length && updated===0`.
    - message suffix `"· via Apify"`.
    - Same return logic as step 1.
  - **Step 3, demo:** for each own account:
    - Take the last 8 snapshots (date desc).
    - avgGain = mean of consecutive diffs, or `max(1, followers*0.001)` when there are none.
    - `rand = mulberry32(hashSeed("sync:{accId}:{YYYY-MM-DD today}"))`.
    - `followers = max(0, round(last + avgGain*(0.7+rand()*0.6)))`.
    - reach, impressions, profileViews and websiteClicks each = `max(0, round(v*(0.85+rand()*0.3)))`, null kept as null.
    - `mediaCount + (rand()<0.3 ? 1 : 0)`; following is copied.
    - Upserts the snapshot `(accId, todayUtc)`.
    - Then SyncJob SUCCESS with message `"{n} akun diperbarui ({platform})"`.
  - **Catch:** creates a FAILED SyncJob (error = message) and returns fail(message, code?).
- **Note:** `syncSocialPublic` and `syncSocialMeta` (social/sync.ts) do the provider writes.
  - **Public (Apify):** upserts today's snapshot (followers, following, mediaCount only), updates the account profile, and upserts posts by `(socialAccountId, externalId)`. Post limit defaults to 20 for competitors and 30 for own accounts.
  - **Meta:** fetches `days = 30` back from latestCompleteDay.
    - Instagram: per-day insight snapshot upserts (reach, impressions, profileViews; creates use the current followers count), today's snapshot upsert, and 50 media with insights.
    - Facebook: page reach = impressionsUnique, posts with mediaType "IMAGE".
    - Accounts without a token are counted as skipped.

### src/features/ads/queries.ts and helpers
- **`getAdAccounts(clientId)`** (:36): ordered by createdAt asc → `{id, externalId, name, currency, hasConnection: connectionId!=null, lastSyncedAt: ISO|null}`.
- **`getAdsDashboard(clientId, range, previous, campaignId?)`** (:107) → `AdsDashboard`
  - **No accounts:** returns `emptyDashboard()`: `isDemo:true`, `deltas.roas:null`, `demographics {hasData:false, ...}`.
  - **Campaigns:** `adCampaign IN accounts`, ordered by name asc. `filterId` = campaignId only if it matches a known campaign; otherwise the filter is silently ignored. The page reads it from `?campaign=`.
  - **Queries, run in parallel:**
    - Insight rows: `adId!=null AND date ∈ range (+campaignId filter)`, asc. Same for the previous range.
    - Demographics: `adDemographic where dateFrom <= range.to AND dateTo >= range.from (+campaignId)`. Overlapping windows; not prorated.
    - adSets and ads, scoped to the campaign when filtered.
  - **KPIs:** `kpis` and `prev` = kpisWithConversion (queries-insights.ts:26): all-row aggregate with conversion-only results, cpr and roas.
  - **daily:** same zeroing rule as the overview; previous series via alignPrevious → `{date, spend, results, spendPrev|null, resultsPrev|null}`.
  - **Deltas:** spend, results, cpr, impressions, reach, linkClicks, ctr, cpc, cpm and frequency via computeDelta. `roas` is computed only if both ROAS values are non-null, else null.
  - **dominantResultType** (queries-insights.ts:33): the resultType with the largest Σresults among conversion rows; falls back to all rows; null if none. Ties keep the first one reached.
  - **Sparks:** spendSpark and resultsSpark are the last 14 days.
  - **breakdown** (queries-breakdown.ts:104): campaign → adSet → ad tree.
    - Each node's KPIs = aggregateAds over all its rows (NOT conversion-filtered).
    - `resultType` = the first non-null in row order.
    - Every level is sorted by spend desc.
  - **campaignShare:** campaigns with spend>0 → `{id, name, spend, results, resultType}`.
  - **topAds** (:144):
    - Per ad, conversion rows only; drop ads with results ≤ 0.
    - `med` = median of results.
    - Keep `results >= med || results >= 5`, sort by cpr asc, take 6.
    - Missing metadata falls back to: name = adId, campaignName "–", status "ACTIVE".
  - **demographics** (queries-demographics.ts:17):
    - byAge, byGender and the age|gender matrix, each summing spend and results.
    - AGE_ORDER `["13-17","18-24","25-34","35-44","45-54","55-64","65+","unknown"]`; GENDER_ORDER `["female","male","unknown"]`; unknown values sort last.
  - **Other:** `hasData = rows||prevRows`; `isDemo = every account lacks a connection`; `account = accounts[0]`; `campaignOptions` = all campaigns.

### src/features/ads/actions.ts ("use server")
- **`requireClientManage(clientId)`** (:17): requireUser (redirects); returns the client for ADMIN or MANAGER, otherwise null.
- **`importAdsCsv(clientId, formData)`** (:31)
  - **Validation:** zod clientId min(1) → VALIDATION; not authorized → UNAUTHORIZED.
  - **File checks:**
    - `file` must be a File with size>0 → else fail("Pilih file CSV terlebih dahulu.", VALIDATION).
    - Size > 10*1024*1024 → "Ukuran file melebihi 10MB".
    - Name not `.csv`, type not containing "csv", and type ≠ text/plain → "File harus berformat .csv".
  - **Parse:** `parseAdsCsv`. 0 rows → fail(warnings[0] ?? "Tidak ada baris data yang bisa dibaca dari file.", VALIDATION).
  - **Persist:** `persistCsvRows`, then SyncJob ADS_CSV_IMPORT SUCCESS with message `"{c} kampanye, {s} set iklan, {a} iklan, {d} hari data, {g} baris demografi."`.
  - Revalidates `/clients/{slug}/ads` and `/clients/{slug}`. Returns ok(ImportSummary `{campaigns, adSets, ads, days, demographics, warnings ≤8}`).
  - On error: FAILED job with the error message; fail(UNKNOWN).
- **`syncAds(clientId)`** (:90)
  - Manage only. No accounts → fail("Belum ada akun iklan untuk disinkronkan.", VALIDATION).
  - **Real path:** if any account has a connection, `syncAdsForClient(days:90)` and revalidate.
    - `real===0` and an error exists → fail(error).
    - `real>0` → ok `{lastSyncedAt: now ISO}`.
    - Otherwise falls through to the demo path.
  - **Demo path:**
    - Sets `lastSyncedAt=now` on all accounts.
    - SyncJob ADS_INSIGHTS SUCCESS, message "Sinkronisasi demo: {campaignCount} kampanye diperbarui.".
    - ok. On error: FAILED job and fail(UNKNOWN).
- **ads/sync.ts**
  - **`syncAdAccount`:** no token → `{mode:"demo"}` and no writes.
    - Window: `[latestCompleteDay-(days-1), latestCompleteDay]`.
    - Upserts campaigns, adSets and ads by `(parent, externalId)`.
    - Daily ad-level insights: find existing by date|campaign|adSet|ad, then update or create in 100-row transactions; rows without an ad are skipped.
    - Campaign-level age/gender demographics for the whole window, keyed `(account, since, until, campaign, age, gender)`.
    - Sets `lastSyncedAt`.
  - **`syncAdsForClient`:** one SyncJob ADS_INSIGHTS per account.
    - status FAILED when there is an error.
    - message "mock" in demo mode, otherwise `"{c} kampanye, {a} iklan, {d} hari · {externalId}"`.
  - **sync-map:** gender is lowercased; empty age or gender → "unknown".
- **csv-import.ts `parseAdsCsv`**
  - **Parsing:** Papa `header:true, skipEmptyLines:"greedy"`; BOM stripped.
  - **Column aliases (EN/ID):**
    - Required: campaignName and spend; missing → warning "Kolom wajib tidak ditemukan: …" and no rows.
    - "amount spent" and "purchases conversion value" match by prefix.
  - **Number parsing** (`parseAdsNumber`):
    - "--", "-", "–" → null.
    - When both "." and "," appear, the last separator is the decimal point.
    - Comma only: `^\d{1,3}(,\d{3})+$` means thousands; a single comma means decimal; otherwise null.
    - Dot only: `^\d{1,3}(\.\d{3})+$` means thousands; more than one dot otherwise means null.
  - **Date parsing:**
    - `YYYY-M-D` prefix.
    - `D/M/YYYY` (or ".") is read as DD/MM unless `b>12 && a<=12`, then MM/DD.
    - Dates are validated as real calendar dates.
  - **Gender:** male/pria/laki-laki/m and female/wanita/perempuan/f; all/semua → null; anything else → unknown.
  - **Age:** all/semua → null; unknown/uncategorized/tidak diketahui → "unknown".
  - **Row kind:** "demographic" if age or gender is set, else "daily". Daily rows without a date are skipped.
  - **resultType normalisation** (substring match, in this order): purchase, messaging_conversation_started, lead, link_click, thruplay, video_view, reach, impressions; else the last `:` / `.` segment with spaces replaced by "_".
  - Integer metrics use `Math.round`. At most 8 warnings.
- **csv-persist.ts `persistCsvRows`**
  - **Account:** first AdAccount (createdAt asc), or create `{externalId:"csv_"+clientId, name:"Impor CSV", currency:"IDR"}`.
  - **Campaigns:** key = campaignExternalId ?? slugify(name). On create, objective comes from resultType: purchase→OUTCOME_SALES, lead→OUTCOME_LEADS, messaging→OUTCOME_ENGAGEMENT, reach/impressions/video_view/thruplay→OUTCOME_AWARENESS, else OUTCOME_TRAFFIC.
  - **Ad sets:** key = campKey|adSetExtId ?? slug. An ad without an ad set goes to the fallback set "(Impor CSV)", slug of `"{campaign} (Impor CSV)"`.
  - **Daily rows:** aggregated per date|campaign|set|ad; createMany in chunks of 500; updates in transactions of 100.
  - **Demographics:** aggregated over the file's min..max date (today if there are no dates).
  - Sets `lastSyncedAt`. Returns `days` = number of distinct dates.

### src/features/seo/queries.ts
- **`getSeoProperty(clientId)`** (:24): first by createdAt asc → `{id, siteUrl, ga4PropertyId, connectionId, auditCadence}|null`.
- **`hasGoogleConnection(clientId)`** (:32): a `connection` exists with provider GOOGLE.
- **`getSeoOverview(propertyId, range, previous)`** (:71) → `SeoOverviewData`
  - **Return shape:**
    - `{hasData, kpis{clicks, impressions, ctr, position}, deltas{clicks, impressions, ctr, position}, sparks{clicks, impressions}}`
    - `daily: SearchDailyPoint[]` = `{date, clicks|null, impressions|null, position|null, clicksPrev|null, impressionsPrev|null, positionPrev|null}`
    - `ga4{hasData, organicSessions, sessions, conversions, organicDelta, conversionsDelta, organicSpark}`
    - `ga4Daily: [{date, organik, lainnya}]`
    - `health`
  - **Reads:**
    - seoDailyMetric for the current range (asc) and previous range.
    - ga4DailyMetric for the current range (asc) and previous range.
    - Latest MOBILE audit.
    - Latest SUCCESS crawl with issues.
  - **Daily series:** `fillDaily(missing:null)` for current and previous; previous is aligned by index. Days GSC did not report stay null. Position per day comes from summing that day's rows (one row per day by the unique key).
  - **GA4:** sums organicSessions, sessions and conversions; deltas for organic and conversions.
    - ga4Daily: `lainnya = max(0, sessions - organic)`. Missing days are coerced to **0** (`?? 0`), not null.
  - **Sparks:** clicks, impressions and organic, with nulls turned into 0.
  - `health` uses the same formula as the overview. `hasData` = current or previous rows present.
- **`dimRows(propertyId, dimension, range)`**: `seoDimensionMetric where {propertyId, dimension, date∈range}` → `{key, clicks, impressions, position}`.
- **`getQueryStats(propertyId, range, previous)`** (:159) → `{rows:QueryAgg[], distribution, opportunities{striking, low_ctr, declining, rising}}`
  - `rows = mergeCurrentPrevious(aggregateDimRows(cur), aggregateDimRows(prev))`.
  - aggregateDimRows: Σclicks, Σimpressions, impression-weighted position, CTR recomputed; sorted by clicks desc then impressions desc.
  - QueryAgg adds `prevClicks|null`, `prevPosition|null`, `prevImpressions|null`; null when the key has no previous data.
- **`getPageStats`** (:173): same as getQueryStats for PAGE, plus `path = urlPath(key)` (pathname+search, "/" if empty, the raw value if it is not a URL).
- **`getDimBreakdown(propertyId, range)`** (:178) → `{devices:DimAgg[], countries:DimAgg[]}`.
- **`getAuditData(propertyId)`** (:278) → `AuditData`
  - **Audits:** latest MOBILE and DESKTOP (serialized; runAt ISO); MOBILE history = `findMany` asc **take 24** (the oldest 24, not the latest).
  - **Crawl:** the latest crawl of any status.
  - **Pages:** ordered by url asc → `{id, url, path, statusCode, titleLength = title?.length ?? 0, metaLength, h1Count, wordCount, imagesMissingAlt, indexable, loadMs}`.
  - **Issues:** ordered by severity asc (enum order ERROR, WARNING, NOTICE), then code asc.
  - **History:** one point per runAt ISO date; the last run of the day wins.
  - **Counts:** `severityCounts {errors, warnings, notices}`; `codeCounts` sorted by count desc.
  - **lastRunAt** = max of the latest mobile and desktop runAt.
- **`getIssueTrend(propertyId, take=30)`** (:357): the last 30 SUCCESS crawls (startedAt desc, then reversed); one point per date (the last crawl of the day wins) → `{date, errors, warnings, notices, pages:pagesCrawled}`.

### src/features/seo/actions.ts ("use server")
- **`guard(clientId, propertyId)`** (:19)
  - No session → fail(UNAUTHORIZED).
  - Non-admin without membership → fail("Anda tidak punya akses ke klien ini.", UNAUTHORIZED). VIEWER is allowed; there is no canManage check.
  - Property must match `{id, clientId}`; otherwise fail("Properti SEO tidak ditemukan.", NOT_FOUND).
- **`runAudit(clientId, propertyId, url?)`** (:40)
  - **Validation:** zod `{clientId min1, propertyId min1, url: string.url().optional()}`.
  - **Throttle:** if the last seoAudit (any strategy) has `runAt` within `AUDIT_COOLDOWN_MS = 10*60*1000` → fail("Audit terakhir dijalankan kurang dari 10 menit lalu. Coba lagi beberapa menit.") with no code.
  - **Run:** `runAuditCore`, then revalidate `/seo/audit` and `/seo`.
  - **Result:**
    - allFailed → fail("Audit gagal dijalankan: " + firstError).
    - Otherwise ok `{message: "Audit selesai" | "Audit selesai sebagian" + ": " + parts.join(" · ") + (err ? " — " + firstErr : ""), audits, pagesCrawled, issues}`.
- **`setAuditCadence({clientId, propertyId, cadence: "NONE"|"WEEKLY"|"DAILY"})`** (:77): guard, then update `seoProperty.auditCadence`, revalidate `/seo/audit`, return ok `{cadence}`.
- **`syncSeo(clientId)`** (:89)
  - Session and membership checks as above.
  - Client missing → NOT_FOUND.
  - Runs `syncSeoForClient(days:90)`; revalidates `/clients/{slug}/seo` ("layout") and `/clients/{slug}`.
  - No properties → fail(auditNoProperty, NOT_FOUND).
  - `results[0].error` → fail(error).
  - Otherwise ok `{mode: results[0].mode, daily: Σ, ga4: Σ}`.
- **audit-core.ts `runAuditCore`** (:40)
  - **Target:** `auditTarget`: override url, else `sc-domain:x` → `https://x/`, else siteUrl.
  - **Setup:** creates SyncJob SEO_AUDIT RUNNING (message = target) and SeoCrawl RUNNING with maxPages 30.
  - **Runs concurrently** (allSettled): PSI mobile, PSI desktop, and `crawlSite(maxPages:30, timeBudgetMs:45_000)`.
  - **Writes:**
    - One SeoAudit per successful PSI (scores, CWV, raw JSON).
    - Crawl pages via createMany, issues via createMany.
    - Crawl updated to SUCCESS with pagesCrawled, or FAILED with the error.
  - **allFailed** = no PSI result AND the crawl was rejected.
  - **Job update:** status FAILED or SUCCESS; message = parts (`"PageSpeed {n}/2 strategi"`, `"Crawl {p} halaman, {i} isu"`) joined by " · "; error = errors joined by " | " sliced to 900 chars.
- **seo/sync.ts**
  - **`syncSeoProperty`:** no token → `{mode:"demo"}` and no writes. Window: last `days` complete days.
    - GSC daily upserts by `(propertyId, date)`.
    - Dimensions query, page, country and device with rowLimit 5000, upserted in transactions of 200.
    - GA4 if `ga4PropertyId` is set (prefixed with "properties/").
  - **`syncSeoForClient`:** SyncJob SEO_GSC per property (message "mock" or `"{d} hari, {n} baris dimensi · {siteUrl}"`), plus SEO_GA4 if a GA4 property is set.
  - **sync-map:**
    - Clicks and impressions use `Math.round`; ctr and position are rounded to 2 decimals.
    - Query keys lowercased, country lowercased, device uppercased.
    - GA4 organic = channel "organic search"; `avgEngagementSec = round2(duration/sessions)`.

---

### src/features/reports/queries.ts
- **`getLatestInsight(clientId, module, range)`** (:13): `aiInsight where {clientId, module, dateFrom=range.from, dateTo=range.to}` (exact match, no language filter), createdAt desc → row|null.
- **`getTopQueryRows(clientId, range, n=5)`** (:25): all properties of the client, QUERY dimension rows in range, grouped by key via aggregateSearch, sorted by clicks desc, top n → `{query, clicks, impressions, ctr, position}`.
- **`getTopPostRows(clientId, range, n=5)`** (:65): posts from own accounts (`account {clientId, isCompetitor:false}`) in range → `{id, platform, username, caption, mediaType, productType, permalink, publishedAt:Date, likes, comments, shares, saves, engagements, reach}`, sorted by engagements desc, top n.
- **`getCampaignRows(clientId, range)`** (:122)
  - Insight rows with `adId!=null AND campaignId!=null`, in range.
  - Grouped per campaign: Σspend, Σimpressions, Σresults; resultType = first non-null.
  - `isConversion = resultType ∉ NON_CONVERSION`; `cpr = isConversion ? safeDiv(spend, results) : 0`.
  - Missing campaign metadata: name "—", objective "", status "ACTIVE".
  - Sorted by spend desc.
- **`getPublishingSummary(clientId, range)`** (:169)
  - Window: `[range.from, range.to+1)`.
  - Published targets: `postTarget status PUBLISHED, publishedAt in window, post.clientId`.
  - Failed count: FAILED targets where `post.scheduledAt` is in the window, OR `scheduledAt` is null and `createdAt` is in the window.
  - Synced posts matched by `externalId IN …` on own accounts.
  - `summarizePublishing` (extras.ts:128):
    - `posts` = distinct postIds.
    - `perPlatform` sorted by count desc, then platform.
    - Stats joined by `socialAccountId:externalId`.
    - mode "engagement" if any match exists, else "recent".
    - Top 5 by engagements desc (ties broken by recency), or by recency.
- **`getSocialDailyFollowers(clientId, range)`** (:210): own accounts' snapshots from `range.from-30` to `range.to`; carried forward per account over each day; output only for days ≥ range.from → `{date, followers:Σ}`.
- **`listReports(clientId)`** (:253): createdAt desc, including createdBy.name → `{id, title, dateFrom, dateTo, modules, language, compare, hasAiSummary: Boolean(aiSummary), createdByName|null, createdAt}`.

### src/features/reports/insights.ts
- **`toRangeInfo`** → `{from, to, days: dayCount, prevFrom, prevTo}` (ISO strings).
- **`buildInsightPayload(clientId, module, range, previous, modules?)`** (:26)
  - **Which modules are included:** a module is included if `module===m`, or if module is OVERVIEW and (`modules` is empty/undefined or includes m).
  - Each block is included only when that overview query's `hasData` is true.
  - **social block:** `{followers, followerGrowth: followersDelta.abs, followerGrowthPct, engagementRate, engagementRateDeltaPct, posts, reach, reachDeltaPct, topPosts (5, caption truncated to 80, type, engagements, likes, comments)}`.
  - **seo block:** `{clicks, clicksDeltaPct, impressions, impressionsDeltaPct, ctr, position, positionDeltaAbs, health, topQueries}`. topQueries: 10 if module==="SEO", else 5.
  - **ads block:** `{currency (client.currency ?? "IDR"), spend, spendDeltaPct, results, resultsDeltaPct, cpr, cprDeltaPct, ctr, cpm, roas, campaigns (first 6: name, objective, spend, results, resultType, cpr)}`.

### src/features/reports/actions.ts ("use server")
- **`parseRange(from, to)`**: parseISODate both; null if either is invalid or `from > to`.
- **`generateInsight(slug, module, rangeFrom, rangeTo, language="id")`** (:29)
  - **Auth:** `requireClientAccess(slug)` (redirects).
  - **Validation:** module ∉ {OVERVIEW, SOCIAL, SEO, ADS} → fail(VALIDATION). Bad range → fail("Periode laporan tidak valid.", VALIDATION).
  - **Language:** "en" stays "en"; anything else becomes "id".
  - **Provider call:** `openrouter.generateInsight({module, language, clientName, range: toRangeInfo, data})` → `{content, model}`.
  - **Writes:** creates an `aiInsight {clientId, module, dateFrom, dateTo, language, model, content}`. Revalidates `/clients/{slug}` and `/clients/{slug}/reports`. Returns ok `{content}`.
  - **Errors:** ProviderError → `fail(t.errors[code] ?? UNKNOWN, code)`; else fail(UNKNOWN). The error is logged.
- **`createReport(slug, _prev, formData)`** (:78)
  - **Auth:** requireClientAccess (any member, VIEWER included).
  - **zod schema:**
    - `title`: string.trim().min(1, "Judul laporan wajib diisi.").max(160)
    - `modules`: array(enum SOCIAL|SEO|ADS).min(1, "Pilih minimal satu modul.")
    - `compare`: boolean (form value "on" or "true")
    - `includeAi`: boolean (same rule)
    - `language`: enum id|en (default "id")
    - `from`, `to`: string (default "")
  - **Validation failure:** fail(first field error, VALIDATION, fieldErrors). Bad range → invalidRange.
  - **AI summary (if includeAi):** `buildInsightPayload(OVERVIEW, modules)` → provider → also caches an AiInsight OVERVIEW row. If this fails, the error is returned and **no report is created**.
  - **Report insert:** `createdById` = the session user only if the user row still exists, else null. Creates `report {clientId, title, dateFrom, dateTo, compare, modules, language, aiSummary, createdById}`. `pdfPath` is never written. DB error → fail(UNKNOWN).
  - **Success:** revalidates `/clients/{slug}/reports`, then `redirect("/clients/{slug}/reports?tab=history&created={id}&from={from}&to={to}")`.
- **`deleteReport(slug, reportId)`** (:153)
  - `!canManage` → fail(UNAUTHORIZED).
  - Report missing or not this client's → `fail(t.errors.NOT_FOUND)` with **no code**.
  - Deletes, revalidates, returns ok `{id}`.

### src/features/reports/sections.ts and extras.ts
- **`getReportSections({clientId, range, previous, compare, modules, lang})`** (sections.ts:14) → `{SOCIAL:ReportSection[], SEO:ReportSection[]}`
  - SOCIAL: `buildPublishingSection(getPublishingSummary)`; null if there are 0 posts.
  - SEO: only if `getSuiteProperty` exists (from seo-suite, out of scope):
    - `buildRankSection(getRankData(pid, range, previous), compare)`
    - `buildBacklinkSection(getBacklinkData(pid, range))`
- **`buildRankSection`** (extras.ts:56)
  - null unless `hasKeywords && latestDate`.
  - KPIs:
    - visibility: formatPercent(v, 1)
    - avgPosition: formatNumber(v, 1), "-" when null, lowerIsBetter
    - top3, top10
    - deltas only when compare is on
  - Table: `topRankRows` (position asc with null = ∞, then volume desc, then keyword), top 10 → keyword truncated to 60, position or "Di luar 100", change text ("Naik n" / "Turun n" / "Tetap" / "Baru masuk" / "Keluar 100 besar" / "-"), volume via formatCompact or "-".
- **`buildBacklinkSection`** (extras.ts:88)
  - null if there is no latest snapshot.
  - KPIs: backlinks and referring domains (compact), domainRank, dofollowShare (%, 1 decimal), new = Σweekly.baru, lost = Σweekly.hilang. No deltas; table null.
- **`buildPublishingSection`** (extras.ts:147)
  - KPIs: posts, "Terbit di {Platform}" per platform, failed count.
  - Table title: "5 postingan terbit dengan interaksi tertinggi" or "5 postingan terbit terbaru".
  - Columns: Platform 16, Post 52 or 66, Date 18, Engagements 14 (engagement mode only).
  - Rows: `(title||body)` with whitespace collapsed, truncated to 70 or 90; formatDateShort(publishedAt).

### src/features/reports/pdf/build-data.ts: `buildReportPdfData(args)` (:48)
- **Args:** `{clientId, clientName, currency, title, range, previous, compare, modules: string[], language, aiSummary|null}`.
- **Language:** "en" if language==="en", else "id".
- **Delta rule:** `d(x) = compare ? x : null`. A KPI shows a delta only when `delta && delta.pct != null`.
  - The string is `formatDeltaPercent(pct)`; `deltaGood = isGoodChange(delta, lowerIsBetter)`.
  - Unlike pages, demo data is not suppressed.
- **SOCIAL module** (if in modules):
  - Data: getOverviewSocial; `getSocialDailyFollowers({from: range.from-1, to})`; getTopPostRows(7).
  - Chart: daily net growth = `followers[i] - followers[i-1]`.
  - If `social.hasData`, the block is:
    - KPIs: Followers (compact), Engagement rate (%), Jangkauan (compact), Postingan (number).
    - Chart only if there are more than 1 point. maxLabel = `formatDeltaNumber(max)+"/hari"`.
    - Top-posts table:
      - columns Platform 14, Caption 44, Tipe 12, Suka 10 right, Komentar 10 right, Interaksi 10 right
      - platform names Instagram / Facebook / TikTok (anything that is not IG or FB becomes TikTok)
      - caption truncated to 64
    - sections = SOCIAL sections.
  - Otherwise a sections-only page if any sections exist.
- **SEO module:**
  - Data: getOverviewSeo + getTopQueryRows(8).
  - **The chart skips days where `clicks==null`** (never draws them as 0). Shown if there are more than 1 reported day; maxLabel compact.
  - KPIs:
    - Klik (compact), Impresi (compact), CTR (%)
    - Posisi rata-rata (1 decimal, lowerIsBetter)
    - "Skor kesehatan" `"{n}/100"` if health is not null (no delta)
  - Table columns: Kata kunci 44, Klik 14, Impresi 14 (compact), CTR 14, Posisi 14 (1 decimal).
  - Same sections-only fallback as SOCIAL.
- **ADS module:**
  - Data: getOverviewAds + getCampaignRows.
  - Included only if `hasData`; there is no sections fallback.
  - KPIs:
    - Belanja iklan (currency, compact)
    - Hasil
    - Biaya / hasil (currency, lowerIsBetter)
    - CTR and CPM (no delta)
    - ROAS `"{2dp}x"` if not null
  - Chart: daily spend, compact currency maxLabel.
  - Campaign table:
    - columns Kampanye 38 (name truncated to 42), Tujuan 14, Belanja 16, Hasil 12, Biaya/hasil 20
    - OBJECTIVE_LABEL {TRAFFIC: Traffic, SALES: Sales, LEADS: Leads, ENGAGEMENT: Engagement, AWARENESS: Awareness, APP_PROMOTION: App}
    - CPR shows "—" when the row is not a conversion.
- **Returns `ReportPdfData`:**
  - `{lang, title, clientName}`
  - `periodLabel` = formatDateRange
  - `comparedLabel` = compare ? `"vs periode sebelumnya: <range>"` : null
  - `generatedLabel` = formatDateTime(now, Asia/Jakarta)
  - `modules`: blocks, in the order SOCIAL, SEO, ADS
  - `aiSummary`
- **Types** (pdf/types.ts):
  - `PdfKpi{label, value, delta?, deltaGood?}`
  - `PdfChart{title, points[{date, value}], maxLabel}`
  - `PdfTable{title, columns[{label, width%, align}], rows:string[][]}`
  - `PdfSection{title, kpis, table|null}`
  - `PdfModuleBlock{key, title, kpis, chart|null, tables, sections}`

### PDF rendering
- **Library:** `@react-pdf/renderer`. `renderToBuffer(<ReportDocument/>)` (pdf/render.tsx:6).
- **Document** (report-document.tsx), A4:
  - Built-in Helvetica / Helvetica-Bold only.
  - `safe()` turns U+2212 into "-", turns NBSP and thin spaces into spaces, and strips anything not encodable in WinAnsi (emoji etc.).
- **Page order:**
  1. Cover: "white" wordmark, title, prepared for, period + compared label, generated on, module titles.
  2. One page per module: KPI tiles 31.5% wide; delta colours good #2f6b3a, bad #8a3434, neutral grey.
  3. Bar chart, 72pt tall, negative values drawn below the baseline.
  4. Tables, then sections.
  5. Footer: `"{footer} · client"` and `"Halaman n / N"`.
  6. Optional "Ringkasan strategis" page: markdown-lite with #/##/### headings, bullets (- * •), numbered lists, **bold**.
  7. Closing page.
- **Storage:** none. The PDF is generated per request and streamed. `Report.pdfPath` exists in the schema but is never written. Nothing goes to `.uploads` (that is `src/lib/storage.ts`, `UPLOAD_DIR` defaulting to `<cwd>/.uploads`, used by other modules).

### src/app/api/reports/[id]/pdf/route.ts: `GET` (:10), `dynamic="force-dynamic"`
- **Lookup:** `report.findUnique({id}, include client)`. Missing → `404 {"error":"NOT_FOUND"}` (JSON).
- **Auth:** then `requireClientAccess(report.client.slug)`, which redirects when there is no session or membership. Any member role works.
- **Build:**
  - range = `{report.dateFrom, report.dateTo}`; previous = previousRange (recomputed, not stored).
  - `buildReportPdfData` uses the client's name and currency plus the report's title, compare, modules, language and aiSummary.
  - Data is recomputed live from current DB state, not snapshotted.
- **Response:** 200 with headers:
  - `Content-Type: application/pdf`
  - `Content-Disposition: attachment; filename="Laporan_{client.slug}_{YYYY-MM-DD from}_{YYYY-MM-DD to}.pdf"`
  - `Cache-Control: no-store`
- **Related, outside the requested scope:** `src/app/api/share/[slug]/pdf/route.ts` is a PIN-gated public share.
  - Errors: 404 NOT_FOUND, 404 DISABLED, 401 PIN_REQUIRED.
  - preset ∈ {7d, 28d, 90d}, default 28d; compare true; modules = client.shareModules; language "id".
  - aiSummary = latest OVERVIEW insight for that exact range; title `"Laporan Performa — {name}"`.
  - Same filename pattern.

### Things to watch when porting
- `sparkline` turns null into 0, and SEO `ga4Daily` turns null into 0. SEO `daily`, `SearchDailyPoint` and the PDF SEO chart keep null.
- Social follower deltas compare growth with previous growth. When the previous growth is 0, `pct` is null.
- SEO `positionDelta` uses previous position 0 when there are no previous rows, so pct is null.
- Ads results, CPR and ROAS are conversion-only at account level. Breakdown node KPIs are not.
- Many actions allow VIEWER: syncSocial, runAudit, setAuditCadence, syncSeo, generateInsight, createReport. Manage-only: addCompetitor, removeCompetitor, importAdsCsv, syncAds, deleteReport.
- Failure styles differ. requireClientAccess and requireUser redirect; the local guard helpers return `ActionResult`.
- Every sync path writes a `SyncJob`. The kinds are SOCIAL_SNAPSHOT, SEO_GSC, SEO_GA4, SEO_AUDIT, ADS_INSIGHTS and ADS_CSV_IMPORT; the statuses are PENDING, RUNNING, SUCCESS and FAILED.
