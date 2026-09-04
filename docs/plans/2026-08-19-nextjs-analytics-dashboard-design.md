# WHITE Analytics — Design Doc (Next.js rebuild)

Tanggal: 2026-08-19  
Status: **Disetujui** (scope: Social + SEO + Ads reporting; arsitektur C: Postgres + Prisma + Auth.js; UI Bahasa Indonesia)

---

## 1. Latar belakang

Aplikasi lama `white_super_app/` adalah Flutter Web SPA (~28k baris, 9 modul) dengan backend Firebase Firestore (`white-superapp`) + cache SharedPreferences, AI via OpenRouter yang dipanggil dari browser, deploy Vercel. Modul yang benar-benar dipakai untuk analitik: **Social** (Instagram Business Discovery, kompetitor), **Ads** (Meta Ads reporting + share link klien). Modul **SEO** hanya kartu "Coming Soon" — tidak ada kode.

Masalah utama yang **tidak dibawa**:

| Kategori | Temuan di app lama |
|---|---|
| Keamanan | API key OpenRouter hardcoded di client (3 file); token Meta plaintext di `AdsSample/`; password user plaintext di Firestore + localStorage; token sosial plaintext di `users/{uid}/social_tokens`; PIN share diverifikasi di client terhadap field publik |
| Data palsu | Growth series sosial disintesis linear (+150/hari); persentase tren KPI dashboard literal (`+12.4%`, `+5.2%`, …); demografi mock disuntik ke laporan klien saat data kosong |
| Rumus/bug | `ctr = results/impressions`; CSV re-import menggandakan data (append tanpa upsert); KPI report mengabaikan date range sementara chart menghormatinya; token FB short-lived disimpan dengan expiry 60 hari karangan; prompt AI sosial memakai system prompt deck Ads (MARP) & model id `openrouter/free` tidak valid; `DateFilter.weekly` tidak berbuat apa-apa |

Yang **dibawa** (konsep): multi-klien + slug + share link + PIN, KPI card dengan delta, hierarki Campaign→AdSet→Ad, CSV import Meta, report PDF bermerek WHITE, AI insight per laporan, role-based access, brand monokrom WHITE.

## 2. Keputusan

| Keputusan | Pilihan | Alasan |
|---|---|---|
| Scope modul | Social + SEO + Ads reporting | Fokus analisa & reporting; Ads sudah matang & sering dipakai agency |
| Sumber SEO | Google Search Console + GA4; PageSpeed/Lighthouse + crawler sendiri | Gratis/resmi; tanpa API berbayar |
| Arsitektur | Next.js + PostgreSQL + Prisma + Auth.js | Pilihan user (kontrol penuh, tanpa BaaS) |
| Bahasa UI | Bahasa Indonesia | Pilihan user; angka `id-ID`, mata uang default IDR per klien |
| DB lokal | PostgreSQL 17 Homebrew yang sudah jalan di `localhost:5432` | Tanpa Docker |
| App Flutter lama | Tidak disentuh; app baru di `white-analytics/` | Migrasi bertahap |

## 3. Stack

- Next.js 16 (App Router, RSC, Server Actions, Route Handlers), TypeScript strict, pnpm
- Tailwind CSS v4 + shadcn/ui, lucide-react, Inter (`next/font`)
- Recharts, TanStack Table, Zod, date-fns
- Prisma 7 + PostgreSQL 17
- Auth.js (Credentials + bcrypt, JWT session). OAuth Google/Meta untuk **data** = flow terpisah per klien (bukan login)
- `@react-pdf/renderer`, papaparse, cheerio, googleapis, fetch (Meta Graph, PageSpeed, OpenRouter)
- Vitest, Playwright (smoke), ESLint/Prettier; Vercel + Vercel Cron

## 4. Arsitektur

```
white-analytics/
  app/                    routing & layout (RSC), route handlers (auth, oauth callback, cron, pdf)
  features/<modul>/       social | seo | ads | reports | clients | admin — komponen, queries, actions
  lib/providers/          adapter per sumber: meta, google (gsc+ga4), pagespeed, crawler, openrouter
                          setiap adapter: interface + implementasi nyata + mock (dipilih via env)
  lib/metrics/            semua rumus (murni, di-test): ER, CTR, CPR, CPC, CPM, frequency, delta periode,
                          bucket posisi, kurva CTR ekspektasi, aturan opportunity, health score
  lib/db/, prisma/        Prisma client, schema, seed, migrasi
  components/ui/          shadcn + primitif dashboard (KpiTile, ChartCard, DataTable, DateRangePicker, EmptyState)
```

