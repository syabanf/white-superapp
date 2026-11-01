/**
 * In-app tutorial for the ERP workspace (/panduan/erp and the contextual help
 * sheet on /erp pages). Labels, buttons and status names below mirror the
 * strings.ts file of each ERP module; when a label changes there, change it here.
 */
import type { GuideSection } from "@/features/guide/content";

export const ERP_GUIDE_SECTIONS: GuideSection[] = [
  {
    id: "erp-mulai",
    title: "Mulai di ERP",
    summary: "Pindah ke ruang kerja ERP, kenali menu per izin, dan baca Beranda ERP.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Buka ruang kerja ERP",
            text: "Klik chip “Ruang kerja” di atas sidebar, lalu pilih “ERP”. Sidebar berganti ke menu Keuangan, SDM, Penjualan, dan Layanan saya. Pilih “Digital Marketing Platform” untuk kembali ke analitik klien.",
          },
          {
            title: "Cek menu yang Anda lihat",
            text: "Menu tampil sesuai izin akun. Keuangan butuh finance.read, SDM butuh hr.read, Payroll butuh hr.payroll, Rekrutmen butuh recruitment.read, Penjualan butuh sales.read, dan Layanan saya butuh self.service. Admin melihat semua menu, termasuk Penjualan → Otomasi.",
          },
          {
            title: "Baca Beranda ERP",
            text: "Kartu “Perlu perhatian” di atas mengumpulkan reimburse menunggu, cuti dan lembur menunggu persetujuan, tugas terlambat, deal macet, piutang lewat jatuh tempo, dan kandidat macet. Di bawahnya ada ringkasan Keuangan, Penjualan, SDM, Kinerja, dan Rekrutmen, lalu kartu “Semua modul” untuk membuka tiap modul.",
          },
          {
            title: "Atur periode",
            text: "Filter periode di Beranda dan dasbor modul punya preset Bulan ini, Bulan lalu, Kuartal ini, Tahun berjalan (YTD), dan 12 bulan terakhir, plus rentang kustom. Pembanding bisa “vs periode lalu” atau “vs tahun lalu”. Klik bulan di grafik untuk membuka data mentahnya, misalnya jurnal atau absensi bulan itu.",
          },
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "finance.write / finance.approve", text: "write untuk membuat draf jurnal dan transaksi kas & bank; approve untuk posting, tutup periode, pembatalan, pembayaran reimburse, dan kebijakan reimburse." },
          { term: "sales.write / sales.approve", text: "write untuk membuat lead, deal, quotation, dan invoice; approve untuk memutuskan permintaan persetujuan diskon." },
          { term: "self.service", text: "Akses ke halaman Layanan saya. Karyawan tanpa hr.read tetap melihat ringkasan Kinerja untuk dirinya sendiri." },
        ],
      },
      {
        kind: "note",
        tone: "warn",
        text: "Kotak merah “Layanan ERP belum terhubung” berarti backend ERP tidak bisa dihubungi. Dasbor dan tabel tidak dimuat sampai layanan kembali, jadi muat ulang halaman setelah backend berjalan. Jika muncul “Anda belum punya izin untuk modul itu”, minta admin menambahkan izinnya di Manajemen User.",
      },
    ],
  },
  {
    id: "erp-keuangan",
    title: "Keuangan: jurnal, kas, hutang-piutang, laporan",
    summary: "Dari bagan akun dan periode fiskal sampai laporan keuangan dan memo CFO.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Siapkan bagan akun",
            text: "Buka Keuangan → Bagan akun. Tambah akun lewat “Akun baru” atau “Impor XLSX” (sheet “coa”, periksa pratinjau sebelum “Impor sekarang”). Kode akun 7 digit. Centang “Akun kas/bank” agar akun muncul di Kas & bank. Tab “Mapping jurnal” mengatur akun yang dipakai posting otomatis per event.",
          },
          {
            title: "Buka tahun fiskal",
            text: "Di Keuangan → Periode fiskal, klik “Tahun fiskal baru”. Sistem membuat dua belas periode bulanan dan hanya periode pertama yang terbuka. Isi “Saldo awal” per akun neraca, lalu “Simpan & posting”. Tahun berikutnya mendapat saran dari neraca akhir tahun sebelumnya.",
          },
          {
            title: "Catat jurnal",
            text: "Keuangan → Jurnal → “Jurnal baru”. Isi tanggal, keterangan, dan minimal dua baris dengan akun berbeda. Simpan sebagai draf, atau langsung “Simpan & posting” bila Anda punya finance.approve. Jurnal draf manual bisa diubah dan dihapus; jurnal posted masuk buku besar dan terkunci.",
          },
          {
            title: "Kas, hutang, dan piutang",
            text: "Kas & bank mencatat kas masuk, kas keluar, dan transfer antar rekening; klik rekening untuk melihat buku kasnya. Hutang usaha (vendor) dan Piutang usaha (klien) memakai alur yang sama: buat tagihan, “Posting”, lalu “Catat pembayaran” atau “Catat penerimaan” dan alokasikan ke tagihan terbuka.",
          },
          {
            title: "Baca laporan",
            text: "Keuangan → Laporan keuangan berisi Neraca saldo, Neraca, Laba rugi, Arus kas, Buku besar, dan Buku pembantu. Analisa keuangan menampilkan rasio profitabilitas, likuiditas, solvabilitas, efisiensi, dan kas dibanding periode sebelumnya, plus “Memo CFO (AI)” lewat tombol “Buat memo”.",
          },
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "Seimbang / Belum seimbang", formula: "Selisih = total debit − total kredit", text: "Jurnal hanya bisa disimpan jika selisihnya nol dan ada minimal satu baris debit dan satu baris kredit." },
          { term: "Pelunasan", text: "Lunas, Sebagian, Jatuh tempo, atau Belum dibayar, dihitung dari sisa tagihan dan tanggal jatuh tempo." },
          { term: "Umur", text: "Tab Umur mengelompokkan sisa tagihan: Belum jatuh tempo, 1–30, 31–60, 61–90, dan lebih dari 90 hari." },
          { term: "Log posting", text: "Tab di halaman Jurnal untuk posting otomatis dari modul lain: Posted, Draf, atau Dilewati. Klik “Coba posting ulang” setelah mapping diperbaiki." },
        ],
      },
      {
        kind: "note",
        tone: "warn",
        text: "Jurnal hanya bisa dibuat di periode terbuka. Form jurnal memberi tahu bila periode untuk tanggal itu belum terbuka, dan pengguna dengan finance.approve bisa langsung klik “Buka periode”. Periode tidak bisa ditutup selama masih ada jurnal draf di dalamnya.",
      },
      {
        kind: "note",
        tone: "info",
        text: "Dasbor keuangan dan semua laporan hanya menghitung jurnal berstatus posted. Draf tidak memengaruhi angka. Membatalkan pembayaran lewat “Batalkan” membuat jurnal pembalik dan mengembalikan alokasinya ke tagihan.",
      },
    ],
  },
  {
    id: "erp-reimburse",
    title: "Reimburse: ajukan, setujui, bayar",
    summary: "Alur klaim biaya kerja dari karyawan, atasan, finance, sampai batch pembayaran.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Karyawan membuat klaim",
            text: "Buka Layanan saya → Reimburse saya → “Klaim baru”. Isi kategori, tanggal transaksi, jumlah, dan keterangan. Klaim tersimpan sebagai draf; unggah foto atau PDF struk di bagian Bukti, lalu klik “Ajukan”.",
          },
          {
            title: "Atasan menyetujui",
            text: "Atasan langsung melihat “Perlu persetujuan Anda” di Reimburse saya dan menu Layanan saya → Persetujuan reimburse. Klik “Setujui” (catatan opsional) atau “Tolak” (alasan wajib).",
          },
          {
            title: "Finance menyetujui",
            text: "Pemegang izin finance.approve membuka Keuangan → Reimburse, tab “Perlu persetujuan”, lalu menyetujui klaim yang sudah disetujui atasan. Karyawan tanpa atasan langsung di data SDM melewati tahap atasan.",
          },
          {
            title: "Finance membayar",
            text: "Di tab “Semua klaim”, centang klaim berstatus Disetujui finance, klik “Bayar”, pilih tanggal bayar dan rekening pembayaran. Klaim dibayar dalam satu batch dan jurnal dibuat dari mapping reimburse. Riwayatnya ada di tab “Batch pembayaran”.",
          },
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "Status klaim", text: "Draf → Diajukan → Disetujui atasan → Disetujui finance → Dibayar. Klaim bisa berakhir Ditolak atau Dibatalkan." },
          { term: "Melebihi batas", text: "Tanda pada klaim yang melewati batas per klaim atau batas per bulan di tab Kebijakan. Batas kosong berarti tanpa batas." },
          { term: "Lewati persetujuan finance", text: "Opsi kebijakan per kategori. Klaim dalam batas langsung siap dibayar setelah disetujui atasan. Klaim yang melebihi batas tetap menunggu persetujuan finance." },
        ],
      },
      {
        kind: "note",
        tone: "warn",
        text: "Klaim yang sudah diajukan tidak bisa diubah kecuali ditolak. Klaim yang dibatalkan tidak bisa diajukan lagi, jadi buat klaim baru bila perlu.",
      },
    ],
  },
  {
    id: "erp-sdm",
    title: "SDM: karyawan, absensi, cuti, lembur, kontrak",
    summary: "Data induk karyawan dan pekerjaan harian HRD, untuk pengguna dengan izin hr.read.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Tambah karyawan",
            text: "SDM → Karyawan → “Karyawan baru”. Kosongkan NIP untuk nomor otomatis dan isi atasan langsung agar struktur organisasi terbentuk. Di profil karyawan, tautkan akun login lewat “Tautkan akun login” supaya karyawan bisa membuka Layanan saya. Tab profil lain: Riwayat, Dokumen, Jadwal, Saldo cuti, Kontrak, dan On/offboarding.",
          },
          {
            title: "Kelola absensi",
            text: "SDM → Absensi punya tab Roster harian, Rekap absensi, Shift, Hari libur, dan Jadwal tim. Klik “Validasi” pada catatan untuk mengoreksi status, atau “Catat absensi” untuk input manual (jam dalam WIB). Rekap bisa diunduh lewat “Unduh XLSX”.",
          },
          {
            title: "Putuskan cuti dan lembur",
            text: "SDM → Cuti dan SDM → Lembur punya tab “Persetujuan tim”. Cuti disetujui dua tahap: atasan lalu HRD. Lembur bisa diajukan karyawan atau ditugaskan perusahaan lewat “Tugaskan lembur”; penugasan perlu dikonfirmasi karyawan.",
          },
          {
            title: "Kontrak dan pengumuman",
            text: "SDM → Kontrak kerja untuk PKWT dan PKWTT: buat draf, “Aktifkan”, lalu “Perpanjang”, “Konversi ke PKWTT”, “Akhiri”, atau “Putus kontrak”. Tab “Segera berakhir” menampilkan kontrak dan probation yang habis dalam rentang terpilih. SDM → Pengumuman menerbitkan informasi ke semua karyawan atau departemen tertentu.",
          },
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "Status cuti", text: "Menunggu atasan → Menunggu HRD → Disetujui. Bisa juga Ditolak atau Dibatalkan." },
          { term: "Status absensi", text: "Hadir, Terlambat, Alpha, Setengah hari, Remote. Roster harian juga memakai Belum absen, Cuti, Libur nasional, Libur, dan Tanpa jadwal." },
          { term: "Status kontrak", text: "Draf, Aktif, Berakhir, Diputus, Dikonversi. Saat kontrak PKWT diakhiri, kompensasi PKWT dihitung otomatis." },
        ],
      },
      {
        kind: "note",
        tone: "info",
        text: "Pinjaman dan kasbon karyawan dikelola di Payroll → tab Pinjaman, karena cicilannya dipotong dari gaji.",
      },
    ],
  },
  {
    id: "erp-payroll",
    title: "Payroll & slip gaji",
    summary: "Proses gaji bulanan dari draf sampai dibayar, untuk pengguna dengan izin hr.payroll.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Lengkapi data gaji",
            text: "Buka Payroll → tab Data gaji, lalu “Gaji baru” per karyawan: gaji pokok, tunjangan, status PTKP, dan kepesertaan BPJS. Versi gaji baru mengakhiri versi aktif sebelumnya. Bonus atau insentif periode ini masuk lewat tab Tambahan. Tarif BPJS, PTKP, lapisan PPh 21, lembur, dan potongan terlambat ada di tab Pengaturan.",
          },
          {
            title: "Buat dan hitung",
            text: "Tab Proses gaji → “Proses gaji baru”, pilih bulan dan tahun. Buka proses itu lalu klik “Hitung” (centang “Sertakan THR” bila perlu). Slip gaji per karyawan langsung terbentuk dan bisa dibuka satu per satu atau diunduh PDF.",
          },
          {
            title: "Setujui lalu tandai dibayar",
            text: "Klik “Setujui” setelah angka benar, lalu “Tandai dibayar” setelah transfer. Cicilan pinjaman periode itu dilunasi otomatis saat ditandai dibayar.",
          },
          {
            title: "Karyawan melihat slip",
            text: "Karyawan membuka Layanan saya → Slip gaji. Halaman ini hanya menampilkan slip dari payroll yang sudah dibayar, dengan filter tahun dan tombol unduh PDF.",
          },
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "Status proses gaji", text: "Draf → Dihitung → Disetujui → Dibayar." },
          { term: "Gaji bersih", text: "Pendapatan (gaji pokok, tunjangan, upah lembur, THR, bonus) dikurangi potongan (BPJS karyawan, Tapera, PPh 21, cuti tanpa gaji, terlambat, cicilan pinjaman). BPJS bagian perusahaan tampil terpisah di “Ditanggung perusahaan”." },
          { term: "Biaya payroll (Beranda)", text: "Gaji bruto ditambah BPJS bagian perusahaan." },
        ],
      },
      {
        kind: "note",
        tone: "warn",
        text: "Setelah disetujui, payroll tidak bisa dihitung ulang. Jika muncul “Data berubah sejak dihitung”, klik “Hitung ulang” sebelum menyetujui.",
      },
    ],
  },
  {
    id: "erp-rekrutmen",
    title: "Rekrutmen: lowongan sampai penawaran",
    summary: "Lowongan, kandidat, pipeline, psikotes, interview, dan talent pool, untuk izin recruitment.read.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Tayangkan lowongan",
            text: "Rekrutmen → Lowongan → “Lowongan baru”. Isi deskripsi, kualifikasi, rentang gaji, dan tanggal tutup, lalu klik “Tayangkan” agar muncul di halaman karier publik. Klik “Tutup lowongan” setelah posisi terisi.",
          },
          {
            title: "Saring kandidat",
            text: "Lamaran dari portal masuk ke Kandidat; tambah manual lewat “Kandidat baru”. Di profil kandidat, jalankan “Analisis” untuk skor kecocokan AI, isi tab Screening, lalu kirim tautan lewat “Kirim psikotes” atau “Kirim interview AI”.",
          },
          {
            title: "Gerakkan pipeline",
            text: "Rekrutmen → Pipeline menampilkan kandidat per tahap. Seret kartu ke kolom lain, atau pakai “Pindah tahap” di profil kandidat.",
          },
          {
            title: "Interview dan penawaran",
            text: "Rekrutmen → Jadwal interview → “Jadwalkan interview” (tatap muka atau online). Setelah wawancara, isi “Scorecard” untuk menyelesaikannya. Di tab Penawaran kandidat, “Buat penawaran” membuat versi baru, dan “Catat tanggapan” mencatat Negosiasi, Diterima, atau Ditolak.",
          },
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "Tahap kandidat", text: "Melamar → Screening → Psikotes → Interview → Penawaran → Diterima. Kandidat juga bisa masuk Talent pool, Ditolak, atau Diarsipkan." },
          { term: "Kandidat macet", text: "Kandidat yang diam di tahap yang sama melewati batas hari tahapnya. Muncul di Beranda ERP dan dasbor Rekrutmen." },
          { term: "Rata-rata hari rekrut", text: "Hari dari lamaran masuk sampai diterima, untuk rekrutan dalam periode. Makin kecil makin baik." },
        ],
      },
      {
        kind: "note",
        tone: "info",
        text: "Talent pool menyimpan kandidat potensial untuk lowongan berikutnya; “Aktifkan ke screening” memasukkannya lagi ke pipeline. Bank psikotes mengelola instrumen (pilihan ganda, PAPI, gambar) dan soalnya. Tes gambar perlu direview manual.",
      },
    ],
  },
  {
    id: "erp-kinerja",
    title: "Kinerja: tugas, KPI, review, 360",
    summary: "Tugas departemen, scorecard KPI bulanan, rapor kuartalan, feedback 360, logbook, dan rekap.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Jadwalkan tugas tim",
            text: "Kinerja → Task tim → “Tugas baru”. Pilih penanggung jawab atau seluruh departemen, pengulangan (Sekali, Harian, Mingguan, Bulanan), dan bila perlu indikator KPI beserta kontribusinya. Sub-tugas dibagi rata bobotnya dan menentukan progres.",
          },
          {
            title: "Atur KPI",
            text: "Kinerja → KPI. Tab Indikator untuk membuat indikator (otomatis atau manual), Bobot departemen untuk memilih indikator per departemen, dan Target untuk target umum, per departemen, atau per karyawan.",
          },
          {
            title: "Hitung scorecard",
            text: "Klik “Jalankan snapshot” untuk menarik angka aktual dari data operasional. Buka scorecard karyawan untuk mengisi aktual indikator manual dan rubrik atasan (1–5), lalu “Finalkan scorecard”.",
          },
          {
            title: "Review dan feedback 360",
            text: "Kinerja → Review kinerja → “Buka siklus” per kuartal; sistem membuat draf review untuk semua karyawan aktif. Kinerja → Feedback 360 → “Siklus baru”, lalu “Tugaskan otomatis” untuk membuat penugasan diri, atasan, bawahan, dan rekan. Setelah penilaian terkumpul, klik “Hitung ringkasan”.",
          },
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "Status tugas", text: "Belum dimulai → Dikerjakan → Tunggu review → Disetujui atau Ditolak. Kontribusi KPI dicatat saat tugas disetujui." },
          { term: "Skor KPI", text: "Skala 0–100 per bulan. Indikator tanpa data periode itu dikeluarkan dan bobotnya dibagi ulang ke indikator lain." },
          { term: "Nilai akhir review", text: "60% hasil kerja (rata-rata KPI kuartal), 30% perilaku, 10% kontribusi." },
          { term: "Nilai akhir 360", text: "Gabungan skor 360 dan KPI dengan bobot yang diatur per siklus. Bobot KPI dan bobot 360 harus berjumlah 100." },
        ],
      },
      {
        kind: "list",
        items: [
          "Logbook: ceklis kerja harian dari template departemen. Tim mengisi lalu “Ajukan logbook”; kepala departemen atau HR memberi “Tandai direview” atau “Tolak”.",
          "Rekap kerja: rekap harian per anggota tim dan rekap bulanan per karyawan atau departemen, dengan “Buat rekap” dan “Unduh PDF”.",
          "Things to improve: item perbaikan dari review, feedback 360, tugas atau logbook yang ditolak, retro, saran AI, dan input manual. Saran AI perlu “Terima” atau “Tolak”.",
        ],
      },
      {
        kind: "note",
        tone: "warn",
        text: "Snapshot tidak menyentuh scorecard berstatus final. Klik “Buka kembali” lebih dulu bila angka final perlu dihitung ulang.",
      },
    ],
  },
  {
    id: "erp-penjualan",
    title: "Penjualan: lead sampai onboarding",
    summary: "Alur lengkap dari lead, deal, quotation, invoice, kontrak, sampai onboarding klien.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Catat lead dan akun",
            text: "Buka Penjualan → Leads. Klik “Tambah lead” atau “Impor” (CSV/XLSX, maksimal 500 baris). Tab Akun dan Kontak menyimpan perusahaan klien dan PIC-nya. Dari detail lead, klik “Buat deal”.",
          },
          {
            title: "Gerakkan deal di pipeline",
            text: "Penjualan → Pipeline: seret kartu deal antar tahap, atau ganti ke tampilan Daftar. Pindah ke tahap menang meminta nilai final; pindah ke tahap kalah meminta alasan. Deal menang memulai onboarding klien otomatis.",
          },
          {
            title: "Susun quotation",
            text: "Penjualan → Quotation → “Quotation baru”. Pilih Pelanggan, lalu pilih deal di “Tarik dari deal”. Tambah item lewat “Tambah dari katalog” atau “Tambah item bebas”, isi Diskon %, nyalakan “Kenakan PPN”, dan susun Termin pembayaran. Tombol “Buat dari deal” di daftar quotation menyusun draf otomatis dari layanan deal, paket, dan diskon retainer.",
          },
          {
            title: "Kirim dan tutup quotation",
            text: "Klik “Kirim” pada draf. Setelah klien menjawab, klik “Diterima klien” atau “Ditolak klien”. “PDF” mengunduh dokumen penawaran. Untuk mengubah quotation yang sudah dikirim, klik “Revisi” sehingga versi baru dibuat dan versi lama berstatus Digantikan.",
          },
          {
            title: "Ajukan invoice dan catat pembayaran",
            text: "Pada quotation diterima, klik “Ajukan invoice dari termin”, atau “Ajukan invoice” di detail deal untuk nominal bebas. Finance membuka Penjualan → Invoice dan klik “Terbitkan”. Detail invoice punya tombol “Cetak”. Pembayaran dicatat di detail deal, bagian Invoice dan pembayaran, lewat “Catat pembayaran”.",
          },
          {
            title: "Kontrak dan onboarding",
            text: "Dari quotation diterima, klik “Buat kontrak”. Pilih template, periksa isi, lalu “Kirim untuk tanda tangan” dan bagikan tautan tanda tangan ke klien. Setelah ditandatangani, klik “Aktifkan”. Onboarding klien berisi checklist brief, aset, kontrak, pembayaran, dan “Buat klien di platform marketing”, plus tautan Portal klien untuk mengisi brief dan mengunggah aset.",
          },
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "Total quotation", formula: "Subtotal − Diskon + PPN", text: "Subtotal = jumlah (Qty × Harga satuan). Diskon = Subtotal × Diskon %. PPN = (Subtotal − Diskon) × PPN %, default 11%." },
          { term: "Termin pembayaran", text: "Persentase semua termin harus berjumlah 100% sebelum quotation bisa disimpan." },
          { term: "Status quotation", text: "Draf → Terkirim → Diterima atau Ditolak. Digantikan berarti sudah ada versi revisi yang lebih baru." },
          { term: "Status invoice", text: "Diajukan (oleh sales), Terkirim (sudah diterbitkan finance), atau Batal. Status bayar: Belum dibayar, Sebagian, Lunas." },
          { term: "Status kontrak", text: "Draf, Dikirim, Ditandatangani, Aktif, Berakhir, Dibatalkan." },
        ],
      },
      {
        kind: "note",
        tone: "warn",
        text: "Diskon yang melewati aturan persetujuan membuat quotation berstatus Menunggu persetujuan dan belum bisa dikirim. Penyetuju memutuskan di tab Persetujuan. Hanya draf yang bisa diubah, dan penerbitan atau pembatalan invoice butuh izin finance.",
      },
      {
        kind: "note",
        tone: "info",
        text: "Penjualan → Forecast & target menampilkan forecast per sales dan target bulanan; “Atur target” mengisi target nilai dan target deal. Analisis funnel ada di dasbor Penjualan.",
      },
    ],
  },
  {
    id: "erp-penjualan-tools",
    title: "Alat penjualan: pitch deck, chatbot, formulir, otomasi",
    summary: "Pendukung tim sales di luar alur deal utama.",
    blocks: [
      {
        kind: "list",
        items: [
          "Pitch deck: klik “Buat pitch deck”, pilih deal dan gaya bahasa (Profesional, Santai, Premium), lalu centang “Sertakan harga” atau “Sertakan studi kasus”. Hasilnya diunduh lewat “Unduh PPTX”; “Buat versi baru” menyimpan versi sebelumnya. Tab Studi kasus mengisi bahan deck.",
          "Chatbot leads: inbox percakapan web dan WhatsApp. Bot menjawab dulu; klik “Ambil alih” untuk membalas sendiri dan “Kembalikan ke bot” setelah selesai. Tab Pengaturan bot mengatur persona, layanan, petunjuk harga, jam kerja, dan kode widget untuk dipasang di website.",
          "Formulir lead: buat formulir publik lewat “Formulir baru”, atur field, sumber lead, dan suhu lead, lalu bagikan tautan publiknya. Setiap kiriman yang diterima membuat lead baru.",
          "Otomasi (khusus admin): aturan workflow dengan pemicu, kondisi, dan aksi (buat tugas, tetapkan pemilik, ubah field, notifikasi, tunggu), aturan skor lead, tahap pipeline beserta probabilitas dan batas diam, serta field kustom.",
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "Status percakapan", text: "Terbuka, Ditangani, Menunggu pelanggan, Selesai." },
          { term: "Status kiriman formulir", text: "Diterima, Spam, Tidak valid." },
          { term: "Batas diam (hari)", text: "Diatur per tahap pipeline. Deal yang melewatinya diberi tanda “Diam n hari” dan muncul sebagai deal macet di Beranda ERP." },
        ],
      },
      {
        kind: "note",
        tone: "info",
        text: "Setelah mengubah aturan skor, klik “Hitung ulang skor” agar skor lead lama ikut diperbarui.",
      },
    ],
  },
  {
    id: "erp-mandiri",
    title: "Layanan saya",
    summary: "Halaman karyawan untuk absensi, cuti, lembur, pinjaman, reimburse, slip gaji, tugas, dan rekap.",
    blocks: [
      {
        kind: "steps",
        items: [
          {
            title: "Clock in dan clock out",
            text: "Buka Layanan saya. Kartu shift hari ini menampilkan jadwal dan toleransi terlambat. Klik “Clock in” saat mulai dan “Clock out” saat pulang. Selfie dan lokasi opsional, tetapi membantu HRD memvalidasi absensi.",
          },
          {
            title: "Ajukan cuti dan lembur",
            text: "Cuti saya menampilkan saldo cuti dan riwayat; klik “Ajukan cuti”. Lembur saya: klik “Ajukan lembur”, atau “Setujui” dan “Tolak” untuk penugasan dari perusahaan. Pengajuan yang masih menunggu bisa dibatalkan lewat “Batalkan”.",
          },
          {
            title: "Ajukan pinjaman",
            text: "Pinjaman saya → “Ajukan pinjaman”: pilih Kasbon, Pinjaman, atau Pinjaman darurat, isi pokok dan tenor (1 sampai 24 bulan). Selama berstatus Menunggu, pinjaman bisa dibatalkan lewat “Batalkan”. Jadwal cicilan muncul setelah disetujui.",
          },
          {
            title: "Pantau tugas dan rekap",
            text: "Task saya mengelompokkan tugas: Terlambat, Hari ini, Mendatang, Menunggu review, Selesai. Kotak “Tambah tugas” memahami penanda seperti “besok”, nama hari, “setiap minggu”, serta ! atau !! untuk prioritas. Rekap harian saya meringkas tugas dan logbook hari ini dan menyimpan catatan pribadi.",
          },
        ],
      },
      {
        kind: "defs",
        items: [
          { term: "Status pinjaman", text: "Menunggu, Berjalan, Lunas, Ditolak, Dibatalkan." },
          { term: "Sisa cuti tahunan", text: "Jatah tahunan dikurangi cuti yang sudah terpakai tahun ini." },
        ],
      },
      {
        kind: "note",
        tone: "warn",
        text: "Pesan “Akun belum terhubung ke data karyawan” berarti akun login Anda belum ditautkan. Minta HRD menautkannya di halaman Karyawan.",
      },
      {
        kind: "note",
        tone: "info",
        text: "Reimburse saya dijelaskan di bagian Reimburse, dan Slip gaji di bagian Payroll & slip gaji.",
      },
    ],
  },
];

function under(pathname: string, prefix: string): boolean {
  return pathname === prefix || pathname.startsWith(`${prefix}/`);
}

const TOOL_PATHS = ["pitch-decks", "chatbot", "forms", "automation"].map((p) => `/erp/sales/${p}`);

/** Which ERP guide section fits the /erp page currently open. */
export function erpGuideSectionForPath(pathname: string): string {
  if (under(pathname, "/erp/finance/reimbursements") || under(pathname, "/erp/me/reimbursements")) return "erp-reimburse";
  if (under(pathname, "/erp/finance")) return "erp-keuangan";
  if (under(pathname, "/erp/hr/payroll") || under(pathname, "/erp/me/payslips")) return "erp-payroll";
  if (under(pathname, "/erp/hr/recruitment")) return "erp-rekrutmen";
  if (under(pathname, "/erp/hr")) return "erp-sdm";
  if (under(pathname, "/erp/performance")) return "erp-kinerja";
  if (TOOL_PATHS.some((p) => under(pathname, p))) return "erp-penjualan-tools";
  if (under(pathname, "/erp/sales")) return "erp-penjualan";
  if (under(pathname, "/erp/me")) return "erp-mandiri";
  return "erp-mulai";
}
