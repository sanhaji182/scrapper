package shopee

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

type searchResponse struct {
	Items []searchItem `json:"items"`
}

type searchItem struct {
	ItemBasic itemBasic `json:"item_basic"`
	AdsItem   itemBasic `json:"ads_keyword_item"`
}

type itemBasic struct {
	ItemID       int64    `json:"itemid"`
	ShopID       int64    `json:"shopid"`
	Name         string   `json:"name"`
	Price        int64    `json:"price"`
	PriceMin     int64    `json:"price_min"`
	PriceBefore  int64    `json:"price_before_discount"`
	Rating       float64  `json:"item_rating.rating_star"`
	RatingInfo   rating   `json:"item_rating"`
	Sold         int      `json:"historical_sold"`
	SoldText     string   `json:"sold"`
	Image        string   `json:"image"`
	ShopLocation string   `json:"shop_location"`
	ShopName     string   `json:"shop_name"`
	IsOfficial   bool     `json:"is_official_shop"`
	IsMall       bool     `json:"shopee_verified"`
	Tier         string   `json:"tier_variations"`
	Categories   []string `json:"categories"`
}

type rating struct {
	RatingStar  float64 `json:"rating_star"`
	RatingCount []int   `json:"rating_count"`
}

func parseProducts(body []byte, keyword string) ([]scraper.Product, error) {
	var response searchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("shopee.parseProducts unmarshal: %w", err)
	}

	products := make([]scraper.Product, 0, len(response.Items))
	seen := make(map[string]bool, len(response.Items))
	for _, item := range response.Items {
		basic := item.ItemBasic
		if basic.Name == "" && item.AdsItem.Name != "" {
			basic = item.AdsItem
		}
		product := basic.toProduct(keyword)
		if product.ID == "" || product.Name == "" || seen[product.ID] {
			continue
		}
		seen[product.ID] = true
		products = append(products, product)
	}
	return products, nil
}

func (i itemBasic) toProduct(keyword string) scraper.Product {
	price := normalizePrice(i.Price)
	if price == 0 {
		price = normalizePrice(i.PriceMin)
	}
	originalPrice := normalizePrice(i.PriceBefore)
	ratingValue := i.Rating
	if ratingValue == 0 {
		ratingValue = i.RatingInfo.RatingStar
	}
	countReview := 0
	if len(i.RatingInfo.RatingCount) > 0 {
		for _, count := range i.RatingInfo.RatingCount {
			countReview += count
		}
	}

	itemID := strconv.FormatInt(i.ItemID, 10)
	shopID := strconv.FormatInt(i.ShopID, 10)
	if i.ItemID == 0 {
		itemID = stableID(keyword + ":" + i.Name)
	}
	id := "shopee-" + shopID + "-" + itemID
	url := "https://shopee.co.id/product/" + shopID + "/" + itemID
	if i.ShopID == 0 {
		url = "https://shopee.co.id/search?keyword=" + strings.ReplaceAll(keyword, " ", "%20")
	}

	return scraper.Product{
		ID:              id,
		Name:            strings.TrimSpace(i.Name),
		Price:           price,
		OriginalPrice:   originalPrice,
		DiscountPercent: discountPercent(originalPrice, price),
		Rating:          ratingValue,
		CountReview:     countReview,
		Sold:            parseSold(i.Sold, i.SoldText),
		URL:             url,
		ImageURL:        imageURL(i.Image),
		ShopName:        strings.TrimSpace(i.ShopName),
		ShopCity:        strings.TrimSpace(i.ShopLocation),
		IsOfficialStore: i.IsOfficial || i.IsMall,
		Marketplace:     MarketplaceName,
	}
}

func normalizePrice(value int64) int64 {
	if value <= 0 {
		return 0
	}
	if value > 100000000 {
		return value / 100000
	}
	return value
}

func discountPercent(originalPrice, price int64) int {
	if originalPrice <= 0 || price <= 0 || originalPrice <= price {
		return 0
	}
	return int((originalPrice - price) * 100 / originalPrice)
}

