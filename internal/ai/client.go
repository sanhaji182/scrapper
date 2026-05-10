package ai

import "context"

type LLMClient interface {
	NormalizeProducts(ctx context.Context, itemsJSON []byte) ([]NormalizedProduct, error)
	SummarizeGroups(ctx context.Context, groupsJSON []byte, userPrompt string) (*AISummaryResult, error)
}
