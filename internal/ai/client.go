package ai

import (
	"context"
	"fmt"
)

type LLMClient interface {
	NormalizeProducts(ctx context.Context, itemsJSON []byte) ([]NormalizedProduct, error)
	SummarizeGroups(ctx context.Context, groupsJSON []byte, userPrompt string) (*AISummaryResult, error)
}

type DummyClient struct{}

func NewDummyClient() *DummyClient {
	return &DummyClient{}
}

func (c *DummyClient) NormalizeProducts(ctx context.Context, itemsJSON []byte) ([]NormalizedProduct, error) {
	return nil, fmt.Errorf("LLM client not configured: please implement real provider or replace DummyClient")
}

func (c *DummyClient) SummarizeGroups(ctx context.Context, groupsJSON []byte, userPrompt string) (*AISummaryResult, error) {
	return nil, fmt.Errorf("LLM client not configured: SummarizeGroups not implemented")
}
