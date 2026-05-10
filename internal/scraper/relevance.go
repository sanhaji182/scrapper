package scraper

import (
	"regexp"
	"sort"
	"strings"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func FilterRelevantProducts(keyword string, products []Product) []Product {
	queryTokens := tokenize(keyword)
	if len(queryTokens) == 0 || len(products) == 0 {
		return products
	}

	type scoredProduct struct {
		product Product
		score   int
		index   int
	}

	scored := make([]scoredProduct, 0, len(products))
	for index, product := range products {
		score := relevanceScore(queryTokens, product)
		if score > 0 {
			scored = append(scored, scoredProduct{product: product, score: score, index: index})
		}
	}
	if len(scored) == 0 {
		return products
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].index < scored[j].index
		}
		return scored[i].score > scored[j].score
	})

	filtered := make([]Product, 0, len(scored))
	for _, item := range scored {
		filtered = append(filtered, item.product)
	}
	return filtered
}

func relevanceScore(queryTokens []string, product Product) int {
	nameTokens := tokenize(product.Name)
	if len(nameTokens) == 0 {
		return 0
	}
	nameSet := make(map[string]bool, len(nameTokens))
	for _, token := range nameTokens {
		nameSet[token] = true
	}

	matches := 0
	for _, token := range queryTokens {
		if nameSet[token] {
			matches++
		}
	}
	if len(queryTokens) > 1 && !nameSet[queryTokens[0]] {
		return 0
	}
	if matches == 0 {
		return 0
	}

	score := matches * 30
	if matches == len(queryTokens) {
		score += 70
	}
	if len(queryTokens) > 0 && nameSet[queryTokens[0]] {
		score += 20
	}
	if strings.Contains(strings.Join(nameTokens, " "), strings.Join(queryTokens, " ")) {
		score += 150
	}
	if product.IsOfficialStore {
		score += 5
	}
	return score
}

func tokenize(value string) []string {
	matches := tokenPattern.FindAllString(strings.ToLower(value), -1)
	tokens := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) <= 1 {
			continue
		}
		tokens = append(tokens, match)
	}
	return tokens
}