Prinsip:
- Semua rahasia hanya di server; token koneksi dienkripsi AES-256-GCM (`ENCRYPTION_KEY`).
- Sinkronisasi: tombol "Sinkronkan" per modul + cron harian `/api/cron/daily` (`CRON_SECRET`); setiap job dicatat di `SyncJob`.
- RBAC di server action & middleware; UI hanya menyembunyikan.
- Provider error dinormalisasi: `RATE_LIMIT | TOKEN_EXPIRED | PERMISSION | NOT_FOUND | NETWORK | UNKNOWN`.

## 5. Model data (Prisma)

- **User**(id, email, name, passwordHash, role `ADMIN|MEMBER`, isActive, createdAt)
- **Client**(id, name, slug, description, logoUrl?, currency `IDR`, timezone `Asia/Jakarta`, shareEnabled, sharePinHash?, shareModules[], createdById)
- **ClientMember**(userId, clientId, role `MANAGER|VIEWER`)
- **Connection**(id, clientId, provider `META|GOOGLE`, accountId, displayName, accessTokenEnc, refreshTokenEnc?, expiresAt?, scopes[], meta Json)
- **SocialAccount**(id, clientId, platform `INSTAGRAM|FACEBOOK|TIKTOK`, externalId, username, displayName, avatarUrl?, isCompetitor, connectionId?)
- **SocialSnapshot**(socialAccountId, date, followers, following?, mediaCount, reach?, impressions?, profileViews?) — unique (accountId, date)
- **SocialPost**(socialAccountId, externalId, caption, mediaType, productType, permalink, mediaUrl, thumbnailUrl, publishedAt, likes, comments, shares, saves, views, reach, impressions) — unique (accountId, externalId)
- **SeoProperty**(id, clientId, siteUrl, ga4PropertyId?, connectionId?)
- **SeoDailyMetric**(propertyId, date, clicks, impressions, ctr, position) — unique (propertyId, date)
- **SeoDimensionMetric**(propertyId, date, dimension `QUERY|PAGE|COUNTRY|DEVICE`, key, clicks, impressions, ctr, position) — unique (propertyId, date, dimension, key)
- **Ga4DailyMetric**(propertyId, date, sessions, organicSessions, users, engagedSessions, conversions, avgEngagementSec)
- **SeoAudit**(propertyId, url, strategy `MOBILE|DESKTOP`, runAt, performance, seo, accessibility, bestPractices, lcpMs, inpMs, cls, fcpMs, ttfbMs, raw Json)
- **SeoCrawl**(propertyId, startedAt, finishedAt, status, pagesCrawled, maxPages) · **SeoCrawlPage**(crawlId, url, statusCode, title, metaDescription, h1Count, wordCount, canonical, indexable, imagesMissingAlt, internalLinks, externalLinks, loadMs) · **SeoIssue**(crawlId, severity `ERROR|WARNING|NOTICE`, code, message, url)
- **AdAccount**(clientId, externalId `act_…`, name, currency, connectionId?)
- **AdCampaign**(adAccountId, externalId, name, objective, status, dailyBudget?) · **AdSet**(campaignId, externalId, name, status) · **Ad**(adSetId, externalId, name, status, thumbnailUrl?)
- **AdDailyInsight**(adAccountId, date, campaignId?, adSetId?, adId?, spend, impressions, reach, clicks, linkClicks, results, resultType) — unique (adAccountId, date, campaignId, adSetId, adId)
- **AdDemographic**(adAccountId, dateFrom, dateTo, campaignId?, age, gender, spend, impressions, results)
- **Report**(clientId, title, dateFrom, dateTo, compare, modules[], aiSummary?, pdfPath?, createdById)
- **AiInsight**(clientId, module, dateFrom, dateTo, language, model, content) · **SyncJob**(clientId?, kind, status, startedAt, finishedAt, error?)

## 6. Struktur halaman

```
/login
/                                Portfolio semua klien
/clients                         daftar + buat klien
/clients/[slug]                  Overview klien (KPI lintas kanal, ringkasan AI, aktivitas sync)
/clients/[slug]/social           tab IG/FB/TikTok
/clients/[slug]/social/competitors
/clients/[slug]/seo              overview GSC + GA4
/clients/[slug]/seo/keywords     explorer query + opportunities
/clients/[slug]/seo/pages
/clients/[slug]/seo/audit        Lighthouse/CWV + isu crawler + histori
/clients/[slug]/ads              Meta Ads + CSV import
/clients/[slug]/reports          report builder + riwayat + share link
/clients/[slug]/settings         koneksi, anggota, PIN
/admin/users
/share/[slug]                    publik, PIN opsional (server), export PDF
```

Global: client switcher (⌘K), date range picker (7/14/28/30/90 hari, bulan lalu, custom) + toggle **bandingkan vs periode sebelumnya**; semua Δ dihitung nyata.

