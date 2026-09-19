# WHITE Analytics

Platform pemasaran digital multi-klien untuk tim agency: **analitik dan laporan Social Media, SEO, Meta Ads**, **publikasi konten**, dan **SEO suite**. Menggantikan aplikasi Flutter Web lama (`../white_super_app`) dengan Next.js.

- Analitik sosial: Instagram, Facebook, TikTok: pertumbuhan followers (snapshot harian nyata), engagement rate, jangkauan, konten teratas, waktu posting terbaik, perbandingan kompetitor.
- SEO: Google Search Console + GA4, explorer kata kunci dengan peluang (striking distance, CTR rendah, menurun), performa halaman, audit Lighthouse/Core Web Vitals + crawler on-page internal.
- Meta Ads: KPI (spend, hasil, CPR, CTR, CPC, CPM, frekuensi, ROAS), drill-down Campaign → Ad Set → Ad, demografi, impor CSV Meta Ads Manager (upsert, anti-duplikat).
- Laporan PDF bermerek WHITE, ringkasan strategis AI, dan **link klien read-only** (`/share/<slug>`) dengan PIN opsional.
- **Publikasi** (`/clients/<slug>/publish`): komposer multi-akun dengan pratinjau per platform, pustaka media, kalender konten bulan/minggu, alur draf → review → disetujui → terjadwal → terbit, komentar internal, dan penjadwal otomatis.
- **SEO suite**: riset kata kunci (volume, KD, CPC, intent, kesenjangan vs kompetitor), rank tracker harian dengan visibilitas dan share of voice, analisa backlink, benchmark domain kompetitor, audit terjadwal.
- **Setup awal** (`/setup`, khusus ADMIN): wizard yang menyimpan kredensial Meta, Google, DataForSEO, Apify, dan OpenRouter secara terenkripsi, lengkap dengan uji koneksi, checklist App Review Meta, dan estimator biaya.
- Notifikasi in-app (permintaan review, hasil publikasi, penurunan peringkat), panduan in-app (`/panduan`), wizard pembuatan project, dan tampilan yang rapi di lebar 375 px.

## Stack

Next.js 16 (App Router, RSC, Turbopack) · TypeScript · Tailwind CSS v4 + shadcn/ui · Recharts · TanStack Table · Prisma 7 + PostgreSQL · Auth.js v5 · `@react-pdf/renderer` · Vitest.

## Menjalankan secara lokal

Prasyarat: Node.js ≥ 20.9, pnpm, PostgreSQL 14+.

```bash
pnpm install
```

Salin `.env.example` menjadi `.env`, lalu isi minimal `DATABASE_URL`, `AUTH_SECRET`, dan `ENCRYPTION_KEY`:

```bash
cp .env.example .env
```

```bash
openssl rand -base64 32
```

Buat database, jalankan migrasi, lalu isi data demo:

```bash
createdb white_analytics && pnpm db:migrate && pnpm db:seed
```

```bash
pnpm dev
```

Buka http://localhost:3000.

### Akun demo (dari seed)

| Email | Kata sandi | Peran |
|---|---|---|
| `admin@white.id` | `admin12345` | ADMIN (semua klien) |
| `tim@white.id` | `member12345` | MEMBER (Kopi Nusantara sebagai Manajer, Adiharjo sebagai Viewer) |

Klien demo: **Kopi Nusantara** (link klien ber-PIN `123456`), **Adiharjo Property**, **Bali Villa Escapes**, masing-masing 120 hari data sosial, SEO, dan iklan.

## Perintah

| Perintah | Fungsi |
|---|---|
| `pnpm dev` | Jalankan dev server |
| `pnpm build` / `pnpm start` | Build & jalankan produksi |
| `pnpm typecheck` | Cek TypeScript |
| `pnpm lint` | ESLint |
| `pnpm test` | Unit test (Vitest) |
| `pnpm sync:smoke` | Uji pipeline sinkronisasi terhadap Postgres dengan adapter mock |
| `pnpm db:migrate` | Migrasi Prisma (dev) |
| `pnpm db:seed` | Isi ulang data demo |
| `pnpm db:studio` | Prisma Studio |
| `pnpm db:reset` | Reset database + seed |

## Mode demo vs data nyata

Aplikasi berjalan penuh tanpa kredensial apa pun: modul menampilkan data seed dan menandainya dengan banner "Data demo". Isi kredensial dari dalam aplikasi di **Admin → Setup awal & integrasi**. Nilainya tersimpan terenkripsi di database, langsung aktif tanpa restart, dan menang atas nilai `.env`. Variabel `.env` di bawah tetap berfungsi sebagai cadangan.

