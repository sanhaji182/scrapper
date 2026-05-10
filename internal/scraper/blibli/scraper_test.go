package blibli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

func TestSearchBuildsRequestAndParsesProducts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("searchTerm") != "iphone" {
			t.Fatalf("searchTerm = %q", r.URL.Query().Get("searchTerm"))
		}
		if r.URL.Query().Get("itemPerPage") != "40" {
			t.Fatalf("itemPerPage = %q", r.URL.Query().Get("itemPerPage"))
		}
		if r.Header.Get("User-Agent") == "" {
			t.Fatal("missing user-agent")
		}
		_, _ = w.Write([]byte(sampleResponse))
	}))
	defer server.Close()

	blibliScraper := newTestScraper(server.URL)
	products, err := blibliScraper.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 10})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d", len(products))
	}
	product := products[0]
	if product.ID != "blibli-BLI-123" {
		t.Fatalf("id = %q", product.ID)
	}
	if product.Price != 14999000 || product.OriginalPrice != 15999000 {
		t.Fatalf("prices = %d/%d", product.Price, product.OriginalPrice)
	}
	if product.Sold != 1200 {
		t.Fatalf("sold = %d", product.Sold)
	}
	if product.Marketplace != MarketplaceName {
		t.Fatalf("marketplace = %q", product.Marketplace)
	}
}

func TestSearchPaginatesWithStartOffset(t *testing.T) {
	var starts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		starts = append(starts, r.URL.Query().Get("start"))
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		_, _ = w.Write([]byte(responseWithProducts(page, page == 1, defaultItemsPerPage)))
	}))
	defer server.Close()

	blibliScraper := newTestScraper(server.URL)
	products, err := blibliScraper.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 45})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(products) != 41 {
		t.Fatalf("len(products) = %d", len(products))
	}
	if len(starts) != 2 || starts[0] != "0" || starts[1] != "40" {
		t.Fatalf("starts = %v", starts)
	}
}

func TestBuildURLSortAndPriceFilters(t *testing.T) {
	blibliScraper := newTestScraper("https://example.com/search")
	requestURL, err := blibliScraper.buildURL(scraper.SearchOptions{Keyword: "iphone", SortBy: scraper.SortByPriceAsc, MinPrice: 1000, MaxPrice: 2000}, 1)
	if err != nil {
		t.Fatalf("buildURL error = %v", err)
	}
	parsed, _ := url.Parse(requestURL)
	query := parsed.Query()
	if query.Get("sort") != "3" || query.Get("minPrice") != "1000" || query.Get("maxPrice") != "2000" || query.Get("start") != "40" {
		t.Fatalf("query = %s", parsed.RawQuery)
	}
}

func TestParseProductsRootProductsFallback(t *testing.T) {
	products, err := parseProducts([]byte(`{"products":[{"sku":"A1","productName":"iPhone Case","finalPrice":"Rp99.000"}]}`), "iphone")
	if err != nil {
		t.Fatalf("parseProducts error = %v", err)
	}
	if len(products) != 1 || products[0].Price != 99000 {
		t.Fatalf("products = %+v", products)
	}
}

func TestParseProductsApifyRawShape(t *testing.T) {
	products, err := parseProducts([]byte(`{"data":{"products":[{"sku":"APF-70017-00242","name":"Mac Mini M4","url":"/p/mac-mini-m4/ps--APF-70017-00242","price":{"priceDisplay":"Rp15.999.000","minPrice":15999000,"discount":5,"strikeThroughPriceDisplay":"Rp16.999.000"},"review":{"rating":4,"count":127,"absoluteRating":4.8},"official":true}]}}`), "mac mini")
	if err != nil {
		t.Fatalf("parseProducts error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d", len(products))
	}
	if products[0].Price != 15999000 || products[0].OriginalPrice != 16999000 || products[0].Rating != 4.8 || products[0].CountReview != 127 {
		t.Fatalf("product = %+v", products[0])
	}
}

func newTestScraper(endpoint string) *Scraper {
	return NewWithClient(http.DefaultClient, endpoint, zap.NewNop(), WithDelay(func(ctx context.Context, delay time.Duration) error { return nil }), WithPageDelay(func() time.Duration { return 0 }))
}

func responseWithProducts(page int, full bool, count int) string {
	products := make([]map[string]interface{}, 0, count)
	limit := 1
	if full {
		limit = count
	}
	for i := 0; i < limit; i++ {
		products = append(products, map[string]interface{}{
			"sku":         "BLI-" + strconv.Itoa(page) + "-" + strconv.Itoa(i),
			"name":        "iPhone " + strconv.Itoa(page) + " " + strconv.Itoa(i),
			"finalPrice":  14999000,
			"productUrl":  "/p/iphone",
			"imageUrl":    "https://www.static-src.com/image.jpg",
			"soldText":    "1 terjual",
			"reviewCount": 1,
		})
	}
	payload, _ := json.Marshal(map[string]interface{}{"data": map[string]interface{}{"products": products}})
	return string(payload)
}

const sampleResponse = `{
  "data": {
    "products": [
      {
        "sku": "BLI-123",
        "name": "Apple iPhone 15 128GB Garansi Resmi",
        "productUrl": "/p/apple-iphone-15",
        "imageUrl": "https://www.static-src.com/image.jpg",
        "finalPrice": "Rp14.999.000",
        "originalPrice": "Rp15.999.000",
        "discountPercentage": "6",
        "rating": "4.8",
        "reviewCount": "1234",
        "soldText": "1,2rb terjual",
        "store": {"name": "Blibli Official Store", "city": "Jakarta", "official": true}
      }
    ]
  }
}`