## 7. Detail modul & rumus

### Social
- Sumber: IG Business (own): `GET /{ig-user-id}?fields=followers_count,media_count,…`, `GET /{ig-user-id}/insights?metric=reach,impressions,profile_views,follower_count&period=day&since&until`, `GET /{ig-user-id}/media?fields=id,caption,like_count,comments_count,media_type,media_product_type,permalink,media_url,thumbnail_url,timestamp&limit=50`, per-media `insights?metric=reach,impressions,saved,shares`. Kompetitor: `GET /{ig-user-id}?fields=business_discovery.username(X){followers_count,media_count,name,profile_picture_url,media.limit(25){…}}`. FB Page: `GET /{page-id}?fields=fan_count,followers_count,name`, `insights?metric=page_impressions_unique,page_post_engagements&period=day`, `posts?fields=message,created_time,permalink_url,shares,reactions.summary(true),comments.summary(true)`. TikTok: adapter placeholder (mock).
- Scope Meta yang dibutuhkan: `pages_show_list, pages_read_engagement, instagram_basic, instagram_manage_insights, business_management` (+ `ads_read` untuk Ads).
- Token: exchange ke long-lived di server (`/oauth/access_token?grant_type=fb_exchange_token`), simpan terenkripsi, expiry nyata.
- Rumus: ER post = (likes+comments+shares+saves)/followers×100; avg ER periode = mean ER post dalam periode; growth = followers(end)−followers(start); growth% = growth/followers(start)×100; Δ = periode ini vs periode sebelumnya sepanjang sama.
- Chart: growth (snapshot harian), reach/impressions harian, ER per tipe konten, heatmap hari×jam posting, top posts.

### SEO
- GSC: `searchanalytics.query` dimensions `date`, `query`, `page`, `country`, `device`; histori 16 bulan saat sync pertama; simpan total harian + top-N per dimensi per hari.
- GA4 Data API: `runReport` metrics `sessions, engagedSessions, totalUsers, conversions, userEngagementDuration` × dimension `date`, `sessionDefaultChannelGroup`, `landingPage`.
- PageSpeed Insights: `GET https://www.googleapis.com/pagespeedonline/v5/runPagespeed?url&strategy&category=performance,seo,accessibility,best-practices` (tanpa key = kuota rendah; `PAGESPEED_API_KEY` opsional).
- Crawler: cheerio, mulai dari sitemap.xml/robots.txt/BFS, batas `maxPages` (default 50); cek status, title (30–60), meta description (70–160), H1 (tepat 1), canonical, robots meta noindex, img alt, broken internal link, word count (<300 = thin), hreflang, JSON-LD.
- Rumus: CTR = clicks/impressions×100; avg position = Σ(position×impressions)/Σimpressions; bucket posisi 1-3 / 4-10 / 11-20 / 21+; **striking distance** = posisi 4–15 & impressions ≥ ambang (default 100/periode); **low CTR** = posisi ≤10 & CTR < 60% kurva ekspektasi (posisi 1: 28%, 2: 15%, 3: 11%, 4: 8%, 5: 7%, 6: 5%, 7: 4%, 8: 3%, 9: 3%, 10: 2.5%); **declining** = clicks turun ≥30% vs periode sebelumnya dengan impressions ≥ ambang; health score = rata-rata skor Lighthouse berbobot (perf 0.3, seo 0.4, a11y 0.15, BP 0.15) dikurangi penalti isu ERROR.
- CWV ambang: LCP good ≤2.5s / poor >4s; INP ≤200ms / >500ms; CLS ≤0.1 / >0.25.

### Ads
- Meta Marketing API v21+: `/{act}/campaigns`, `/adsets`, `/ads?fields=…,creative{thumbnail_url}`, `/{act}/insights?level=ad&time_increment=1&fields=campaign_id,adset_id,ad_id,spend,impressions,reach,clicks,inline_link_clicks,actions,frequency&time_range`, breakdown `age,gender`.
- CSV import ekspor Meta: mapping kolom (Campaign name, Ad set name, Ad name, Day/Reporting starts, Amount spent, Impressions, Reach, Link clicks, Results, Result type, Age, Gender) → upsert by (ad, date).
- Rumus: CPR = spend/results; **CTR = linkClicks/impressions×100**; CPC = spend/linkClicks; CPM = spend/impressions×1000; frequency = impressions/reach; Δ vs periode sebelumnya.

