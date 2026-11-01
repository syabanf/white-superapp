/**
 * Guide section for the client content workflow: editorial plans, the content
 * board, design tasks, the brand asset library and monthly content reports.
 * Brand Kit and the AI design studio live in the "konten-ai" section.
 * Labels mirror src/features/content/strings.ts.
 */
import type { GuideSection } from "@/features/guide/content";

export const CONTENT_GUIDE: GuideSection = {
  id: "konten",
  title: "Produksi konten: EP, papan, desain, aset, laporan",
  summary: "Alur konten klien dari editorial plan bulanan sampai terbit dan dilaporkan.",
  blocks: [
    {
      kind: "steps",
      items: [
        {
          title: "Buat editorial plan",
          text: "Buka Konten → Editorial plan, klik “Editorial plan baru”, dan pilih bulan. Lengkapi brief: tujuan, pilar konten, frekuensi per minggu, dan gaya bahasa. Tambah konten satu per satu lewat “Tambah konten”, atau klik “Buat draf dengan AI” agar AI mengusulkan konten dari brief dan post terbaik klien.",
        },
        {
          title: "Ajukan dan setujui EP",
          text: "Klik “Ajukan review” saat EP siap. Penyetuju memilih “Setujui” atau “Minta revisi” (catatan wajib, EP kembali ke draf). Orang yang mengajukan tidak bisa menyetujui EP-nya sendiri; persetujuan harus dari orang lain atau admin.",
        },
        {
          title: "Kerjakan di papan konten",
          text: "Konten → Papan konten menampilkan kolom dari Ide sampai Terbit. Seret kartu ke kolom yang disorot atau pakai menu di kartu. Perpindahan yang tersedia mengikuti peran Anda, dan pindah ke Revisi butuh catatan. Di detail konten, setiap simpan caption membuat versi baru yang bisa dipulihkan lewat “Pakai versi ini”.",
        },
        {
          title: "Minta dan review desain",
          text: "Dari detail konten, klik “Buat tugas desain” dan isi brief, referensi, ukuran, dan tenggat. Desainer membuka Konten → Desain, klik “Mulai kerjakan”, lalu “Unggah versi baru”; untuk carousel, urutan file adalah urutan slide. Reviewer memberi masukan, meminta revisi, atau menyetujui desain.",
        },
        {
          title: "Jadwalkan ke Publikasi",
          text: "Setelah desain final disetujui, klik “Jadwalkan” di detail konten. Desain diunggah ke Publikasi, post dibuat, disetujui, dan dijadwalkan otomatis. Klik “Sinkronkan status” untuk menarik hasil publikasi dan tautan post.",
        },
      ],
    },
    {
      kind: "defs",
      items: [
        { term: "Status EP", text: "Draf → Menunggu review → Disetujui. Brief bisa diubah selama EP masih draf." },
        { term: "Status konten", text: "Ide → Copywriting → Desain → Review → Disetujui → Terjadwal → Terbit. Revisi memulai putaran baru; Gagal berarti publikasi tidak berhasil." },
        { term: "Status desain", text: "Belum mulai → Dikerjakan → Menunggu review → Perlu revisi atau Disetujui." },
        { term: "Terlambat", text: "Konten yang melewati tenggat tahapnya saat ini: tenggat copy untuk Ide dan Copywriting, tenggat desain untuk Desain, tenggat review untuk Review dan Revisi." },
        { term: "Engagement rate", formula: "engagement ÷ jangkauan", text: "Engagement = suka + komentar + bagikan + simpan dari post terbit." },
      ],
    },
    {
      kind: "list",
      items: [
        "Aset brand (Konten → Aset): pustaka logo, brand guideline, font, foto, dan template klien, dikelompokkan dalam folder dan tag. Klik “Unggah aset” (maksimal 25 MB per file). “Versi baru” menjadikan file baru versi aktif, dan versi lama tetap bisa diunduh.",
        "Laporan konten (Konten → Laporan): klik “Buat laporan” untuk bulan yang sudah berjalan. Laporan merangkum rencana vs terbit, ketepatan waktu, revisi, engagement per platform dan format, post terbaik dan terlemah, serta ringkasan dan rekomendasi AI. Klik “Hitung ulang” setelah data berubah, dan “Unduh PDF” untuk klien.",
        "Ringkasan konten (Konten): dasbor tab Ringkasan, Produksi, Kalender, Performa, serta Aset & desain. Periodenya mengikuti pemilih tanggal di bilah atas.",
      ],
    },
    {
      kind: "note",
      tone: "warn",
      text: "Merevisi caption konten yang sudah disetujui mengembalikannya ke Review. Konten tidak bisa dijadwalkan sebelum ada desain final yang disetujui.",
    },
    {
      kind: "note",
      tone: "info",
      text: "Pesan “Layanan konten tidak dapat dihubungi” berarti backend Mixist sedang tidak berjalan. Muat ulang halaman setelah layanan kembali.",
    },
  ],
};
