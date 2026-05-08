package tokopedia

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

type legacyTokopediaResponse []struct {
	Data struct {
		SearchProductV5 legacySearchProductV5 `json:"searchProductV5"`
	} `json:"data"`
}

type currentTokopediaResponse struct {
	Data struct {
		SearchProductV5 currentSearchProductV5 `json:"searchProductV5"`
	} `json:"data"`
}

type legacySearchProductV5 struct {
	Data struct {
		Products    []legacyTokopediaProduct `json:"products"`
		HasNextPage bool                     `json:"hasNextPage"`
	} `json:"data"`
}

type currentSearchProductV5 struct {
	Header struct {
		TotalData int `json:"totalData"`
	} `json:"header"`
	Data struct {
		Products []currentTokopediaProduct `json:"products"`
	} `json:"data"`
}

type legacyTokopediaProduct struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	URL                  string          `json:"url"`
	ImageURL             string          `json:"imageUrl"`
	Price                string          `json:"price"`
	SlashedPrice         string          `json:"slashedPrice"`
	DiscountedPercentage json.RawMessage `json:"discountedPercentage"`
	RatingAverage        json.RawMessage `json:"ratingAverage"`
	CountReview          json.RawMessage `json:"countReview"`
	CountSold            json.RawMessage `json:"countSold"`
	Shop                 struct {
		Name          string `json:"name"`
		City          string `json:"city"`
		OfficialStore bool   `json:"officialStore"`
	} `json:"shop"`
}

type currentTokopediaProduct struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	MediaURL struct {
		Image    string `json:"image"`
		Image300 string `json:"image300"`
	} `json:"mediaURL"`
	Shop struct {
		Name string `json:"name"`
		City string `json:"city"`
		Tier int    `json:"tier"`
	} `json:"shop"`
	Price struct {
		Text               string `json:"text"`
		Number             int64  `json:"number"`
		Original           string `json:"original"`
		DiscountPercentage int    `json:"discountPercentage"`
	} `json:"price"`
	Rating      json.RawMessage `json:"rating"`
	LabelGroups []struct {
		Position string `json:"position"`
		Title    string `json:"title"`
	} `json:"labelGroups"`
}

func parseProducts(body []byte) ([]scraper.Product, bool, error) {
	if len(body) > 0 && body[0] == '[' {
		return parseLegacyProducts(body)
	}
	return parseCurrentProducts(body)
}

func parseLegacyProducts(body []byte) ([]scraper.Product, bool, error) {
	var response legacyTokopediaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, false, fmt.Errorf("parse tokopedia legacy response: %w", err)
	}
	if len(response) == 0 {
		return nil, false, nil
	}

	products := response[0].Data.SearchProductV5.Data.Products
	parsed := make([]scraper.Product, 0, len(products))
	for _, product := range products {
		parsed = append(parsed, scraper.Product{
			ID:              product.ID,
			Name:            product.Name,
			Price:           parsePrice(product.Price),
			OriginalPrice:   parsePrice(product.SlashedPrice),
			DiscountPercent: parseJSONInt(product.DiscountedPercentage),
			Rating:          parseJSONFloat(product.RatingAverage),
			CountReview:     parseJSONInt(product.CountReview),
			Sold:            parseSold(product.CountSold),
			URL:             product.URL,
			ImageURL:        product.ImageURL,
			ShopName:        product.Shop.Name,
			ShopCity:        product.Shop.City,
			IsOfficialStore: product.Shop.OfficialStore,
			Marketplace:     "tokopedia",
		})
	}

	return parsed, response[0].Data.SearchProductV5.Data.HasNextPage, nil
}

func parseCurrentProducts(body []byte) ([]scraper.Product, bool, error) {
	var response currentTokopediaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, false, fmt.Errorf("parse tokopedia current response: %w", err)
	}

	products := response.Data.SearchProductV5.Data.Products
	parsed := make([]scraper.Product, 0, len(products))
	for _, product := range products {
		imageURL := product.MediaURL.Image
		if imageURL == "" {
			imageURL = product.MediaURL.Image300
		}
		parsed = append(parsed, scraper.Product{
			ID:              product.ID,
			Name:            product.Name,
			Price:           parseCurrentPrice(product.Price.Text, product.Price.Number),
			OriginalPrice:   parsePrice(product.Price.Original),
			DiscountPercent: product.Price.DiscountPercentage,
			Rating:          parseJSONFloat(product.Rating),
			CountReview:     parseReviewFromLabels(product.LabelGroups),
			Sold:            parseSoldFromLabels(product.LabelGroups),
			URL:             product.URL,
			ImageURL:        imageURL,
			ShopName:        product.Shop.Name,
			ShopCity:        product.Shop.City,
			IsOfficialStore: product.Shop.Tier == 2,
			Marketplace:     "tokopedia",
		})
	}

	hasNextPage := len(products) == defaultRowsPerPage
	if response.Data.SearchProductV5.Header.TotalData > 0 {
		hasNextPage = len(products) > 0
	}
	return parsed, hasNextPage, nil
}

func parseCurrentPrice(text string, number int64) int64 {
	if number > 0 {
		return number
	}
	return parsePrice(text)
}

func parsePrice(value string) int64 {
	replacer := strings.NewReplacer("Rp", "", ".", "", ",", "", " ", "")
	cleaned := replacer.Replace(value)
	if cleaned == "" {
		return 0
	}
	number, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil {
		return 0
	}
	return number
}

func parseRating(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	rating, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return rating
}

func parseJSONFloat(raw json.RawMessage) float64 {
	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		return parseRating(stringValue)
	}
	var floatValue float64
	if err := json.Unmarshal(raw, &floatValue); err == nil {
		return floatValue
	}
	return 0
}

func parseJSONInt(raw json.RawMessage) int {
	var intValue int
	if err := json.Unmarshal(raw, &intValue); err == nil {
		return intValue
	}
	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		number, err := strconv.Atoi(strings.TrimSpace(stringValue))
		if err == nil {
			return number
		}
	}
	return 0
}

func parseSold(raw json.RawMessage) int {
	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		return parseSoldString(stringValue)
	}
	return parseJSONInt(raw)
}

func parseSoldFromLabels(labels []struct {
	Position string `json:"position"`
	Title    string `json:"title"`
}) int {
	for _, label := range labels {
		if strings.Contains(strings.ToLower(label.Title), "terjual") {
			return parseSoldString(label.Title)
		}
	}
	return 0
}

func parseReviewFromLabels(labels []struct {
	Position string `json:"position"`
	Title    string `json:"title"`
}) int {
	for _, label := range labels {
		lowerTitle := strings.ToLower(label.Title)
		if strings.Contains(lowerTitle, "ulasan") || strings.Contains(lowerTitle, "review") {
			return parseSoldString(lowerTitle)
		}
	}
	return 0
}

func parseSoldString(value string) int {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "terjual")
	value = strings.TrimSuffix(value, "terjual")
	value = strings.ReplaceAll(value, "ulasan", "")
	value = strings.ReplaceAll(value, "review", "")
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	multiplier := 1.0
	if strings.HasSuffix(value, "rb") {
		multiplier = 1000
		value = strings.TrimSuffix(value, "rb")
	}
	value = regexp.MustCompile(`[^0-9,.]`).ReplaceAllString(value, "")
	value = strings.ReplaceAll(value, ",", ".")
	if value == "" {
		return 0
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return int(number * multiplier)
}
