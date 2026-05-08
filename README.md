# Tokopedia Scraper Service

## Overview

Tokopedia Scraper Service adalah REST API yang memungkinkan submit scraping job ke Tokopedia, mirip cara kerja Apify Actor. Setiap job berjalan async di background worker dan hasilnya bisa diambil kapanpun via API.

Service ini menerima request pencarian produk, membuat run di PostgreSQL, mengirim job ke Redis melalui Asynq, lalu worker melakukan scraping Tokopedia GraphQL API dan menyimpan hasilnya kembali ke database.

## Architecture

```text
Client → POST /v1/scrape/tokopedia/search
            ↓
         Run dibuat di DB (status: QUEUED)
            ↓
         Job di-enqueue ke Redis (Asynq)
            ↓
         Worker mengambil job → scraping Tokopedia GraphQL API
            ↓
         Hasil disimpan ke DB (status: SUCCEEDED)
            ↓
Client → GET /v1/runs/:id → dapat hasil produk
```

## Quick Start

```bash
git clone <repo>
cd tokopedia-scraper
cp .env.example .env
make dev
make migrate
make seed
```

`make dev` menjalankan PostgreSQL, Redis, API, dan worker. Jalankan `make migrate` setelah service database healthy untuk membuat tabel `runs`.

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

### Submit Tokopedia Search Job

`POST /v1/scrape/tokopedia/search`

```bash
curl -X POST http://localhost:8080/v1/scrape/tokopedia/search \
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

`sort_by` menerima `relevancy`, `price_asc`, `price_desc`, atau `latest`.

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
| `PROXY_LIST` | Comma-separated proxy list untuk scraper | Empty |
| `REQUEST_TIMEOUT_SEC` | Timeout per request scraper dalam detik | `15` |
| `WORKER_CONCURRENCY` | Jumlah concurrent worker Asynq | `5` |
| `ALLOWED_ORIGINS` | Comma-separated CORS allowed origins | `*` |

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
