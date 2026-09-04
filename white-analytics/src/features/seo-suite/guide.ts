/**
 * In-app guide section for the SEO suite. The integrator appends it to GUIDE_SECTIONS and maps
 * /seo/{research,rank,backlinks,competitors} → "seo-suite" in guideSectionForPath.
 * Formulas quoted here must match src/lib/metrics/seo-suite.ts.
 */
import type { GuideSection } from "@/features/guide/content";

export const SEO_SUITE_GUIDE: GuideSection = {
  id: "seo-suite",
  title: "SEO Suite: riset, peringkat, backlink, kompetitor",
  summary: "Empat halaman ala Semrush yang menempel pada properti SEO klien — dengan pelacak harian otomatis.",
  blocks: [
    {
      kind: "p",
      text: "Riset Kata Kunci mencari ide dari satu kata kunci awal; Peringkat melacak posisi kata kunci pilihan setiap hari; Backlink memantau profil tautan; Kompetitor Domain membandingkan domain Anda dengan hingga 5 pesaing. Semua data pihak ketiga datang dari DataForSEO (pasar default: Indonesia, bahasa Indonesia — diatur per properti).",
    },
    {
      kind: "defs",
      items: [
        {
          term: "Visibilitas",
          formula: "rata-rata expectedCtr(posisi) dari semua kata kunci yang dilacak · tidak terindeks = 0",
          text: "Memakai kurva CTR ekspektasi yang sama dengan halaman Kata Kunci (posisi 1 ≈ 28 %, 3 ≈ 11 %, 10 ≈ 2,5 %, 11–20 = 1,5 %, >20 = 1 %). Semua kata kunci di posisi 1 → visibilitas 28 %. Angka ini menimbang posisi menurut peluang klik, jadi naik dari #8 ke #3 terasa jauh lebih besar daripada dari #40 ke #35.",
        },
        {
          term: "Posisi rata-rata",
          formula: "mean(posisi) untuk kata kunci yang masuk 100 besar",
          text: "Lebih rendah lebih baik; kata kunci yang tidak terindeks tidak ikut dihitung (agar rata-rata tidak melonjak saat satu kata kunci hilang).",
        },
        {
          term: "Δ periode",
          formula: "posisi awal periode − posisi terkini",
          text: "Positif berarti naik peringkat. “Terbaik” adalah posisi terendah yang pernah dicapai dalam periode.",
        },
        {
          term: "Share of voice",
          formula: "Σ expectedCtr(posisi) per domain di 10 besar, dibagi total",
          text: "Dihitung dari daftar 10 besar semua kata kunci yang dilacak pada snapshot terakhir. Delapan domain teratas ditampilkan, sisanya digabung ke “Lainnya”.",
        },
        {
          term: "KD (Keyword Difficulty)",
          formula: "0–100 · <30 mudah · 30–59 sedang · 60–79 sulit · ≥80 sangat sulit",
          text: "Perkiraan seberapa sulit masuk 10 besar, dari kekuatan halaman yang sudah ada di sana. Cari kata kunci volume tinggi dengan KD rendah.",
        },
        {
          term: "Intent",
          text: "Informasi (ingin tahu: “cara”, “apa itu”), Navigasi (mencari merek/situs tertentu), Komersial (membandingkan: “terbaik”, “review”), Transaksi (siap membeli: “harga”, “beli”, “promo”).",
        },
        {
          term: "Dofollow / nofollow",
          text: "Tautan dofollow meneruskan otoritas ke situs Anda; nofollow tidak (tetapi masih membawa kunjungan). Pangsa dofollow = dofollow ÷ total backlink.",
        },
        {
          term: "Domain Rank (DR)",
          text: "Skor otoritas 0–100 dari kekuatan profil backlink sebuah domain. Backlink dari DR tinggi bernilai lebih besar.",
        },
        {
          term: "Toxic",
          formula: "spam score ≥ 60",
          text: "Tautan dari situs berkualitas rendah/spam (link farm, PBN). Pertimbangkan untuk di-disavow bila jumlahnya besar.",
        },
      ],
    },
    {
      kind: "steps",
      items: [
        {
          title: "Riset → Lacak",
          text: "Masukkan kata kunci awal, pilih mode (Ide / Pertanyaan / Terkait), saring dengan volume minimum, KD maksimum, intent, dan kata yang harus/tidak boleh muncul. Centang kata kunci lalu klik “Lacak” — kata kunci masuk ke pelacak peringkat dengan volume, KD, dan intent yang sudah tersimpan. Hasil riset disimpan 7 hari agar API tidak dipanggil berulang.",
        },
        {
          title: "Kesenjangan kata kunci",
          text: "Di bagian bawah halaman Riset, pilih satu kompetitor domain. “Hilang” = kompetitor peringkat tapi Anda tidak (peluang konten baru); “Lemah” = keduanya peringkat, kompetitor lebih tinggi; “Kuat” = Anda lebih tinggi; “Bersama” = semua yang diperingkat keduanya.",
        },
        {
          title: "Pelacak harian",
          text: "Setiap hari cron memeriksa posisi seluruh kata kunci yang dilacak (per perangkat: mobile/desktop), menyimpan 10 besar tiap SERP untuk share of voice, mengambil ringkasan backlink & tautan baru/hilang, lalu memotret metrik domain Anda dan semua kompetitor. Tombol “Perbarui sekarang” menjalankan hal yang sama secara manual (jeda 10 menit).",
        },
        {
          title: "Notifikasi",
          text: "Anda mendapat notifikasi “Peringkat turun” bila sebuah kata kunci turun lebih dari 5 posisi, keluar dari 10 besar, atau hilang dari 100 besar dibanding kemarin; dan “Backlink hilang” bila lebih dari 5 tautan hilang dalam sehari.",
        },
        {
          title: "Jadwal audit",
          text: "Di halaman Audit Situs, pilih Tidak / Mingguan / Harian. Cron harian akan menjalankan audit Lighthouse + crawl sesuai jadwal, dan grafik “Tren isu” menunjukkan jumlah isu per crawl.",
        },
      ],
    },
    {
      kind: "note",
      tone: "info",
      text: "Tanda “Data demo” pada halaman suite berarti DATAFORSEO_LOGIN/PASSWORD belum diisi: volume, posisi, dan backlink adalah data sintetis yang deterministik (selalu sama untuk kata kunci/domain yang sama). Begitu kredensial diisi, pelacak harian mulai mengisi data nyata tanpa mengubah alur apa pun.",
    },
    {
      kind: "note",
      tone: "warn",
      text: "Dengan kredensial nyata, setiap pemeriksaan posisi dan riset berbayar per panggilan. Jumlah kata kunci yang dilacak × hari = biaya bulanan; mulai dari 30–50 kata kunci prioritas.",
    },
  ],
};
