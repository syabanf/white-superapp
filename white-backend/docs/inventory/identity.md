# Backend behaviour inventory for the Go port (white-analytics)

Repo root: `/Users/syabanf/Documents/works/WHITE-Superapps-main/white-analytics`. All paths below are relative to it.

**Stack:**
- next-auth `5.0.0-beta.32`
- bcryptjs `^3.0.3`
- zod `^4.4.3`. This is Zod v4: default English error messages and v4 `.url()`/`.email()` semantics.
- Prisma 7.9 with `@prisma/adapter-pg` (`src/lib/db.ts`, uses `DATABASE_URL`).

**Server actions in general:**
- They return `ActionResult`. The UI shows the `error` string to users.
- `revalidatePath` is a Next cache call with no DB effect. In Go it becomes a no-op or a cache bust.
- A `redirect()` inside an action means an HTTP redirect, not a JSON result.

---

## GLOBAL / CROSS-CUTTING

### src/lib/action-result.ts
- **Types:**
  - `ActionResult<T> = {ok:true, data:T} | {ok:false, error:string, code?: ProviderErrorCode|"VALIDATION"|"UNAUTHORIZED", fieldErrors?: Record<string,string[]>}`
  - `ProviderErrorCode = "RATE_LIMIT"|"TOKEN_EXPIRED"|"PERMISSION"|"NOT_FOUND"|"NETWORK"|"UNKNOWN"`
- `ok(data)` (l.11) returns `{ok:true,data}`.
- `fail(error, code?, fieldErrors?)` (l.15) returns `{ok:false,error,code,fieldErrors}`. `code` and `fieldErrors` may be undefined.
- `class ProviderError(provider, code, message, {requiresReconnect?, cause?})` (l.20):
  - `requiresReconnect` defaults to true when code is TOKEN_EXPIRED or PERMISSION.
  - `isProviderError(e)` is an instanceof check.
- **`fieldErrors` construction (`zodErrors`):** key = `String(issue.path[0] ?? "form")`, value = list of issue messages. It exists as a copy in admin/actions.ts, clients/action-helpers.ts and clients/wizard-actions.ts. Nested errors collapse to the top-level key, e.g. `memberships`.
- **Error strings** (`src/i18n/id.ts` l.523+, `t.errors`):

  | Key | Message |
  |---|---|
  | RATE_LIMIT | "Batas permintaan API tercapai. Coba lagi beberapa menit." |
  | TOKEN_EXPIRED | "Koneksi akun kedaluwarsa. Hubungkan ulang akun." |
  | PERMISSION | "Izin tidak mencukupi. Periksa akses akun yang dihubungkan." |
  | NOT_FOUND | "Data tidak ditemukan." |
  | NETWORK | "Gangguan jaringan. Periksa koneksi dan coba lagi." |
  | UNKNOWN | "Terjadi kesalahan tak terduga." |
  | VALIDATION | "Periksa kembali data yang dimasukkan." |
  | UNAUTHORIZED | "Anda tidak punya akses." |

- **Other strings used:**

  | Key | Message |
  |---|---|
  | `t.auth.invalid` | "Email atau kata sandi salah" |
  | `t.auth.inactive` | "Akun Anda dinonaktifkan. Hubungi admin." (defined but never returned) |
  | `t.auth.unauthorized` | "Anda tidak punya akses ke halaman ini." |
  | `t.admin.emailTaken` | "Email sudah terdaftar" |
  | `t.admin.cannotDeleteSelf` | "Anda tidak dapat menghapus akun sendiri." |
  | `t.clients.slugTaken` | "Slug sudah dipakai" |
  | `t.share.pinWrong` | "PIN salah" |
  | `t.share.pinTooMany` | "Terlalu banyak percobaan. Coba lagi beberapa menit." |
  | `t.share.notFound` | "Dashboard tidak ditemukan" |

- **Admin strings** (`src/features/admin/strings.ts`):
  - nameRequired = "Nama wajib diisi"
  - emailInvalid = "Email tidak valid"
  - passwordMin = "Kata sandi minimal 8 karakter"
  - cannotEditSelf = "Anda tidak dapat menonaktifkan atau menurunkan peran akun sendiri."
- **Client strings** (`src/features/clients/strings.ts`):
  - wizard.pickOneChannel (l.47) = "Pilih minimal satu kanal."
  - form.nameRequired (l.86) = "Nama klien wajib diisi"
  - form.slugRequired = "Slug wajib diisi"
  - form.slugInvalid = "Slug hanya boleh huruf kecil, angka, dan tanda hubung"
  - form.websiteInvalid (l.90) = "URL website tidak valid (awali dengan https://)"
  - settings.duplicate (l.127) = "Sudah terdaftar untuk klien ini"
  - settings.pinInvalid (l.139) = "PIN harus 6 digit angka"
  - settings.modulesMin (l.141) = "Pilih minimal satu modul"
- **Unique violations:** detected as Prisma `P2002` → Postgres unique_violation 23505.

### Passwords / hashing
- bcryptjs, cost **10**, everywhere: `hash(pw, 10)`, verified with `compare`.
- Used for user passwords (`User.passwordHash`) and share PINs (`Client.sharePinHash`).
- Seed users (prisma/seed.ts l.101-107):
  - admin@mixist.id / admin12345 (ADMIN)
  - tim@mixist.id / member12345 (MEMBER)
  - aloha.mrwhite@gmail.com (ADMIN)
  - The first seeded client has share PIN "123456".

### Enums (prisma/schema.prisma)
- `UserRole`: ADMIN, MEMBER (default MEMBER)
- `ClientMemberRole`: MANAGER, VIEWER (default VIEWER)
- `ConnectionProvider`: META, GOOGLE
- `SocialPlatform`: INSTAGRAM, FACEBOOK, TIKTOK
- `ModuleKey`: SOCIAL, SEO, ADS (Client.shareModules is `ModuleKey[]`, default `[SOCIAL,SEO,ADS]`)
- Constants (`src/features/clients/constants.ts`):
  - CLIENT_CURRENCIES = IDR, USD, SGD, AUD, EUR
  - CLIENT_TIMEZONES = Asia/Jakarta, Asia/Makassar, Asia/Jayapura, Asia/Singapore, Asia/Kuala_Lumpur, Australia/Sydney
  - SHARE_MODULE_KEYS = SOCIAL, SEO, ADS
  - SOCIAL_PLATFORMS = INSTAGRAM, FACEBOOK, TIKTOK
  - MEMBER_ROLES = MANAGER, VIEWER

