package ai

import "github.com/sonick/tokopedia-scraper/internal/scraper"

type NormalizedProduct struct {
	SourceProductID string   `json:"source_product_id"`
	Marketplace     string   `json:"marketplace"`
	URL             string   `json:"url"`
	Brand           string   `json:"brand"`
	Model           string   `json:"model"`
	Variant         string   `json:"variant"`
	CategoryPath    string   `json:"category_path"`
	ImportantSpecs  []string `json:"important_specs"`
	CanonicalKey    string   `json:"canonical_key"`
}

type GroupedItem struct {
	ProductID       string  `json:"product_id"`
	Marketplace     string  `json:"marketplace"`
	Name            string  `json:"name"`
	Price           int64   `json:"price"`
	OriginalPrice   int64   `json:"original_price"`
	DiscountPercent int     `json:"discount_percent"`
	Rating          float64 `json:"rating"`
	CountReview     int     `json:"count_review"`
	ShopName        string  `json:"shop_name"`
	IsOfficialStore bool    `json:"is_official_store"`
	URL             string  `json:"url"`
}

type ProductGroup struct {
	GroupID        string        `json:"group_id"`
	CanonicalName  string        `json:"canonical_name"`
	Brand          string        `json:"brand"`
	Model          string        `json:"model"`
	Variant        string        `json:"variant"`
	CategoryPath   string        `json:"category_path"`
	ImportantSpecs []string      `json:"important_specs"`
	Items          []GroupedItem `json:"items"`
	MinPrice       int64         `json:"min_price"`
	MaxPrice       int64         `json:"max_price"`
	AvgPrice       float64       `json:"avg_price"`
	BestPriceID    string        `json:"best_price_id"`
}

func FromProduct(p scraper.Product) GroupedItem {
	return GroupedItem{
		ProductID:       p.ID,
		Marketplace:     p.Marketplace,
		Name:            p.Name,
		Price:           p.Price,
		OriginalPrice:   p.OriginalPrice,
		DiscountPercent: p.DiscountPercent,
		Rating:          p.Rating,
		CountReview:     p.CountReview,
		ShopName:        p.ShopName,
		IsOfficialStore: p.IsOfficialStore,
		URL:             p.URL,
	}
}
