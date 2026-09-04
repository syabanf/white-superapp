/**
 * UI strings BARU khusus modul Social — melengkapi `t.social` di `src/i18n/id.ts`
 * (file i18n bersama tidak boleh diubah selama pengerjaan paralel).
 */
export const s = {
  platformTabsAria: "Pilih platform",

  // KPI hints
  followersHint:
    "Jumlah followers pada akhir periode. Delta membandingkan pertumbuhan periode ini dengan pertumbuhan periode sebelumnya.",
  growthHint: "Selisih followers antara awal dan akhir periode.",
  erHint:
    "Rata-rata (suka+komentar+bagikan+simpan) / followers × 100 per postingan yang terbit dalam periode.",
  reachHint: "Total jangkauan harian akun selama periode.",
  impressionsHint: "Total impresi harian akun selama periode.",
  postsHint: "Jumlah postingan yang terbit dalam periode.",

  // Charts
  reachChartDesc: "Jangkauan (area) dan impresi (garis) per hari — satuan sama",
  heatmapValue: "Rata-rata interaksi",
  heatmapCount: "Jumlah postingan",
  hour: "Jam",
  day: "Hari",
  days: ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"],
  index: "Indeks",
  indexBase: "Awal periode = 100",
  noPosts: "Tidak ada postingan pada periode ini.",
  post: "postingan",

  // Sync
  syncSuccess: "Sinkronisasi selesai",
  postsUpdated: "post diperbarui",
  viaApify: "via Apify",
  syncNoAccount: "Tidak ada akun untuk disinkronkan.",
  accountsUpdated: "akun diperbarui",

  // Competitors
  account: "Akun",
  avgEr: "ER rata-rata",
  addCompetitorDesc:
    "Masukkan username Instagram kompetitor (akun bisnis/kreator). Dalam mode demo, riwayat data akan dibuat otomatis.",
  usernameInvalid: "Username tidak valid. Gunakan 2–30 karakter huruf, angka, titik, atau garis bawah.",
  competitorExists: "Kompetitor dengan username ini sudah ada.",
  competitorAdded: "Kompetitor ditambahkan",
  competitorRemoved: "Kompetitor dihapus",
  removeConfirmTitle: "Hapus kompetitor?",
  removeConfirmDesc:
    "Semua data snapshot dan postingan kompetitor ini akan ikut terhapus. Tindakan ini tidak dapat dibatalkan.",
  topCompetitorPosts: "Postingan teratas kompetitor",
  topCompetitorPostsDesc: "Postingan terbaik lintas kompetitor berdasarkan total interaksi",
  noCompetitors: "Belum ada kompetitor",
  noCompetitorsDesc: "Tambahkan akun Instagram kompetitor untuk mulai membandingkan performa.",
  ownAccountMissing: "Akun Instagram sendiri belum terhubung — tabel hanya menampilkan kompetitor.",
} as const;
