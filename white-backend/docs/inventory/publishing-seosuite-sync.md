# Backend behaviour inventory: white-analytics (publishing, seo-suite, sync, cron, connections, media, storage)

All paths are relative to `/Users/syabanf/Documents/works/WHITE-Superapps-main/white-analytics/`. Line numbers are 1-based.

**Shared conventions**
- **Action return shape.** `ActionResult<T>` (`src/lib/action-result.ts`) is `{ok:true,data}` or `{ok:false,error:string,code?,fieldErrors?}`.
  - `code` is one of `"RATE_LIMIT"|"TOKEN_EXPIRED"|"PERMISSION"|"NOT_FOUND"|"NETWORK"|"UNKNOWN"|"VALIDATION"|"UNAUTHORIZED"`.
  - Provider adapters throw `ProviderError(provider, code, message, {requiresReconnect})`. `requiresReconnect` defaults to true when code is TOKEN_EXPIRED or PERMISSION.
- **Generic error strings** (`src/i18n/id.ts:523`), `t.errors`:
  - `VALIDATION`: "Periksa kembali data yang dimasukkan."
  - `UNAUTHORIZED`: "Anda tidak punya akses."
  - `NOT_FOUND`: "Data tidak ditemukan."
  - `UNKNOWN`: "Terjadi kesalahan tak terduga."
  - `RATE_LIMIT`: "Batas permintaan API tercapai. Coba lagi beberapa menit."
  - `TOKEN_EXPIRED`: "Koneksi akun kedaluwarsa. Hubungkan ulang akun."
  - `PERMISSION`: "Izin tidak mencukupi. Periksa akses akun yang dihubungkan."
  - `NETWORK`: "Gangguan jaringan. Periksa koneksi dan coba lagi."
- **Session.** `getSessionUser()` (`src/lib/rbac.ts:20`) returns null when there is no session or the user has `isActive=false`. It returns `{id,email,name,role}`, where role is `UserRole` = `ADMIN|MEMBER`. `ClientMemberRole` = `MANAGER|VIEWER`.
- **Notifications** (`src/lib/notify.ts`):
  - `notify(userIds, {type,title,body?,href?,clientId?})` dedupes ids and runs `notification.createMany`, with body defaulting to "".
  - `clientAudience(clientId, except?)` = active ADMIN users plus active members of the client.
  - `clientApprovers(clientId, except?)` = active ADMIN users plus active MANAGER members.
- **Token encryption** (`src/lib/crypto.ts`): AES-256-GCM. `ENCRYPTION_KEY` is used directly if it base64-decodes to 32 bytes, otherwise it is sha256-hashed. Ciphertext is `base64(iv[12] + tag[16] + data)`.
- **Dates** (`src/lib/dates.ts`): `todayUtc()` is UTC midnight. `latestCompleteDay()` = `todayUtc() - 1 day`. `addDays` works on UTC calendar days.
- **Relevant Prisma enums** (`prisma/schema.prisma:17-104`):
  - `PostStatus`: DRAFT, IN_REVIEW, APPROVED, SCHEDULED, PUBLISHING, PUBLISHED, FAILED, REJECTED
  - `MediaKind`: IMAGE, VIDEO
  - `SocialPlatform`: INSTAGRAM, FACEBOOK, TIKTOK
  - `JobStatus`: PENDING, RUNNING, SUCCESS, FAILED (default PENDING)
  - `ConnectionProvider`: META, GOOGLE
  - `DeviceType`: DESKTOP, MOBILE
  - `KeywordIntent`: INFORMATIONAL, NAVIGATIONAL, COMMERCIAL, TRANSACTIONAL
  - `AuditCadence`: NONE, WEEKLY, DAILY
  - `AuditStrategy`: MOBILE, DESKTOP
  - `IssueSeverity`: ERROR, WARNING, NOTICE
- **SyncJob row** (`schema:607`): `{id, clientId?, kind:String, status:JobStatus, startedAt default now(), finishedAt?, message?, error?}`, index `(clientId, startedAt)`.

---

## src/features/publishing/lib.ts (pure rules, shared by actions and scheduler)

**Constants**
- `PLATFORMS = ["INSTAGRAM","FACEBOOK","TIKTOK"]` (:9). An unknown platform is coerced to `"INSTAGRAM"` everywhere (`isPlatform` fallback).
- `PLATFORM_LIMITS` (:22):
  - INSTAGRAM `{caption:2200, hashtags:30, media:{min:1,max:10}, videoOnly:false}`
  - FACEBOOK `{caption:63206, hashtags:null, media:{min:0,max:10}, videoOnly:false}`
  - TIKTOK `{caption:2200, hashtags:null, media:{min:1,max:1}, videoOnly:true}`
- `MIN_SCHEDULE_LEAD_MS = 5*60*1000`; `MAX_PUBLISH_ATTEMPTS = 3` (:28-29).
- `countHashtags`: regex `/(^|\s)#[\p{L}\p{N}_]+/gu`.
- `captionFor(master, override)` returns the override if `override.trim()` is non-empty, otherwise master.

**`validatePost(input, {level:"draft"|"full" (default "full"), requireSchedule?, now?})`** (:118)
- Input is `{body, targets:[{platform, bodyOverride}], media:[{kind}], scheduledAt?, linkUrl?, firstComment?}`.
- Always checked:
  - `linkUrl` must be an http(s) URL → "Tautan harus berupa URL http(s) yang valid."
  - `firstComment.length > 2200` → "Komentar pertama melebihi 2.200 karakter."
- Full level only:
  - no targets → "Pilih minimal satu akun tujuan."
  - empty body and no media → "Isi caption atau tambahkan media."
- When `requireSchedule`:
  - missing → "Tentukan jadwal terbit."
  - `< now+5min` → "Jadwal minimal 5 menit dari sekarang."
- Per unique `platform:caption` pair, always checked:
  - caption > limit → "Caption melebihi batas {n id-ID} karakter."
  - hashtags > limit → "Maksimal {n} hashtag."
  - media > max → "Maksimal {max} media."
- Per platform, full level only:
  - IG: 0 media → "Instagram membutuhkan minimal satu gambar atau video."; more than 1 media and more than 10 → "Carousel Instagram berisi 2–10 gambar."; more than 1 media with any video → "Carousel Instagram hanya boleh berisi gambar."
  - TIKTOK: unless exactly 1 media and it is a video → "TikTok membutuhkan tepat satu video."
  - FACEBOOK: a video combined with more than 1 media → "Facebook hanya menerima satu video per post."
- Returns `{ok, issues:[{scope:"post"|Platform, field?, message}]}`.

**State machine `TRANSITIONS`** (:158). These are human-driven transitions; PUBLISHING → PUBLISHED/FAILED/SCHEDULED is system-only.

| From | Allowed targets |
|---|---|
| DRAFT | IN_REVIEW, APPROVED |
| IN_REVIEW | APPROVED, REJECTED, DRAFT |
| REJECTED | DRAFT, IN_REVIEW |
| APPROVED | SCHEDULED, PUBLISHING, DRAFT |
| SCHEDULED | APPROVED, PUBLISHING |
| PUBLISHING | (none) |
| PUBLISHED | (none) |
| FAILED | PUBLISHING, SCHEDULED, DRAFT |

**Permission helpers**
- `canTransition(from, to, {role, isAuthor})`: VIEWER is always false. `to` must be in `TRANSITIONS[from]`. If `to ∈ {APPROVED, REJECTED}` and role is MANAGER and the user is the author → false. ADMIN may approve their own post.
- `EDITABLE_STATUSES = [DRAFT, REJECTED, IN_REVIEW, APPROVED]`; `canEdit` = not VIEWER and status in that list.
- `statusAfterEdit`: any editable status → DRAFT.
- `RESCHEDULABLE_STATUSES = [DRAFT, APPROVED, SCHEDULED]`; `canReschedule` = not VIEWER and status in that list.
- `canDelete` = not VIEWER and status ≠ PUBLISHING.

**`resolvePostOutcome(targets[{status,attempts}], {maxAttempts=3, manual})`** (:216), in order:
1. Targets non-empty and all PUBLISHED → PUBLISHED.
2. No FAILED targets → FAILED if there are zero targets, otherwise SCHEDULED.
3. `manual`, or any failed target has `attempts >= max` → FAILED.
4. Otherwise → SCHEDULED.

**UTM**
- `UTM_KEYS = [source, medium, campaign, term, content]`.
- `buildUtmUrl` sets `utm_<k>` on the URL, overwriting existing params. An invalid or non-http URL is returned trimmed and unchanged.
- `defaultUtm(platform)` = `{source: platform.toLowerCase() || "social", medium: "social"}`.

**`postInstant`** = `publishedAt ?? scheduledAt ?? createdAt`.

## src/features/publishing/access.ts

- **`resolvePublishingAccess(clientId)`** (:17)
  - Reads the session user, then `client.findUnique {slug, timezone}`.
  - ADMIN → role `"ADMIN"`. Otherwise reads `clientMember(userId_clientId)`: MANAGER → `"MANAGER"`, any other membership → `"VIEWER"`.
  - Returns null when there is no user, no client, or no membership.
  - Returns `{userId, userName, role, clientId, slug, timezone}`.
- **`revalidatePublishing(slug, postId?)`** (:32) revalidates `/clients/{slug}/publish`, `.../publish/posts`, `.../publish/media`, and `.../publish/posts/{postId}` when a postId is given.
- **`issuesToMessage(prefix, issues)`** returns `prefix + " " + unique messages`. Platform-scoped messages are prefixed `"Instagram: "`, `"Facebook: "` or `"TikTok: "`.

## src/features/publishing/actions.ts ("use server")

