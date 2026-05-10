package lazada

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

func isCaptchaBody(body []byte) bool {
	lower := bytes.ToLower(body)
	return bytes.Contains(lower, []byte("_____tmd_____")) || bytes.Contains(lower, []byte("x5secdata")) || bytes.Contains(lower, []byte("action\":\"captcha"))
}

func extractJSONBody(body []byte) []byte {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '<' {
		return body
	}
	patterns := [][]byte{
		[]byte("window.pageData ="),
		[]byte("window.pageData="),
		[]byte("__moduleData__ ="),
		[]byte("__moduleData__="),
	}
	for _, pattern := range patterns {
		index := bytes.Index(body, pattern)
		if index < 0 {
			continue
		}
		start := bytes.IndexByte(body[index+len(pattern):], '{')
		if start < 0 {
			continue
		}
		start += index + len(pattern)
		if extracted := balancedJSONObject(body[start:]); len(extracted) > 0 {
			return extracted
		}
	}
	return body
}

func balancedJSONObject(body []byte) []byte {
	depth := 0
	inString := false
	escaped := false
	for index, char := range body {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}
		switch char {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return body[:index+1]
			}
		}
	}
	return nil
}

type searchResponse struct {
	Mods struct {
		ListItems []item `json:"listItems"`
	} `json:"mods"`
	Data struct {
		Products []item `json:"products"`
	} `json:"data"`
	Products []item `json:"products"`
	Items    []item `json:"items"`
}

type item struct {
	ID                string        `json:"itemId"`
	SKU               string        `json:"sku"`
	Name              string        `json:"name"`
	Title             string        `json:"title"`
	ItemURL           string        `json:"itemUrl"`
	ProductURL        string        `json:"productUrl"`
	Image             string        `json:"image"`
	ImageURL          string        `json:"imageUrl"`
	Price             flexiblePrice `json:"price"`
	PriceShow         string        `json:"priceShow"`
	OriginalPrice     flexiblePrice `json:"originalPrice"`
	OriginalPriceShow string        `json:"originalPriceShow"`
	Discount          string        `json:"discount"`
	RatingScore       flexibleFloat `json:"ratingScore"`
	RatingAverage     flexibleFloat `json:"ratingAverage"`
	Review            flexibleInt   `json:"review"`
	ReviewCount       flexibleInt   `json:"reviewCount"`
	Sold              flexibleInt   `json:"sold"`
	SoldText          string        `json:"soldText"`
	Location          string        `json:"location"`
	SellerName        string        `json:"sellerName"`
	ShopName          string        `json:"shopName"`
	IsOfficialStore   bool          `json:"isOfficialStore"`
	IsLazMall         bool          `json:"isLazMall"`
}

type flexiblePrice int64

type flexibleInt int

type flexibleFloat float64

func parseProducts(body []byte, keyword string) ([]scraper.Product, error) {
	if isCaptchaBody(body) {
		return nil, fmt.Errorf("lazada anti-bot captcha page; residential proxy or browser session is required")
	}
	jsonBody := extractJSONBody(body)
	var response searchResponse
	if err := json.Unmarshal(jsonBody, &response); err != nil {
		return nil, fmt.Errorf("lazada.parseProducts unmarshal: %w", err)
	}
	items := response.Mods.ListItems
	if len(items) == 0 {
		items = response.Data.Products
	}
	if len(items) == 0 {
		items = response.Products
	}
	if len(items) == 0 {
		items = response.Items
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

func (i item) toProduct(keyword string) scraper.Product {
	name := firstNonEmpty(i.Name, i.Title)
	id := firstNonEmpty(i.ID, i.SKU)
	if id == "" {
		id = stableID(keyword + ":" + name)
	}
	price := int64(i.Price)
	if price == 0 {
		price = parsePrice(i.PriceShow)
	}
	originalPrice := int64(i.OriginalPrice)
	if originalPrice == 0 {
		originalPrice = parsePrice(i.OriginalPriceShow)
	}
	rating := float64(i.RatingScore)
	if rating == 0 {
		rating = float64(i.RatingAverage)
	}
	return scraper.Product{
		ID:              "lazada-" + id,
		Name:            strings.TrimSpace(name),
		Price:           price,
		OriginalPrice:   originalPrice,
		DiscountPercent: firstPositiveInt(parseDiscount(i.Discount), discountPercent(originalPrice, price)),
		Rating:          rating,
		CountReview:     firstPositiveInt(int(i.Review), int(i.ReviewCount)),
		Sold:            firstPositiveInt(int(i.Sold), parseSoldText(i.SoldText)),
		URL:             productURL(firstNonEmpty(i.ItemURL, i.ProductURL)),
		ImageURL:        firstNonEmpty(i.ImageURL, i.Image),
		ShopName:        strings.TrimSpace(firstNonEmpty(i.ShopName, i.SellerName)),
		ShopCity:        strings.TrimSpace(i.Location),
		IsOfficialStore: i.IsOfficialStore || i.IsLazMall,
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

func parseDiscount(value string) int {
	match := regexp.MustCompile(`[0-9]+`).FindString(value)
	if match == "" {
		return 0
	}
	parsed, err := strconv.Atoi(match)
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
	if strings.Contains(value, "rb") || strings.Contains(value, "ribu") || strings.Contains(value, "k") {
		multiplier = 1000
	} else if strings.Contains(value, "jt") || strings.Contains(value, "juta") || strings.Contains(value, "m") {
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
		return "https://www.lazada.co.id"
	}
	if strings.HasPrefix(value, "//") {
		return "https:" + value
	}
	if strings.HasPrefix(value, "http") {
		return value
	}
	if strings.HasPrefix(value, "/") {
		return "https://www.lazada.co.id" + value
	}
	return "https://www.lazada.co.id/" + value
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