### Relevant models (key fields)
- **User**
  - Fields: id cuid, email unique, name, passwordHash, role, isActive (default true), createdAt, updatedAt.
  - On user delete:
    - Cascades to ClientMember and Notification.
    - Sets null on Client.createdById, Report.createdById, MediaAsset.uploadedById, ContentPost.createdById/approvedById, PostComment.userId, PostActivity.userId.
- **Client**
  - Fields: id, name, slug unique, description (default ""), logoUrl?, websiteUrl?, industry?, currency (default "IDR"), timezone (default "Asia/Jakarta"), shareEnabled (default false), sharePinHash?, shareModules, createdById?.
  - Every child relation cascades on delete.
- **ClientMember:** PK (userId, clientId), role, createdAt. Index on clientId.
- **Connection**
  - Fields: id, clientId, provider, accountId, displayName, accessTokenEnc, refreshTokenEnc?, expiresAt?, scopes `String[]` (default []), meta Json?.
  - Unique (clientId, provider, accountId).
  - SocialAccount, SeoProperty and AdAccount have `connectionId` with SetNull on delete.
- **SocialAccount:** id, clientId, platform, externalId, username, displayName, avatarUrl?, biography?, isCompetitor (default false), connectionId?. Unique (clientId, platform, externalId).
- **SeoProperty:** id, clientId, siteUrl, ga4PropertyId?, connectionId?, locationCode (default 2360), languageCode (default "id"), auditCadence (default NONE). Unique (clientId, siteUrl).
- **AdAccount:** id, clientId, externalId, name, currency (default "IDR"), connectionId?, lastSyncedAt?. Unique (clientId, externalId).
- **Notification:** id, userId, clientId?, type, title, body (default ""), href?, readAt?, createdAt. Index (userId, readAt, createdAt).
- **IntegrationSetting:** provider (string PK), publicId?, secretEnc?, meta Json?, verifiedAt?, updatedById? (plain string, no FK), createdAt, updatedAt.
- **AppSetting:** key (string PK), value Json, updatedAt.
- **SyncJob:** clientId?, kind, status, startedAt, finishedAt.
- **Date columns** in SocialSnapshot, SeoDailyMetric and AdDailyInsight are `@db.Date`.

---

## AUTH

### src/auth.config.ts (edge-safe config)
- `pages.signIn = "/login"`.
- **Session:** `session: { strategy: "jwt", maxAge: 1209600 }` (14 days); `trustHost: true`.
  - No database sessions and no adapter.
  - Auth.js defaults apply (not in repo code):
    - Encrypted JWE cookie `authjs.session-token`, or `__Secure-authjs.session-token` over https.
    - Key derived from `AUTH_SECRET`.
    - updateAge 24h.
- **`callbacks.authorized`** (l.14): public when any of these hold, otherwise requires `auth.user`:
  - `pathname === "/login"`
  - `startsWith("/share/")`
  - `startsWith("/api/auth")`
  - `startsWith("/api/cron")`
  - `startsWith("/api/share")`
  - Effectively unused, because proxy.ts passes its own handler. With a custom handler Auth.js ignores a boolean false from `authorized`.
- **`callbacks.jwt`** (l.26): on sign-in (`user` present) it sets:
  - `token.id = user.id`
  - `token.role = user.role ?? "MEMBER"`
  - `token.name = user.name`
- **JWT token contents:** Auth.js default claims `name`, `email`, `sub` (= user id), `iat`, `exp`, `jti`, plus custom `id` and `role`.
- **`callbacks.session`** (l.34): `session.user.id = token.id`; `session.user.role = token.role ?? "MEMBER"`.
  - Session shape: `{user:{id, role, name, email}, expires}`.
  - Types are in `src/types/next-auth.d.ts`.

### src/auth.ts
- Credentials provider "Email & kata sandi". Schema `credentialsSchema` (l.8): `email: z.string().email()`, `password: z.string().min(1)`.
- **`authorize(raw)`** (l.22):
  1. safeParse; failure returns null.
  2. `email = parsed.email.toLowerCase().trim()`. Zod email validation runs before the trim, so surrounding spaces fail.
  3. `db.user.findUnique({where:{email}})`.
  4. Missing user or `!isActive` returns null.
  5. `bcrypt.compare(password, passwordHash)`; false returns null.
  6. Returns `{id, email, name, role}`.
- Exports `handlers, auth, signIn, signOut`.

### src/app/api/auth/[...nextauth]/route.ts:1-3
- `GET`/`POST` = NextAuth handlers. These are the standard Auth.js endpoints under `/api/auth/*`: csrf, signin/credentials, callback/credentials, session, signout, providers.

### src/features/auth/actions.ts
- **`loginAction(_prev: LoginState, formData)`** (l.16), with `LoginState = {error?:string}|undefined`.
  - Schema (l.10): `email: z.string().email()`, `password: z.string().min(1)`, `callbackUrl: z.string().optional()`. Empty callbackUrl becomes undefined.
  - Invalid input returns `{error: t.auth.invalid}`.
  - `redirectTo` = callbackUrl if it starts with "/", otherwise "/". `//host` would pass this check; Auth.js's default redirect callback restricts to same origin.
  - Calls `signIn("credentials", {email, password, redirectTo})`.
    - On success Auth.js sets the session cookie and redirects (the thrown redirect is rethrown).
    - On `AuthError` (CredentialsSignin, wrong or inactive account) it returns `{error:"Email atau kata sandi salah"}`. There is no separate inactive message.
- **`logoutAction()`** (l.40): `signOut({redirectTo:"/login"})`, which clears the cookie.

### src/proxy.ts (Next 16 middleware)
- **Matcher:** `/((?!_next/static|_next/image|favicon.ico|logo.*|.*\.(?:png|jpg|jpeg|svg|webp|ico|txt|xml)$).*)`
- Cookie-only JWT check, no DB.
- **Public paths:** same list as `authorized`: `/login`, `/share/*`, `/api/auth*`, `/api/cron*`, `/api/share*`.
- **Not logged in on a non-public path:** 302 to `/login`, with `?callbackUrl=<pathname+search>` unless that equals "/".
- **Logged in on `/login`:** redirect to "/".
- **Logged in, `restoreSavedFilters`** (l.38):
  - Calendar route: regex `^/clients/[^/]+/publish/?$`. Uses cookie `white_calendar_filters` with keys view, date, platform, status.
  - Date-range route: `/`, or `^/clients/[^/]+(/|$)` excluding `/settings`, `/publish` and `/seo/(research|backlinks|competitors)`. Uses cookie `white_date_range` with keys from, to, preset, compare.
  - If the URL already has any of the keys, nothing happens.
  - Otherwise it does `JSON.parse(decodeURIComponent(cookie))` and sets each value that is a non-empty string other than "ALL".
  - Redirects only if the query changed. Parse errors are ignored.
