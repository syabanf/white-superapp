# WHITE Analytics → Platform Pemasaran Digital — Design Doc (Fase A + B)

Tanggal: 2 September 2026. Melanjutkan `2026-08-19-nextjs-analytics-dashboard-design.md`.

> **Status: Fase A + B SELESAI (2 September 2026).** 9 rute baru (`/publish`, `/publish/posts`, `/publish/posts/[id]`, `/publish/new`, `/publish/media`, `/seo/research`, `/seo/rank`, `/seo/backlinks`, `/seo/competitors`) + 2 cron + `/api/media`. 157 unit test hijau, typecheck/lint bersih, build produksi sukses, semua rute 200 dan tanpa scroll horizontal di 375 px. Publish nyata & data SEO nyata menunggu kredensial (lihat §7).

## 1. Tujuan

Mengubah WHITE Analytics dari **dashboard pelaporan** menjadi **platform yang bekerja**: gabungan
inti Hootsuite (publikasi & kalender konten dengan alur persetujuan) dan inti Semrush (riset kata
kunci, rank tracking, backlink, benchmark kompetitor). Analitik yang sudah ada tetap menjadi
fondasi; modul baru menempel pada `Client`, `SocialAccount`, dan `SeoProperty` yang sudah ada.

Fase yang dikerjakan sekarang: **A (Publishing) dan B (SEO Suite) secara paralel**. Fase C
(inbox & listening) dan D (content intelligence) menyusul.

## 2. Keputusan

| # | Keputusan | Alasan |
|---|---|---|
| 1 | Modul baru = folder fitur baru: `features/publishing`, `features/seo-suite`. Tidak menyentuh modul lama kecuali titik integrasi yang disepakati. | Dua agen bekerja paralel tanpa konflik file. |
| 2 | Semua data pihak ketiga lewat adapter `src/lib/providers/*` dengan `mock` deterministik sebagai default. | Publish nyata butuh App Review Meta; data SERP/volume/backlink butuh langganan DataForSEO. Produk harus utuh & bisa didemo tanpa keduanya. |
| 3 | Satu penyedia data SEO: **DataForSEO** (Basic auth, bayar per panggilan). | Satu API menutup keyword ideas, search volume, SERP, backlinks, domain overview. |
| 4 | Post yang dibuat di app = `ContentPost` (+ `PostTarget` per akun tujuan). `SocialPost` tetap = riwayat hasil sinkronisasi. | Siklus hidup berbeda; jangan dicampur. |
| 5 | Media disimpan lewat `src/lib/storage.ts` (disk lokal, disajikan `/api/media/[...key]`). | Meta mengambil media via URL publik. Ganti ke S3/R2/Blob = satu file. |
| 6 | Jadwal publish dieksekusi cron `GET /api/cron/publish` tiap 5 menit; tugas harian SEO suite menumpang `GET /api/cron/daily`. | Vercel Cron; Hobby plan hanya harian → catatan deploy. |
| 7 | Notifikasi in-app lewat `src/lib/notify.ts` → tabel `Notification`; lonceng di topbar. | Alur persetujuan & alarm peringkat butuh kanal ke orang. |
| 8 | Peran: ADMIN & MANAGER membuat/menyetujui/menjadwalkan; VIEWER hanya melihat. Penyetuju ≠ penulis (kecuali ADMIN). | Four-eyes ringan, cocok ukuran agency. |
| 9 | Mobile-ready adalah syarat setiap halaman baru (375 px: tanpa scroll horizontal, aksi utama lengket di bawah bila relevan). | Permintaan pengguna. |

## 3. Model data (ringkasan — sumber kebenaran: `prisma/schema.prisma`, migrasi `platform_expansion`)

**Publishing**: `MediaAsset` (pustaka), `ContentPost` (status `PostStatus`: DRAFT → IN_REVIEW → APPROVED → SCHEDULED → PUBLISHING → PUBLISHED | FAILED; REJECTED), `PostTarget` (per akun tujuan; hasil publish per target), `PostMedia`, `PostComment` (thread internal), `PostActivity` (jejak audit).

**SEO suite**: `TrackedKeyword` (+ metrik riset ter-cache), `RankSnapshot` (posisi harian, `topResults` top-10), `KeywordResearch` (cache hasil riset per seed), `BacklinkSnapshot` (harian), `Backlink` (baris tautan, `isLost`, `spamScore`), `CompetitorDomain`, `DomainSnapshot` (domain sendiri + kompetitor). `SeoProperty` bertambah `locationCode`, `languageCode`, `auditCadence`.

**Lintas**: `Notification`.

## 4. Rute & kepemilikan

```
src/app/(app)/clients/[slug]/publish/            features/publishing   (agen A)
  page.tsx (kalender)  posts/  posts/[id]/  new/  media/
src/app/(app)/clients/[slug]/seo/{research,rank,backlinks,competitors}/   features/seo-suite (agen B)
src/lib/providers/social-publisher/*             agen A
src/lib/providers/dataforseo/*                   agen B
prisma/seed-publishing.ts · prisma/seed-seo-suite.ts   (dipanggil dari seed.ts oleh integrator)
```