**Shared helpers**
- `loadPost(postId)` (:34) loads `contentPost` including targets (account platform, isCompetitor, clientId) and media ordered by `order asc` (with asset.kind). It then calls `resolvePublishingAccess`. It returns null (surfaced as NOT_FOUND) if the post or access is missing. `isAuthor = post.createdById === access.userId`.
- `setStatus(post, access, status, activityType, {data, meta})` (:54) runs one `$transaction` with three writes, then calls `revalidatePublishing`:
  - `contentPost.update({status, ...data})`
  - `postTarget.updateMany({where:{postId, status:{not:"PUBLISHED"}}, data:{status}})`
  - `postActivity.create({type, meta: meta ?? {from, to}})`
- Strings (`strings.ts` `p.err`):
  - `notFound`: "Post tidak ditemukan."
  - `transition`: "Aksi tidak diizinkan untuk status post saat ini."
  - `accountNotOwn`: "Salah satu akun bukan milik klien ini."
  - `mediaNotOwn`: "Salah satu media bukan milik klien ini."
  - `fixIssues`: "Perbaiki dulu isu berikut:"
  - `noFailedTargets`: "Tidak ada target yang gagal."
  - `nothingToPublish`: "Tidak ada target untuk diterbitkan."
  - `p.cannotEditStatus`: "Post dengan status ini tidak bisa diubah. Batalkan jadwal atau duplikat post."
  - `p.selfApprove`: "Penyetuju harus orang lain (kecuali Admin)."
- Notification titles (`p.notif`):
  - "Post menunggu review: {t}" (body "Buka untuk menyetujui atau menolak.")
  - "Post disetujui: {t}"
  - "Post ditolak: {t}"
  - "Post terbit: {t}"
  - "Post gagal terbit: {t}"
- Notification post title = `title || body.slice(0,40) || id`; href = `/clients/{slug}/publish/posts/{id}`.

**Exported actions**
- **`savePost(input)` (:94)**
  - Zod `saveSchema` (:78):
    - `clientId` min1; `id` min1 optional
    - `title` max160, default ""; `body` max65000, default ""
    - `targets[]` max20 of `{socialAccountId min1, bodyOverride max65000 nullable optional}`, default []
    - `mediaIds[]` max10 of min1, default []
    - `mediaAlt` record(string → string max500) optional
    - `linkUrl` max2000 nullable optional
    - `utm` partial `{source, medium, campaign, term, content}`, each max100
    - `firstComment` max2200 nullable optional
    - `labels[]` max20 of `trim().min1.max40`, default []
    - `scheduledAt` `z.iso.datetime({offset:true})` nullable optional
  - Errors:
    - parse fail → `t.errors.VALIDATION` / "VALIDATION"
    - no access or VIEWER → "UNAUTHORIZED"
    - accounts not found for `(id in, clientId, isCompetitor:false)` count mismatch → `accountNotOwn` VALIDATION
    - `mediaAsset (id in, clientId)` count mismatch → `mediaNotOwn` VALIDATION
    - `validatePost` level "draft" fails → issuesToMessage VALIDATION
  - Normalisation:
    - `utm` is kept only if some value is truthy
    - `labels` trimmed and deduped
    - `linkUrl` and `firstComment` trimmed, empty → null
    - `bodyOverride` trimmed, empty → null
    - media rows are `{assetId, order: index}` after dedupe
  - `altUpdates`: `mediaAsset.update altText` (trimmed, empty → null) for ids that are in the selected mediaIds.
  - **Update path** (`d.id`):
    - `findFirst {id, clientId}`; missing → NOT_FOUND.
    - `!canEdit` → `cannotEditStatus` VALIDATION.
    - `status = statusAfterEdit` (always DRAFT).
    - One `$transaction`:
      - `contentPost.update`: base fields, `status`, `utm ?? Prisma.JsonNull`, and `approvedById/approvedAt = null` if the post was moving from non-DRAFT to DRAFT
      - `postTarget.deleteMany` where `socialAccountId notIn` the new list
      - `postTarget.upsert` per target on `(postId_socialAccountId)`, with status
      - `postMedia.deleteMany`, then `postMedia.createMany`
      - the alt updates
      - `postActivity "EDITED" {from, to}`
    - Returns `{id, status}`.
  - **Create path**: `contentPost.create` with status DRAFT, `createdById`, nested targets (status DRAFT), media, and activity `"CREATED"` (no meta). Alt updates run in a separate `$transaction`. Returns `{id, status:"DRAFT"}`.
  - Calls `revalidatePublishing`. `scheduledAt` is stored without any status change; it acts as a planned date.
- **`submitForReview(postId)` (:186)**
  - Requires `canTransition → IN_REVIEW`, else `transition` UNAUTHORIZED. Then `validatePost` full, else VALIDATION.
  - `setStatus IN_REVIEW`, activity `"SUBMITTED"`, `reviewNote = null`.
  - Notifies `clientApprovers(clientId, except actor)` with type `POST_REVIEW`.
- **`approvePost(postId)` (:204)**
  - Requires `canTransition → APPROVED`; on failure returns `selfApprove` if the actor is a MANAGER author, else `transition` (UNAUTHORIZED). Then `validatePost` full.
  - `setStatus APPROVED`, activity `"APPROVED"`, `{approvedById: actor, approvedAt: now, reviewNote: null}`.
  - Notifies the author with `POST_APPROVED` if the author is not the actor.
- **`rejectPost({postId, note})` (:222)**
  - Zod: `postId` min1; `note` `trim().min1.max1000`.
  - Requires `canTransition → REJECTED`, with the same selfApprove handling.
  - `setStatus REJECTED`, activity `"REJECTED"`, `data {reviewNote: note}`, `meta {note}`.
  - Notifies the author with `POST_REJECTED`, body = note.
- **`schedulePost({postId, scheduledAt})` (:246)**
  - Zod: `postId` min1; `scheduledAt` `z.iso.datetime({offset:true})`.
  - Requires `canTransition → SCHEDULED` (from APPROVED or FAILED). Then `validatePost` full with `requireSchedule` (≥ now + 5 min).
  - `setStatus SCHEDULED`, activity `"SCHEDULED"`, `data {scheduledAt}`, `meta {scheduledAt: ISO}`.
- **`unschedulePost(postId)` (:260)**: requires status === SCHEDULED and `canTransition → APPROVED`. `setStatus APPROVED`, activity `"UNSCHEDULED"`.
- **`reschedulePost({postId, scheduledAt})` (:270)**
  - Requires `canReschedule` (DRAFT, APPROVED or SCHEDULED; not VIEWER).
  - If status is SCHEDULED and the new time is `< now + 5 min` → VALIDATION "...Jadwal minimal 5 menit dari sekarang."
  - `$transaction`: `contentPost.update {scheduledAt}` (status unchanged) plus activity `"RESCHEDULED" {from: ISO|null, to: ISO}`.
- **`backToDraft(postId)` (:292)**: requires `canTransition → DRAFT` (from IN_REVIEW, REJECTED, APPROVED or FAILED). `setStatus DRAFT`, activity `"BACK_TO_DRAFT"`, clears `approvedById/approvedAt`.
- **`publishNow(postId)` (:303)**
  - Requires `canTransition → PUBLISHING` (from APPROVED, SCHEDULED or FAILED).
  - Zero targets → `nothingToPublish` VALIDATION. Then `validatePost` full.
  - Calls `publishPost(id, {manual:true, userId})` and revalidates.
  - A null result (lost the claim) → `transition` UNAUTHORIZED. Otherwise returns `ok(PublishPostResult)`.
- **`retryPost(postId)` (:317)**: status must be FAILED, else `noFailedTargets` VALIDATION. Delegates to `publishNow`.
- **`duplicatePost(postId)` (:326)**
  - VIEWER → UNAUTHORIZED.
  - Creates a new DRAFT post: title `"{title} (salinan)"` or "", copies body, linkUrl, utm, firstComment, labels, targets (status DRAFT, bodyOverride), media (assetId, order).
  - `scheduledAt` is not copied. Activity `"CREATED" {duplicatedFrom}`.
  - Returns `{id}`.
- **`deletePost(postId)` (:352)**: requires `canDelete` (not PUBLISHING, not VIEWER). `contentPost.delete` (children cascade). No storage deletion.
- **`addPostComment({postId, body})` (:364)**
  - Zod: `body` `trim().min1.max2000`.
  - Any access role, including VIEWER, may comment.
  - `postComment.create`, then revalidate.

**PostActivity.type values actually written**: CREATED, EDITED, SUBMITTED, APPROVED, REJECTED, SCHEDULED, UNSCHEDULED, RESCHEDULED, BACK_TO_DRAFT, PUBLISHED, PUBLISH_FAILED, PUBLISH_RETRY.

## src/features/publishing/actions-media.ts ("use server")

- **`uploadMedia(formData)` (:30)**
  - Expects `clientId` (string) and one or more `files` entries (File-like).
  - Missing clientId or zero files → VALIDATION. No access or VIEWER → UNAUTHORIZED.
  - Validates **every** file before storing any:
    - `f.size > MAX_UPLOAD_BYTES` (25 MiB) → `"Ukuran file melebihi 25 MB. ({name})"`
    - `sniffContentType(bytes)` null → `"Jenis file tidak didukung. ({name})"`
    - The browser MIME type is ignored; the sniffed type is authoritative.
  - Then, per file:
    - `putObject(buf, {clientId, contentType: sniffed})`
    - `mediaAsset.create {clientId, kind: VIDEO if type starts with "video/" else IMAGE, url, filename: name.slice(0,200), mimeType, sizeBytes, uploadedById}`
  - Calls `revalidatePublishing(slug)`.
  - Returns `{assets: MediaRow[]}` with `usageCount = 0`.
  - Not transactional across files.