- Otherwise `NextResponse.next()`.

---

## src/lib/rbac.ts (authorization layer)
- **`SessionUser = {id, email, name, role}`**
- **`getSessionUser()`** (l.20, cached per request):
  - Reads `auth()`, then `session.user.id`.
  - Runs `db.user.findUnique({where:{id}, select:{id,email,name,role,isActive}})`.
  - Missing or inactive returns null (treated as signed out). Deactivation takes effect on the next request despite JWT.
- **`requireUser()`** (l.33): null → `redirect("/login")`.
- **`requireAdmin()`** (l.39): role ≠ ADMIN → `redirect("/?unauthorized=1")`.
- **`isAdmin(user)`** (l.45): returns `role === "ADMIN"`.
- **`requireClientAccess(slug)`** (l.60, cached):
  1. requireUser.
  2. `client.findUnique({where:{slug}})`; missing → `redirect("/clients?missing=1")`.
  3. ADMIN returns `{client, role:"ADMIN", canManage:true}`.
  4. Otherwise `clientMember.findUnique({where:{userId_clientId}})`; missing → `redirect("/?unauthorized=1")`.
  5. Returns `{client, role: membership.role, canManage: role==="MANAGER"}`.
- **`listAccessibleClients()`** (l.73):
  - ADMIN: `client.findMany({orderBy:{name:"asc"}})`.
  - Otherwise: `where:{members:{some:{userId}}}`, also ordered by name asc.
- **Membership model:**
  - Global role ADMIN sees and manages everything.
  - MEMBER sees only clients where a ClientMember row exists.
  - MANAGER membership = can manage the client (settings, members, share, resources). VIEWER = read-only.
- **Action-level checks** return `fail(UNAUTHORIZED)` and never redirect:
  - "admin actor" = `getSessionUser()` non-null with role ADMIN.
  - `manageContext` is defined below.

---

## src/lib/crypto.ts
- **Algorithm:** AES-256-GCM.
- **Key derivation `key()`** (l.9):
  - Missing `process.env.ENCRYPTION_KEY` throws Error("ENCRYPTION_KEY belum diset").
  - Base64-decode it; if the result is exactly 32 bytes, use it.
  - Otherwise key = SHA-256(raw env string as UTF-8).
- **`encrypt(plain)`** (l.17):
  - IV = 12 random bytes.
  - Output = `base64(iv[12] || authTag[16] || ciphertext)`, standard base64 with padding. No AAD.
- **`decrypt(payload)`** (l.25): splits `iv = buf[0:12]`, `tag = buf[12:28]`, `data = buf[28:]`. Throws on auth failure.
- **`randomPin()`** (l.36): `String(floor(100000 + Math.random()*900000))`, a 6-digit number. Not cryptographic; not used in the scoped files.

## src/lib/notify.ts
- **`notify(userIds, {type, title, body?, href?, clientId?})`** (l.18):
  - Dedupes ids and drops falsy ones; if none remain, returns 0.
  - `notification.createMany` with one row per user: `{userId, clientId, type, title, body: body ?? "", href}`.
  - Returns the count.
  - `type` is a free-form string, e.g. POST_REVIEW, POST_APPROVED, POST_PUBLISHED, POST_FAILED, RANK_DROP, BACKLINK_LOST.
- **`clientAudience(clientId, except?)`** (l.35): union of active ADMIN users and the client's members whose user is active, minus `except`.
- **`clientApprovers(clientId, except?)`** (l.46): union of active ADMINs and the client's MANAGER members whose user is active, minus `except`.
- None of the scoped features (auth, admin, clients, notifications, setup, share, guide) call `notify`. It is used by other modules.

---

## ADMIN — src/features/admin

### actions.ts
- **Auth (`requireAdminActor`):** `getSessionUser()` must have role ADMIN, otherwise `fail("Anda tidak punya akses.","UNAUTHORIZED")`.
- **Schemas:**
  - `membershipSchema`: `{clientId: string min1, role: enum MANAGER|VIEWER}`.
  - `baseUserSchema`:
    - `name`: trim, min1 ("Nama wajib diisi"), max80
    - `email`: trim, toLowerCase, email ("Email tidak valid")
    - `role`: enum ADMIN|MEMBER
    - `isActive`: boolean
    - `memberships`: array of membershipSchema, default []
  - `createUserSchema` = base + `password`: string min8 ("Kata sandi minimal 8 karakter"), max100.
  - `updateUserSchema` = base + `password`: `"" | string min8 max100`, default "".
  - `UserFormInput` = input type of updateUserSchema.

- **`createUserAction(input)`** (l.56), returns `ActionResult<{id}>`:
  1. Auth check, then validate. Failure → `fail(VALIDATION msg, "VALIDATION", fieldErrors)`.
  2. `user.findUnique({where:{email}})`; if it exists → `fail("Email sudah terdaftar","VALIDATION",{email:[same]})`.
  3. `hash(password,10)`; `user.create` with `{name, email, passwordHash, role, isActive, memberships:{create:[{clientId, role}]}}` (nested insert of ClientMember).
  4. P2002 → emailTaken. This also fires if the memberships contain a duplicate clientId (composite PK). Any other error → `fail(UNKNOWN)` with no code (e.g. bad clientId FK).
  5. `revalidatePath("/admin/users")`, then `ok({id})`.
  - Memberships are not stripped for ADMIN role.

- **`updateUserAction(userId, input)`** (l.84), returns `ActionResult<void>`. Checks in order:
  1. Auth check.
  2. Validate.
  3. `user.findUnique({id})`; missing → `fail(NOT_FOUND,"NOT_FOUND")`.
  4. Self-protection: if `userId === actor.id` and (`!isActive` or `role !== "ADMIN"`) → `fail(cannotEditSelf,"VALIDATION")`.
  5. Email owner check: if another user has the email → emailTaken with fieldErrors.email.
  6. Transaction:
     - `user.update` with `{name, email, role, isActive}`, plus `passwordHash: hash(pw,10)` only when the password is non-empty.
     - `clientMember.deleteMany({where:{userId}})`.
     - If memberships exist, `clientMember.createMany({data:[{userId, clientId, role}], skipDuplicates:true})`.
  7. P2002 → emailTaken; other errors → UNKNOWN.
  8. `revalidatePath("/admin/users")`, then `ok(undefined)`.

