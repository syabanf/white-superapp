/**
 * SEO module — NEW UI strings (Bahasa Indonesia).
 * Existing keys live in src/i18n/id.ts under t.seo — do not duplicate them here.
 * This file exists so parallel agents don't conflict on id.ts.
 */
export const s = {
  // ── Ringkasan SEO ──────────────────────────────────────────
  clicksCardDesc: "Klik organik harian vs periode sebelumnya",
  impressionsCardDesc: "Impresi harian vs periode sebelumnya",
  positionCardDesc: "Posisi rata-rata harian — lebih rendah lebih baik",
  ga4CardTitle: "Sesi organik vs total sesi",
  ga4CardDesc: "Sesi harian GA4: organik vs kanal lain",
  deviceDesc: "Pangsa klik per perangkat",
  countryDesc: "Pangsa klik per negara",
  tablet: "Tablet",
  queriesUnit: "kata kunci",
  healthHint:
    "Gabungan skor Lighthouse (mobile): performa 30%, SEO 40%, aksesibilitas 15%, praktik terbaik 15%, dikurangi penalti isu crawl.",
  organicHint: "Sesi dari kanal Organic Search (GA4).",
  conversionsHint: "Total konversi GA4 (semua kanal).",
  clicksHint: "Klik organik dari Google Search Console.",
  impressionsHint: "Berapa kali situs muncul di hasil pencarian Google.",
  positionHint: "Posisi rata-rata berbobot impresi. Lebih rendah lebih baik.",

  // ── Explorer kata kunci ────────────────────────────────────
  allDesc: "Semua kata kunci pada periode ini, diurutkan berdasarkan klik.",
  bucket: "Bucket",
  potentialHint: "Perkiraan tambahan klik bila CTR mencapai kurva ekspektasi (CTR rendah) atau posisi naik ke top 3 (striking distance).",
  searchQuery: "Cari kata kunci…",
  searchPage: "Cari halaman…",
  searchUrl: "Cari URL…",
  deltaVsPrev: "Δ vs periode sebelumnya",

  // ── Halaman ────────────────────────────────────────────────
  clicksPerPage: "Klik per halaman",
  clicksPerPageDesc: "10 halaman teratas berdasarkan klik organik",
  allPages: "Semua halaman",

  // ── Audit situs ────────────────────────────────────────────
  auditThrottled: "Audit terakhir dijalankan kurang dari 10 menit lalu. Coba lagi beberapa menit.",
  auditNoProperty: "Properti SEO tidak ditemukan.",
  auditUnauthorized: "Anda tidak punya akses ke klien ini.",
  auditDone: "Audit selesai",
  auditPartial: "Audit selesai sebagian",
  auditAllFailed: "Audit gagal dijalankan",
  psiLabel: "PageSpeed",
  crawlLabel: "Crawl",
  historyDesc: "Skor Lighthouse (Mobile) dari waktu ke waktu",
  crawlSummary: "Ringkasan crawl",
  duration: "Durasi",
  issuesTab: "Isu",
  pagesTab: "Halaman",
  crawlPagesTitle: "Halaman hasil crawl",
  crawlPagesDesc: "Detail on-page tiap URL yang di-crawl",
  topCodes: "Isu terbanyak",
  topCodesDesc: "Jumlah isu per kode",
  codeCol: "Kode",
  messageCol: "Pesan",
  titleLenCol: "Title",
  metaLenCol: "Meta",
  wordsCol: "Kata",
  imgsNoAltCol: "Img tanpa alt",
  indexableCol: "Indexable",
  loadMsCol: "Muat",
  cooldownNote: "Audit berikutnya dapat dijalankan ±10 menit setelah audit terakhir.",
  noCrawl: "Belum ada crawl",
  strategyLabel: "Strategi",
  charsUnit: "karakter",
  syncDoneReal: (daily: number, ga4: number) => `Search Console ${daily} hari, GA4 ${ga4} hari diperbarui.`,
  syncDoneDemo: "Belum ada koneksi Google untuk properti ini — data demo tetap dipakai. Hubungkan di Pengaturan Klien.",
} as const;
