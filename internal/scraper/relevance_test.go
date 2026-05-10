package scraper

import "testing"

func TestFilterRelevantProductsPrioritizesFullContext(t *testing.T) {
	products := []Product{
		{ID: "wrong", Name: "iPhone 15 Pro Max Case"},
		{ID: "partial", Name: "OnePlus 9 Pro not iPhone 15"},
		{ID: "exact", Name: "OnePlus 15 5G 12GB 256GB"},
		{ID: "exact-2", Name: "Oneplus 15 Global ROM"},
	}

	filtered := FilterRelevantProducts("oneplus 15", products)

	if len(filtered) != 3 {
		t.Fatalf("len(filtered) = %d, want 3", len(filtered))
	}
	if filtered[0].ID != "exact" || filtered[1].ID != "exact-2" {
		t.Fatalf("full context products not prioritized: %+v", filtered)
	}
	for _, product := range filtered {
		if product.ID == "wrong" {
			t.Fatalf("irrelevant product should be filtered out: %+v", filtered)
		}
	}
}

func TestFilterRelevantProductsFallsBackWhenNoMatch(t *testing.T) {
	products := []Product{{ID: "a", Name: "Random Item"}}
	filtered := FilterRelevantProducts("iphone", products)
	if len(filtered) != 1 || filtered[0].ID != "a" {
		t.Fatalf("fallback changed products: %+v", filtered)
	}
}
