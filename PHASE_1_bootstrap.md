# PHASE 1 — Bootstrap & Config

## Objective
Inisialisasi project Go dari nol: module, folder structure, dan config loader.

## Scope
File yang dibuat di fase ini:
- `go.mod` + `go.sum`
- `.env.example`
- `internal/config/config.go`

Tidak lebih dari itu. Jangan implement logic lain dulu.

---

## Step 1.1 — Init Go Module

```bash
go mod init github.com/[username]/tokopedia-scraper
```

Lalu buat seluruh folder structure (folder kosong, cukup taruh `.gitkeep`):
```
cmd/api/
cmd/worker/
internal/config/
internal/scraper/tokopedia/
internal/run/
internal/queue/
internal/proxy/
internal/middleware/
db/migrations/
```

---

## Step 1.2 — Install Dependencies

```bash
go get github.com/labstack/echo/v4
go get github.com/labstack/echo/v4/middleware
go get github.com/hibiken/asynq
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/redis/go-redis/v9
go get go.uber.org/zap
go get github.com/joho/godotenv
go get github.com/google/uuid
go get github.com/stretchr/testify
```

---

## Step 1.3 — File: `.env.example`

```env
PORT=8080
DATABASE_URL=postgres://scraper:scraper@localhost:5432/tokopedia_scraper?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
PROXY_LIST=
REQUEST_TIMEOUT_SEC=15
WORKER_CONCURRENCY=5
ALLOWED_ORIGINS=*
```

---

## Step 1.4 — File: `internal/config/config.go`

```go
package config

import (
    "fmt"
    "os"
    "strconv"
    "strings"

    "github.com/joho/godotenv"
)

type Config struct {
    Port              string
    DatabaseURL       string
    RedisAddr         string
    RedisPassword     string
    ProxyList         []string
    RequestTimeoutSec int
    WorkerConcurrency int
    AllowedOrigins    []string
}

func Load() (*Config, error) {
    _ = godotenv.Load() // ignore error jika .env tidak ada (pakai OS env)

    cfg := &Config{
        Port:              getEnv("PORT", "8080"),
        DatabaseURL:       getEnv("DATABASE_URL", ""),
        RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
        RedisPassword:     getEnv("REDIS_PASSWORD", ""),
        RequestTimeoutSec: getEnvInt("REQUEST_TIMEOUT_SEC", 15),
        WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 5),
    }

    if cfg.DatabaseURL == "" {
        return nil, fmt.Errorf("config: DATABASE_URL is required")
    }

    rawProxy := getEnv("PROXY_LIST", "")
    if rawProxy != "" {
        cfg.ProxyList = strings.Split(rawProxy, ",")
    }

    rawOrigins := getEnv("ALLOWED_ORIGINS", "*")
    cfg.AllowedOrigins = strings.Split(rawOrigins, ",")

    return cfg, nil
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

func getEnvInt(key string, fallback int) int {
    if v := os.Getenv(key); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    return fallback
}
```

---

## Verification

Setelah selesai, jalankan:
```bash
go build ./...
```
Harus sukses tanpa error. Update checklist di AGENTS.md: Phase 1 ✅