- **`setUserActiveAction(userId, isActive: boolean)`** (l.127):
  1. Auth check. If self and `!isActive` → cannotEditSelf with "VALIDATION".
  2. `user.update({where:{id}, data:{isActive}})`. Any error → `fail(NOT_FOUND,"NOT_FOUND")`.
  3. revalidate `/admin/users`, then ok.

- **`deleteUserAction(userId)`** (l.140):
  1. Auth check. If self → `fail("Anda tidak dapat menghapus akun sendiri.","VALIDATION")`.
  2. findUnique; missing → NOT_FOUND.
  3. `user.delete` (cascade and set-null as listed under the models above).
  4. revalidate `/admin/users`, then ok. Delete errors are not caught.

### queries.ts
- **`listUsersWithClients()`** (l.17), cached:
  - Calls `requireAdmin()`, which redirects on failure.
  - `user.findMany({orderBy:{createdAt:"asc"}, include:{memberships:{include:{client:{select:{id,name,slug}}}, orderBy:{client:{name:"asc"}}}}})`.
  - Returns `AdminUserRow[] = {id, name, email, role, isActive, createdAt ISO, clients:[{id, name, slug, role: MANAGER|VIEWER}]}`.
- **`listClientOptions()`** (l.40): `requireAdmin()`, then `client.findMany({orderBy:{name:"asc"}, select:{id,name,slug}})`.
- Page `src/app/(app)/admin/users/page.tsx` calls `requireAdmin()`, then both queries.

---

## CLIENTS — src/features/clients

### action-helpers.ts (not a server action module)
- `zodErrors`, `isUniqueViolation` (P2002).
- `emptyToNull(v)`: trims; empty string becomes null.
- **`manageContext(clientId)`** (l.33):
  1. `getSessionUser()`; null → null.
  2. `client.findUnique({id}, select slug)`; missing → null.
  3. ADMIN → `{userId, isAdmin:true, clientId, slug}`.
  4. Otherwise `clientMember.findUnique(userId_clientId)` must have role MANAGER, else null.
  - Callers map null to `fail(UNAUTHORIZED,"UNAUTHORIZED")`. A missing client therefore reports UNAUTHORIZED, not NOT_FOUND.
- **`revalidateClientPaths(slug)`:** revalidates `/`, `/clients`, `/clients/{slug}` and `/clients/{slug}/settings`.

### actions.ts
- **`clientFormSchema`** (l.28):
  - `name`: trim, min1 (nameRequired), max80
  - `slug`: trim, toLowerCase, min2 (slugRequired), max60, regex `^[a-z0-9]+(?:-[a-z0-9]+)*$` (slugInvalid)
  - `description`: trim, max500, default ""
  - `industry`: trim, max60, default ""
  - `websiteUrl`: `"" | string trim url (websiteInvalid)`, default "". Zod v4 `.url()` accepts any scheme that `new URL` parses.
  - `currency`: enum CLIENT_CURRENCIES
  - `timezone`: enum CLIENT_TIMEZONES

- **`createClientAction(input)`** (l.46):
  1. Auth: session user must be ADMIN, else UNAUTHORIZED.
  2. Validate. Slug pre-check via `client.findUnique({slug})`; taken → `fail("Slug sudah dipakai","VALIDATION",{slug:[…]})`.
  3. `client.create` with `{name, slug, description, industry: emptyToNull, websiteUrl: emptyToNull, currency, timezone, createdById: user.id}`. shareEnabled and shareModules use DB defaults (false; [SOCIAL,SEO,ADS]).
  4. P2002 → slugTaken; other errors → UNKNOWN.
  5. revalidate `/` and `/clients`, then **redirect `/clients/{slug}`**. No ok result is returned.

- **`updateClientAction(clientId, input)`** (l.76), returns `{slug}`:
  1. `manageContext` (ADMIN or MANAGER). Validate.
  2. If the slug changed, run the pre-check (slugTaken).
  3. `client.update` with the same fields as create, minus createdById.
  4. P2002 → slugTaken; else UNKNOWN.
  5. `revalidateClientPaths(old)`, plus `(new)` if changed. Returns `ok({slug})`.

- **`deleteClientAction(clientId)`** (l.108):
  1. ADMIN only.
  2. findUnique; missing → NOT_FOUND.
  3. `client.delete`, which cascades to all children.
  4. revalidate `/` and `/clients`, then ok.

- **`memberSchema`:** `{userId: string min1, role: enum MANAGER|VIEWER}`.
- **`addMemberAction(clientId, {userId, role})`** (l.126):
  1. manageContext. Validate.
  2. `user.findUnique({id})`; missing → NOT_FOUND. isActive is not checked.
  3. `clientMember.upsert` on `userId_clientId`: create `{userId, clientId, role}`, update `{role}`.
  4. `revalidateClientPaths`, then ok.
- **`updateMemberRoleAction(clientId, {userId, role})`** (l.142): manageContext, validate, then `clientMember.update({where:{userId_clientId}, data:{role}})`. Any error → NOT_FOUND. Then revalidate and ok.
- **`removeMemberAction(clientId, userId)`** (l.159):
  - manageContext. No validation.
  - `clientMember.delete(userId_clientId)`; error → NOT_FOUND.
  - Then revalidate and ok.
  - A MANAGER can remove themselves or other managers; nothing guards against it.

- **`shareSchema`** (l.173):
  - `shareEnabled`: boolean
  - `pin`: `"" | regex ^\d{6}$` (pinInvalid)
  - `modules`: array of enum SOCIAL|SEO|ADS, min1 (modulesMin)
- **`updateShareSettingsAction(clientId, input)`** (l.181):
  1. manageContext. Validate.
  2. `sharePinHash = pin === "" ? null : bcrypt(pin,10)`. **An empty PIN clears any existing PIN**; it does not keep it.
  3. `client.update({shareEnabled, sharePinHash, shareModules: modules})`.
  4. `revalidateClientPaths` plus `/share/{slug}`, then ok.
  - Re-saving the same PIN produces a new salted hash, which invalidates existing unlock cookies.

### wizard-actions.ts
- **`profileSchema`** (l.29): identical to clientFormSchema.
- **`wizardCreateClientAction(input, types: ProjectType[])`** (l.57):
  1. ADMIN only, else UNAUTHORIZED.
  2. Validate types with `projectTypesSchema = z.array(z.enum(["SEO","SOCIAL","ADS"])).min(1)` (duplicates not removed). Failure → `fail("Pilih minimal satu kanal.","VALIDATION")`.
  3. Validate profile; failure → VALIDATION with fieldErrors.
  4. Slug pre-check → slugTaken.
  5. `client.create` with `{…profile fields, shareEnabled:false, shareModules: types, createdById}` (select id).
  6. If types include "SEO" and a website is non-empty: `seoProperty.create({clientId, siteUrl: website})`. Not in a transaction; if this throws, the client row remains.
  7. P2002 → slugTaken; other errors → UNKNOWN.
  8. revalidate `/` and `/clients`, then **redirect `/clients/new?step=sumber&client={slug}`**.

