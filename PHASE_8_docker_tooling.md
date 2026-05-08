# PHASE 8 — Docker, Makefile & Tooling

## Prerequisite
Phase 1-7 selesai. `go build ./...` sukses.

## Objective
Containerisasi service dan buat tooling untuk development workflow.

## Scope
- `docker-compose.yml`
- `Dockerfile.api`
- `Dockerfile.worker`
- `Makefile`
- `README.md`

---

## Step 8.1 — File: `Dockerfile.api`

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/api /bin/api
EXPOSE 8080
CMD ["/bin/api"]
```

---

## Step 8.2 — File: `Dockerfile.worker`

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker ./cmd/worker

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/worker /bin/worker
CMD ["/bin/worker"]
```

---

## Step 8.3 — File: `docker-compose.yml`

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: scraper
      POSTGRES_PASSWORD: scraper
      POSTGRES_DB: tokopedia_scraper
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U scraper -d tokopedia_scraper"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  api:
    build:
      context: .
      dockerfile: Dockerfile.api
    ports:
      - "8080:8080"
    env_file: .env
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    env_file: .env
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
```

---

## Step 8.4 — File: `Makefile`

```makefile
.PHONY: dev down build test migrate seed logs-api logs-worker lint

dev:
	docker compose up --build

down:
	docker compose down

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test:
	go test ./... -v -cover

migrate:
	docker compose exec postgres psql -U scraper -d tokopedia_scraper -f /dev/stdin < db/migrations/001_create_runs.sql

seed:
	curl -s -X POST http://localhost:8080/v1/scrape/tokopedia/search \
		-H "Content-Type: application/json" \
		-d '{"keyword":"laptop gaming","max_items":30,"sort_by":"price_asc"}' | jq .

logs-api:
	docker compose logs -f api

logs-worker:
	docker compose logs -f worker

lint:
	golangci-lint run ./...
```

---

## Step 8.5 — File: `README.md`

Buat README yang memuat:

### 1. Overview
Tokopedia Scraper Service adalah REST API yang memungkinkan submit scraping job
ke Tokopedia, mirip cara kerja Apify Actor. Setiap job berjalan async di background
worker dan hasilnya bisa diambil kapanpun via API.

### 2. Architecture
```
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

### 3. Quick Start
```bash
git clone <repo>
cd tokopedia-scraper
cp .env.example .env
make dev         # start semua service
make migrate     # buat tabel runs
make seed        # submit test job
```

### 4. API Reference
Dokumentasikan semua 4 endpoint dengan contoh curl dan response JSON.

### 5. Configuration
Tabel semua env variable, description, dan default value.

---

## Verification

```bash
make build     # binary harus berhasil dibuat
make dev       # semua container harus healthy
make migrate   # tabel runs terbuat
make seed      # dapat run_id di response
```

Tunggu beberapa detik, lalu:
```bash
curl http://localhost:8080/v1/runs/{run_id_dari_seed} | jq .
```

Status harus berubah dari QUEUED → RUNNING → SUCCEEDED, dan `result` berisi array produk.

Update checklist AGENTS.md: Phase 8 ✅