### Reports & Share
- Report builder: modul + periode + compare + ringkasan AI → preview → PDF (`@react-pdf/renderer`) → unduh/simpan; riwayat.
- Share `/share/[slug]`: read-only, modul sesuai `shareModules`, date range preset, PIN di-hash (bcrypt) + rate limit; tidak ada token/rahasia yang terekspos.
- AI insight: OpenRouter (model via `OPENROUTER_MODEL`), system prompt terpisah per modul (social/seo/ads/overview), bahasa ID/EN, cache di `AiInsight`.

## 8. Desain visual — tone withwhite.id

Referensi tone: **withwhite.id** (situs agency WHITE). Diadopsi 2026-08-21 menggantikan arah monokrom awal.

**Prinsip:** kanvas cetak-biru (blueprint) yang tenang, satu aksen biru royal yang tegas, tipografi tebal rapat, dan label teknis monospace. Data adalah satu-satunya elemen yang "berisik".

| Elemen | Nilai |
|---|---|
| Kanvas halaman | `#f4f4f2` + grid blueprint (minor 14px, mayor 112px, biru 5%/9%) — utilitas `.blueprint` |
| Kartu | `#ffffff`, hairline `#e2e2dd`, radius 14px |
| Aksen brand | `#1b4de4` (light) · `#5b84ff` mark / `#2e5be8` tombol (dark) |
| Teks | `#0a0a0a` · sekunder `#5f6470` |
| Judul | Inter bold, tracking `-0.035em`, disertai garis biru pendek di bawahnya |
| Label mikro | Geist Mono, 10px, uppercase, tracking `0.14em` — utilitas `.label-mono` |
| Tombol | pill (`rounded-full`); primer biru solid |
| Ornamen teknis | crop-mark sudut (`.crop-marks`) pada KPI tile & panel hero; strip penggaris (`.ruler-x`) di topbar/hero |
| Sidebar | putih, divider hairline antar item, penanda **titik biru** untuk item aktif (bukan blok terisi), label grup mono, CTA pill di bawah |
| Mode gelap | kanvas `#0b0c0e`, kartu `#131417`; ramp sequential **dibalik** (lebih terang = lebih besar) |

**Chart** (tervalidasi ulang dengan validator dataviz, semua gate lolos di kedua mode):
- Slot kategorikal 1 = biru brand. Light: `#1b4de4, #eb6834, #1baf7a, #eda100, #e87ba4, #008300, #4a3aa7, #e34948`. Dark: `#5b84ff, #d95926, #199e70, #c98500, #d55181, #008300, #9085e9, #e66767`.
- Ramp sequential berbasis hue brand (`--seq-100..700`); ordinal minimal step 250 (light) / 550 (dark) agar ≥ 2:1.
- Warna status tetap terpisah dan selalu berpasangan ikon + label.
- Angka ringkas memakai NBSP (`1,3 rb`) agar label sumbu tidak pecah dua baris.

## 9. Error handling & keamanan
Zod pada semua input; server action mengembalikan `ActionResult<T>` bertipe; error provider dinormalisasi → banner + CTA reconnect; token terenkripsi; env server-only; RBAC di server; PIN di-hash + rate limit; cron secret; sync job dicatat, kegagalan satu klien tidak menghentikan lainnya.

## 10. Testing
Vitest: rumus & aturan (`lib/metrics`), CSV importer, mapper GSC/GA4, pemeriksa HTML crawler, PIN. Playwright smoke: login → klien → tiap modul render dengan seed → share + PIN. CI: typecheck, lint, test.

## 11. Migrasi & tahapan
- `scripts/migrate-firestore.ts`: ekspor JSON Firestore → Client (name, slug, description, PIN di-hash), User (role dipetakan: admin→ADMIN, lainnya→MEMBER; password plaintext lama di-hash saat migrasi, sarankan reset), data Ads lama best-effort.
- **Fase 1**: scaffold, DB + Prisma + seed realistis, auth, shell/UI semua halaman, chart/tabel, report builder + PDF, share link, adapter mock + PageSpeed & crawler nyata, unit test rumus.
- **Fase 2**: OAuth Meta & Google nyata (butuh App ID/secret & Google Cloud OAuth client), cron Vercel, migrasi data lama, deploy.

## 12. Referensi app lama (untuk migrasi)
- Firebase project `white-superapp`; Facebook App ID `913443368363662` (SDK v20); AdSample memakai app `1727203858114267` (berbeda).
- Firestore: `projects/{id}` (+ subkoleksi campaigns/adSets/ads/dailyMetrics/demographics), `users/{id}`, `users/{uid}/social_tokens/{platform}`.
- Share link lama: `white-superapp.vercel.app/#/shared/s/{slug}` → baru `/share/{slug}`.
- Role lama: admin, adsOnly, projectSpecific, socialFeatureOnly, trialSocialAccount (kuota 10 pencarian) → baru ADMIN/MEMBER + ClientMember MANAGER/VIEWER.
