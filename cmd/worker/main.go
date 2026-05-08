package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/config"
	"github.com/sonick/tokopedia-scraper/internal/proxy"
	"github.com/sonick/tokopedia-scraper/internal/queue"
	"github.com/sonick/tokopedia-scraper/internal/run"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
	tokopedia "github.com/sonick/tokopedia-scraper/internal/scraper/tokopedia"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect postgres", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Fatal("postgres ping failed", zap.Error(err))
	}
	logger.Info("connected to postgres")

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
