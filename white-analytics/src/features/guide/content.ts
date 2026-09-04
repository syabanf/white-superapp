/**
 * In-app guide content. One source of truth: the full /panduan page and the
 * contextual help sheet in the topbar both render from here.
 *
 * Formulas quoted below must match src/lib/metrics/* — if a formula changes
 * there, change it here too (tests/guide.test.ts checks the ids stay wired).
 */

export type GuideBlock =
  | { kind: "p"; text: string }
  | { kind: "steps"; items: { title: string; text: string }[] }
  | { kind: "list"; items: string[] }
  | { kind: "defs"; items: { term: string; formula?: string; text: string }[] }
  | { kind: "note"; tone: "info" | "warn"; text: string };

export type GuideSection = {
  id: string;
  title: string;
  summary: string;
  blocks: GuideBlock[];
};

import { SEO_SUITE_GUIDE } from "@/features/seo-suite/guide";
import { PUBLISHING_GUIDE } from "@/features/publishing/guide";

export const GUIDE_SECTIONS: GuideSection[] = [
  {
    id: "mulai",
    title: "Mulai cepat",
    summary: "Tiga langkah dari nol sampai laporan pertama.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Buat project",
            text: "Klik “Klien baru” di sidebar. Pilih jenis project (SEO, Social Media, Meta Ads, atau gabungan) — langkah berikutnya menyesuaikan pilihan itu. Isi nama klien; alamat dashboard dibuat otomatis dari nama.",
          },
          {
            title: "Hubungkan sumber data",
            text: "Di langkah “Sumber data”, hubungkan Meta (Instagram/Facebook/Ads) dan Google (Search Console/GA4). Belum punya kredensial? Lewati saja — dashboard tetap terbuka dengan data demo, dan Anda bisa menghubungkannya nanti lewat Pengaturan Klien.",
          },
          {
            title: "Buat laporan",
            text: "Buka menu Laporan, pilih modul dan periode, lalu unduh PDF. Untuk klien, aktifkan tautan read-only di tab “Link klien” — mereka bisa melihat dashboard tanpa akun.",
          },
        ],
      },
      {
        kind: "note",
        tone: "info",
        text: "Tanda “Data demo” pada sebuah halaman berarti angkanya contoh, bukan data akun Anda. Tanda itu hilang begitu sumber data nyata terhubung.",
      },
    ],
  },
  {
    id: "periode",
    title: "Periode & perbandingan",
    summary: "Cara kerja filter tanggal di kanan atas — berlaku untuk seluruh halaman.",
    blocks: [
      {
        kind: "p",
        text: "Filter tanggal di topbar mengatur SEMUA angka, grafik, dan tabel di halaman yang sedang dibuka. Jadi KPI dan grafik tidak akan pernah berbeda periode.",
      },
      {
        kind: "list",
        items: [
          "Preset cepat: 7, 14, 28, 30, 90 hari, bulan ini, bulan lalu, 12 bulan.",
          "Rentang kustom lewat kalender; periode selalu berakhir kemarin karena data hari berjalan belum lengkap.",
          "Sakelar “vs periode sebelumnya” menghidupkan badge naik/turun. Pembandingnya adalah periode sepanjang sama tepat sebelum periode terpilih.",
        ],
      },
      {
        kind: "note",
        tone: "info",
        text: "Badge hijau tidak selalu berarti baik. Untuk metrik biaya dan posisi, turun justru bagus — sistem sudah membalik warnanya secara otomatis.",
      },
    ],
  },
  {
    id: "social",
    title: "Social Media",
    summary: "Followers, engagement, jangkauan, konten teratas, dan kompetitor.",
    blocks: [
      {
        kind: "p",
        text: "Pilih platform lewat tab (Instagram, Facebook, TikTok). Angka besar di atas adalah KPI utama; sisanya ada di strip tipis di bawahnya.",
      },
      {
        kind: "defs",
        items: [
          {
            term: "Engagement rate",
            formula: "(suka + komentar + bagikan + simpan) ÷ followers × 100",
            text: "Dihitung per postingan, lalu dirata-ratakan untuk periode terpilih.",
          },
          {
            term: "Pertumbuhan followers",
            text: "Selisih followers akhir dan awal periode. Diambil dari snapshot harian, bukan perkiraan.",
          },
          {
            term: "Waktu posting terbaik",
            text: "Peta hari × jam (WIB) berisi rata-rata interaksi. Makin gelap makin tinggi. Gunakan untuk menjadwalkan konten.",
          },
        ],
      },
      {
        kind: "p",
        text: "Halaman Kompetitor membandingkan akun klien dengan akun pesaing. Grafik pertumbuhannya diindeks ke 100 di awal periode supaya akun besar dan kecil bisa dibandingkan adil.",
      },
    ],
  },
  {
    id: "seo",
    title: "SEO",
    summary: "Search Console, GA4, audit teknis, dan peluang kata kunci.",
    blocks: [
      {
        kind: "defs",
        items: [
          { term: "CTR", formula: "klik ÷ impresi × 100", text: "Berapa persen orang yang melihat hasil Anda lalu mengkliknya." },
          {
            term: "Posisi rata-rata",
            formula: "Σ(posisi × impresi) ÷ Σ impresi",
            text: "Dibobot impresi, jadi kata kunci besar lebih berpengaruh. Angka lebih kecil = lebih baik.",
          },
          { term: "Skor kesehatan", formula: "performa 30% + SEO 40% + aksesibilitas 15% + praktik terbaik 15% − penalti isu", text: "Ringkasan satu angka dari audit terakhir." },
        ],
      },
      {
        kind: "p",
        text: "Halaman Kata Kunci punya empat tab peluang yang sudah disaring otomatis:",
      },
      {
        kind: "defs",
        items: [
          { term: "Striking distance", text: "Posisi 4–15 dengan impresi tinggi. Paling cepat memberi hasil — sedikit dorongan bisa masuk 3 besar." },
          { term: "CTR rendah", text: "Sudah di halaman 1 tapi jarang diklik. Biasanya cukup perbaiki judul dan deskripsi halaman." },
          { term: "Menurun", text: "Klik turun ≥30% dibanding periode sebelumnya. Periksa apakah ada perubahan konten atau pesaing baru." },
          { term: "Naik", text: "Klik naik ≥30%. Perkuat yang sedang bekerja." },
        ],
      },
      {
        kind: "p",
        text: "Audit Situs menjalankan Lighthouse (mobile & desktop) sekaligus crawler internal yang memeriksa title, meta description, H1, canonical, gambar tanpa alt, tautan rusak, dan konten tipis. Tombol “Jalankan audit” bekerja tanpa kredensial apa pun.",
      },
      {
        kind: "note",
        tone: "warn",
        text: "Search Console biasanya telat 2–3 hari. Hari yang belum dilaporkan sengaja dibiarkan kosong pada grafik — bukan nol — supaya tidak terbaca seolah trafik anjlok.",
      },
    ],
  },
  {
    id: "ads",
    title: "Meta Ads",
    summary: "Belanja, hasil, efisiensi biaya, dan impor CSV.",
    blocks: [
      {
        kind: "defs",
        items: [
          { term: "Biaya / hasil (CPR)", formula: "belanja ÷ hasil", text: "Metrik efisiensi utama. Makin rendah makin baik." },
          { term: "CTR", formula: "klik tautan ÷ impresi × 100", text: "Memakai klik tautan, bukan semua klik — jadi mencerminkan minat sebenarnya." },
          { term: "CPC", formula: "belanja ÷ klik tautan", text: "Biaya per klik tautan." },
          { term: "CPM", formula: "belanja ÷ impresi × 1.000", text: "Biaya per seribu tayangan." },
          { term: "Frekuensi", formula: "impresi ÷ jangkauan", text: "Rata-rata berapa kali satu orang melihat iklan. Di atas 3 biasanya tanda iklan mulai jenuh." },
        ],
      },
      {
        kind: "note",
        tone: "info",
        text: "Kampanye awareness (hasilnya berupa jangkauan) sengaja dikeluarkan dari perhitungan “Hasil” dan CPR, supaya satuannya tidak tercampur dengan konversi.",
      },
      {
        kind: "p",
        text: "Belum menghubungkan akun? Gunakan “Impor CSV” dengan berkas ekspor dari Meta Ads Manager. Kolom dikenali otomatis (Inggris maupun Indonesia), dan mengimpor berkas yang sama dua kali tidak akan menggandakan data.",
      },
    ],
  },
  {
    id: "laporan",
    title: "Laporan & tautan klien",
    summary: "PDF untuk dikirim, atau dashboard read-only untuk dibuka klien.",
    blocks: [
      {
        kind: "steps",
        items: [
          { title: "Buat laporan PDF", text: "Menu Laporan → tab “Buat laporan”. Pilih modul, periode mengikuti filter global, lalu unduh. Riwayat laporan tersimpan di tab sebelahnya." },
          { title: "Ringkasan strategis", text: "Analisa naratif berbasis angka periode terpilih. Bisa disertakan ke dalam PDF." },
          { title: "Tautan klien", text: "Tab “Link klien” → aktifkan, salin URL. Tambahkan PIN 6 digit bila perlu. Klien hanya bisa melihat; tidak ada tombol ubah dan tidak ada data akun lain." },
        ],
      },
    ],
  },
  {
    id: "akses",
    title: "Akses & peran",
    summary: "Siapa boleh melihat dan mengubah apa.",
    blocks: [
      {
        kind: "defs",
        items: [
          { term: "Admin", text: "Akses penuh ke semua klien, bisa membuat/menghapus klien dan mengelola user." },
          { term: "Anggota", text: "Hanya melihat klien tempat ia didaftarkan." },
          { term: "Manajer (per klien)", text: "Boleh mengubah pengaturan klien tersebut." },
          { term: "Viewer (per klien)", text: "Hanya melihat." },
        ],
      },
      {
        kind: "note",
        tone: "info",
        text: "Menonaktifkan user di Manajemen User langsung berlaku pada permintaan berikutnya — sesi lamanya ikut ditolak.",
      },
      {
        kind: "p",
        text: "Setiap aksi yang berhasil atau gagal diiringi nada singkat. Bisa dimatikan lewat menu akun di kiri bawah → “Suara”.",
      },
    ],
  },
  PUBLISHING_GUIDE,
  SEO_SUITE_GUIDE,
  {
    id: "integrasi",
    title: "Setup awal & integrasi",
    summary: "Kredensial mana yang dibutuhkan, apa yang tetap jalan tanpa kredensial, dan mengapa publikasi butuh App Review.",
    blocks: [
      {
        kind: "p",
        text: "Menu Admin → Setup awal & integrasi menyimpan kredensial level aplikasi (terenkripsi) dan langsung mengaktifkan adapter nyata — tanpa restart. Nilai yang disimpan di sini menang atas variabel .env.",
      },
      {
        kind: "defs",
        items: [
          { term: "Meta (App ID/Secret)", text: "Dipakai untuk OAuth klien, analitik IG/FB, Meta Ads, dan publikasi. Publikasi sungguhan baru berjalan setelah izin instagram_content_publish & pages_manage_posts disetujui lewat App Review dan aplikasi dalam mode Live — sampai saat itu post dikirim lewat simulasi." },
          { term: "Google (Client ID/Secret)", text: "OAuth untuk Search Console dan GA4 per klien." },
          { term: "DataForSEO", text: "Sumber volume kata kunci, SERP harian, backlink, dan overview domain. Bayar per panggilan; gunakan estimator biaya di wizard sebelum menambah banyak kata kunci." },
          { term: "Apify", text: "Data publik lewat actor Apify: posisi SERP harian untuk rank tracker (dipakai bila DataForSEO kosong) dan followers/post publik Instagram, TikTok, Facebook untuk akun klien & kompetitor — tanpa App Review. Bayar per hasil; reach/impressions tetap butuh Meta." },
          { term: "OpenRouter", text: "Ringkasan AI di laporan. Tanpa kunci, ringkasan memakai templat." },
          { term: "TikTok", text: "Analitik nyata bila Apify diisi (profil & video publik); publikasi belum tersedia (butuh persetujuan Content Posting API)." },
        ],
      },
      { kind: "note", tone: "info", text: "Tanpa satu pun kredensial, seluruh aplikasi tetap bisa dipakai dengan data demo — cocok untuk onboarding tim sebelum kredensial siap." },
    ],
  },
];

export const GUIDE_IDS = GUIDE_SECTIONS.map((s) => s.id);

/** Which guide section is most useful for the page currently open. */
export function guideSectionForPath(pathname: string): string {
  if (pathname === "/clients/new") return "mulai";
  if (pathname.startsWith("/setup")) return "integrasi";
  if (pathname.includes("/publish")) return "publikasi";
  if (pathname.includes("/social")) return "social";
  if (/\/seo\/(research|rank|backlinks|competitors)(\/|$)/.test(pathname)) return "seo-suite";
  if (pathname.includes("/seo")) return "seo";
  if (pathname.includes("/ads")) return "ads";
  if (pathname.includes("/reports")) return "laporan";
  if (pathname.includes("/settings")) return "akses";
  if (pathname.startsWith("/admin")) return "akses";
  return "mulai";
}

export function getGuideSection(id: string): GuideSection | undefined {
  return GUIDE_SECTIONS.find((s) => s.id === id);
}