- **`addExternalMedia(input)` (:76)**
  - Zod: `clientId` min1; `url` `z.url({protocol:/^https$/}).max(2000)`; `kind` enum `[IMAGE, VIDEO]`; `altText` max500 optional; `tags[]` max20 of `trim().min1.max40`.
  - Parse fail → "URL harus diawali https://" (VALIDATION).
  - Filename = last path segment, URI-decoded, `slice(0,200)`, falling back to "media".
  - MIME derived from extension:
    - VIDEO → `video/quicktime` if `.mov`, else `video/mp4`
    - IMAGE → `png` / `webp` / `gif`, else `image/jpeg`
  - The remote file is not fetched or validated; `sizeBytes` defaults to 0.
  - Returns `{asset}`.
- **`updateMedia({id, altText?, tags?})` (:108)**
  - Zod: `altText` max500 nullable optional; `tags` as above.
  - Missing asset → NOT_FOUND. VIEWER → UNAUTHORIZED.
  - `altText` trimmed, empty → null (only when provided). Tags deduped.
- **`deleteMedia(id)` (:126)**
  - If `_count.usages > 0` → "Media masih dipakai oleh post. Lepaskan dari post terlebih dahulu." (VALIDATION).
  - Deletes the DB row, then the stored file if `keyFromUrl(url)` matches (external URLs are skipped).

## src/features/publishing/scheduler.ts (publish flow)

**`publishPost(postId, {now?, manual?, userId?, claimed?, syncJob? (default true)})` (:37)**

1. **Claim.** Unless `claimed`, runs `contentPost.updateMany({where:{id, status in CLAIMABLE=["APPROVED","SCHEDULED","FAILED"]}, data:{status:"PUBLISHING"}})`. If `count ≠ 1` it returns null.
2. **Load.** Loads the post with client `{id, slug}`, targets with account and connection, and media ordered by `order` (asset url, kind).
3. **Link.** `link = buildUtmUrl(linkUrl, parseUtm(utm))`.
4. **Pending targets.** Pending = targets with status ≠ PUBLISHED; they are bulk-set to PUBLISHING. Already-PUBLISHED targets count as published outcomes.
5. **Per pending target:**
   - `text = captionFor(body, bodyOverride)`; if a link exists and the text does not already contain it, `text = text.trim() + "\n\n" + link` (or just link).
   - `useReal = isPublisherConfigured()` (`META_APP_ID && META_APP_SECRET`) **and** `account.connectionId` is set. Otherwise `mockPublisher` is used. This means credentials present but no connection silently uses the mock.
   - When real, `getConnectionToken(connectionId)` supplies the token (may throw TOKEN_EXPIRED).
   - `PublishInput = {platform, accessToken, accountExternalId, text, media, linkUrl, firstComment}`; firstComment is set for INSTAGRAM only.
   - Success → `postTarget.update {status: PUBLISHED, externalId, permalink, error: null, publishedAt: now, attempts+1}`.
   - Error → `postTarget.update {status: FAILED, error: msg.slice(0,500), attempts+1}`.
     - `msg` = `"{t.errors[code]} ({e.message})"` for a ProviderError, else `e.message`, else UNKNOWN.
     - The error entry is `{target: "{platform lower} @{username}", error}`.
6. **Resolve.** `status = resolvePostOutcome(outcomes, {manual, maxAttempts:3})`.
7. **Persist post and activity.**
   - `contentPost.update {status, publishedAt: status=PUBLISHED ? now : null}`.
   - `postActivity {userId ?? null, type: PUBLISHED|PUBLISH_FAILED|PUBLISH_RETRY, meta {published, failed, errors, manual, mock: !isPublisherConfigured()}}`.
8. **SyncJob.** If `syncJob !== false`: `syncJob.create {kind: "PUBLISH", status: FAILED if post FAILED else SUCCESS, startedAt: now, finishedAt, message: "{title||id}: {n} terbit, {m} gagal[ (mock)]", error: errors[0]?.error}`.
9. **Notify.** `clientAudience(clientId)` (actor not excluded) gets `POST_PUBLISHED`, or `POST_FAILED` with body = first error. Nothing is sent on SCHEDULED (retry).
10. **Return** `{status: "PUBLISHED"|"FAILED"|"SCHEDULED", published, failed, errors:[{target, error}]}`.

Notes:
- `attempts` is never reset, including on reschedule or back-to-draft. After a target has accumulated 3 attempts, the next cron failure goes straight to FAILED.
- On SCHEDULED retry, `scheduledAt` is unchanged (already ≤ now), so the next 5-minute cron run picks it up again. Failed targets keep FAILED target status meanwhile.

**`publishDuePosts({now?, limit=20})` (:156)**, the cron entry point:
1. Candidates: `contentPost.findMany({where:{status:"SCHEDULED", scheduledAt:{lte:now}}, orderBy:{scheduledAt:"asc"}, take:limit})`.
2. Claims each sequentially with a conditional `updateMany` (same where clause plus id) → PUBLISHING; keeps rows where `count === 1`.
3. Re-reads claimed ids still in PUBLISHING.
4. Publishes each sequentially with `{now, claimed:true, syncJob:false}` (non-manual, so retries are possible).
5. A thrown exception sets the post to FAILED and writes activity `PUBLISH_FAILED {error}`; both writes swallow their own errors.
6. Per client, one `syncJob {kind:"PUBLISH", status: FAILED if failed>0 && published==0 else SUCCESS, message "{p} terbit · {f} gagal · {r} diulang[ (mock)]"}`.
7. Returns `{attempted, published, failed}`.

**Platforms real vs mock**
- `src/lib/providers/social-publisher/`:
  - **Real** (Graph v21.0, `real.ts`):
    - **Instagram**: one media item → container (`image_url`, or `media_type: REELS` + `video_url`). Several media → child containers (`is_carousel_item`, video children use `media_type: VIDEO`) plus a `CAROUSEL` container with children. Then `media_publish {creation_id}`, `GET permalink`, and a best-effort first comment.
    - Video containers are polled every 3000 ms, up to 40 times. ERROR or EXPIRED throws UNKNOWN; timeout throws NETWORK.
    - **Facebook**:
      - video → `/{page}/videos {file_url, description}`
      - 1 image → `/{page}/photos {url, message}`, using `post_id ?? id`
      - more than 1 image → unpublished photos plus `/{page}/feed` with `attached_media[i]`
      - otherwise → `/{page}/feed {message, link}`
      - Then `GET permalink_url` and a best-effort first comment.
    - **TikTok** always throws `PERMISSION` "Publikasi TikTok belum tersedia — menunggu persetujuan Content Posting API".
  - **Mock** (`mock.ts`):
    - Deterministic success: `externalId = mock_{platform}_{hash36}{hash%997 base36}`, with a platform-specific permalink.
    - A caption containing `"[fail]"` throws `ProviderError("mock-publisher","UNKNOWN","Publikasi gagal (simulasi — caption mengandung [fail])")`.

## src/features/publishing/queries.ts (server-only, React `cache`, no auth inside; callers check)

- **`getOwnAccounts(clientId)` (:157)**: `socialAccount {clientId, isCompetitor:false}`, ordered by `platform asc, username asc`. Returns `{id, platform, username, displayName, avatarUrl}[]`.
- **`getBestTimes(clientId, tz, now)` (:167)**
  - Reads `socialPost` where `account {clientId, isCompetitor:false}`, ordered by `publishedAt desc`, take 400.
  - `postingHeatmap` → `bestPostingTimes` (`src/lib/metrics/publishing.ts`):
    - score = `avgEngagement * ln(1 + posts)`
    - minPosts = 2, n = 3, distinct weekdays first
    - defaults: Mon 11:00, Wed 19:00, Fri 12:00
  - `nextOccurrences` uses a 5-minute lead.
  - Returns `{day, hour, at: ISO, fromHistory: score > 0}[]`.
- **`listMedia(clientId)` (:179)**: `mediaAsset {clientId}` ordered by `createdAt desc`, with `_count.usages`. Returns `MediaRow[]`.
- **`getPostDetail(clientId, postId)` (:188)**
  - `findFirst {id, clientId}`, including createdBy and approvedBy names, targets with account info, media (order asc, with usage count), comments (createdAt asc) and activities (createdAt desc).
  - Targets are sorted by platform.
  - Returns `PostDetail` or null.
- **`listPosts(clientId)` (:249)**
  - All posts, ordered by `[scheduledAt desc, createdAt desc]`, including the first media item.
  - Rows: `{id, title, body, bodyShort (truncate 90), status, at: postInstant, scheduledAt, publishedAt, updatedAt, platforms (unique), author, thumbnailUrl (thumb ?? url), failedTargets}`.
  - Also returns `counts`: per-status counts for all 8 statuses.
- **`getCalendarPosts(clientId, from, to)` (:278)**
  - Where clause: `OR[{publishedAt in [from,to)}, {publishedAt null AND scheduledAt in [from,to)}, {both null AND createdAt in [from,to)}]`.
  - Sorted by `at` asc. Returns `{id, title, bodyShort (60), status, at, platforms, thumbnailUrl}`.
  - Range comes from `time.ts` `rangeForDays(days, tz)`: client-tz midnights converted to UTC.
- **`getPublishingStats(clientId, now)` (:306)**, five counts:
  - `scheduled7d`: SCHEDULED with `scheduledAt in [now, now+7d)`
  - `inReview`: IN_REVIEW
  - `published30d`: PUBLISHED with `publishedAt ≥ now-30d`
  - `failed`: FAILED
  - `drafts`: DRAFT

## src/features/publishing/posts-tabs.ts

