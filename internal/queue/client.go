package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
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
		asynq.TaskID(runID),
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
