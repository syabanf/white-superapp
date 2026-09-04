# WHITE Super App

Platform pemasaran digital untuk tim agency WHITE: analitik & laporan **Social Media, SEO, Meta Ads**, publikasi konten (komposer, kalender, persetujuan, penjadwalan), dan SEO suite (riset kata kunci, rank tracker, backlink, kompetitor domain).

| Folder | Isi |
|---|---|
| [`white-analytics/`](white-analytics/) | Aplikasi Next.js 16 + PostgreSQL/Prisma + Auth.js — **aplikasi utama**. Mulai dari [white-analytics/README.md](white-analytics/README.md); kontrak kode di [CONVENTIONS.md](white-analytics/CONVENTIONS.md). |
| [`docs/plans/`](docs/plans/) | Design doc & rencana implementasi (rebuild 2026-08-19, ekspansi platform 2026-09-02). |
| `.claude/launch.json` | Konfigurasi dev server (`pnpm --dir white-analytics dev --port 3100`). |

## Menjalankan

```bash
cd white-analytics
pnpm install
cp .env.example .env   # isi DATABASE_URL, AUTH_SECRET, ENCRYPTION_KEY (lihat komentar di file)
pnpm db:migrate && pnpm db:seed
pnpm dev --port 3100
```

Login demo: `admin@white.id / admin12345`. Kredensial integrasi (Meta, Google, DataForSEO, Apify, OpenRouter) diisi dari dalam aplikasi: **Admin → Setup awal & integrasi**.

## Verifikasi

```bash
pnpm typecheck && pnpm lint && pnpm test   # unit
pnpm sync:smoke                            # pipeline sinkronisasi vs DB (adapter mock)
pnpm build
```

## Aplikasi Flutter lama

`white_super_app/` (dashboard Meta Ads generasi pertama) belum dipublikasikan di repo ini karena masih berisi kunci API hard-coded; lihat catatan di `.gitignore`. Rotasi kunci dan pindahkan ke env sebelum menambahkannya.