### wizard.ts (pure step model, used by the `/clients/new` page)
- **Steps:** `["jenis","profil","sumber","akses","selesai"]`.
  - needsClient: sumber, akses, selesai.
  - needsTypes: profil.
  - `wizardStepSchema` falls back to "jenis" on invalid input.
- **`parseTypes("SEO,ADS")`:** splits on comma, trims, uppercases, returns in canonical order SEO, SOCIAL, ADS, drops invalid entries.
- **`resolveStep(requested, {hasClient, hasTypes})`:**
  - needsClient and no client → "profil" if hasTypes, else "jenis".
  - needsTypes and no types → "jenis".
  - Step without needsClient while the client exists → "sumber".
  - Otherwise the requested step.
- **`wizardHref(step, {slug?, types?})`:** `/clients/new?step=…&types=…&client=slug`.
- `websiteRequiredFor(types)` = includes SEO (UI-only; the server schema keeps website optional).
- `sourceSectionsFor(types)` returns `{social, seo, ads}` booleans.
- **Page gating** (`src/app/(app)/clients/new/page.tsx`):
  - `requireUser`; non-ADMIN → `redirect("/clients?unauthorized=1")`.
  - Looks up the client by `?client=` slug.
  - Loads `getClientSettings` for steps sumber, akses, selesai, and `listUserOptions` for akses.

