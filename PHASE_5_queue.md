# PHASE 5 — Job Queue (Asynq)

## Prerequisite
Phase 1-4 selesai. Tests pass.

## Objective
Implementasi job queue menggunakan Asynq: client untuk enqueue dan worker untuk proses job.

## Scope
- `internal/queue/client.go`
- `internal/queue/worker.go`

---

## Step 5.1 — File: `internal/queue/client.go`

```go
package queue

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/hibiken/asynq"
    "github.com/[username]/tokopedia-scraper/internal/scraper"
)

const (
    TaskTokopediaSearch = "tokopedia:search"
    QueueScraper        = "scraper"
    JobTimeout          = 5 * time.Minute
    MaxRetry            = 3
)

type JobPayload struct {
    RunID   string                `json:"run_id"`
    Options scraper.SearchOptions `json:"options"`
}

type Client struct {
    asynq *asynq.Client
}

func NewClient(redisAddr, redisPassword string) *Client {
    return &Client{
        asynq: asynq.NewClient(asynq.RedisClientOpt{
            Addr:     redisAddr,
            Password: redisPassword,
        }),
    }
}

func (c *Client) EnqueueScrapeJob(ctx context.Context, runID string, opts scraper.SearchOptions) error {
    payload, err := json.Marshal(JobPayload{RunID: runID, Options: opts})
    if err != nil {
        return fmt.Errorf("queue.EnqueueScrapeJob marshal: %w", err)
    }

    task := asynq.NewTask(TaskTokopediaSearch, payload,
        asynq.Queue(QueueScraper),
        asynq.TaskID(runID),           // idempotency: satu runID = satu job
        asynq.MaxRetry(MaxRetry),
        asynq.Timeout(JobTimeout),
    )

    _, err = c.asynq.EnqueueContext(ctx, task)
    if err != nil {
        return fmt.Errorf("queue.EnqueueScrapeJob enqueue: %w", err)
    }
    return nil
}

func (c *Client) Close() error {
    return c.asynq.Close()
}
```

---

## Step 5.2 — File: `internal/queue/worker.go`

```go
package queue

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/hibiken/asynq"
    "go.uber.org/zap"
    "github.com/[username]/tokopedia-scraper/internal/run"
    "github.com/[username]/tokopedia-scraper/internal/scraper"
)

type Worker struct {
    server  *asynq.Server
    mux     *asynq.ServeMux
    logger  *zap.Logger
    repo    run.Repository
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

    // Update status RUNNING
    if err := w.repo.UpdateStatus(ctx, payload.RunID, run.StatusRunning, ""); err != nil {
        return fmt.Errorf("worker: update status running: %w", err)
    }

    // Eksekusi scraping
    s, ok := w.scrapers["tokopedia"]
    if !ok {
        errMsg := "tokopedia scraper not registered"
        _ = w.repo.UpdateStatus(ctx, payload.RunID, run.StatusFailed, errMsg)
        return fmt.Errorf(errMsg)
    }

    products, err := s.Search(ctx, payload.Options)
    if err != nil {
        log.Error("scrape failed", zap.Error(err))
        _ = w.repo.UpdateStatus(ctx, payload.RunID, run.StatusFailed, err.Error())
        return fmt.Errorf("worker: scrape: %w", err)
    }

    // Simpan hasil
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
```

---

## Verification

```bash
go build ./...
```
Update checklist AGENTS.md: Phase 5 ✅