func imageURL(image string) string {
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}
	if strings.HasPrefix(image, "http") {
		return image
	}
	return "https://down-id.img.susercontent.com/file/" + image
}

func parseSold(value int, text string) int {
	if value > 0 {
		return value
	}
	text = strings.ToLower(strings.TrimSpace(text))
	multiplier := 1
	if strings.Contains(text, "rb") || strings.Contains(text, "k") {
		multiplier = 1000
	}
	text = strings.NewReplacer("terjual", "", "+", "", "rb", "", "k", "", ",", ".").Replace(text)
	parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil {
		return 0
	}
	return int(parsed * float64(multiplier))
}

func stableID(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func parseSEOProducts(body []byte, keyword string) []scraper.Product {
	content := htmlUnescape(string(body))
	blocks := splitProductBlocks(content)
	products := make([]scraper.Product, 0, len(blocks))
	seen := make(map[string]bool, len(blocks))
	for _, block := range blocks {
		name := firstMatch(block, `aria-label="View product: ([^"]+)"`)
		href := firstMatch(block, `href="(/[^"]+-i\.[0-9]+\.[0-9]+[^"]*)"`)
		if name == "" || href == "" {
			continue
		}
		shopID, itemID := parseSEOIDs(href)
		id := "shopee-" + shopID + "-" + itemID
		if seen[id] {
			continue
		}
		seen[id] = true
		price := parseSEOPrice(block)
		product := scraper.Product{
			ID:              id,
			Name:            strings.TrimSpace(name),
			Price:           price,
			DiscountPercent: parseSEODiscount(block),
			Rating:          parseSEOFloat(firstMatch(block, `>([0-9]+(?:\.[0-9]+)?)</div></div></div></div><div class="flex items-center space-x-1 max-w-full"`)),
			Sold:            parseSold(0, firstMatch(block, `([0-9.,]+\s*(?:RB|rb|k|K)?\+?\s*terjual)`)),
			URL:             "https://shopee.co.id" + stripHTMLAttribute(href),
			ImageURL:        stripHTMLAttribute(firstMatch(block, `src="(https://down-id\.img\.susercontent\.com/file/[^"]+)"`)),
			ShopCity:        strings.TrimSpace(firstMatch(block, `aria-label="location-([^"]+)"`)),
			Marketplace:     MarketplaceName,
		}
		products = append(products, product)
	}
	return scraper.FilterRelevantProducts(keyword, products)
}

func splitProductBlocks(content string) []string {
	parts := strings.Split(content, `aria-label="Product card"`)
	if len(parts) <= 1 {
		parts = strings.Split(content, `aria-label="View product:`)
		for index := 1; index < len(parts); index++ {
			parts[index] = `aria-label="View product:` + parts[index]
		}
	}
	if len(parts) <= 1 {
		return nil
	}
	return parts[1:]
}

func firstMatch(value, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(value)
	if len(matches) < 2 {
		return ""
	}
	return htmlUnescape(matches[1])
}

func parseSEOIDs(href string) (string, string) {
	re := regexp.MustCompile(`-i\.([0-9]+)\.([0-9]+)`)
	matches := re.FindStringSubmatch(href)
	if len(matches) < 3 {
		return "0", stableID(href)
	}
	return matches[1], matches[2]
}

func parseSEOPrice(block string) int64 {
	value := firstMatch(block, `<span class="truncate text-base/5 font-medium">([^<]+)</span>`)
	if value == "" {
		value = firstMatch(block, `Rp\s*([0-9.]+)`)
	}
	return int64(parseDigits(value))
}

func parseSEODiscount(block string) int {
	value := firstMatch(block, `aria-label="-([0-9]+)%"`)
	return parseDigits(value)
}

func parseSEOFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseDigits(value string) int {
	value = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func stripHTMLAttribute(value string) string {
	value = strings.ReplaceAll(value, "&amp;", "&")
	value = strings.ReplaceAll(value, "&#x27;", "'")
	return value
}

func htmlUnescape(value string) string {
	return html.UnescapeString(value)
}
