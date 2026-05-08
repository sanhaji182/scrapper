package run

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

type Repository interface {
	Create(ctx context.Context, marketplace string, input scraper.SearchOptions) (*Run, error)
	GetByID(ctx context.Context, id string) (*Run, error)
	List(ctx context.Context, limit, offset int) ([]Run, int, error)
	UpdateStatus(ctx context.Context, id string, status Status, errMsg string) error
	SaveResult(ctx context.Context, id string, products []scraper.Product) error
	Delete(ctx context.Context, id string) error
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) Create(ctx context.Context, marketplace string, input scraper.SearchOptions) (*Run, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("repository.Create marshal input: %w", err)
	}

	run := &Run{}
	err = r.pool.QueryRow(ctx, `
        INSERT INTO runs (marketplace, input_json)
        VALUES ($1, $2)
        RETURNING id, status, marketplace, input_json, item_count, created_at
    `, marketplace, inputBytes).Scan(
		&run.ID, &run.Status, &run.Marketplace,
		&run.InputJSON, &run.ItemCount, &run.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.Create: %w", err)
	}
	return run, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*Run, error) {
	run := &Run{}
	err := r.pool.QueryRow(ctx, `
        SELECT id, status, marketplace, input_json, result_json,
               error_message, item_count, created_at, started_at, finished_at
        FROM runs WHERE id = $1
    `, id).Scan(
		&run.ID, &run.Status, &run.Marketplace, &run.InputJSON, &run.ResultJSON,
		&run.ErrorMessage, &run.ItemCount, &run.CreatedAt, &run.StartedAt, &run.FinishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}
	return run, nil
}

func (r *postgresRepository) List(ctx context.Context, limit, offset int) ([]Run, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM runs").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository.List count: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
        SELECT id, status, marketplace, input_json, error_message,
               item_count, created_at, started_at, finished_at
        FROM runs ORDER BY created_at DESC LIMIT $1 OFFSET $2
    `, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.List query: %w", err)
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(
			&run.ID, &run.Status, &run.Marketplace, &run.InputJSON,
			&run.ErrorMessage, &run.ItemCount, &run.CreatedAt, &run.StartedAt, &run.FinishedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("repository.List scan: %w", err)
		}
		runs = append(runs, run)
	}
	return runs, total, nil
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, id string, status Status, errMsg string) error {
	var startedAt *time.Time
	if status == StatusRunning {
		now := time.Now()
		startedAt = &now
	}
	_, err := r.pool.Exec(ctx, `
        UPDATE runs SET status = $1, error_message = $2, started_at = COALESCE(started_at, $3)
        WHERE id = $4
    `, status, errMsg, startedAt, id)
	if err != nil {
		return fmt.Errorf("repository.UpdateStatus: %w", err)
	}
	return nil
}

func (r *postgresRepository) SaveResult(ctx context.Context, id string, products []scraper.Product) error {
	resultBytes, err := json.Marshal(products)
	if err != nil {
		return fmt.Errorf("repository.SaveResult marshal: %w", err)
	}
	now := time.Now()
	_, err = r.pool.Exec(ctx, `
        UPDATE runs
        SET status = $1, result_json = $2, item_count = $3, finished_at = $4
        WHERE id = $5
    `, StatusSucceeded, resultBytes, len(products), now, id)
	if err != nil {
		return fmt.Errorf("repository.SaveResult: %w", err)
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM runs WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("repository.Delete: %w", err)
	}
	return nil
}
