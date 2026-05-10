# Tokopedia Scraper Service

## Overview

Tokopedia Scraper Service adalah REST API yang memungkinkan submit scraping job marketplace Indonesia, mirip cara kerja Apify Actor. Setiap job berjalan async di background worker dan hasilnya bisa diambil kapanpun via API.

Service ini menerima request pencarian produk, membuat run di PostgreSQL, mengirim job ke Redis melalui Asynq, lalu worker melakukan scraping marketplace API dan menyimpan hasilnya kembali ke database.

## Architecture

```text
Client → POST /v1/scrape/{marketplace}/search
            ↓
         Run dibuat di DB (status: QUEUED)
            ↓
         Job di-enqueue ke Redis (Asynq)
            ↓
         Worker mengambil job → scraping marketplace API
            ↓
         Hasil disimpan ke DB (status: SUCCEEDED)
            ↓
Client → GET /v1/runs/:id → dapat hasil produk
```

## Quick Start End-to-End

### 1. Prasyarat

- Docker dan Docker Compose untuk menjalankan PostgreSQL, Redis, API, dan worker.
- Go 1.22+ untuk build/test backend secara lokal.
- Node.js 20+ untuk menjalankan dashboard Next.js.
- `jq` opsional untuk membaca response JSON di terminal.

### 2. Setup Environment

```bash
git clone <repo>
cd tokopedia-scraper
cp .env.example .env
```

Minimal `.env` sudah cukup untuk mode Docker karena `docker-compose.yml` akan override host PostgreSQL dan Redis menjadi service internal container.

Jika ingin memakai AI summary/normalizer, isi provider dan API key di `.env`:

```env
AI_PROVIDER=openai
AI_API_KEY=sk-...
AI_MODEL=gpt-4.1-mini
```

Untuk Shopee, isi `SHOPEE_COOKIE_HEADER` jika request anonymous diblokir. Untuk marketplace yang ketat, isi `PROXY_LIST` dengan proxy residential/mobile.

### 3. Jalankan Backend Lengkap via Docker

```bash
make dev
```

Command ini menjalankan:

- PostgreSQL di `localhost:5432`
- Redis di `localhost:6379`
- API service di `http://localhost:8080`
- Worker Asynq untuk memproses scraping job

Buka terminal kedua untuk menjalankan migration:

```bash
make migrate
```

Cek health API:

```bash
curl http://localhost:8080/health | jq .
```

### 4. Submit Job dari API

Pilih salah satu marketplace: `tokopedia`, `shopee`, `blibli`, atau `lazada`.

```bash
curl -s -X POST http://localhost:8080/v1/scrape/blibli/search \
  -H "Content-Type: application/json" \
  -d '{
    "keyword": "laptop gaming",
    "max_items": 30,
    "sort_by": "price_asc",
    "min_price": 0,
    "max_price": 0
  }' | jq .
```

Simpan `run_id` dari response, lalu cek status sampai `SUCCEEDED` atau `FAILED`:

```bash
curl http://localhost:8080/v1/runs/{run_id} | jq .
```

Lihat semua run:

```bash
curl "http://localhost:8080/v1/runs?limit=20&offset=0" | jq .
```

Pantau log API dan worker:

```bash
make logs-api
make logs-worker
```

### 5. Jalankan Dashboard Next.js

Pastikan backend Docker masih berjalan, lalu buka terminal baru:

```bash
cd dashboard
npm install
npm run dev
```

Buka dashboard di `http://localhost:3000`. Dashboard akan memanggil API backend di `http://localhost:8080` untuk submit job, melihat riwayat run, menampilkan produk, normalisasi, dan AI summary.

Jika dashboard memakai env khusus, buat `dashboard/.env.local`:

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

### 6. Validasi Build dan Test

Backend:

```bash
go test ./...
go build ./...
```

Dashboard:

```bash
cd dashboard
npm run build
```

