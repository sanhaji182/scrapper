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

	"github.com/sonick/tokopedia-scraper/internal/config"
	appMiddleware "github.com/sonick/tokopedia-scraper/internal/middleware"
	"github.com/sonick/tokopedia-scraper/internal/queue"
	"github.com/sonick/tokopedia-scraper/internal/run"
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

	runRepo := run.NewRepository(pool)
	queueClient := queue.NewClient(cfg.RedisAddr, cfg.RedisPassword)
	defer queueClient.Close()

	e := echo.New()
	e.HideBanner = true
	appMiddleware.Register(e, logger, cfg.AllowedOrigins)

	runHandler := run.NewHandler(runRepo, queueClient, logger)
	runHandler.RegisterRoutes(e)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(httpStatusOK, map[string]string{"status": "ok"})
	})

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

const httpStatusOK = 200
