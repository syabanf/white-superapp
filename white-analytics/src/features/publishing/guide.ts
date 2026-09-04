import type { GuideSection } from "@/features/guide/content";

/**
 * Guide section for the Publishing module. The integrator appends this to
 * GUIDE_SECTIONS and maps `/publish` paths to `id: "publikasi"` in
 * `guideSectionForPath`.
 */
export const PUBLISHING_GUIDE: GuideSection = {
  id: "publikasi",
  title: "Publikasi & kalender konten",
  summary: "Buat post untuk banyak akun sekaligus, lewati alur persetujuan, lalu jadwalkan — semua dari satu kalender.",
  blocks: [
    {
      kind: "steps",
      items: [
        {
          title: "Buat post",
          text: "Klik “Buat post” di kalender. Pilih akun tujuan (hanya akun milik klien), tulis satu caption utama, dan sesuaikan per platform bila perlu. Tambahkan media dari pustaka atau unggah langsung; tautan bisa diberi UTM otomatis.",
        },
        {
          title: "Kirim untuk review",
          text: "Simpan sebagai draf kapan saja. Saat siap, kirim untuk review — penyetuju mendapat notifikasi di lonceng. Aturan platform (Instagram wajib media, TikTok tepat satu video, batas karakter) dicek di tahap ini.",
        },
        {
          title: "Setujui atau tolak",
          text: "Admin dan Manager dapat menyetujui. Manager tidak bisa menyetujui post buatannya sendiri (four-eyes ringan); Admin boleh. Penolakan selalu disertai catatan yang dikirim ke penulis.",
        },
        {
          title: "Jadwalkan",
          text: "Post yang disetujui dijadwalkan minimal 5 menit ke depan, dalam zona waktu klien. Tiga saran “waktu terbaik” diambil dari riwayat engagement akun klien. Chip di kalender bisa diseret ke hari lain untuk mengubah jadwal.",
        },
        {
          title: "Terbit",
          text: "Penjadwal berjalan tiap beberapa menit dan menerbitkan post yang jatuh tempo ke setiap akun tujuan. Target yang gagal dicoba ulang sampai 3 kali; setelah itu post ditandai Gagal dan bisa dicoba lagi manual dari halaman detail.",
        },
      ],
    },
    {
      kind: "defs",
      items: [
        { term: "Draf", text: "Masih disunting; belum masuk alur review." },
        { term: "Menunggu review", text: "Menunggu keputusan penyetuju." },
        { term: "Disetujui", text: "Lolos review, belum punya jadwal aktif." },
        { term: "Terjadwal", text: "Akan diterbitkan otomatis pada waktu yang ditentukan." },
        { term: "Terbit / Gagal", text: "Hasil publikasi per akun tujuan tercatat di halaman detail, lengkap dengan tautan atau pesan error." },
      ],
    },
    {
      kind: "list",
      items: [
        "Peran: Admin & Manager membuat, menyunting, menjadwalkan, dan menyetujui. Viewer hanya melihat dan berkomentar.",
        "Menyunting post yang sudah direview mengembalikannya ke Draf — persetujuan harus diulang.",
        "Semua jam di modul ini memakai zona waktu klien (Pengaturan Klien), bukan zona perangkat Anda.",
        "Komentar internal tidak pernah ikut terbit; komentar pertama (Instagram) diposting otomatis setelah post tayang.",
      ],
    },
    {
      kind: "note",
      tone: "info",
      text: "Tanda “Data demo” pada halaman publikasi berarti kredensial Meta belum diatur: post tetap melewati seluruh alur, tetapi publikasinya disimulasikan (tautan hasil bersifat contoh). Caption yang mengandung “[fail]” sengaja digagalkan untuk menguji alur coba ulang.",
    },
  ],
};
