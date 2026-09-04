# WHITE Analytics — Rencana Implementasi Fase 1

Mengacu ke design doc `2026-08-19-nextjs-analytics-dashboard-design.md`. Semua pekerjaan di folder baru `white-analytics/`.

> **Status: Fase 1 SELESAI (19 Agustus 2026).** Semua rute berjalan (24/24 smoke test), 88 unit test hijau, typecheck & lint bersih, build produksi sukses. Yang tersisa untuk Fase 2: OAuth Meta/Google nyata (butuh kredensial), penjadwalan cron di Vercel, migrasi data Firestore, deploy. Detail hasil ada di bagian "Hasil akhir" di bawah.

## Versi paket (dicek 2026-08-19)
next 16.3 · react 19.2 · prisma 7.9 + @prisma/adapter-pg · next-auth 5.0.0-beta (Auth.js v5) · tailwindcss 4 · shadcn 4 · recharts 3 · @tanstack/react-table 8 (v9 API belum kompatibel) · zod 4 · date-fns 4 · cheerio 1.2 · papaparse 5.6 · @react-pdf/renderer 4.6 · bcryptjs 3 · vitest 4 · nuqs 2.9 (URL state) · next-themes · sonner

Google (GSC/GA4/PSI) dan Meta Graph dipanggil dengan `fetch` REST langsung (tanpa `googleapis` yang berat).

## Struktur folder
```
white-analytics/
  prisma/schema.prisma, prisma/seed.ts, prisma.config.ts
  src/app/(auth)/login
  src/app/(app)/… (layout dengan shell)   src/app/share/[slug]   src/app/api/…
  src/components/ui (shadcn) · src/components/dashboard (KpiTile, ChartCard, DataTable, DateRangePicker, EmptyState, PageHeader, Sparkline)
  src/components/shell (Sidebar, Topbar, ClientSwitcher, ThemeToggle, UserMenu)
  src/features/{clients,social,seo,ads,reports,admin}/{components,queries.ts,actions.ts}
  src/lib/{db,auth,metrics,format,dates,providers/{meta,google,pagespeed,crawler,openrouter},crypto,rbac,action-result}
  src/i18n/id.ts (kamus copy UI Bahasa Indonesia)
  tests/ (vitest)
  scripts/migrate-firestore.ts
```

## Langkah

### Tahap 0 — Fondasi (berurutan)
1. Scaffold `pnpm create next-app` (TS, Tailwind, ESLint, App Router, src/, alias `@/*`); init shadcn; pasang semua dependensi.
2. DB `white_analytics` di Postgres lokal; `prisma/schema.prisma` lengkap sesuai design §5; migrasi awal; Prisma client singleton (adapter-pg).
3. Token desain WHITE (CSS variables light/dark), Inter, `next-themes`; komponen shadcn yang dibutuhkan.
4. `lib/format` (id-ID, IDR, compact, persen, tanggal), `lib/dates` (preset, periode sebelumnya, parsing dari URL via nuqs), `lib/metrics` (semua rumus) + unit test.
5. Auth.js: Credentials + bcrypt, JWT, `auth()` helper, middleware proteksi `(app)`, halaman login, RBAC helper (`requireUser`, `requireAdmin`, `requireClientAccess`).
6. Shell: Sidebar (collapsible, nav per klien + global), Topbar (ClientSwitcher ⌘K, DateRangePicker + compare, status sync, theme, user menu). Layout `(app)`, layout `clients/[slug]`.
7. Primitif dashboard: KpiTile (delta + sparkline), ChartCard, DataTable (TanStack: sort, filter, pagination, export CSV), EmptyState, PageHeader, Section, StatusBadge, ProviderErrorBanner.
8. Seed realistis: 3 klien × 120 hari (social snapshots/posts/kompetitor, GSC harian + dimensi, GA4, audit + crawl + isu, ads hierarki + insight harian + demografi), user admin/member, laporan contoh.
9. `CONVENTIONS.md` untuk kontrak antar modul (pola query server, action, i18n, komponen, warna chart).
10. Contoh end-to-end: halaman Overview klien (`/clients/[slug]`) memakai semua primitif — jadi acuan modul lain.