- `POSTS_TABS = [ALL, DRAFT, IN_REVIEW, APPROVED, SCHEDULED, PUBLISHED, FAILED]`.
- `TAB_STATUSES`:
  - ALL: null (everything)
  - DRAFT: [DRAFT, REJECTED]
  - IN_REVIEW: [IN_REVIEW]
  - APPROVED: [APPROVED]
  - SCHEDULED: [SCHEDULED, PUBLISHING]
  - PUBLISHED: [PUBLISHED]
  - FAILED: [FAILED]
- `isPostsTab` validates `?tab=`. `tabCount` sums the counts for a tab.

## src/features/publishing/time.ts (pure)

- `toZoned` uses `Intl` en-US h23, weekday Mon=0.
- `fromZoned(date, time, tz)` uses a two-pass offset correction.
- `monthGrid` is 6×7, Monday-first.
- `rangeForDays` returns `[fromZoned(first,"00:00"), fromZoned(last+1,"00:00"))`.
- `nextOccurrences(slots, tz, from, leadMs=5min)` searches up to 8 days ahead.
- `tzLabel` maps Asia/Jakarta → WIB, Asia/Makassar → WITA, Asia/Jayapura → WIT.

---

## src/lib/storage.ts

- **`ALLOWED` MIME → extension**: `image/jpeg→jpg`, `image/png→png`, `image/webp→webp`, `image/gif→gif`, `video/mp4→mp4`, `video/quicktime→mov`.
- `MAX_UPLOAD_BYTES = 25*1024*1024`.
- Root directory: `process.env.UPLOAD_DIR || <cwd>/.uploads`.
- **`safeKey`** regex `/^[A-Za-z0-9_-]+\/[A-Za-z0-9_.-]+$/`, else throws `"INVALID_KEY"`.
- **`sniffContentType(bytes)` (:46), magic bytes**:
  - JPEG: `FF D8 FF`
  - PNG: `0x89 "PNG" 0D 0A 1A 0A` (8 bytes)
  - GIF: `"GIF87a"` or `"GIF89a"`
  - WEBP: `"RIFF"` at 0–4 and `"WEBP"` at 8–12
  - MP4/MOV: `"ftyp"` at bytes 4–8, then `video/quicktime` if bytes 8–12 start with `"qt"`, else `video/mp4`
  - Anything else → null.
- **`publicUrl(key)`** = `${NEXT_PUBLIC_APP_URL without trailing /}/api/media/${key}`.
- **`putObject(data, {clientId, contentType})` (:72)**
  - Throws `"UNSUPPORTED_TYPE"` if the extension is unknown or `sniff(data) !== contentType`.
  - Throws `"TOO_LARGE"` if over 25 MiB.
  - Key = `${clientId}/${randomUUID()}.${ext}`. Writes the file with mkdir -p.
  - Returns `{key, url, sizeBytes, contentType}`.
- **`getObject(key)`** returns `{body, contentType (from extension, else application/octet-stream), etag: sha1("${size}-${mtimeMs}") hex}`, or null.
- **`deleteObject`** unlinks and ignores errors.
- **`keyFromUrl`** regex `/\/api\/media\/([A-Za-z0-9_-]+\/[A-Za-z0-9_.-]+)$/`.

## src/app/api/media/[...key]/route.ts

- **`GET`** (:10): public, no auth. Unguessable UUID keys are treated as equivalent to signed URLs; Meta Graph fetches media by URL.
- Joins the key segments with "/" and calls `getObject`.
  - Missing (or invalid key, caught) → 404 "Not found".
  - `If-None-Match === etag` → 304.
- Otherwise 200 with headers `Content-Type`, `Content-Length`, `Cache-Control: public, max-age=31536000, immutable`, `ETag`.

---

## src/lib/connections.ts

**`getConnectionToken(connectionId)` (:20)**
- Returns null if the id is empty, the connection is not found, or the provider app is not configured:
  - GOOGLE: `GOOGLE_CLIENT_ID && GOOGLE_CLIENT_SECRET`
  - META: `META_APP_ID && META_APP_SECRET`
- `REFRESH_SKEW_MS = 60_000`. "Expiring" = `expiresAt` is set and `expiresAt - now < 60s`. A null `expiresAt` never expires.
- Not expiring → `{token: decrypt(accessTokenEnc), connectionId}`.
- GOOGLE with a refresh token:
  - `refreshGoogleAccessToken`: POST `https://oauth2.googleapis.com/token`, `grant_type=refresh_token`, 15 s timeout.
  - Failure: `invalid_grant` → TOKEN_EXPIRED, else PERMISSION.
  - Success: `connection.update {accessTokenEnc: encrypt(new), expiresAt: now + expiresIn*1000 or null}`. The refresh token is not rotated.
- Otherwise throws `ProviderError(provider.toLowerCase(), "TOKEN_EXPIRED", "Token koneksi kedaluwarsa — hubungkan ulang di Pengaturan Klien.")`. Meta long-lived tokens (~60 days) cannot be refreshed silently.

## src/app/api/connections/_lib/access.ts

- **`appBaseUrl`**: `NEXT_PUBLIC_APP_URL` without trailing slash, falling back to the request origin.
- **`checkClientManage(slug)`**:
  - slug must match `/^[a-z0-9-]+$/`, else null
  - no session → `"unauthenticated"`
  - client not found → null
  - ADMIN → ok; otherwise membership role must be MANAGER, else null
  - ok returns `{clientId, slug, userId}`.
- Cookies: `META_NONCE_COOKIE = "meta_oauth_nonce"`, `GOOGLE_NONCE_COOKIE = "google_oauth_nonce"`. Options `{httpOnly, sameSite: "lax", secure: NODE_ENV==="production", path: "/", maxAge: 600}`.

## OAuth state (`src/lib/providers/oauth-state/index.ts`)

- State = `base64url(JSON{slug, nonce, ts})` + "." + `HMAC-SHA256(body, AUTH_SECRET)` (base64url).
- Nonce = `randomBytes(16)` hex.
- Verify uses `timingSafeEqual` and requires: slug matching `/^[a-z0-9-]+$/`, a non-empty nonce, `ts` not more than 60 s in the future, and age ≤ `OAUTH_STATE_MAX_AGE_MS` (10 min).
- `signOAuthState` throws if the secret is missing.

## Connection routes

**`src/app/api/connections/google/start/route.ts` — `GET ?client=<slug>`**
1. `checkClientManage`: unauthenticated → redirect `/login`; not allowed → `/clients?missing=1`.
2. Google not configured → `/clients/{slug}/settings?demo=1&provider=google`.
3. Otherwise signs state, redirects to the Google auth URL and sets the nonce cookie.
   - Auth URL params: `access_type=offline`, `prompt=consent`, `include_granted_scopes=true`, `response_type=code`.
   - Scopes: `webmasters.readonly`, `analytics.readonly`, `openid`, `email`.
   - `redirect_uri` = `{base}/api/connections/google/callback`.

**`src/app/api/connections/google/callback/route.ts` — `GET`**
1. Invalid state → `/clients?missing=1`.
2. Cookie nonce ≠ state nonce → `{settings}?error=oauth_state`.
3. `checkClientManage` again (same redirects); not configured → demo redirect.
4. `error` param or no `code` → `?error=oauth_denied`.
5. Exchanges the code at `https://oauth2.googleapis.com/token` (`grant_type=authorization_code`), then fetches userinfo from `https://openidconnect.googleapis.com/v1/userinfo`.
6. `connection.upsert` on unique `(clientId, provider: GOOGLE, accountId: userinfo.sub)`:
   - Payload: `displayName = email ?? name ?? "Akun Google"`, `accessTokenEnc`, `expiresAt = now + expiresIn` or null, `scopes` (returned scopes or `GOOGLE_SCOPES`).
   - `refreshTokenEnc` is updated only when Google returns a new one; create always sets it (possibly null).
7. Success → `?connected=google`. Errors → `?error={ProviderError.code | "UNKNOWN"}`. The nonce cookie is deleted in both cases.
- Google error mapping (`mapGoogleError`): 429 or `RESOURCE_EXHAUSTED` → RATE_LIMIT; `invalid_grant` or 401 → TOKEN_EXPIRED; 403, `access_denied` or `PERMISSION_DENIED` → PERMISSION; 404 → NOT_FOUND.

**`src/app/api/connections/meta/start/route.ts`**
- Same flow as the Google start route.
- Dialog: `https://www.facebook.com/v21.0/dialog/oauth`.
- Scopes (comma-joined): `pages_show_list, pages_read_engagement, instagram_basic, instagram_manage_insights, business_management, ads_read`.

**`src/app/api/connections/meta/callback/route.ts`**
- Same guards as the Google callback.
- Token flow: code → short-lived token (`/oauth/access_token`) → long-lived token (`grant_type=fb_exchange_token`) → `/me?fields=id,name`.
- `connection.upsert` on `(clientId, META, me.id)`, payload `{displayName: me.name, accessTokenEnc, refreshTokenEnc: null, expiresAt, scopes: META_SCOPES}`.
- Redirects to `?connected=meta`.
- Meta error mapping:
  - 429 or code 4/17/32/613 → RATE_LIMIT
  - code 190 or `OAuthException` → TOKEN_EXPIRED
  - 403, code 10 or 200–299 → PERMISSION
  - 404 or code 803 → NOT_FOUND

**Outside the requested folders, for completeness** (`src/features/clients/actions-resources.ts`):
- `disconnectConnectionAction` (:22) deletes a connection.
- `simulateConnectionAction` (:35) is demo mode. It upserts accountId `"demo-meta"` or `"demo-google"` with `accessTokenEnc = encrypt("mock")`, expiry now + 60 days, and the provider scopes.

## src/lib/integrations.ts

