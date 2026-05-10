package blibli

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

type searchResponse struct {
	Data struct {
		Products []productItem `json:"products"`
	} `json:"data"`
	Products []productItem `json:"products"`
}

type productItem struct {
	ID              string          `json:"id"`
	SKU             string          `json:"sku"`
	Name            string          `json:"name"`
	ProductName     string          `json:"productName"`
	URL             string          `json:"url"`
	ProductURL      string          `json:"productUrl"`
	Price           flexiblePrice   `json:"price"`
	PriceDetail     priceDetail     `json:"-"`
	FinalPrice      flexiblePrice   `json:"finalPrice"`
	StrikePrice     flexiblePrice   `json:"strikePrice"`
	OriginalPrice   flexiblePrice   `json:"originalPrice"`
	Review          reviewDetail    `json:"review"`
	Discount        flexibleInt     `json:"discount"`
	DiscountPercent flexibleInt     `json:"discountPercentage"`
	Rating          flexibleFloat   `json:"rating"`
	ReviewCount     flexibleInt     `json:"reviewCount"`
	Sold            flexibleInt     `json:"sold"`
	SoldText        string          `json:"soldText"`
	Images          []string        `json:"images"`
	ImageURL        string          `json:"imageUrl"`
	Store           storeItem       `json:"store"`
	Merchant        storeItem       `json:"merchant"`
	Official        bool            `json:"official"`
	OfficialStore   bool            `json:"officialStore"`
	Raw             json.RawMessage `json:"-"`
}

func (p *productItem) UnmarshalJSON(data []byte) error {
	type alias productItem
	var raw struct {
		*alias
		Price json.RawMessage `json:"price"`
	}
	raw.alias = (*alias)(p)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.Price) == 0 || string(raw.Price) == "null" {
		return nil
	}
	var detail priceDetail
	if err := json.Unmarshal(raw.Price, &detail); err == nil && (detail.MinPrice > 0 || detail.PriceDisplay != "" || detail.Discount > 0) {
		p.PriceDetail = detail
		return nil
	}
	var price flexiblePrice
	if err := json.Unmarshal(raw.Price, &price); err == nil {
		p.Price = price
	}
	return nil
}

type priceDetail struct {
	PriceDisplay               string        `json:"priceDisplay"`
	MinPrice                   flexiblePrice `json:"minPrice"`
	Discount                   flexibleInt   `json:"discount"`
	StrikeThroughPriceDisplay  string        `json:"strikeThroughPriceDisplay"`
	StrikeThroughPrice         flexiblePrice `json:"strikeThroughPrice"`
}

type reviewDetail struct {
	Rating         flexibleFloat `json:"rating"`
	Count          flexibleInt   `json:"count"`
	AbsoluteRating flexibleFloat `json:"absoluteRating"`
}

type storeItem struct {
	Name     string `json:"name"`
	City     string `json:"city"`
	Location string `json:"location"`
	Official bool   `json:"official"`
}

type flexiblePrice int64

type flexibleInt int

type flexibleFloat float64

func parseProducts(body []byte, keyword string) ([]scraper.Product, error) {
	var response searchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("blibli.parseProducts unmarshal: %w", err)
	}
	items := response.Data.Products
	if len(items) == 0 {
		items = response.Products
	}

	products := make([]scraper.Product, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		product := item.toProduct(keyword)
		if product.ID == "" || product.Name == "" || seen[product.ID] {
			continue
		}
		seen[product.ID] = true
		products = append(products, product)
	}
	return products, nil
}