### 7. Stop Sistem

```bash
make down
```

Jika ingin menghapus data PostgreSQL juga:

```bash
docker compose down -v
```

## Quick Commands

```bash
cp .env.example .env
make dev
make migrate
make seed
```

`make seed` submit sample Tokopedia scraping job. Untuk Blibli/Shopee, gunakan endpoint `POST /v1/scrape/{marketplace}/search`.


## Marketplace Cookie & Proxy Tips

Cookie tidak wajib disimpan di `.env`. Untuk project open-source, buka halaman `Pengaturan` di dashboard lalu isi cookie runtime untuk marketplace yang membutuhkan. Cookie ini hanya disimpan di memory API process dan ikut dikirim ke job berikutnya lewat queue payload. Jika container API restart, isi ulang dari dashboard.

| Marketplace | Cookie Browser | Proxy | Catatan |
| --- | --- | --- | --- |
| Tokopedia | Tidak perlu | Opsional | Paling stabil untuk search publik. |
| Blibli | Tidak wajib | Opsional | Biasanya jalan tanpa cookie; proxy membantu kalau kena `403`. |
| Shopee | Sering perlu | Disarankan | Bisa isi `Shopee Cookie Header` di dashboard atau `SHOPEE_COOKIE_HEADER` di `.env`. |
| Lazada | Sangat disarankan | Residential/mobile disarankan | Tanpa session/proxy sering diarahkan ke captcha `_____tmd_____/punish`. |

Cara ambil cookie: buka marketplace di browser, cari produk, selesaikan captcha kalau muncul, buka DevTools → Network → klik request search/catalog → copy header `Cookie`, lalu paste ke dashboard `Pengaturan`. Jangan commit cookie ke Git.

## API Reference

### Health Check

```bash
curl http://localhost:8080/health | jq .
```

Response:

```json
{
  "status": "ok"
}
```

### Submit Search Job

`POST /v1/scrape/{marketplace}/search`

Marketplace yang didukung: `tokopedia`, `shopee`, `blibli`, `lazada`.

```bash
curl -X POST http://localhost:8080/v1/scrape/blibli/search \
  -H "Content-Type: application/json" \
  -d '{
    "keyword": "laptop gaming",
    "max_items": 30,
    "sort_by": "price_asc",
    "min_price": 0,
    "max_price": 0
  }' | jq .
```

Response:

```json
{
  "message": "Job submitted successfully",
  "run_id": "0d7f7da5-3b6f-42ac-8102-8de4aa0c2db8",
  "status": "QUEUED"
}
```

`sort_by` menerima `relevancy`, `price_asc`, `price_desc`, atau `latest`. Blibli search memakai paging 40 produk per halaman. Lazada search memakai endpoint katalog AJAX Lazada Indonesia dengan paging per halaman, sort `priceasc`/`pricedesc`, dan price range, mengikuti pola umum actor Apify Lazada sebagai referensi.

### List Runs

`GET /v1/runs?limit=20&offset=0`

```bash
curl "http://localhost:8080/v1/runs?limit=20&offset=0" | jq .
```

Response:

```json
{
  "limit": 20,
  "offset": 0,
  "runs": [
    {
      "id": "0d7f7da5-3b6f-42ac-8102-8de4aa0c2db8",
      "status": "SUCCEEDED",
      "marketplace": "tokopedia",
      "input": {
        "keyword": "laptop gaming",
        "max_items": 30,
        "sort_by": "price_asc",
        "min_price": 0,
        "max_price": 0
      },
      "item_count": 30,
      "created_at": "2026-05-08T10:00:00Z",
      "started_at": "2026-05-08T10:00:02Z",
      "finished_at": "2026-05-08T10:00:15Z"
    }
  ],
  "total": 1
}
```

### Get Run Detail

`GET /v1/runs/:id`

