package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

const maxItemsPerChunk = 40

type SaveNormalizedFunc func(ctx context.Context, runID string, normalizedJSON []byte) error

func NormalizeRun(ctx context.Context, runID string, status string, resultJSON []byte, client LLMClient, save SaveNormalizedFunc) ([]ProductGroup, error) {
	if status != "SUCCEEDED" {
		return nil, fmt.Errorf("NormalizeRun: run status must be SUCCEEDED, got %s", status)
	}
	if len(resultJSON) == 0 {
		return nil, fmt.Errorf("NormalizeRun: run has no result_json")
	}

	var products []scraper.Product
	if err := json.Unmarshal(resultJSON, &products); err != nil {
		return nil, fmt.Errorf("NormalizeRun: unmarshal products: %w", err)
	}

	var allNormalized []NormalizedProduct
	for i := 0; i < len(products); i += maxItemsPerChunk {
		end := i + maxItemsPerChunk
		if end > len(products) {
			end = len(products)
		}
		chunk := products[i:end]

		payload := struct {
			Items []scraper.Product `json:"items"`
		}{Items: chunk}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("NormalizeRun: marshal chunk: %w", err)
		}

		normalizedChunk, err := client.NormalizeProducts(ctx, payloadBytes)
		if err != nil {
			return nil, fmt.Errorf("NormalizeRun: LLM normalize chunk %d: %w", i/maxItemsPerChunk, err)
		}
		allNormalized = append(allNormalized, normalizedChunk...)
	}

	groups, err := groupNormalized(products, allNormalized)
	if err != nil {
		return nil, fmt.Errorf("NormalizeRun: grouping: %w", err)
	}

	groupsBytes, err := json.Marshal(groups)
	if err != nil {
		return nil, fmt.Errorf("NormalizeRun: marshal groups: %w", err)
	}
	if err := save(ctx, runID, groupsBytes); err != nil {
		return nil, fmt.Errorf("NormalizeRun: save normalized: %w", err)
	}

	return groups, nil
}

func groupNormalized(products []scraper.Product, normalized []NormalizedProduct) ([]ProductGroup, error) {
	prodByID := make(map[string]scraper.Product, len(products))
	for _, p := range products {
		prodByID[p.ID] = p
	}

	groupsMap := make(map[string]*ProductGroup)

	for _, n := range normalized {
		if n.CanonicalKey == "" {
			continue
		}
		p, ok := prodByID[n.SourceProductID]
		if !ok {
			continue
		}

		g, ok := groupsMap[n.CanonicalKey]
		if !ok {
			g = &ProductGroup{
				GroupID:        n.CanonicalKey,
				CanonicalName:  buildCanonicalName(n.Brand, n.Model, n.Variant),
				Brand:          n.Brand,
				Model:          n.Model,
				Variant:        n.Variant,
				CategoryPath:   n.CategoryPath,
				ImportantSpecs: append([]string{}, n.ImportantSpecs...),
			}
			groupsMap[n.CanonicalKey] = g
		}

		item := FromProduct(p)
		g.Items = append(g.Items, item)
	}

	var groups []ProductGroup
	for _, g := range groupsMap {
		var sum int64
		var bestPrice int64
		var bestID string
		for i, item := range g.Items {
			price := item.Price
			sum += price
			if i == 0 || price < bestPrice {
				bestPrice = price
				bestID = item.ProductID
			}
		}
		if len(g.Items) > 0 {
			g.MinPrice = g.Items[0].Price
			g.MaxPrice = g.Items[0].Price
			for _, item := range g.Items[1:] {
				if item.Price < g.MinPrice {
					g.MinPrice = item.Price
				}
				if item.Price > g.MaxPrice {
					g.MaxPrice = item.Price
				}
			}
			g.AvgPrice = float64(sum) / float64(len(g.Items))
		}
		g.BestPriceID = bestID
		groups = append(groups, *g)
	}

	return groups, nil
}

func buildCanonicalName(brand, model, variant string) string {
	name := ""
	if brand != "" {
		name += brand
	}
	if model != "" {
		if name != "" {
			name += " "
		}
		name += model
	}
	if variant != "" {
		if name != "" {
			name += " "
		}
		name += variant
	}
	return name
}