### actions-resources.ts (all use manageContext → UNAUTHORIZED; all end with `revalidateClientPaths` and `ok(undefined)`)
- **`disconnectConnectionAction(clientId, connectionId)`** (l.22): `connection.findUnique(id)`; missing or clientId mismatch → NOT_FOUND. Then `connection.delete`, which sets null on linked social, SEO and ad rows' connectionId.
- **`simulateConnectionAction(clientId, provider)`** (l.35), demo mode:
  - provider is validated with `z.enum(["META","GOOGLE"])`; failure → VALIDATION.
  - accountId = "demo-meta" or "demo-google".
  - Payload: `{displayName:"Akun demo", accessTokenEnc: encrypt("mock"), refreshTokenEnc:null, expiresAt: now+60 days, scopes}`.
    - META_SCOPES = [pages_show_list, pages_read_engagement, instagram_basic, instagram_manage_insights, business_management, ads_read]
    - GOOGLE_SCOPES = [https://www.googleapis.com/auth/webmasters.readonly, https://www.googleapis.com/auth/analytics.readonly, openid, email]
  - `connection.upsert` on `clientId_provider_accountId`: create `{clientId, provider, accountId, …payload}`, update with the payload.
- **`socialAccountSchema`:**
  - `platform`: enum INSTAGRAM|FACEBOOK|TIKTOK
  - `username`: trim, min1 (message "Nama klien wajib diisi"), max60, then transforms: strip leading "@"s, lowercase
  - `displayName`: trim, max80, default ""
- **`addSocialAccountAction(clientId, {platform, username, displayName?})`** (l.71):
  1. `connection.findFirst({clientId, provider:"META"})` (no ordering).
  2. `socialAccount.create` with:
     - `clientId`, `platform`
     - `externalId: "${platform.toLowerCase()}:${username}"`
     - `username`
     - `displayName: displayName || username`
     - `isCompetitor: false`
     - `connectionId: conn?.id ?? null`
  3. P2002 → `fail("Sudah terdaftar untuk klien ini","VALIDATION",{username:[…]})`; else UNKNOWN.
- **`removeSocialAccountAction(clientId, accountId)`** (l.101): NOT_FOUND if the account is missing, belongs to another client, or `isCompetitor`. Otherwise delete.
- **`seoPropertySchema`:**
  - `siteUrl`: trim, min4, refine (websiteInvalid). Passes if it matches `/^sc-domain:[a-z0-9.-]+$/i` or parses as a URL with protocol http: or https:.
  - `ga4PropertyId`: trim, max30, default "".
- **`addSeoPropertyAction(clientId, {siteUrl, ga4PropertyId?})`** (l.128):
  - `connection.findFirst({clientId, provider:"GOOGLE"})`.
  - `seoProperty.create({clientId, siteUrl, ga4PropertyId: emptyToNull, connectionId})`.
  - P2002 → duplicate, fieldErrors.siteUrl.
- **`removeSeoPropertyAction(clientId, propertyId)`** (l.154): ownership check → NOT_FOUND, then delete.
- **`adAccountSchema`:** `externalId` trim min2 max40; `name` trim min1 (nameRequired) max80.
- **`addAdAccountAction(clientId, {externalId, name})`** (l.169):
  - Runs in parallel `client.findUnique(select currency)` and `connection.findFirst({clientId, provider:"META"})`.
  - `adAccount.create({clientId, externalId, name, currency: client.currency ?? "IDR", connectionId})`.
  - P2002 → duplicate, fieldErrors.externalId.
- **`removeAdAccountAction(clientId, adAccountId)`** (l.196): ownership check → NOT_FOUND, then delete.

### queries.ts (no auth inside, except `listClientsWithMeta`; callers gate)
- **`getPortfolioTotals(clientIds, range, previous)`** (l.21), returns `{followers, followersDelta, clicks, clicksDelta, spend, spendDelta}`:
  1. Empty ids → zeros with `computeDelta(0,0)`.
  2. `socialAccount.findMany({clientId in ids, isCompetitor:false}, select id)`.
  3. In parallel:
     - `socialSnapshot.findMany({socialAccountId in, date gte addDays(previous.from,-1), lte range.to}, orderBy date asc, select socialAccountId,date,followers)`, skipped if no accounts.
     - `seoDailyMetric.aggregate(_sum clicks)` where `property.clientId in ids`, current range and previous range.
     - `adDailyInsight.aggregate(_sum spend)` where `adAccount.clientId in ids`, `adId not null` (ad-level rows only), current and previous range.
  4. `followersAt(d)` = sum over accounts of the latest snapshot with date ≤ d.
  5. Values:
     - now = followersAt(range.to)
     - start = followersAt(range.from − 1)
     - prevStart = followersAt(previous.from − 1)
     - followersDelta = `computeDelta(now − start, start − prevStart)`
  - **`computeDelta(cur, prev)`** (`src/lib/metrics/delta.ts`): `abs = cur − prev`; `pct = prev==0 ? null : abs/|prev|*100`; direction = flat if |abs| < 1e-9, otherwise up or down.
  - Date ranges are UTC midnights. `previousRange(r)` = same length ending the day before `r.from`.
  - Caller: `src/app/(app)/page.tsx`.
- **`getLatestSyncByClient(clientIds)`** (l.83): `syncJob.groupBy({by:["clientId"], where:{clientId in}, _max:{startedAt}})`, returns `{[clientId]: ISO}`.
- **`listClientsWithMeta()`** (l.115):
  - `requireUser()`.
  - where = `{}` for ADMIN, otherwise `{members:{some:{userId}}}`.
  - orderBy name asc; include `_count.members` and the members filtered to the current user (role).
  - Returns `ClientRow = {id, name, slug, description, industry, websiteUrl, logoUrl, currency, timezone, createdAt ISO, membersCount, myRole}`. myRole = ADMIN for admins, else the membership role or "VIEWER".
- **`getClientSettings(clientId)`** (l.162), 5 parallel queries:
  - connections: where clientId, orderBy createdAt asc, select {id, provider, accountId, displayName, expiresAt, scopes, createdAt}. Dates serialised as ISO; expiresAt may be null.
  - socialAccounts: `{clientId, isCompetitor:false}`, orderBy [platform asc, username asc], select {id, platform, username, displayName}.
  - seoProperties: orderBy siteUrl asc, select {id, siteUrl, ga4PropertyId}.
  - adAccounts: orderBy name asc, select {id, externalId, name, currency}.
  - members: `clientMember` where clientId, orderBy createdAt asc, include user {id, name, email, isActive}. Mapped to `{userId, role, name, email, isActive, createdAt ISO}`.
  - Gated by callers: `requireClientAccess` on the settings page, or admin on the wizard page.
- **`listUserOptions()`** (l.215): `user.findMany({isActive:true}, orderBy name asc, select {id, name, email})`. Settings page calls it only when canManage.

---

## NOTIFICATIONS — src/features/notifications
- **`markNotificationRead(id)`** (actions.ts l.10):
  1. `getSessionUser`; null → UNAUTHORIZED.
  2. `z.string().min(1)`; failure → VALIDATION.
  3. `notification.updateMany({where:{id, userId: me, readAt:null}, data:{readAt: now}})`. A foreign or unknown id is a silent no-op.
  4. `revalidatePath("/","layout")`, then `ok(undefined)`.
- **`markAllNotificationsRead()`** (l.21): `updateMany({userId: me, readAt:null} → readAt now})`, then revalidate layout, then `ok({count})`.
- **`getNotificationFeed(userId, limit=20)`** (queries.ts l.19):
  - `notification.findMany({where:{userId}, orderBy createdAt desc, take limit, select id,type,title,body,href,createdAt,readAt})`.
  - Plus `count({userId, readAt:null})`.
  - Returns `{unread, items:[{id, type, title, body, href|null, createdAt ISO, read: readAt!==null}]}`.
  - Called from `src/app/(app)/layout.tsx` with the session user id.

---

## SETUP — src/features/setup + src/lib/integrations.ts

### integrations.ts (registry)
- **Keys:** META, GOOGLE, DATAFORSEO, APIFY, OPENROUTER.

  | Key | publicEnv | secretEnv | publicRequired |
  |---|---|---|---|
  | META | META_APP_ID | META_APP_SECRET | true |
  | GOOGLE | GOOGLE_CLIENT_ID | GOOGLE_CLIENT_SECRET | true |
  | DATAFORSEO | DATAFORSEO_LOGIN | DATAFORSEO_PASSWORD | true |
  | APIFY | APIFY_ACTORS | APIFY_TOKEN | false |
  | OPENROUTER | OPENROUTER_MODEL | OPENROUTER_API_KEY | false |

- **`isConfiguredPair(def, publicId, secret)`**: secret must be truthy; then if publicRequired, publicId must also be truthy.
- **`maskSecret(s)`**: length ≤ 8 → "••••"; otherwise first 4 + "••••" + last 4.
- **Meta review:**
  - META_REVIEW_STATUSES = NOT_REQUESTED, SUBMITTED, APPROVED.
  - META_PERMISSIONS (name / unlocks / required):
    - pages_show_list, pages_read_engagement, instagram_basic, instagram_manage_insights → social, required
    - business_management, ads_read → ads, required
    - pages_manage_posts, instagram_content_publish → publish, required
    - instagram_manage_comments, pages_messaging → engagement, optional
  - `parseMetaReview(raw)` keeps only valid status values and coerces appLive and businessVerified to booleans.
  - `reviewProgress`:
    - approved = number of required permissions with status APPROVED; total = number required (8).
    - publishReady (and adsReady) = every required permission in that area is APPROVED and appLive is true.
- **Cost estimators** (pure, 30 days):
  - `estimateSeoCost`. Default prices: serp 0.002, backlink 0.02/day, domain 0.01/domain/day, research 0.05/run.
    - domains = 1 + competitors
    - rank = kw·30·serp
    - backlinks = 30·b
    - domains cost = domains·30·d
    - research = runs·r
    - totalIdr = totalUsd·usdIdr
  - `estimateApifyCost`. Prices: serp 0.0035, instagram 0.0023.
    - serp = kw·30·0.0035
    - social = accounts·30·(1 + posts)·0.0023
  - Inputs are clamped with `max(0, …)`.

### src/lib/integrations.ts
- **Credential storage:** IntegrationSetting rows, keyed by provider.
  - `secretEnc = encrypt(secret)`; publicId is plaintext.
  - Effective credentials live in `process.env`. DB values override deployed env.
- **`applyRow(row)`**:
  - If publicId is set → `env[publicEnv] = publicId`.
  - If secretEnc is set → `env[secretEnv] = decrypt(...)`. A decrypt failure is ignored.
- **`hydrateIntegrationEnv()`**: `integrationSetting.findMany()`, then `applyRow` for each.
  - Called on boot by `src/instrumentation.ts` (nodejs runtime only). Errors are only warned.
- **`getIntegrationStatuses()`** (cached), per key:
  - row = DB row, if any.
  - secret = decrypt(row.secretEnc), or null on failure.
  - `fromDb = row && (row.publicId || secret)`.
  - `publicId = (fromDb ? row.publicId : null) ?? env[publicEnv] ?? null`.
  - `effSecret = secret ?? env[secretEnv] ?? null`.
  - `configured = isConfiguredPair(def, publicId, effSecret)`.
  - `source = fromDb ? "db" : configured ? "env" : "none"`.
  - Returns `{key, configured, source, publicId, secretMasked: effSecret ? mask : null, meta: row?.meta ?? null, verifiedAt ISO|null}`.
- **`saveIntegration(key, {publicId?, secret?, meta?, verifiedAt?, updatedById?})`**:
  - Builds a partial update:
    - publicId: only if not undefined; "" becomes null.
    - secretEnc: encrypted only if secret is a non-empty string; set to null when secret === null.
    - meta, verifiedAt: only if not undefined.
    - updatedById: only if truthy.
  - `integrationSetting.upsert({where:{provider:key}, create:{provider:key,…data}, update:data})`.
  - Then applyRow. If secret === null, delete the secretEnv var; if publicId is null or "", delete the publicEnv var.
- **`clearIntegration(key)`**: `deleteMany({provider:key})`, then delete **both** env vars from the process, including deployed env fallbacks until restart.
- **Workspace setup flag:** `AppSetting` key `"setup"`, value JSON `{completedAt: ISO|null, skippedAt: ISO|null}`.
  - `getSetupState()` returns both fields, defaulting to null.
  - `markSetup(kind)` reads current state, sets `[kind] = now ISO`, and upserts.
- **How "configured" is decided for the workspace wizard:**
  - Portfolio page (`src/app/(app)/page.tsx` l.31-35): if the user is ADMIN and has zero accessible clients, and both `completedAt` and `skippedAt` are null → `redirect("/setup")`.
  - Per-integration "configured" = `getIntegrationStatuses().configured`, as above.

### setup/actions.ts (auth: `getSessionUser` role ADMIN, else `fail(UNAUTHORIZED)`)
- **`credentialSchema`:**
  - `key`: enum INTEGRATION_KEYS
  - `publicId`: string trim max300, optional
  - `secret`: string trim max2000, optional
  - `verify`: boolean, default false
- **`saveCredentialAction(input)`** (l.52), returns `ActionResult<{verified: VerifyResult|null}>`:
  1. Validation failure → VALIDATION (no fieldErrors).
  2. `saveIntegration(key, {publicId (undefined if absent; "" clears it), secret: secret || undefined (blank keeps the stored secret), updatedById})`.
  3. If verify:
     - Read `env[publicEnv]` and `env[secretEnv]` after the save.
     - Run `runVerify(key, pub, sec)`:
       - No secret → `{ok:false, message: VALIDATION msg}`.
       - META needs publicId, else `{ok:false, message:"App ID"}`. GOOGLE needs it ("Client ID"). DATAFORSEO needs it ("Login (email akun)").
       - APIFY and OPENROUTER need only the secret.
     - `saveIntegration(key, {verifiedAt: ok ? now : null})`.
  4. revalidate `/setup` and `/` (layout).
- **`VerifyResult = {ok, message, detail?}`.** Provider calls (`src/lib/providers/verify.ts`), 12 s timeout, no-store:
  - **META:** `GET https://graph.facebook.com/v21.0/oauth/access_token?client_id&client_secret&grant_type=client_credentials`.
    - Ok if status 200 with `access_token`.
    - Otherwise "Meta menolak: {error.message}", or "App ID atau App Secret tidak cocok.".
  - **GOOGLE:** `POST https://oauth2.googleapis.com/token`, form body `{client_id, client_secret, grant_type: authorization_code, code: "white-analytics-probe", redirect_uri: "http://localhost/probe"}`.
    - Error invalid_grant or redirect_uri_mismatch → ok.
    - invalid_client or unauthorized_client → fail.
    - Anything else → "Respons tak terduga dari Google (…)".
  - **DATAFORSEO:** `GET https://api.dataforseo.com/v3/appendix/user_data` with Basic auth.
    - 401 → fail.
    - 200 with `tasks[0].status_code == 20000` → ok, with balance from `tasks[0].result[0].money.balance` (detail `balanceUsd`).
  - **OPENROUTER:** `GET https://openrouter.ai/api/v1/auth/key` with Bearer key. 200 → ok (with `data.label`); 401 → "Kunci OpenRouter ditolak.".
  - **APIFY:** `fetchApifyUsage(token)` from `src/lib/providers/apify/client` (not read here). Message "Token Apify valid (user, plan). Pemakaian bulan ini $x dari $y.".
  - Network exceptions → "Tidak bisa menghubungi X: {err}".
- **`clearCredentialAction(key)`** (l.78): validate key (VALIDATION), `clearIntegration`, revalidate, ok.
- **`reviewSchema`:**
  - `permissions`: `z.record(string max80, enum NOT_REQUESTED|SUBMITTED|APPROVED)`
  - `appLive`: boolean
  - `businessVerified`: boolean
- **`saveMetaReviewAction(input)`** (l.94):
  - `prev` = the current META row's meta object, or {}.
  - `saveIntegration("META", {meta:{...prev, ...input}, updatedById})`. The permissions map is replaced wholesale.
  - Upsert can create a META row without credentials; source stays "none" or "env".
- **`finishSetupAction()`** (l.106): non-admin → `redirect("/login")`. Then `markSetup("completedAt")`, revalidate, `redirect("/clients/new")`.
- **`skipSetupAction()`** (l.114): same, with `markSetup("skippedAt")` and `redirect("/")`.

### setup/queries.ts
- **`getSetupOverview()`**: no auth inside; the page calls `requireAdmin()`.
  - Fetches `getIntegrationStatuses()` and `getSetupState()`.
  - Counts: `trackedKeyword.count()`, `competitorDomain.count()`, `client.count()`, `socialAccount.count()`. These are global, all clients, including competitors.
  - Returns `{statuses, metaReview: parseMetaReview(statuses.META.meta), setup, appUrl: NEXT_PUBLIC_APP_URL without trailing "/", counts}`.
  - The page also reads `DATAFORSEO_USD_IDR` (default 16000).

### setup/steps.ts
- **Steps:** meta, google, seo-data, apify, ai, selesai. Invalid input falls back to "meta".
- `setupHref("meta") = "/setup"`; otherwise `/setup?step=X`.

---

## SHARE — src/features/share, src/app/share, src/app/api/share

### How share links work
- **No token:** the public URL is `/share/{client.slug}` (and `/api/share/{slug}/pdf`). There is no expiry on the link.
- Access requires `Client.shareEnabled = true`.
- If `sharePinHash` is set, the visitor needs the unlock cookie.
- **Cookie:**
  - Name: `share_{clientId}`.
  - Value: hex `HMAC-SHA256(key = AUTH_SECRET ?? ENCRYPTION_KEY ?? "white-share-dev-secret", message = the bcrypt sharePinHash string)`.
  - Attributes: httpOnly, SameSite=Lax, Secure only when NODE_ENV=production, Max-Age 43200 (12 h), Path=/.
  - Changing or clearing the PIN changes the hash, which invalidates all issued cookies.

### access.ts
- `shareCookieName(clientId)` and `shareCookieValue(pinHash)`, as above.
- `getShareClient(slug)` = `client.findUnique({slug})` (full row).
- `isShareUnlocked(client)`: no pinHash → true; otherwise the cookie value must equal `shareCookieValue(pinHash)` (plain `===`).
- **`getShareContext(slug)`** returns one of:
  - `{status:"not-found"}`
  - `{status:"disabled", client}` when `!shareEnabled`
  - `{status:"locked", client}`
  - `{status:"ok", client}`

### actions.ts
- **Rate limit:** in-memory Map on globalThis, per server instance.
  - Key `${slug}:${ip}`. ip = first value of `x-forwarded-for`, else `x-real-ip`, else "local".
  - Window 10 minutes, MAX 5.
  - First hit creates `{count:1, resetAt: now+10min}` and is allowed.
  - Later hits increment; blocked when count > 5, so the 6th attempt in the window is blocked.
  - Every call counts, including successes and malformed input. Success does not reset the bucket.
- **`verifySharePin(slug, pin)`** (l.38):
  1. Rate limited → `fail("Terlalu banyak percobaan…","RATE_LIMIT")`.
  2. pin must match `^\d{6}$`, else `fail("PIN salah","VALIDATION")`.
  3. `client.findUnique({slug}, select id, shareEnabled, sharePinHash)`. Missing or disabled → `fail("Dashboard tidak ditemukan","NOT_FOUND")`.
  4. No pinHash → `redirect("/share/{slug}")`.
  5. `bcrypt.compare` false → `fail("PIN salah")` with no code.
  6. Success: set the cookie, then `redirect("/share/{slug}")`.
- **`verifySharePinForm(slug, _prev, formData)`**: reads `formData.pin` and calls `verifySharePin`.

### queries.ts (public view models; depend on out-of-scope `features/overview/queries` and `features/reports/queries`)
- **`getShareSocial(clientId, range, previous)`**:
  - Uses `getOverviewSocial`, `getSocialDailyFollowers(clientId, range)` and `getTopPostRows(clientId, range, 10)`.
  - Returns null if `!social.hasData`.
  - Otherwise `{followers, followersDelta, engagementRate, engagementRateDelta, reach, reachDelta, posts, postsDelta, daily:[{date, followers}], topPosts:[{id, platform, username, caption: truncate(caption,90), type: contentType(p), likes, comments, engagements, publishedAt ISO}]}`.
- **`getShareSeo`**:
  - Uses `getOverviewSeo` and `getTopQueryRows(…, 10)`. Null if no data.
  - Otherwise `{clicks, impressions, ctr, position, their Deltas, daily:[{date, clicks|null, impressions|null}], topQueries:[{query, clicks, impressions, ctr, position}]}`.
- **`getShareAds(…, currency)`**:
  - Uses `getOverviewAds` and `getCampaignRows`. Null if no data.
  - Otherwise `{currency, spend, spendDelta, results, resultsDelta, cpr, cprDelta, ctr, daily:[{date, spend, results}], campaigns:[{id, name, objective, spend, results, resultType, cpr, isConversion}]}`.
- **Page** `src/app/share/[slug]/page.tsx`:
  - Status not-found, disabled or locked → the corresponding UI.
  - preset from `?preset`, allowed 7d, 28d, 90d; default 28d.
  - `range = rangeForPreset`: end = yesterday UTC; 7d = end−6..end, 28d = end−27..end, 90d = end−89..end.
  - `previous = previousRange(range)`.
  - Loads only modules present in `client.shareModules`.
  - Metadata sets robots noindex, nofollow.

### src/app/api/share/[slug]/pdf/route.ts — `GET` (l.12), `dynamic = "force-dynamic"`, public (no session)
- not-found → 404 `{"error":"NOT_FOUND"}`.
- disabled → 404 `{"error":"DISABLED"}`.
- locked → 401 `{"error":"PIN_REQUIRED"}`.
- Otherwise:
  1. preset ∈ {7d, 28d, 90d}, default 28d. Compute range and previous.
  2. `insight = getLatestInsight(client.id, "OVERVIEW", range)` (AiInsight).
  3. `buildReportPdfData({clientId, clientName, currency, title: "Laporan Performa — {name}", range, previous, compare:true, modules: client.shareModules, language:"id", aiSummary: insight?.content ?? null})`.
  4. `renderReportPdf(data)`.
  5. 200 response:
     - `Content-Type: application/pdf`
     - `Content-Disposition: attachment; filename="Laporan_{slug}_{YYYY-MM-DD from}_{YYYY-MM-DD to}.pdf"`
     - `Cache-Control: no-store`

---

## GUIDE — src/features/guide
- No server actions, queries or DB access. Static content only.
- `content.ts`: `GUIDE_SECTIONS` (merges `SEO_SUITE_GUIDE` and `PUBLISHING_GUIDE`), `GUIDE_IDS`, `getGuideSection(id)` (find by id).
- **`guideSectionForPath(path)`**, first match wins:
  - `/clients/new` → mulai
  - starts with `/setup` → integrasi
  - contains `/publish` → publikasi
  - contains `/social` → social
  - matches `/seo/(research|rank|backlinks|competitors)(/|$)` → seo-suite
  - contains `/seo` → seo
  - contains `/ads` → ads
  - contains `/reports` → laporan
  - contains `/settings` → akses
  - starts with `/admin` → akses
  - otherwise → mulai
- `strings.ts`: UI labels only, no error codes.

---

## Behaviour to keep in the Go port
1. Every action re-checks the DB user (`isActive`) on each request. JWT claims alone are never trusted for role.
2. `manageContext` returns UNAUTHORIZED, not NOT_FOUND, for an unknown clientId.
3. An empty share PIN clears the PIN. Any save of a PIN re-hashes it and invalidates cookies.
4. The admin user update replaces all memberships (delete all, then insert) inside a transaction.
5. Setup saves mutate process-wide env/config. `clearCredentialAction` also wipes deployed env fallbacks for that provider until restart.
6. The share PIN rate limiter is per instance and in memory, and counts every attempt.
7. Validation error messages for min/max/enum that have no custom message come from Zod v4's English defaults, keyed by the top-level field.