- **Model** `IntegrationSetting {provider PK, publicId?, secretEnc?, meta?, verifiedAt?, updatedById?}`. A DB value wins over env and is copied into `process.env` (`def.publicEnv` / `def.secretEnv` from `@/features/setup/integrations`).
- **`hydrateIntegrationEnv()`** (:51): `findMany`, applies every row, returns the row count. A decrypt failure leaves env untouched.
- **`getIntegrationStatuses()`** (:57, cached): per `INTEGRATION_KEYS` returns `{key, configured (isConfiguredPair), source: "db"|"env"|"none", publicId, secretMasked, meta, verifiedAt ISO}`.
- **`saveIntegration(key, {publicId?, secret?, meta?, verifiedAt?, updatedById?})`** (:89)
  - Upserts by provider. `publicId` "" → null.
  - A non-empty `secret` is encrypted. `secret === null` clears the stored secret **and** deletes the env var.
  - `publicId` null or "" deletes the public env var.
- **`clearIntegration(key)`** (:107): `deleteMany` plus deletes both env vars.
- **`getSetupState()`** (:117): `appSetting key="setup"` → `{completedAt, skippedAt}`.
- **`markSetup(kind)`** (:123): upserts the timestamp ISO.

---

## src/features/seo-suite/access.ts

- **`REFRESH_COOLDOWN_MS = 10*60*1000`** (:12).
- **`guardProperty(clientId, propertyId, {manage?})`** (:16)
  - No user → `t.errors.UNAUTHORIZED` / "UNAUTHORIZED".
  - Non-ADMIN without membership → "Anda tidak punya akses ke klien ini." (UNAUTHORIZED).
  - `manage` and role ≠ MANAGER → "Hanya peran ADMIN/MANAGER yang dapat mengubah." (UNAUTHORIZED).
  - `seoProperty.findFirst {id, clientId}` missing → "Properti SEO tidak ditemukan." (NOT_FOUND).
  - Returns `{userId, property:{id, siteUrl, locationCode, languageCode, clientId, slug}}`.
- **`startJob(clientId, kind, message?)`** (:33)
  - Looks up the latest SyncJob of that kind for the client, any status, by `startedAt desc`.
  - If it started less than 10 min ago → `fail("Pembaruan terakhir kurang dari 10 menit lalu. Coba lagi beberapa menit.")` with no code.
  - Otherwise creates `syncJob {status: "RUNNING", message}` and returns its id.
- **`finishJob(id, outcome)`**: `{status: "SUCCESS", finishedAt, message}` or `{status: "FAILED", finishedAt, error: slice(0,900)}`.
- **`errorMessage`** returns `e.message` (or UNKNOWN). **`errorCode`** returns the ProviderError code or undefined.

## src/features/seo-suite/actions.ts ("use server")

**Schemas**
- `ideaSchema` (:15): `keyword` min1 max120; `volume` int ≥ 0 nullable optional; `difficulty` 0–100 nullable optional; `cpc` ≥ 0 nullable optional; `intent` enum `[INFORMATIONAL, NAVIGATIONAL, COMMERCIAL, TRANSACTIONAL]` nullable optional.
- `revalidateSuite(slug)` revalidates `/clients/{slug}/seo/rank` and `/clients/{slug}/seo/research`.

**`insertTracked(property, ideas, device, tags)`** (:39, internal)
- `trackedKeyword.createMany({data:[{propertyId, keyword: trim().toLowerCase(), locationCode, languageCode (from property), device, tags, volume, difficulty: round, cpc, intent}], skipDuplicates:true})`.
- The unique key is `(propertyId, keyword, locationCode, device)`.
- If `count > 0`, it diffs ids before and after and calls `trackRanks(property, {date: todayUtc(), keywordIds: newIds})`; errors are swallowed.
- Returns `count`.

**Exported actions**
- **`trackKeywords(input)`** (:70)
  - Zod: `{clientId, propertyId, ideas[] min1 max100, device enum [DESKTOP, MOBILE] default MOBILE, tags[] max10 of min1 max30, default []}`.
  - `guardProperty` with manage. Returns `{added}`. A provider error → `fail(message, code)`.
- **`addTrackedKeywords(input)`** (:93)
  - Zod: `{clientId, propertyId, text min1 max20000, device default MOBILE, tags string max500 default ""}`.
  - `parseKeywordLines(text, 100)`: split on `/[\n,]+/`, normalise (trim, collapse spaces, lowercase), drop entries longer than 120 chars, dedupe, cap 100.
  - Empty → "Masukkan minimal satu kata kunci (maks. 100 per tambah)." (VALIDATION).
  - Guard with manage, then enrichment via `seoData.keywordMetrics(keywords, loc, lang)` (failure ignored).
  - `parseTags`: split on `/[,\n]+/`, lowercase, length ≤ 30, unique, ≤ 10.
  - Returns `{added}`.
- **`deleteTrackedKeyword({clientId, keywordId})`** (:123): missing keyword → NOT_FOUND. Guard with manage on the keyword's property. Deletes the row (snapshots cascade).
- **`updateKeywordTags({clientId, keywordId, tags: string max500})`** (:137): guard with manage, `parseTags`, update. Returns `{tags}`.
- **`refreshRanks({clientId, propertyId})`** (:153)
  - Guard **without** manage, so VIEWER can refresh.
  - `startJob(clientId, "SEO_SUITE_RANKS", siteUrl)`, then `trackRanks(property, {date: todayUtc()})`.
  - `finishJob` ok with message `"{n} kata kunci"`, or failure. Returns `{updated}`.

## src/features/seo-suite/actions-domains.ts ("use server")

- **`refreshBacklinks({clientId, propertyId})`** (:18)
  - Guard without manage; `startJob "SEO_SUITE_BACKLINKS"`; `tracker.refreshBacklinks(property, todayUtc())`.
  - Finish message `"+{added} / −{lost}"`. Revalidates `/clients/{slug}/seo/backlinks`.
  - Returns `{added, lost}`.
- **`addCompetitorDomain({clientId, propertyId, domain min1 max253})`** (:40)
  - `normalizeDomain` null → "Format domain tidak valid." (VALIDATION).
  - Guard with manage.
  - Domain equal to `siteHost(siteUrl)` → "Format domain tidak valid.".
  - Already exists → "Domain sudah ada.".
  - `count >= MAX_COMPETITOR_DOMAINS` (5) → "Maksimal 5 kompetitor domain.".
  - `competitorDomain.create`, then `refreshDomains(p, today, [domain])` (errors swallowed).
  - Revalidates `/seo/competitors` and `/seo/research`. Returns `{id, domain}`.
- **`removeCompetitorDomain({clientId, id})`** (:68)
  - Missing → NOT_FOUND. Guard with manage.
  - `$transaction [competitorDomain.delete, domainSnapshot.deleteMany {propertyId, domain}]`.
  - Revalidates the same two paths.
- **`refreshDomains({clientId, propertyId})`** (:84)
  - Guard without manage; `startJob "SEO_SUITE_DOMAINS"`.
  - Finish message `"{n} domain"`. Revalidates `/seo/competitors`. Returns `{domains}`.

## src/features/seo-suite/tracker.ts (auth-free write paths)

**`trackRanks(property, {date, keywordIds?})`** (:19), rank snapshot logic
1. Loads keywords `{id, keyword, device}` (optionally filtered by `keywordIds`). None → `{updated: 0, drops: []}`.
2. `targetDomain = siteHost(siteUrl)`.
3. Previous positions = `rankSnapshot` where `date = date - 1 day`, exactly.
4. Batches of `SERP_BATCH = 20`, each calling `seoData.serpRanks([{keyword, device}], {locationCode, languageCode, targetDomain})`.
5. Per result:
   - `rankSnapshot.upsert` on `(keywordId, date)` with `{position, url, serpFeatures, topResults: top10}`.
   - `updated++`.
   - A drop is recorded only if a snapshot existed yesterday **and** `isRankDropAlert(prev, cur)`.
6. Returns `{updated, drops: [{keyword, prev, cur}]}`.

**`refreshBacklinks(property, date)`** (:57)
1. In parallel: `seoData.backlinkSummary(domain)`, `seoData.backlinks(domain, {limit: 300, mode: "all"})`, and existing `backlink` rows.
2. In chunks of 25, `backlink.upsert` on `(propertyId, sourceUrl, targetUrl)`:
   - Create sets all fields. Update sets `{anchor, dofollow, domainRank, spamScore, lastSeen, isLost}`.
   - `firstSeen` and `lastSeen` fall back to the date when missing.
3. Counting:
   - `added` = keys not present before.
   - `lost` = keys that were present with `isLost = false` and are now `isLost`.
   - Links absent from the provider list are **not** marked lost.
