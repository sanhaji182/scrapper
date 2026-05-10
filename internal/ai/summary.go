package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

type SaveAISummaryFunc func(ctx context.Context, runID string, summaryJSON []byte) error

func SummarizeRun(ctx context.Context, runID string, normalizedJSON []byte, client LLMClient, userPrompt string, save SaveAISummaryFunc) (*AISummaryResult, error) {
	if len(normalizedJSON) == 0 {
		return nil, fmt.Errorf("SummarizeRun: normalized_json is empty; run NormalizeRun first")
	}

	var groups []ProductGroup
	if err := json.Unmarshal(normalizedJSON, &groups); err != nil {
		return nil, fmt.Errorf("SummarizeRun: unmarshal normalized_json: %w", err)
	}

	payload := struct {
		Groups []ProductGroup `json:"groups"`
	}{Groups: groups}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("SummarizeRun: marshal groups: %w", err)
	}

	result, err := client.SummarizeGroups(ctx, payloadBytes, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("SummarizeRun: LLM summarize: %w", err)
	}

	resBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("SummarizeRun: marshal result: %w", err)
	}
	if err := save(ctx, runID, resBytes); err != nil {
		return nil, fmt.Errorf("SummarizeRun: save ai summary: %w", err)
	}

	return result, nil
}
