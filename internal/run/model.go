package run

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusQueued    Status = "QUEUED"
	StatusRunning   Status = "RUNNING"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
	StatusTimedOut  Status = "TIMED_OUT"
)

type Run struct {
	ID           string          `json:"id" db:"id"`
	Status       Status          `json:"status" db:"status"`
	Marketplace  string          `json:"marketplace" db:"marketplace"`
	InputJSON    json.RawMessage `json:"input" db:"input_json"`
	ResultJSON   json.RawMessage `json:"result,omitempty" db:"result_json"`
	ErrorMessage string          `json:"error_message,omitempty" db:"error_message"`
	ItemCount    int             `json:"item_count" db:"item_count"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty" db:"started_at"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty" db:"finished_at"`
}

func (r *Run) IsTerminal() bool {
	return r.Status == StatusSucceeded ||
		r.Status == StatusFailed ||
		r.Status == StatusTimedOut
}