4. First run (no existing rows): `new = round(newLast30/30)`, `lost = round(lostLast30/30)`.
5. `backlinkSnapshot.upsert` on `(propertyId, date)` with `{backlinks, referringDomains, dofollow, nofollow, newBacklinks, lostBacklinks, domainRank, toxicShare}`.
6. Returns `{lost, added}` (the snapshot's values).

**`refreshDomains(property, date, only?)`** (:103)
- Domains = `only` or `[own host, ...competitors]`.
- Sequentially calls `seoData.domainOverview` for each, then `domainSnapshot.upsert` on `(propertyId, domain, date)` with `{organicKeywords, organicTraffic, organicCost, top3, top10, top100, backlinks, referringDomains, domainRank}`.
- Returns the count.

## src/features/seo-suite/daily.ts

**`runSeoSuiteDaily({now?, clientId?})`** (:25)
1. `date = todayUtc(now)`. Properties = `seoProperty.findMany` (filtered by client if given).
2. Per property, `syncJob.create {kind: "SEO_SUITE_DAILY", status: "RUNNING", message: siteUrl}`, then four independent steps (each error is pushed to `errors` as `"ranks: …"`, `"backlinks: …"`, `"domains: …"`, `"audit: …"`):
   1. **Ranks**: `trackRanks`. If there are drops, notify `clientAudience` with type `RANK_DROP`, title "Peringkat turun", body `"{n} kata kunci turun tajam sejak kemarin: {up to 3 × 'kw (prev|–→cur|–)'}"`, href `/clients/{slug}/seo/rank`.
   2. **Backlinks**: `refreshBacklinks`. If `lost > BACKLINK_LOST_THRESHOLD` (5), notify `BACKLINK_LOST`, "Backlink hilang", body `"{n} backlink hilang dalam sehari terakhir."`, href `.../seo/backlinks`.
   3. **Domains**: `refreshDomains`.
   4. **Audit**: last `seoAudit.runAt` → `shouldAudit(cadence, last, now)`:
      - NONE → false
      - DAILY → no previous audit, or more than 20 h since the last
      - WEEKLY → no previous audit, or more than 6 days since the last
      - When due → `runAuditCore(p.id)`.
3. Updates the job: `status = FAILED` if there are errors and no notes, else SUCCESS. `message = notes.join(" · ")`, `error = errors.join(" | ").slice(0,900)`.
4. Housekeeping: `keywordResearch.deleteMany {fetchedAt < date - 30 days}`.
5. Returns `{properties, ranks, backlinks, domains, audits}`.

## src/features/seo-suite/queries.ts (server-only, React `cache`, no auth inside)

- **`getSuiteProperty(clientId)`** (:31): first `seoProperty` by `createdAt asc`. Returns `{id, siteUrl, host, locationCode, languageCode, auditCadence}`.
- **`isSuiteDemo()`**: `!isDataForSeoConfigured()`.
- **`lastJobAt(clientId, kinds[])`** (:43): latest SyncJob `startedAt` ISO for any of the given kinds, any status.
- **`getResearch(clientId, seed, mode, loc, lang)`** (:58), keyword research cache
  - Cache key `seed = "{mode}:{seed}"`.
  - Newest cached row by `fetchedAt desc`; if fresh, returns `{ideas, fetchedAt, fromCache: true}`. Fresh = `isCacheFresh`: `0 ≤ age < RESEARCH_MAX_AGE_DAYS (7) × 86400000`.
  - Otherwise `seoData.keywordIdeas(seed, {loc, lang, mode, limit: 100})` → `keywordResearch.create {provider: "dataforseo"|"mock", results}` → `deleteMany` of other rows with the same key.
  - Research modes: `RESEARCH_MODES = ["ideas", "questions", "related"]`; `normalizeMode` defaults to "ideas".
- **`getTrackedKeywordSet(propertyId)`**: lowercased keyword list.
- **`getCompetitorDomains(propertyId)`**: ordered by `createdAt asc`, `{id, domain}`.
- **`getKeywordGap(own, comp, loc, lang)`**: `domainKeywords` ×2 (limit 200), then `keywordGap`:
  - missing = competitor ranks, own does not
  - weak = own position > competitor position
  - strong = otherwise
  - shared = all keywords both rank for
  - Each set sorted by volume desc; shared volume = max of the two.
  - Live provider call, not cached in the DB.
- **`getRankData(propertyId, range, previous)`** (:118)
  - Windows: `curEnd = range.to + 1 day`, `prevEnd = previous.to + 1 day`, `sparkFrom = curEnd - 29`; snapshots are loaded from `min(previous.from, sparkFrom)` to `curEnd`.
  - Keywords ordered by `createdAt asc`.
  - KPIs are taken on `latestSnapshotDate ≤ curEnd` (and `≤ prevEnd` for the previous window).
  - Formulas (`src/lib/metrics/seo-suite.ts`):
    - `visibility` = mean of `expectedCtr(pos)`, contributing 0 for null or > 100.
    - `expectedCtr`: 1:28, 2:15, 3:11, 4:8, 5:7, 6:5, 7:4, 8:3, 9:3, 10:2.5, 11–20:1.5, otherwise 1 (position rounded, min 1).
    - `avgPosition` = mean over positions ≤ 100.
    - `top3` / `top10` = counts.
  - Per-row `keywordRank`: current = latest ≤ to; start = first in `[from, to]`; best = min position in window; `change = classifyRankChange(start, cur)`; spark = `101 - clamp(pos, 1, 100)` (0 if unranked).
  - Strip: `improved` = up + new; `declined` = down + lost.
  - `distribution` buckets: `"1-3"`, `"4-10"`, `"11-20"`, `"21-50"`, `"51-100"`, `"unranked"`.
  - Share of voice: per-domain sum of `expectedCtr(pos)` over the latest top10 lists; top 8 plus "Lainnya".
  - `classifyRankChange` gives up / down / flat / new / lost / none, with `delta = prev - cur`.
  - `isRankDropAlert(prev, cur)`: true if lost, or if down and (`cur - prev > 5` or (`prev ≤ 10` and `cur > 10`)).
- **`getBacklinkData(propertyId, range)`** (:209)
  - Latest snapshot. Series = snapshots in `[range.from, range.to + 1]` ascending. Links ordered by `[isLost asc, domainRank desc]`.
  - Comparison snapshot = latest with `date ≤ latest - 30 days`; deltas via `computeDelta`.
  - `dofollowShare` = `dofollow / (dofollow + nofollow) × 100`.
  - Weekly `{baru, hilang}` grouped by `weekKey`.
  - `topAnchors` / `topDomains`: top 8 from live links.
  - Helper constants: `TOXIC_SPAM_SCORE = 60`; the "new" view = `firstSeen ≤ 30 days` ago.
- **`getCompetitorData(property, range)`** (:275)
  - Domain snapshots in range, plus `seoData.competitors(...)` (errors → []).
  - Latest per domain via `distinct: ["domain"]` ordered by date desc.
  - Plots own domain plus top 3 competitors by latest `organicTraffic`, using keys `d0..d3`.
  - Suggestions exclude known domains, `slice(0, 6)`.

## DataForSEO and Apify budgeting and caching

- **Adapter selection** (`src/lib/providers/dataforseo/index.ts`), resolved lazily per call via a Proxy (`lazy.ts`):
  - `DATAFORSEO_LOGIN && DATAFORSEO_PASSWORD` → real client.
  - Else `APIFY_TOKEN` → mock for everything **except** `serpRanks`, which uses the Apify Google Search Scraper.
  - Else full mock.
- **There is no monetary budget or quota tracking.** Cost is controlled by:
  - the 10-minute SyncJob cooldown per (client, kind) on manual refreshes (`startJob`)
  - the 10-minute audit cooldown on `SeoAudit.runAt` (`AUDIT_COOLDOWN_MS`)
  - the 7-day `KeywordResearch` cache, purged after 30 days by the daily job
  - a maximum of 100 keywords per add and 100 ideas per track call
  - a maximum of 5 competitor domains
  - a backlink list limit of 300
  - SERP batches of 20 (tracker), and within the real client 5 parallel single-task calls
  - the per-client daily cron fan-out
- **Real DataForSEO** (`real.ts`):
  - Basic auth, `POST https://api.dataforseo.com{path}`, 60 s timeout. `USD_TO_IDR = Number(env.DATAFORSEO_USD_IDR ?? 16000)`, used for `cpc` and `organicCost`.
  - Error mapping:
    - HTTP 401 → TOKEN_EXPIRED; 402/403 → PERMISSION; 429 → RATE_LIMIT; 404 → NOT_FOUND
    - envelope `status_code ≠ 20000`: 40200/40201 → PERMISSION; 40202 → RATE_LIMIT
    - task status: 40402 → NOT_FOUND; 402xx → PERMISSION
  - Endpoints:
    - `dataforseo_labs/google/keyword_ideas/live`
    - `keyword_suggestions/live` (questions: filter `search_volume > 0`, then regex `/^(cara|apa|bagaimana|kenapa|mengapa|berapa|apakah|dimana|di mana|kapan|siapa|how|what|why|where|when|who)\b/i`)
    - `related_keywords/live` (depth 2)
    - `historical_search_volume/live`, falling back to `keywords_data/google_ads/search_volume/live`
    - `bulk_keyword_difficulty/live`
    - `serp/google/organic/live/regular` (depth 100, one task per call)
    - `backlinks/summary/live`
    - `backlinks/timeseries_new_lost_summary/live` (last 30 days, monthly)
    - `backlinks/backlinks/live` (`order_by rank desc`)
    - `dataforseo_labs/google/domain_rank_overview/live`
    - `ranked_keywords/live`
    - `competitors_domain/live` (limit 10)
  - Normalisation: `domainRank = round(rank / 10)`; `toxicShare = min(1, spam / 100)`; SERP position = `rank_group ?? rank_absolute` of the first organic result where `host === target` or ends with `.{target}`.
- **Apify** (`src/lib/providers/apify/client.ts`):
  - `runActor` = `POST https://api.apify.com/v2/acts/{owner~name}/run-sync-get-dataset-items?token&timeout(120 default)&memory=1024&clean=true&format=json[&limit]`, with fetch timeout = `timeout + 15 s`.
  - Status mapping: 401/403 → TOKEN_EXPIRED; 402 → PERMISSION ("Kuota/kredit Apify habis"); 404 → NOT_FOUND; 408 → NETWORK; 429 → RATE_LIMIT.
  - SERP: one run per device, with all keywords newline-joined. Input `{maxPagesPerQuery: 1, resultsPerPage: 100, mobileResults, countryCode}` (location map: 2360→id, 2840→us, 2702→sg, 2458→my, 2826→gb, 2036→au, 2392→jp, 2356→in, default id). Timeout 180 s.
  - Default actors (overridable via `APIFY_ACTORS` JSON):
    - serp: `apify/google-search-scraper`
    - instagramProfile: `apify/instagram-profile-scraper`
    - instagramPosts: `apify/instagram-scraper`
    - tiktok: `clockworks/tiktok-scraper`
    - facebookPage: `apify/facebook-pages-scraper`
    - facebookPosts: `apify/facebook-posts-scraper`

## Audit and crawler (`src/features/seo/audit-core.ts`, `src/features/seo/actions.ts`, `src/lib/providers/crawler/*`, `src/lib/providers/pagespeed/index.ts`)

**`runAuditCore(propertyId, url?)`** (`audit-core.ts:40`)
1. Target = `url`, or `https://{domain}/` for `sc-domain:` properties, or `siteUrl`.
2. Creates `syncJob {kind: "SEO_AUDIT", status: "RUNNING", message: target}` and `seoCrawl {status: "RUNNING", maxPages: 30}`.
3. `allSettled` of `runPagespeed(mobile)`, `runPagespeed(desktop)` and `crawlSite(target, {maxPages: 30, timeBudgetMs: 45000})`.
4. Each successful PSI result → `seoAudit.create {strategy: MOBILE|DESKTOP, performance, seo, accessibility, bestPractices, lcpMs, inpMs, cls, fcpMs, ttfbMs, tbtMs, speedIndexMs, raw}`.
5. Crawl OK → `seoCrawlPage.createMany`, `seoIssue.createMany`, `seoCrawl` → SUCCESS with `pagesCrawled`. Crawl failed → `seoCrawl` FAILED with error.
6. `allFailed` = no PSI result **and** the crawl was rejected.
7. Job update: status FAILED if allFailed, else SUCCESS. Message `"PageSpeed {n}/2 strategi · Crawl {p} halaman, {i} isu"`. Error = `errorParts.join(" | ").slice(0,900)`.
8. Returns `{slug, target, allFailed, parts, errorParts, audits, pagesCrawled, issues}`.

**Actions in `src/features/seo/actions.ts`**
- **`runAudit(clientId, propertyId, url?)`** (:40)
  - Zod: `url` `z.string().url().optional()`. Any member may run it (no MANAGER check).
  - Cooldown: last `seoAudit.runAt` less than 10 min ago → "Audit terakhir dijalankan kurang dari 10 menit lalu. Coba lagi beberapa menit.".
  - Revalidates `/seo/audit` and `/seo`.
  - allFailed → "Audit gagal dijalankan: {first error}". Otherwise message "Audit selesai" or "Audit selesai sebagian" + parts.
- **`setAuditCadence({clientId, propertyId, cadence: enum NONE|WEEKLY|DAILY})`** (:77): any member (guard has no role check). Updates `seoProperty.auditCadence`.
- **`syncSeo(clientId)`** (:89): any member. Calls `syncSeoForClient(clientId, {days: 90})`.

**Health score** (`src/lib/metrics/seo.ts:233`)
- `base = perf*0.3 + seo*0.4 + a11y*0.15 + bestPractices*0.15`.
- `penalty = clamp(errors*2, 0, 20) + clamp(warnings*0.5, 0, 10)`.
- `score = round(clamp(base - penalty, 0, 100))`; null if there is no audit.
- Inputs (`src/features/seo/queries.ts:72`): the latest MOBILE audit, and issues from the latest SUCCESS crawl.

**PageSpeed** (`pagespeed/index.ts`)
- `GET https://www.googleapis.com/pagespeedonline/v5/runPagespeed?url&strategy&category=performance,seo,accessibility,best-practices[&key=PAGESPEED_API_KEY]`, 60 s timeout.
- Category score = `round(score × 100)`.
- Error codes: 429 → RATE_LIMIT; 403 → PERMISSION; 404 → NOT_FOUND.

**Crawler** (`crawler/index.ts`)
- BFS, same-origin only, concurrency 4, request timeout 10 s.
- `maxPages` default 30, hard cap 60. Time budget default 45 s; workers stop when less than 1.5 s is left.
- Respects robots.txt (UA token `whiteanalyticsbot`, UA string `"WHITEAnalyticsBot/1.0 (+https://white.id/bot)"`; longest match wins, allow wins ties).
- Sitemap seeds: up to 4 documents and 3 child sitemaps each, capped at `maxPages × 2`.
- Queue cap: `maxPages × 6`. Redirects followed manually, max 5.
- Binary extensions are skipped.
- Broken-link pass: already-seen 4xx/5xx targets, plus HEAD checks of up to 15 unchecked targets (falls back to GET on 405/501).
- Zero pages crawled → `ProviderError("crawler", "NETWORK", ...)`.

**Issue checks** (`analyze.ts`). Thresholds: `TITLE_MAX` 60, `TITLE_MIN` 30, `META_DESC_MAX` 160, `THIN_WORDS` 300, `SLOW_MS` 1500.
- Content checks run only when status < 400.

| Code | Severity |
|---|---|
| MISSING_TITLE | ERROR |
| TITLE_TOO_LONG | WARNING |
| TITLE_TOO_SHORT | WARNING |
| MISSING_META_DESCRIPTION | WARNING |
| META_DESCRIPTION_TOO_LONG | NOTICE |
| MISSING_H1 | ERROR |
| MULTIPLE_H1 | WARNING |
| IMG_MISSING_ALT | WARNING |
| MISSING_CANONICAL | NOTICE |
| NOINDEX (meta robots or X-Robots-Tag) | WARNING |
| THIN_CONTENT | WARNING |
| NO_STRUCTURED_DATA | NOTICE |
| MIXED_CONTENT | ERROR |
| SLOW_RESPONSE (loadMs > 1500, any status) | WARNING |
| REDIRECT_CHAIN (more than 1 hop) | WARNING |
| BROKEN_INTERNAL_LINK | ERROR |

---

## src/features/sync/policy.ts

- **`RETENTION_DAYS`** (:2) = `{ syncJobs: 90, readNotifications: 30, anyNotifications: 90, seoSnapshots: 760 }`.
- **`retentionCutoffs(now)`**: `now - days × 86_400_000` for each.
- **`planDailyRun(clients[{id, lastRunAt}], {concurrency, budgetMs, perClientMs})`** (:21)
  - Sorts by `lastRunAt` ascending (null counts as 0, so first).
  - `waves = max(1, floor(budgetMs / perClientMs))`; `capacity = waves × max(1, concurrency)`.
  - Returns `{run: sorted[0:capacity], deferred: rest}`.

## src/features/sync/retention.ts

**`runRetention(now)`** (:12), all deletes in parallel:

| Cutoff | Delete |
|---|---|
| 90 days | `syncJob.deleteMany {startedAt < cutoff}` |
| 30 days | `notification.deleteMany {readAt < cutoff}` |
| 90 days | `notification.deleteMany {createdAt < cutoff}` |
| 760 days | `rankSnapshot.deleteMany {date < cutoff}` |
| 760 days | `backlinkSnapshot.deleteMany {date < cutoff}` |
| 760 days | `domainSnapshot.deleteMany {date < cutoff}` |

- Returns `{syncJobs, notifications (read + old), rankSnapshots, backlinkSnapshots, domainSnapshots}`.
- `KeywordResearch` older than 30 days is deleted separately in `runSeoSuiteDaily`.

## src/features/sync/daily.ts

**`runDailySync({clientId?})`** (:24). Loads clients (one, or all), each `{id, slug}`, and processes each sequentially.

**Social block**
- For each platform in `[INSTAGRAM, FACEBOOK, TIKTOK]`:
  - Non-TIKTOK → `syncSocialMeta({clientId, platform})`.
  - If Apify is configured → `syncSocialPublic({clientId, platform, includeCompetitors: true})`.
- Results are aggregated into `updated`, `posts` and `failed` (`"{platform} @{user}: {err}"`). `real` = any `updated > 0`.
- If `real || failed.length`: `syncJob {kind: SOCIAL_SNAPSHOT, status: FAILED if updated==0 && failed.length else SUCCESS, finishedAt, message: "{updated} akun, {posts} post", error: failed.join("; ").slice(0,1000)}`.
- Otherwise `mockJob`: `syncJob {kind: SOCIAL_SNAPSHOT, status: SUCCESS, startedAt = finishedAt = now, message: "mock"}`.
- A thrown exception is recorded in the summary only (no SyncJob).

**SEO block**: `syncSeoForClient(id, {days: 30})`; the summary kind is SEO_GSC.

**Ads block**: `syncAdsForClient(id, {days: 30})`; the summary kind is ADS_INSIGHTS.

**Return** `{clients, summary: [{client: slug, kind, status: SUCCESS|FAILED, mode: real|demo, message}]}`.

## Per-module sync files

**`src/features/social/sync.ts`**
- **`syncSocialPublic({clientId, platform, includeCompetitors?, postLimit?})`** (:15)
  - Accounts for the client and platform; competitors are excluded only when `includeCompetitors === false`.
  - Per account, Apify `publicSocial.profile` and `posts` (limit = `postLimit ?? (competitor ? 20 : 30)`), then one `$transaction`:
    - `socialAccount.update {displayName, biography, avatarUrl}`
    - `socialSnapshot.upsert` on `(socialAccountId, today)` with `{followers, following, mediaCount}`
    - `socialPost.upsert` on `(socialAccountId, externalId)`; update sets `{caption, likes, comments, shares, saves, views, thumbnailUrl, syncedAt}`
  - Returns `{updated, posts, failed: [{username, error}]}`.
- **`syncSocialMeta({clientId, platform: INSTAGRAM|FACEBOOK, days=30, provider?, token?})`** (:85)
  - Own accounts only. `getConnectionToken(connectionId)` null → `skipped++`.
  - Window: `until = latestCompleteDay`, `since = until - (days - 1)`.
  - **INSTAGRAM**: `getInstagramAccount`, `getInstagramInsights(since, until)`, `getInstagramMedia(50)`.
    - Updates the account profile.
    - Daily snapshot upserts with `{reach, impressions, profileViews}`; create also sets followers, following, mediaCount.
    - Today's snapshot upsert with followers, following, mediaCount.
    - `socialPost` upserts with likes, comments, shares, saves, views, reach, impressions.
  - **FACEBOOK**: `getFacebookPage`, `getPageInsights`, `getPagePosts(50)`.
    - Daily snapshot `reach = impressionsUnique`; today's followers.
    - Posts with `mediaType "IMAGE"`, `likes = reactions`.
  - Returns `{updated, posts, skipped, failed}`.

**`src/features/seo/sync.ts`**
- **`syncSeoProperty(property, {days=30, provider?, token?})`** (:21)
  - No token → `{mode: "demo", ...}`; nothing is written.
  - Window: `latestCompleteDay - (days - 1)` to `latestCompleteDay`.
  - GSC `querySearchAnalytics` with dimensions `["date"]` → `seoDailyMetric.upsert (propertyId, date)`.
  - Then for each dim in `[query, page, country, device]`: dimensions `["date", dim]`, `rowLimit 5000` → `seoDimensionMetric.upsert (propertyId, date, dimension, key)` in `$transaction` chunks of 200.
    - Key normalisation: query and country lowercased, device uppercased.
    - Values: clicks and impressions rounded; ctr and position rounded to 2 decimals.
  - GA4, if `ga4PropertyId` is set (`properties/` prefix added if missing): `runGa4Report` with metrics `[sessions, engagedSessions, totalUsers, conversions, userEngagementDuration]`, dimensions `[date, sessionDefaultChannelGroup]`, limit 10000.
    - Aggregated per date; `organicSessions` = channel `"organic search"`; `avgEngagementSec = duration / sessions`.
    - `ga4DailyMetric.upsert`.
- **`syncSeoForClient(clientId, {days})`** (:87)
  - Per property, an exception becomes `{mode: "real", error}`.
  - Writes `syncJob {kind: "SEO_GSC", status: FAILED if error else SUCCESS, finishedAt, message: "mock" or "{daily} hari, {dims} baris dimensi · {siteUrl}", error}`.
  - If `ga4PropertyId` is set, also writes `syncJob {kind: "SEO_GA4", same status, message: "mock" or "{ga4} hari · {ga4PropertyId}"}`.
  - Returns `{properties, real, results}`.

**`src/features/ads/sync.ts`**
- **`syncAdAccount(account, {days=30, adapter?})`** (:17)
  - No token → demo. Otherwise `getMetaMarketing(token)`.
  - Upserts `adCampaign (adAccountId, externalId)`, `adSet (campaignId, externalId)` and `ad (adSetId, externalId)`, skipping orphans.
  - Ad-level daily insights (`timeIncrement 1`) are aggregated per (date, campaign, adset, ad). Existing rows are matched by key and updated, otherwise created. `$transaction` chunks of 100. Rows without an ad id are skipped.
  - Campaign demographics (breakdowns age and gender over the window): `findFirst {adAccountId, dateFrom, dateTo, campaignId, age, gender}`, then update or create.
  - Sets `adAccount.lastSyncedAt`.
- **`syncAdsForClient`** (:102): per account, `syncJob {kind: "ADS_INSIGHTS", status, message: "mock" or "{c} kampanye, {a} iklan, {d} hari · {externalId}", error}`.

**Manual sync actions that also write SyncJobs**
- **`src/features/social/actions.ts:182` `syncSocial({clientId, platform enum})`**
  - Any member, including VIEWER. No own accounts → `s.syncNoAccount` NOT_FOUND.
  - Tries, in order:
    1. Meta, if any account has a connection: SOCIAL_SNAPSHOT job with message `"... (platform · Meta)"`.
    2. Apify: SOCIAL_SNAPSHOT job.
    3. Demo extrapolation of the last 8 snapshots with deterministic jitter (`mulberry32(hashSeed("sync:{acc}:{date}"))`): SOCIAL_SNAPSHOT SUCCESS.
  - A thrown exception writes a FAILED job with error `t.errors[code]`.
- **`src/features/ads/actions.ts:90` `syncAds(clientId)`**
  - ADMIN or MANAGER only.
  - Real path when a connection exists: `syncAdsForClient(days: 90)`.
  - Demo path: touches `lastSyncedAt` and writes an ADS_INSIGHTS SUCCESS job.
- **`src/features/ads/actions.ts:31` `importAdsCsv`**: maximum 10 MB; writes an `ADS_CSV_IMPORT` job with SUCCESS or FAILED.

## SyncJob kind list (every value written)

| Kind | Status values | Written by |
|---|---|---|
| `SOCIAL_SNAPSHOT` | SUCCESS/FAILED | `sync/daily.ts`, `social/actions.ts` |
| `SEO_GSC` | SUCCESS/FAILED | `seo/sync.ts` |
| `SEO_GA4` | SUCCESS/FAILED | `seo/sync.ts` |
| `SEO_AUDIT` | RUNNING → SUCCESS/FAILED | `seo/audit-core.ts` |
| `ADS_INSIGHTS` | SUCCESS/FAILED | `ads/sync.ts`, `ads/actions.ts` |
| `ADS_CSV_IMPORT` | SUCCESS/FAILED | `ads/actions.ts` |
| `PUBLISH` | SUCCESS/FAILED | `publishing/scheduler.ts` |
| `SEO_SUITE_RANKS` | RUNNING → SUCCESS/FAILED | `seo-suite/actions.ts` |
| `SEO_SUITE_BACKLINKS` | RUNNING → SUCCESS/FAILED | `seo-suite/actions-domains.ts` |
| `SEO_SUITE_DOMAINS` | RUNNING → SUCCESS/FAILED | `seo-suite/actions-domains.ts` |
| `SEO_SUITE_DAILY` | RUNNING → SUCCESS/FAILED | `seo-suite/daily.ts` |

PENDING is never written explicitly. The schema comment at `schema:610` lists only six kinds and is stale.

---

## src/app/api/cron/daily/route.ts — daily cron dispatcher

- Route config: `dynamic = "force-dynamic"`, `maxDuration = 300`. Scheduled `"0 2 * * *"` in `vercel.json`.
- Auth: `process.env.CRON_SECRET` must be set **and** the `authorization` header must equal exactly `Bearer ${CRON_SECRET}`. Otherwise 401 `{error: "UNAUTHORIZED"}`.
- Constants: `CONCURRENCY = 4`, `BUDGET_MS = 270_000`, `PER_CLIENT_MS = 90_000`. So waves = 3 and capacity = 12 clients per run.
- `attempt(fn)` wraps any thrown error as `{error: message}`.

**With `?client=<id>`** (per-client worker)
- `sync = attempt(runDailySync({clientId}))`, then `seoSuite = attempt(runSeoSuiteDaily({now: startedAt, clientId}))`, sequentially.
- Response 200 `{ok: true, client, sync, seoSuite}`. It is always 200 unless the function itself crashes.

**Without `?client=`** (dispatcher)
1. `retention = attempt(runRetention(startedAt))`.
2. In parallel: `client.findMany {id, slug}` and `syncJob.groupBy({by: ["clientId"], where: {kind: "SEO_GSC"}, _max: {startedAt}})`. The latter becomes `lastRunAt`.
   - Clients with no SEO properties never get an SEO_GSC job, so they always sort first.
3. `planDailyRun(...)`.
4. Four workers pull from the queue. Each fetches `${url.origin}/api/cron/daily?client=${encodeURIComponent(id)}` with the same Bearer header and `cache: "no-store"`. Each result is `{client: slug, ok: status === 200, status (0 on fetch error), ms}`.
5. Response `{ok: all ok, ranAt: ISO, retention, clients: total count, results, deferred: [slugs]}`.

## src/app/api/cron/publish/route.ts

- Route config: `force-dynamic`, `maxDuration = 300`. `vercel.json` schedule `"*/5 * * * *"`; a comment notes Vercel Hobby only allows daily, so an external scheduler should be pointed here instead.
- Same `CRON_SECRET` Bearer check (401 `{error: "UNAUTHORIZED"}`).
- Calls `publishDuePosts({now})`, which uses the default limit of 20.
- Returns `{ok: true, ranAt, attempted, published, failed}`.

---

## Behaviours to preserve or decide on in the Go port

**Silent mock publishing.** In `scheduler.ts:76`, if Meta app credentials exist but a target account has no `connectionId`, the post is published with the mock publisher and reported as success. The activity `mock` flag stays false in that case.

**Permissions**
- VIEWER can call `refreshRanks`, `refreshBacklinks`, `refreshDomains`, `syncSocial` and `addPostComment`.
- Any member (VIEWER included) can call `runAudit`, `setAuditCadence` and `syncSeo`.

**Cooldowns and counters**
- The suite refresh cooldown counts any SyncJob of that kind regardless of status, including RUNNING and cron-written jobs.
- `startJob`'s throttle failure has no `code`.
- `PostTarget.attempts` is never reset.

**Misleading counts in summaries**
- When `runDailySync` finds zero SEO properties or zero ad accounts, the summary reports "demo/mock" but no SyncJob is written.
- `refreshBacklinks` never marks links that disappeared from the provider list as lost.
- `trackRanks` counts `updated` before the upsert succeeds.

**No auth on queries or media**
- Publishing and seo-suite query functions do no authorisation; the calling pages are expected to use `rbac` helpers.
- The media route is fully public.
