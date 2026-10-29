# Deploy backend ke produksi

Aplikasi Next.js jalan di Vercel. Backend Go (gateway, 12 service, scheduler, NATS) jalan di satu server Docker, dengan Caddy di depan gateway untuk HTTPS. Postgres tetap memakai database yang sama dengan Next.js.

```
Browser ─► Vercel (Next.js) ──GATEWAY_URL──► Caddy :443 ─► gateway ─► service gRPC ─► Postgres (sama dengan Vercel)
```

## Alur otomatis

Setiap push ke `main` yang mengubah `white-backend/**` menjalankan workflow **Backend deploy** (`.github/workflows/backend-deploy.yml`):

1. `test`: `go vet` dan seluruh test terhadap Postgres sementara.
2. `images`: satu image per service ke `ghcr.io/syabanf/mixist-<service>`, bertag `latest` dan SHA commit.
3. `deploy`: menyalin `docker-compose.prod.yml`, `Caddyfile` dan `deploy.sh` ke server lewat SSH, lalu menjalankan `deploy.sh` dengan tag SHA itu. Job ini hanya jalan bila variabel `DEPLOY_HOST` sudah diisi.

Workflow juga bisa dijalankan manual dari tab Actions (Run workflow).

## Persiapan sekali saja

### 1. Server

Butuh Linux dengan Docker Engine dan plugin Compose, minimal 2 vCPU dan 2 GB RAM. Buka port 80 dan 443.

```bash
sudo adduser --disabled-password deploy
sudo usermod -aG docker deploy
sudo mkdir -p /opt/mixist-backend && sudo chown deploy: /opt/mixist-backend
```

Tambahkan kunci publik SSH untuk GitHub Actions ke `~deploy/.ssh/authorized_keys`.

Bila package GHCR masih private, login sekali sebagai user `deploy` memakai personal access token dengan scope `read:packages`:

```bash
echo <TOKEN> | docker login ghcr.io -u syabanf --password-stdin
```

### 2. Domain

Buat A record, misalnya `api.agency-system.reddie.id`, ke IP server. Kalau domainnya lewat Cloudflare, set ke DNS only (awan abu-abu) dulu supaya Caddy bisa mengambil sertifikat.

### 3. File `.env` di server

```bash
cd /opt/mixist-backend
cp .env.prod.example .env   # file contoh ikut tersalin oleh workflow
nano .env
```

Yang wajib:

| Variabel | Isi |
|---|---|
| `API_DOMAIN` | domain dari langkah 2 |
| `DATABASE_URL` | sama persis dengan `DATABASE_URL` di Vercel |
| `INTERNAL_API_SECRET` | sama persis dengan di Vercel |
| `ENCRYPTION_KEY` | sama persis dengan di Vercel |
| `AUTH_SECRET` | `openssl rand -base64 32` |
| `APP_BASE_URL`, `CORS_ORIGINS` | `https://agency-system.reddie.id` |

Saat start pertama, tiap service membuat skemanya sendiri di database itu dan identity menyalin user dari tabel `public."User"`.

### 4. GitHub

Di Settings, buka Secrets and variables, lalu Actions:

- Variables: `DEPLOY_HOST` (IP atau hostname server). Opsional: `DEPLOY_USER` (bawaan `deploy`), `DEPLOY_PATH` (bawaan `/opt/mixist-backend`).
- Secrets: `DEPLOY_SSH_KEY`, kunci privat pasangan kunci publik di langkah 1.

Job deploy memakai environment `production`. Tambahkan required reviewer di Settings > Environments bila deploy perlu persetujuan.

### 5. Vercel

Di Project Settings > Environment Variables (Production):

- `GATEWAY_URL=https://api.agency-system.reddie.id`
- `INTERNAL_API_SECRET` sama dengan di server.

Redeploy Next.js setelah mengubahnya.

## Perintah di server

```bash
cd /opt/mixist-backend
./deploy.sh                          # tarik image terbaru dan restart
TAG=<sha-commit> ./deploy.sh         # rollback ke build tertentu
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f gateway identity
```

## Cek setelah deploy

```bash
curl https://api.agency-system.reddie.id/healthz   # ok
```

Lalu buka `https://agency-system.reddie.id/erp/finance`. Banner "Layanan ERP belum terhubung" harus hilang.

## Catatan

- File unggahan (struk, slip gaji, dokumen) tersimpan di volume Docker `uploads`. Backup volume ini bersama database.
- `docker-compose.yml` di folder ini untuk lokal: membangun image dari source dan ikut menjalankan Postgres.
