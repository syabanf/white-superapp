# WHITE Backend (Go)

Backend microservice untuk WHITE Analytics, ditulis ulang dari server action Next.js + Prisma di `../white-analytics`. Platform punya dua workspace: Digital Marketing dan ERP (keuangan, SDM, kinerja, penjualan, porting dari NüHabit). Service Go berjalan di atas satu PostgreSQL dengan skema per service, NATS JetStream untuk kerja asinkron, gRPC antar service, dan grpc-gateway sebagai REST untuk frontend.

| Service | Port | Tanggung jawab |
|---|---|---|
| `gateway` | 8080 (HTTP) | REST/JSON dari gRPC, OpenAPI, alur OAuth, cookie share, file media, unduhan PDF |
| `identity` | 9001 | Login (JWT EdDSA + refresh token), user, klien, anggota, notifikasi, akses share, flag setup |
| `integrations` | 9002 | Koneksi OAuth Meta/Google (token terenkripsi), kredensial integrasi, verifikasi, checklist App Review |
| `analytics` | 9003 | Social, SEO (GSC/GA4, audit, crawl), Meta Ads, ringkasan, sinkronisasi harian |
| `publishing` | 9004 | Pustaka media, komposer post, alur persetujuan, kalender, penjadwal terbit |
| `seosuite` | 9005 | Riset kata kunci, rank tracker, backlink, kompetitor domain |
| `reports` | 9006 | Laporan PDF dan ringkasan AI, disusun dari service lain |
| `content` | 9011 | Digital Marketing: editorial plan dengan draft AI, alur edit dan approval konten, tugas desain, aset brand, laporan konten, jadwal feed dan story ke publishing |
| `finance` | 9007 | ERP: bagan akun, jurnal, periode fiskal, kas-bank, hutang dan piutang, laporan keuangan, reimbursement, analisa rasio dengan AI |
| `hr` | 9008 | ERP: karyawan, absensi, jadwal, cuti, lembur, kontrak, pengumuman, payroll (PPh21, BPJS), pinjaman, layanan mandiri |
| `recruitment` | 9009 | ERP: lowongan, kandidat dan pipeline, talent pool, psikotes, wawancara AI, jadwal wawancara, offer letter, portal karier publik |
| `sales` | 9010 | ERP: lead, pipeline, quotation, tagihan, kontrak dengan tanda tangan digital, pitch deck, onboarding klien, chatbot |
| `performance` | 9012 | ERP: KPI, review, 360, tugas departemen, logbook, rekap harian dan bulanan, things to improve |
| `scheduler` | - | Tick harian per klien, tick publish 5 menit, tick retensi, tick ERP (15 menit, per jam, harian) |

Rincian desain ada di [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), aturan kode di [docs/CONVENTIONS.md](docs/CONVENTIONS.md), dan inventaris perilaku aplikasi lama di `docs/inventory/`.

## Menjalankan secara lokal

Prasyarat: Go 1.26, PostgreSQL 17, NATS (`brew install nats-server`), `protoc` (hanya bila mengubah `.proto`).

```bash
cp .env.example .env            # isi DATABASE_URL, AUTH_SECRET, ENCRYPTION_KEY
nats-server -js &               # atau: docker compose -f deploy/docker-compose.yml up nats postgres
make run-identity               # tiap service: make run-<nama>; migrasi jalan otomatis saat start
make run-integrations
make run-analytics
make run-publishing
make run-seosuite
make run-reports
make run-content
make run-hr                     # ERP: hr dulu, finance dan performance membaca direktori karyawannya
make run-finance
make run-recruitment
make run-sales
make run-performance
make run-scheduler
make run-gateway                # REST + dokumentasi API (Swagger UI) di http://localhost:8080
```

Atau semuanya sekaligus dengan `./scripts/dev-up.sh` (matikan dengan `./scripts/dev-down.sh`). Kalau port 8080 sudah dipakai aplikasi lain, isi `HTTP_ADDR=:8380` di `.env`.

Setiap binary menerima `serve` (default: migrasi + gRPC + worker), `migrate`, atau `worker`.

Dengan Docker:

```bash
make up
```

Data demo ERP dan konten untuk agency WHITE (aman dijalankan ulang):

```bash
for s in hr finance recruitment sales performance content; do go run ./cmd/$s seed; done
```

Rencana dan keputusan ERP ada di [docs/ERP_PLAN.md](docs/ERP_PLAN.md).

## Migrasi dari database Prisma

Migrasi `00002_import_prisma.sql` di tiap service menyalin baris dari tabel Prisma di skema `public` ke skema service, bila tabel itu ada. Id dipertahankan, jadi referensi antar service tetap valid. Jalankan sekali pada database yang sama dengan Next.js:

```bash
for s in identity integrations analytics publishing seosuite reports; do go run ./cmd/$s migrate; done
```

Nilai `ENCRYPTION_KEY` harus sama dengan milik Next.js agar token koneksi lama bisa didekripsi.

## Pengembangan

```bash
make generate     # protoc (gen/go, gen/openapi) + sqlc
make test         # unit test
WHITE_TEST_DATABASE_URL=postgresql://... make test   # plus integration test
make vet
make build        # bin/<service>
```

Login awal memakai akun yang sudah ada di tabel user (hash bcrypt sama). Token akses berumur 15 menit, refresh token 14 hari dan dirotasi setiap dipakai.
