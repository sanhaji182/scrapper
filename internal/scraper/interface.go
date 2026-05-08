package scraper

import (
	"context"
	"fmt"
)

const (
	SortByRelevancy = "relevancy"
	SortByPriceAsc  = "price_asc"
	SortByPriceDesc = "price_desc"
	SortByLatest    = "latest"
)

type Product struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Price           int64   `json:"price"`
	OriginalPrice   int64   `json:"original_price"`
	DiscountPercent int     `json:"discount_percent"`
	Rating          float64 `json:"rating"`
	CountReview     int     `json:"count_review"`
	Sold            int     `json:"sold"`
	URL             string  `json:"url"`
	ImageURL        string  `json:"image_url"`
	ShopName        string  `json:"shop_name"`
	ShopCity        string  `json:"shop_city"`
	IsOfficialStore bool    `json:"is_official_store"`
	Marketplace     string  `json:"marketplace"`
}

type SearchOptions struct {
	Keyword  string `json:"keyword"`
	MaxItems int    `json:"max_items"`
	SortBy   string `json:"sort_by"`
	MinPrice int64  `json:"min_price"`
	MaxPrice int64  `json:"max_price"`
}

func (o *SearchOptions) Validate() error {
	if o.Keyword == "" {
		return fmt.Errorf("keyword is required")
	}
	if len(o.Keyword) > 100 {
		return fmt.Errorf("keyword max 100 characters")
	}
	if o.MaxItems <= 0 {
		o.MaxItems = 50
	}
	if o.MaxItems > 200 {
		o.MaxItems = 200
	}
	if o.SortBy == "" {
		o.SortBy = SortByRelevancy
	}
	validSorts := map[string]bool{
		SortByRelevancy: true,
		SortByPriceAsc:  true,
		SortByPriceDesc: true,
		SortByLatest:    true,
	}
	if !validSorts[o.SortBy] {
		return fmt.Errorf("sort_by must be one of: relevancy, price_asc, price_desc, latest")
	}
	if o.MinPrice > 0 && o.MaxPrice > 0 && o.MinPrice > o.MaxPrice {
		return fmt.Errorf("min_price cannot be greater than max_price")
	}
	return nil
}

type MarketplaceScraper interface {
	Search(ctx context.Context, opts SearchOptions) ([]Product, error)
	Name() string
}