func (p productItem) toProduct(keyword string) scraper.Product {
	name := firstNonEmpty(p.Name, p.ProductName)
	id := firstNonEmpty(p.ID, p.SKU)
	if id == "" {
		id = stableID(keyword + ":" + name)
	}
	price := int64(p.FinalPrice)
	if price == 0 {
		price = int64(p.PriceDetail.MinPrice)
	}
	if price == 0 {
		price = int64(p.Price)
	}
	originalPrice := int64(p.OriginalPrice)
	if originalPrice == 0 {
		originalPrice = int64(p.PriceDetail.StrikeThroughPrice)
	}
	if originalPrice == 0 {
		originalPrice = parsePrice(p.PriceDetail.StrikeThroughPriceDisplay)
	}
	if originalPrice == 0 {
		originalPrice = int64(p.StrikePrice)
	}
	rating := float64(p.Review.AbsoluteRating)
	if rating == 0 {
		rating = float64(p.Rating)
	}
	if rating == 0 {
		rating = float64(p.Review.Rating)
	}
	shop := p.Store
	if shop.Name == "" {
		shop = p.Merchant
	}
	return scraper.Product{
		ID:              "blibli-" + id,
		Name:            strings.TrimSpace(name),
		Price:           price,
		OriginalPrice:   originalPrice,
		DiscountPercent: firstPositiveInt(int(p.DiscountPercent), int(p.Discount), int(p.PriceDetail.Discount), discountPercent(originalPrice, price)),
		Rating:          rating,
		CountReview:     firstPositiveInt(int(p.ReviewCount), int(p.Review.Count)),
		Sold:            firstPositiveInt(int(p.Sold), parseSoldText(p.SoldText)),
		URL:             productURL(firstNonEmpty(p.ProductURL, p.URL)),
		ImageURL:        imageURL(p),
		ShopName:        strings.TrimSpace(shop.Name),
		ShopCity:        strings.TrimSpace(firstNonEmpty(shop.City, shop.Location)),
		IsOfficialStore: p.Official || p.OfficialStore || shop.Official,
		Marketplace:     MarketplaceName,
	}
}

func (p *flexiblePrice) UnmarshalJSON(data []byte) error {
	value, err := parseNumber(data)
	if err != nil {
		return err
	}
	*p = flexiblePrice(value)
	return nil
}

func (p *flexibleInt) UnmarshalJSON(data []byte) error {
	value, err := parseNumber(data)
	if err != nil {
		return err
	}
	*p = flexibleInt(value)
	return nil
}

func (p *flexibleFloat) UnmarshalJSON(data []byte) error {
	var number float64
	if err := json.Unmarshal(data, &number); err == nil {
		*p = flexibleFloat(number)
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return nil
	}
	text = strings.ReplaceAll(strings.TrimSpace(text), ",", ".")
	if text == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil
	}
	*p = flexibleFloat(parsed)
	return nil
}

func parseNumber(data []byte) (int64, error) {
	var number int64
	if err := json.Unmarshal(data, &number); err == nil {
		return number, nil
	}
	var decimal float64
	if err := json.Unmarshal(data, &decimal); err == nil {
		return int64(decimal), nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return 0, nil
	}
	return parsePrice(text), nil
}

func parsePrice(value string) int64 {
	cleaned := regexp.MustCompile(`[^0-9]`).ReplaceAllString(value, "")
	if cleaned == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseSoldText(value string) int {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return 0
	}
	multiplier := 1
	if strings.Contains(value, "rb") || strings.Contains(value, "ribu") {
		multiplier = 1000
	} else if strings.Contains(value, "jt") || strings.Contains(value, "juta") {
		multiplier = 1000000
	}
	match := regexp.MustCompile(`[0-9]+([,.][0-9]+)?`).FindString(value)
	if match == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(strings.ReplaceAll(match, ",", "."), 64)
	if err != nil {
		return 0
	}
	return int(parsed * float64(multiplier))
}

func productURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "https://www.blibli.com"
	}
	if strings.HasPrefix(value, "http") {
		return value
	}
	if strings.HasPrefix(value, "/") {
		return "https://www.blibli.com" + value
	}
	return "https://www.blibli.com/" + value
}

func imageURL(p productItem) string {
	if p.ImageURL != "" {
		return p.ImageURL
	}
	if len(p.Images) > 0 {
		return p.Images[0]
	}
	return ""
}

func discountPercent(originalPrice, price int64) int {
	if originalPrice <= 0 || price <= 0 || originalPrice <= price {
		return 0
	}
	return int((originalPrice - price) * 100 / originalPrice)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func stableID(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}
