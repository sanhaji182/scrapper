package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/run"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

type Worker struct {
	server   *asynq.Server
	mux      *asynq.ServeMux
	logger   *zap.Logger
	repo     run.Repository
	scrapers map[string]scraper.MarketplaceScraper
}

func NewWorker(
	redisAddr, redisPassword string,
	concurrency int,
	logger *zap.Logger,
	repo run.Repository,
	scrapers map[string]scraper.MarketplaceScraper,
) *Worker {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				QueueScraper: 10,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.Error("task failed", zap.String("type", task.Type()), zap.Error(err))
			}),
		},
	)

	w := &Worker{
		server:   srv,
		mux:      asynq.NewServeMux(),
		logger:   logger,
		repo:     repo,
		scrapers: scrapers,
	}

	w.mux.HandleFunc(TaskTokopediaSearch, w.handleTokopediaSearch)
	return w
}

func (w *Worker) handleTokopediaSearch(ctx context.Context, t *asynq.Task) error {
	var payload JobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("worker.handleTokopediaSearch unmarshal: %w", err)
	}

	log := w.logger.With(zap.String("run_id", payload.RunID))
	log.Info("starting scrape job", zap.String("keyword", payload.Options.Keyword))

	if err := w.repo.UpdateStatus(ctx, payload.RunID, run.StatusRunning, ""); err != nil {
		return fmt.Errorf("worker: update status running: %w", err)
	}

	s, ok := w.scrapers["tokopedia"]
	if !ok {
		errMsg := "tokopedia scraper not registered"
		_ = w.repo.UpdateStatus(ctx, payload.RunID, run.StatusFailed, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	products, err := s.Search(ctx, payload.Options)
	if err != nil {
		log.Error("scrape failed", zap.Error(err))
		_ = w.repo.UpdateStatus(ctx, payload.RunID, run.StatusFailed, err.Error())
		return fmt.Errorf("worker: scrape: %w", err)
	}

	if err := w.repo.SaveResult(ctx, payload.RunID, products); err != nil {
		_ = w.repo.UpdateStatus(ctx, payload.RunID, run.StatusFailed, err.Error())
		return fmt.Errorf("worker: save result: %w", err)
	}

	log.Info("scrape completed", zap.Int("item_count", len(products)))
	return nil
}

func (w *Worker) Start() error {
	return w.server.Run(w.mux)
}

func (w *Worker) Shutdown() {
	w.server.Shutdown()
}