Titik integrasi yang dipegang integrator (bukan agen): schema, `nav-config`, `i18n/id.ts` (kunci nav), topbar, `guide/content.ts` (menyambungkan `GuideSection` yang diekspor tiap modul), rute cron, `seed.ts`, `CONVENTIONS.md`, lonceng notifikasi.

Kontrak entry point untuk cron:
- `features/publishing/scheduler.ts` → `publishDuePosts({ now?, limit? }) → { attempted, published, failed }`
- `features/seo-suite/daily.ts` → `runSeoSuiteDaily({ now? }) → { properties, ranks, backlinks, domains, audits }`

## 5. Adapter

**`social-publisher`** — `publish(input) → { externalId, permalink }`. Real: Instagram Graph (container → `media_publish`, carousel via children, polling status video), Facebook Page (`/feed`, `/photos`), komentar pertama via `/{id}/comments`. TikTok: `ProviderError(PERMISSION)` sampai Content Posting API disetujui. Mock: sukses deterministik; teks mengandung `[fail]` → gagal (untuk uji alur retry).

**`dataforseo`** — `keywordIdeas`, `keywordMetrics`, `serpRanks`, `backlinkSummary`, `backlinks`, `domainOverview`, `domainKeywords`, `competitors`. Real: REST v3 (`/v3/dataforseo_labs/google/*`, `/v3/serp/google/organic/live/regular`, `/v3/backlinks/*`). Mock: kata kunci Indonesia turunan seed, volume log-normal, KD berkorelasi volume, CPC IDR, posisi random-walk stabil per hari.

Env: `DATAFORSEO_LOGIN/PASSWORD`, `UPLOAD_DIR`; Meta memakai `META_APP_ID/SECRET` yang sudah ada.

## 5b. Setup awal (wizard level aplikasi, `/setup`)

Prasyarat integrasi dipindahkan dari catatan deploy ke dalam produk. Sekali per ruang kerja, ADMIN melewati lima langkah: **Meta** (App ID/Secret + uji koneksi + checklist App Review per izin + status Live/Business Verification + catatan TikTok), **Google** (OAuth client), **Data SEO** (DataForSEO login/password + uji koneksi yang menampilkan saldo + estimator biaya bulanan), **AI** (OpenRouter), **Selesai** (ringkasan mode nyata/demo). Kredensial disimpan terenkripsi (`IntegrationSetting`) dan dihidrasi ke `process.env` saat boot (`instrumentation.ts`) dan saat disimpan; adapter tetap membaca env. Semua langkah boleh dilewati. Admin pertama tanpa klien diarahkan ke `/setup`; wizard project (`/clients/new`) menautkan ke sana bila kredensial belum ada.

## 5c. Apify sebagai sumber data publik (3 September 2026)

Permintaan pengguna: pakai Apify API. Peran: (1) **SERP** — `apify/google-search-scraper` menggantikan DataForSEO untuk posisi harian bila DataForSEO kosong (volume/KD/backlink tetap DataForSEO); (2) **social publik** — profil & post Instagram/TikTok/Facebook untuk akun klien dan kompetitor tanpa App Review (followers, jumlah post, likes/komentar/share/views; reach & impressions tetap dari Meta Insights). Sinkronisasi harian lewat `/api/cron/daily` dan tombol "Sinkronkan". Token disimpan dari wizard setup (langkah 04 "Data publik (Apify)") dengan uji token + pemakaian bulanan dan estimator biaya. Prioritas: DataForSEO > Apify untuk SERP; Meta Insights + Apify untuk social. Publikasi tetap Meta.

## 5d. Sinkronisasi data nyata (3 September 2026)

Jalur nyata untuk tiga sumber resmi kini ada dan teruji dengan adapter mock (`pnpm sync:smoke`): **GSC + GA4** (harian, dimensi query/page/country/device, GA4 per kanal → sesi organik), **Meta Ads** (hierarki kampanye/set/iklan, insight harian level iklan, demografi usia×gender), dan **Meta Insights** akun sendiri (reach, impressions, profile views, insight per post). Semuanya menyala otomatis begitu klien punya `Connection` (OAuth dari Pengaturan Klien) dan kredensial aplikasi terisi di `/setup`; refresh token Google ditangani otomatis, token Meta yang kedaluwarsa memunculkan "Hubungkan ulang". Tanpa koneksi, cron mencatat `mock` dan data seed tetap dipakai. Publikasi IG/FB sudah tersambung ke token koneksi yang sama dan menunggu App Review.

## 6. Bukan bagian fase ini

Inbox/DM & listening (Fase C), pengoptimal on-page & asisten konten AI (Fase D), penagihan, multi-workspace, integrasi LinkedIn/X.

## 7. Risiko & catatan deploy

- Publish nyata IG/FB butuh izin `instagram_content_publish`, `pages_manage_posts`, `pages_read_engagement` → App Review (1–3 minggu). Semua alur dibangun & diuji dengan mock.
- Volume/SERP/backlink nyata butuh akun DataForSEO; biaya per panggilan kecil, tapi rank tracker harian × jumlah kata kunci perlu anggaran.
- Cron 5 menit tidak tersedia di Vercel Hobby; alternatif: worker eksternal memanggil `/api/cron/publish`.
- Penyimpanan media lokal tidak persisten di serverless → ganti ke object storage sebelum produksi.