```bash
curl http://localhost:8080/v1/runs/0d7f7da5-3b6f-42ac-8102-8de4aa0c2db8 | jq .
```

Response saat berhasil:

```json
{
  "id": "0d7f7da5-3b6f-42ac-8102-8de4aa0c2db8",
  "status": "SUCCEEDED",
  "marketplace": "tokopedia",
  "input": {
    "keyword": "laptop gaming",
    "max_items": 30,
    "sort_by": "price_asc",
    "min_price": 0,
    "max_price": 0
  },
  "result": [
    {
      "id": "123",
      "name": "Laptop Gaming",
      "price": 14999000,
      "original_price": 15999000,
      "discount_percent": 6,
      "rating": 4.8,
      "count_review": 1234,
      "sold": 1200,
      "url": "https://www.tokopedia.com/example/product",
      "image_url": "https://images.tokopedia.net/example.jpg",
      "shop_name": "Official Store",
      "shop_city": "Jakarta",
      "is_official_store": true,
      "marketplace": "tokopedia"
    }
  ],
  "item_count": 30,
  "created_at": "2026-05-08T10:00:00Z",
  "started_at": "2026-05-08T10:00:02Z",
  "finished_at": "2026-05-08T10:00:15Z"
}
```

Jika status belum `SUCCEEDED`, field `result` disembunyikan.

### Delete Run

`DELETE /v1/runs/:id`

```bash
curl -i -X DELETE http://localhost:8080/v1/runs/0d7f7da5-3b6f-42ac-8102-8de4aa0c2db8
```

Response:

```http
HTTP/1.1 204 No Content
```

## Configuration

| Variable | Description | Default |
| --- | --- | --- |
| `PORT` | Port HTTP API server | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `REDIS_ADDR` | Redis address untuk Asynq | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | Empty |
| `PROXY_LIST` | Comma-separated proxy list untuk scraper, disarankan residential/mobile untuk marketplace yang ketat | Empty |
| `SHOPEE_COOKIE_HEADER` | Cookie header Shopee dari browser jika API Shopee balas 403 | Empty |
| `REQUEST_TIMEOUT_SEC` | Timeout per request scraper dalam detik | `15` |
| `WORKER_CONCURRENCY` | Jumlah concurrent worker Asynq | `5` |
| `ALLOWED_ORIGINS` | Comma-separated CORS allowed origins | `*` |


## Shopee Scraping Notes

Shopee lebih ketat daripada Tokopedia. Pendekatan actor Shopee populer di Apify juga mengandalkan cookie agar request terlihat seperti sesi browser. Service ini mendukung dua mode:

1. Tanpa cookie: worker mencoba anonymous session otomatis.
2. Dengan cookie: isi `SHOPEE_COOKIE_HEADER` di `.env`, lalu rebuild worker.

Contoh:

```env
SHOPEE_COOKIE_HEADER=SPC_F=...; REC_T_ID=...; csrftoken=...
```

Jangan commit cookie asli ke Git. Jika masih `403`, cookie mungkin expired, tidak cocok dengan IP/proxy, atau Shopee meminta browser fingerprint/proxy residential.

## Development Commands

```bash
make dev          # start semua service dengan Docker Compose
make down         # stop semua service
make build        # build binary API dan worker ke bin/
make test         # run semua test
make migrate      # apply migration runs ke PostgreSQL container
make seed         # submit sample Tokopedia scraping job
make logs-api     # follow log API
make logs-worker  # follow log worker
make lint         # run golangci-lint
```

## Verification Flow

```bash
cp .env.example .env
make dev
make migrate
make seed
```

Ambil `run_id` dari response `make seed`, tunggu beberapa detik, lalu cek hasil:

```bash
curl http://localhost:8080/v1/runs/{run_id_dari_seed} | jq .
```

Status akan berubah dari `QUEUED` → `RUNNING` → `SUCCEEDED`, dan `result` berisi array produk jika scraping berhasil.
