# WHITE Analytics

Dashboard agency multi-klien untuk **analisa & reporting Social Media, SEO, dan Meta Ads**. Menggantikan aplikasi Flutter Web lama (`../white_super_app`) dengan Next.js.

- Analitik sosial: Instagram, Facebook, TikTok — pertumbuhan followers (snapshot harian nyata), engagement rate, jangkauan, konten teratas, waktu posting terbaik, perbandingan kompetitor.
- SEO: Google Search Console + GA4, explorer kata kunci dengan peluang (striking distance, CTR rendah, menurun), performa halaman, audit Lighthouse/Core Web Vitals + crawler on-page internal.
- Meta Ads: KPI (spend, hasil, CPR, CTR, CPC, CPM, frekuensi, ROAS), drill-down Campaign → Ad Set → Ad, demografi, impor CSV Meta Ads Manager (upsert, anti-duplikat).
- Laporan PDF bermerek WHITE, ringkasan strategis AI, dan **link klien read-only** (`/share/<slug>`) dengan PIN opsional.

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

Klien demo: **Kopi Nusantara** (link klien ber-PIN `123456`), **Adiharjo Property**, **Bali Villa Escapes** — masing-masing 120 hari data sosial, SEO, dan iklan.

## Perintah

| Perintah | Fungsi |
|---|---|
| `pnpm dev` | Jalankan dev server |
| `pnpm build` / `pnpm start` | Build & jalankan produksi |
| `pnpm typecheck` | Cek TypeScript |
| `pnpm lint` | ESLint |
| `pnpm test` | Unit test (Vitest) |
| `pnpm db:migrate` | Migrasi Prisma (dev) |
| `pnpm db:seed` | Isi ulang data demo |
| `pnpm db:studio` | Prisma Studio |
| `pnpm db:reset` | Reset database + seed |

## Mode demo vs data nyata

Aplikasi berjalan penuh tanpa kredensial apa pun: modul menampilkan data seed dan menandainya dengan banner "Data demo". Integrasi nyata aktif begitu variabel lingkungan diisi:

| Variabel | Mengaktifkan |
|---|---|
| `META_APP_ID`, `META_APP_SECRET` | OAuth Meta → Instagram/Facebook insight & Meta Ads insight |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | OAuth Google → Search Console & GA4 |
| `PAGESPEED_API_KEY` | Kuota lebih tinggi untuk audit Lighthouse (audit tetap jalan tanpa key) |
| `OPENROUTER_API_KEY` | Ringkasan strategis AI (tanpa key: ringkasan template berbasis angka nyata) |
| `CRON_SECRET` | Endpoint `GET /api/cron/daily` (snapshot harian) |

Audit situs (PageSpeed + crawler on-page) **tidak** memerlukan kredensial dan sudah nyata sejak awal.

Semua rahasia hanya dibaca di server. Token koneksi disimpan terenkripsi (AES-256-GCM) di kolom `Connection.*Enc`.

## Struktur

```
prisma/            schema, migrasi, seed
src/app/(app)      halaman di dalam shell (sidebar + topbar)
src/app/share      dashboard publik klien
src/app/api        route handler (auth, koneksi OAuth, PDF, cron, AI)
src/features/*     modul: overview, social, seo, ads, reports, clients, admin
src/lib/metrics    semua rumus (murni, di-test)
src/lib/providers  adapter Meta, Google, PageSpeed, crawler, OpenRouter (real + mock)
src/components     ui (shadcn), dashboard (KPI/chart/tabel), shell
src/i18n/id.ts     seluruh teks UI (Bahasa Indonesia)
```

Kontrak pengembangan ada di [CONVENTIONS.md](CONVENTIONS.md); desain & rencana ada di `../docs/plans/`.

## Deploy

Vercel: set semua env di atas (`NEXT_PUBLIC_APP_URL` = domain produksi), gunakan Postgres terkelola (Neon/Vercel Postgres), jalankan `pnpm prisma migrate deploy` saat build. Cron harian sudah terdaftar di `vercel.json`.

## Migrasi dari aplikasi Flutter lama

`scripts/migrate-firestore.ts` mengimpor ekspor JSON Firestore lama (`projects`, `users`) menjadi `Client` dan `User`. Password lama yang tersimpan plaintext akan di-hash saat impor — sarankan semua user melakukan reset setelah migrasi.
