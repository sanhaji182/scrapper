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
	TaskMarketplaceSearch = "marketplace:search"
	TaskTokopediaSearch   = "tokopedia:search"
	QueueScraper          = "scraper"
	JobTimeout            = 5 * time.Minute
	MaxRetry              = 3
)

type JobPayload struct {
	RunID        string                `json:"run_id"`
	Marketplace  string                `json:"marketplace"`
	Options      scraper.SearchOptions `json:"options"`
	CookieHeader string                `json:"cookie_header,omitempty"`
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
	return c.EnqueueMarketplaceScrapeJob(ctx, runID, "tokopedia", opts)
}

func (c *Client) EnqueueMarketplaceScrapeJob(ctx context.Context, runID, marketplace string, opts scraper.SearchOptions) error {
	return c.EnqueueMarketplaceScrapeJobWithCookie(ctx, runID, marketplace, opts, "")
}

func (c *Client) EnqueueMarketplaceScrapeJobWithCookie(ctx context.Context, runID, marketplace string, opts scraper.SearchOptions, cookieHeader string) error {
	payload, err := json.Marshal(JobPayload{RunID: runID, Marketplace: marketplace, Options: opts, CookieHeader: cookieHeader})
	if err != nil {
		return fmt.Errorf("queue.EnqueueMarketplaceScrapeJobWithCookie marshal: %w", err)
	}

	task := asynq.NewTask(TaskMarketplaceSearch, payload,
		asynq.Queue(QueueScraper),
		asynq.TaskID(runID),
		asynq.MaxRetry(MaxRetry),
		asynq.Timeout(JobTimeout),
	)

	_, err = c.asynq.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("queue.EnqueueMarketplaceScrapeJobWithCookie enqueue: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	return c.asynq.Close()
}