| Integrasi | Variabel | Mengaktifkan |
|---|---|---|
| Meta | `META_APP_ID`, `META_APP_SECRET` | OAuth klien, insight Instagram/Facebook (reach, impressions), Meta Ads, publikasi IG/FB. Publikasi sungguhan butuh izin `instagram_content_publish` dan `pages_manage_posts` lewat App Review. Sebelum itu post dikirim lewat simulasi. |
| Google | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | OAuth klien, sinkronisasi harian Search Console dan GA4 |
| DataForSEO | `DATAFORSEO_LOGIN`, `DATAFORSEO_PASSWORD` | Volume kata kunci, KD, SERP harian, backlink, overview domain (bayar per panggilan) |
| Apify | `APIFY_TOKEN`, `APIFY_ACTORS` (opsional) | Posisi SERP saat DataForSEO kosong, serta profil dan post publik Instagram, TikTok, Facebook untuk akun klien dan kompetitor |
| OpenRouter | `OPENROUTER_API_KEY`, `OPENROUTER_MODEL` | Ringkasan strategis AI (tanpa kunci: ringkasan templat dari angka nyata) |
| PageSpeed | `PAGESPEED_API_KEY` | Kuota audit Lighthouse lebih tinggi (audit tetap jalan tanpa kunci) |
| Cron | `CRON_SECRET` | `GET /api/cron/daily` dan `GET /api/cron/publish` |
| Media | `UPLOAD_DIR` | Lokasi file upload (default `.uploads/`). Ganti `src/lib/storage.ts` ke S3/R2 untuk produksi. |

Publikasi ke TikTok belum tersedia karena butuh persetujuan Content Posting API. Audit situs (PageSpeed + crawler on-page) tidak memerlukan kredensial.

Semua rahasia hanya dibaca di server. Token koneksi dan kredensial integrasi disimpan terenkripsi (AES-256-GCM). Upload media divalidasi dari isi file (magic bytes), dan semua respons membawa CSP, `X-Frame-Options`, serta `Referrer-Policy`.

## Cron

| Endpoint | Jadwal | Tugas |
|---|---|---|
| `/api/cron/daily` | 02:00 UTC | Retensi data, lalu satu pemanggilan per klien (`?client=<id>`, paralel 4, klien yang paling lama menunggu didahulukan): sinkronisasi Social/GSC/GA4/Ads dan job SEO suite (peringkat, backlink, domain, audit terjadwal) |
| `/api/cron/publish` | tiap 5 menit | Menerbitkan post yang jatuh tempo. Klaim atomik mencegah terbit ganda. |

Retensi: `SyncJob` 90 hari, notifikasi terbaca 30 hari (semua notifikasi 90 hari), snapshot peringkat/backlink/domain 760 hari. Angkanya ada di `src/features/sync/policy.ts`.

## Struktur

```
prisma/            schema, migrasi, seed
src/app/(app)      halaman di dalam shell (sidebar + topbar)
src/app/share      dashboard publik klien
src/app/api        route handler (auth, koneksi OAuth, PDF, cron, AI)
src/features/*     modul: overview, social, seo, seo-suite, ads, publishing, reports, clients, setup, sync, notifications, guide, admin
src/lib/metrics    semua rumus (murni, di-test)
src/lib/providers  adapter Meta Graph/Marketing/publisher, Google, DataForSEO, Apify, PageSpeed, crawler, OpenRouter (real + mock)
src/components     ui (shadcn), dashboard (KPI/chart/tabel), shell
src/i18n/id.ts     seluruh teks UI (Bahasa Indonesia)
```

Kontrak pengembangan ada di [CONVENTIONS.md](CONVENTIONS.md); desain & rencana ada di `../docs/plans/`.

## Deploy

Vercel: set *Root Directory* ke `white-analytics`, isi `DATABASE_URL`, `AUTH_SECRET`, `ENCRYPTION_KEY`, `CRON_SECRET`, dan `NEXT_PUBLIC_APP_URL` (domain produksi). Gunakan Postgres terkelola (Neon/Vercel Postgres) dan jalankan `pnpm prisma migrate deploy` saat build. Kedua cron terdaftar di `vercel.json`. Jadwal 5 menit butuh paket Pro; di paket Hobby panggil `/api/cron/publish` dari penjadwal eksternal. Ganti penyimpanan media ke object storage sebelum produksi karena disk serverless tidak persisten.

CI (`.github/workflows/ci.yml`) menjalankan typecheck, lint, unit test, `sync:smoke` dengan service Postgres, dan build pada setiap push dan pull request.

## Migrasi dari aplikasi Flutter lama

`scripts/migrate-firestore.ts` mengimpor ekspor JSON Firestore lama (`projects`, `users`) menjadi `Client` dan `User`. Password lama yang tersimpan plaintext akan di-hash saat impor. Sarankan semua user melakukan reset setelah migrasi.
