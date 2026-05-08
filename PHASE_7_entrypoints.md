# PHASE 7 — Entry Points (cmd/api & cmd/worker)

## Prerequisite
Phase 1-6 selesai. `go build ./...` sukses.

## Objective
Wire semua dependencies di entry point API server dan Worker.

## Scope
- `cmd/api/main.go`
- `cmd/worker/main.go`

---

## Step 7.1 — File: `cmd/api/main.go`

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/labstack/echo/v4"
    "go.uber.org/zap"

    "github.com/[username]/tokopedia-scraper/internal/config"
    appMiddleware "github.com/[username]/tokopedia-scraper/internal/middleware"
    "github.com/[username]/tokopedia-scraper/internal/queue"
    "github.com/[username]/tokopedia-scraper/internal/run"
)

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    cfg, err := config.Load()
    if err != nil {
        logger.Fatal("failed to load config", zap.Error(err))
    }

    // Connect PostgreSQL
    pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
    if err != nil {
        logger.Fatal("failed to connect postgres", zap.Error(err))
    }
    defer pool.Close()

    if err := pool.Ping(context.Background()); err != nil {
        logger.Fatal("postgres ping failed", zap.Error(err))
    }
    logger.Info("connected to postgres")

    // Init dependencies
    runRepo := run.NewRepository(pool)
    queueClient := queue.NewClient(cfg.RedisAddr, cfg.RedisPassword)
    defer queueClient.Close()

    // Setup Echo
    e := echo.New()
    e.HideBanner = true
    appMiddleware.Register(e, logger, cfg.AllowedOrigins)

    // Register routes
    runHandler := run.NewHandler(runRepo, queueClient, logger)
    runHandler.RegisterRoutes(e)

    // Health check endpoint
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(200, map[string]string{"status": "ok"})
    })

    // Graceful shutdown
    go func() {
        if err := e.Start(":" + cfg.Port); err != nil {
            logger.Info("server stopped", zap.Error(err))
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := e.Shutdown(ctx); err != nil {
        logger.Fatal("server forced to shutdown", zap.Error(err))
    }
    logger.Info("server exited")
}
```

---

## Step 7.2 — File: `cmd/worker/main.go`

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"

    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"

    "github.com/[username]/tokopedia-scraper/internal/config"
    "github.com/[username]/tokopedia-scraper/internal/proxy"
    "github.com/[username]/tokopedia-scraper/internal/queue"
    "github.com/[username]/tokopedia-scraper/internal/run"
    "github.com/[username]/tokopedia-scraper/internal/scraper"
    tokopedia "github.com/[username]/tokopedia-scraper/internal/scraper/tokopedia"
)

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    cfg, err := config.Load()
    if err != nil {
        logger.Fatal("failed to load config", zap.Error(err))
    }

    // Connect PostgreSQL
    pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
    if err != nil {
        logger.Fatal("failed to connect postgres", zap.Error(err))
    }
    defer pool.Close()

    if err := pool.Ping(context.Background()); err != nil {
        logger.Fatal("postgres ping failed", zap.Error(err))
    }
    logger.Info("connected to postgres")

    // Init dependencies
    proxyMgr := proxy.NewManager(cfg.ProxyList)
    tokopediaScraper := tokopedia.New(cfg.RequestTimeoutSec, proxyMgr, logger)

    scrapers := map[string]scraper.MarketplaceScraper{
        "tokopedia": tokopediaScraper,
    }

    runRepo := run.NewRepository(pool)
    worker := queue.NewWorker(
        cfg.RedisAddr, cfg.RedisPassword,
        cfg.WorkerConcurrency,
        logger, runRepo, scrapers,
    )

    logger.Info("starting worker", zap.Int("concurrency", cfg.WorkerConcurrency))

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-quit
        logger.Info("shutting down worker...")
        worker.Shutdown()
    }()

    if err := worker.Start(); err != nil {
        logger.Fatal("worker error", zap.Error(err))
    }
    logger.Info("worker exited")
}
```

> ⚠️ Pastikan `tokopedia.New(timeoutSec int, proxyMgr *proxy.Manager, logger *zap.Logger)` 
> konstruktor sudah ada di `scraper/tokopedia/scraper.go`. Tambahkan jika belum.

---

## Verification

```bash
go build ./cmd/api/...
go build ./cmd/worker/...
```
Kedua binary harus build sukses. Update checklist AGENTS.md: Phase 7 ✅