### Tahap 1 — Modul (paralel, per feature folder)
- **Clients & Portfolio**: `/` portfolio, `/clients` list+create dialog, `/clients/[slug]/settings` (info, anggota, koneksi mock, share PIN), `/admin/users`.
- **Social**: `/clients/[slug]/social` (tab platform, KPI Δ, growth, reach/impr, ER per tipe, heatmap, top posts), `/social/competitors` (tambah/hapus, tabel banding, overlay growth). Adapter Meta (mock + kerangka nyata + exchange token).
- **SEO**: `/seo` overview (KPI Δ, clicks/impr, posisi, distribusi, device/negara, GA4), `/seo/keywords` (explorer + opportunities striking distance/low CTR/declining), `/seo/pages`, `/seo/audit` (gauge, CWV, isu, histori, **Run audit nyata** via PageSpeed + crawler cheerio sebagai job server action). Adapter Google (mock + kerangka nyata).
- **Ads**: `/ads` (KPI Δ, spend & results sebagai dua grafik terpisah (satu sumbu), drill-down campaign→adset→ad, demografi, top ads, **CSV import** upsert). Adapter Meta Marketing (mock + kerangka nyata).
- **Reports & Share & AI**: `/reports` (builder: modul, periode, compare, AI; preview; PDF `@react-pdf/renderer` via route handler; riwayat), `/share/[slug]` (PIN server-side + rate limit, tab modul, export PDF), `/api/ai/insight` (OpenRouter, mock jika tanpa key, cache AiInsight), `/api/cron/daily` (kerangka).

### Tahap 2 — Integrasi & verifikasi
- Typecheck, lint, vitest hijau; jalankan dev server; verifikasi tiap halaman di browser (screenshot), perbaiki.
- README (setup, env, seed, jalankan), `.env.example`, `scripts/migrate-firestore.ts` (kerangka + mapping).

## Kontrak penting
- Semua teks UI dari `src/i18n/id.ts` (Bahasa Indonesia).
- Query data hanya di server (`queries.ts`, RSC/server actions); komponen chart adalah client component yang menerima data siap-pakai.
- Periode & compare dibaca dari search params `?from&to&compare=1` (nuqs), default 28 hari terakhir.
- Delta selalu dihitung dengan `lib/metrics/delta.ts` (periode ini vs sebelumnya sepanjang sama).
- Tidak menambah dependensi tanpa dicatat di laporan.

---

## Hasil akhir (19 Agustus 2026)

### Rute yang berjalan
`/login` · `/` (Portofolio) · `/clients` · `/clients/[slug]` (Ringkasan) · `/social` (+`?platform=`) · `/social/competitors` · `/seo` · `/seo/keywords` (+`?view=`) · `/seo/pages` · `/seo/audit` · `/ads` · `/reports` (tab builder/history/insight/share) · `/settings` · `/admin/users` · `/share/[slug]` (publik, PIN opsional) · API: `/api/auth/*`, `/api/connections/{meta,google}/{start,callback}`, `/api/reports/[id]/pdf`, `/api/share/[slug]/pdf`, `/api/cron/daily`.

### Yang nyata (bukan mock) sejak Fase 1
- **Audit SEO**: PageSpeed Insights (mobile + desktop) dan crawler on-page sendiri (robots.txt, sitemap, BFS same-origin, 16 kode isu).
- **Impor CSV Meta Ads**: parser header EN/ID, upsert anti-duplikat (verifikasi: impor dua kali → jumlah baris tetap).
- **PDF laporan & PDF share** (`@react-pdf/renderer`), **PIN share** (bcrypt + rate limit), **RBAC** di server, **enkripsi token** AES-256-GCM.
- **Ringkasan strategis**: template berbasis angka nyata bila `OPENROUTER_API_KEY` kosong; OpenRouter bila diisi.

### Masih mock (aktif otomatis saat kredensial diisi)
Meta Graph (Instagram/Facebook), Meta Marketing (Ads), Google Search Console & GA4 — masing-masing punya adapter `real.ts` lengkap dengan pemetaan error; `index.ts` memilih real/mock dari env. TikTok tetap mock (Research API perlu persetujuan).

### Verifikasi
- `pnpm typecheck`, `pnpm lint` (0 error), `pnpm test` (88 test), `pnpm build` — semua lolos.
- `pnpm smoke` (skrip baru, login + cek 24 rute + proteksi rute tanpa sesi) — 24/24 OK.
- Pemeriksaan visual di browser: light & dark, 1440px; PDF diperiksa halaman per halaman.

### Catatan teknis penting
- TanStack Table dipasang di **v8** (v9 punya API baru yang belum kompatibel dengan pola shadcn).
- Metrik "hasil"/CPR mengecualikan kampanye awareness (`resultType` reach/impressions) agar satuan tidak tercampur.
- CTR iklan memakai `linkClicks/impressions` (memperbaiki rumus lama `results/impressions`).
- Grafik selalu satu sumbu Y; dua ukuran berbeda skala dirender sebagai dua grafik berdampingan.
