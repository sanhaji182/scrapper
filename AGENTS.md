# TOKOPEDIA SCRAPER SERVICE — PROJECT CONTEXT
> File ini dibaca otomatis oleh Codex CLI setiap sesi. Jangan hapus.

## Project Overview
Kamu membangun **Tokopedia Scraper Service** — mini Apify platform untuk scraping produk Tokopedia.
Sistem ini berupa REST API yang menerima job scraping, menjalankannya di background worker,
dan menyimpan hasilnya agar bisa diambil kapanpun.

## Tech Stack
- **Language:** Go 1.22+
- **HTTP:** Echo v4
- **Queue:** Asynq (Redis-backed)
- **Database:** PostgreSQL (pgx/v5)
- **Cache:** Redis (go-redis/v9)
- **Logger:** Uber Zap
- **Config:** godotenv
- **Container:** Docker + docker-compose
- **Module:** `github.com/[username]/tokopedia-scraper`

## Project Structure (target akhir)
```
tokopedia-scraper/
├── cmd/api/main.go
├── cmd/worker/main.go
├── internal/
│   ├── config/config.go
│   ├── scraper/interface.go
│   ├── scraper/tokopedia/scraper.go
│   ├── scraper/tokopedia/parser.go
│   ├── scraper/tokopedia/scraper_test.go
│   ├── run/model.go
│   ├── run/repository.go
│   ├── run/handler.go
│   ├── queue/client.go
│   ├── queue/worker.go
│   ├── proxy/manager.go
│   └── middleware/middleware.go
├── db/migrations/001_create_runs.sql
├── docker-compose.yml
├── Dockerfile.api
├── Dockerfile.worker
├── Makefile
├── .env.example
└── README.md
```

## Global Rules (wajib diikuti di semua fase)
1. Semua fungsi I/O terima `context.Context` sebagai parameter pertama
2. Error di-wrap dengan `fmt.Errorf("...: %w", err)`
3. Gunakan zap structured logging, bukan `fmt.Println`
4. Tidak ada global state — semua dependency inject via constructor
5. Semua constant (task names, status values) sebagai typed constants
6. Setelah setiap fase: `go build ./...` harus sukses

## Build Progress Tracker
- [x] Phase 1 — Bootstrap & Config
- [x] Phase 2 — Core Structs & Interfaces
- [x] Phase 3 — Database Migration & Repository
- [ ] Phase 4 — Tokopedia Scraper
- [x] Phase 5 — Job Queue
- [x] Phase 6 — REST API Handlers
- [x] Phase 7 — Proxy Manager
- [x] Phase 8 — Entry Points
- [x] Phase 9 — Docker & Tooling
- [x] Phase 10 — AI Normalizer
- [x] Phase 11 — AI Summary

## Updated Folder Structure (Phase 10–12)

internal/ai/
  client.go        // LLMClient interface + OpenAI impl
  normalizer.go    // NormalizeRun, GroupNormalizedProducts
  summary.go       // SummarizeRun, AISummaryResult
  types.go         // NormalizedProduct, ProductGroup, GroupedItem

apps/dashboard/    // Repo frontend terpisah (Next.js 14)

## New Endpoints (Phase 10–11)
POST /v1/runs/:id/normalize
GET  /v1/runs/:id/normalized
POST /v1/runs/:id/ai-summary
GET  /v1/runs/:id/ai-summary

## Global Rules Tambahan
- Semua call ke LLM wajib lewat LLMClient interface, tidak boleh direct HTTP call
- Output LLM selalu di-parse dan divalidasi sebelum disimpan ke DB
- Prompt template disimpan sebagai const string di file masing-masing (normalizer.go/summary.go)
