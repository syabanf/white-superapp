# Role matrix (draf)

Draf untuk tindak lanjut MoM 4 November 2026, poin 7. Empat stakeholder yang disebut di MoM belum dipetakan: kolom contoh di bawah perlu divalidasi bersama Mixist.

## Izin yang ada di sistem

Admin memegang semua izin. Pengguna lain mendapat izin yang diberikan Admin di Manajemen User.

| Izin | Yang dibuka |
|---|---|
| `self.service` | Absensi, cuti, reimburse, dan tugas milik sendiri |
| `finance.read` | Melihat semua halaman Keuangan, invoice, dan pembayaran |
| `finance.write` | Draf jurnal, kas dan bank, unggah mutasi, cocokkan mutasi ke jurnal, abaikan mutasi, revisi dan terbitkan invoice, faktur pajak, catat pembayaran, setujui follow-up WhatsApp, anggaran, vendor, buat dan ajukan PR dan PO, dokumen komersial, jadikan dokumen sebagai draf tagihan |
| `finance.approve` | Posting jurnal, tutup periode, bayar reimburse, cocokkan mutasi ke invoice, buat jurnal dari mutasi, kelola jurnal berulang dan aset, setujui atau tolak PR dan PO (pembuatnya tidak bisa memutuskan dokumen sendiri, kecuali Admin) |
| `sales.read`, `sales.write`, `sales.approve` | Lead, pipeline, quotation, dan persetujuan diskon |
| `hr.read`, `hr.write`, `hr.approve`, `hr.payroll` | Data karyawan, persetujuan cuti dan lembur, payroll |
| `recruitment.read`, `recruitment.write` | Kandidat dan lowongan |
| Peran Admin | Manajemen User dan Audit trail |

## Contoh pemetaan yang perlu divalidasi

| Peran | Izin |
|---|---|
| Pemilik | Admin |
| Finance | `finance.read`, `finance.write`, `finance.approve`, `self.service` |
| Sales | `sales.read`, `sales.write`, `self.service` |
| Tim | `self.service` |

## Audit trail

Gateway mencatat setiap POST, PUT, PATCH, dan DELETE: waktu, pengguna, peran, alamat, hasil, IP, dan user agent. Isi permintaan dan query string tidak disimpan. Tidak ada use case yang mengubah atau menghapus catatan.

Belum tercakup:

- Fitur marketing yang langsung memakai Prisma di Next.js tidak lewat gateway.
- Permintaan dari server Next.js tercatat dengan IP server tersebut.
- Membaca data tidak dicatat.
