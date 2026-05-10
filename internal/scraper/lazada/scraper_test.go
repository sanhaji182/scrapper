package lazada

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
		if r.URL.Query().Get("ajax") != "true" {
			t.Fatalf("ajax = %q", r.URL.Query().Get("ajax"))
		}
		if r.URL.Query().Get("q") != "iphone" {
			t.Fatalf("q = %q", r.URL.Query().Get("q"))
		}
		if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			t.Fatalf("x-requested-with = %q", r.Header.Get("X-Requested-With"))
		}
		_, _ = w.Write([]byte(sampleResponse))
	}))
	defer server.Close()

	lazadaScraper := newTestScraper(server.URL)
	products, err := lazadaScraper.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 10})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d", len(products))
	}
	product := products[0]
	if product.ID != "lazada-123" || product.Price != 14999000 || product.OriginalPrice != 15999000 {
		t.Fatalf("product = %+v", product)
	}
	if product.Sold != 1200 || product.Marketplace != MarketplaceName {
		t.Fatalf("product = %+v", product)
	}
}

func TestSearchPaginates(t *testing.T) {
	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages = append(pages, r.URL.Query().Get("page"))
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		_, _ = w.Write([]byte(responseWithProducts(page, page == 1, defaultItemsPerPage)))
	}))
	defer server.Close()

	lazadaScraper := newTestScraper(server.URL)
	products, err := lazadaScraper.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 45})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(products) != 41 {
		t.Fatalf("len(products) = %d", len(products))
	}
	if len(pages) != 2 || pages[0] != "1" || pages[1] != "2" {
		t.Fatalf("pages = %v", pages)
	}
}

func TestBuildURLSortAndPriceFilters(t *testing.T) {
	lazadaScraper := newTestScraper("https://example.com/catalog/")
	requestURL, err := lazadaScraper.buildURL(scraper.SearchOptions{Keyword: "iphone", SortBy: scraper.SortByPriceAsc, MinPrice: 1000, MaxPrice: 2000}, 2)
	if err != nil {
		t.Fatalf("buildURL error = %v", err)
	}
	parsed, _ := url.Parse(requestURL)
	query := parsed.Query()
	if query.Get("sort") != "priceasc" || query.Get("price") != "1000-2000" || query.Get("page") != "2" {
		t.Fatalf("query = %s", parsed.RawQuery)
	}
}

func TestParseProductsCaptchaPage(t *testing.T) {
	_, err := parseProducts([]byte(`<script>window._config_ = {"action":"captcha"}; location.href="/_____tmd_____/punish?x5secdata=abc"</script>`), "iphone")
	if err == nil {
		t.Fatal("expected captcha error")
	}
}

func TestParseProductsHTMLPageData(t *testing.T) {
	body := []byte(`<html><script>window.pageData = {"mods":{"listItems":[{"itemId":"HTML1","name":"iPhone 15","priceShow":"Rp15.000.000"}]}};</script></html>`)
	products, err := parseProducts(body, "iphone")
	if err != nil {
		t.Fatalf("parseProducts error = %v", err)
	}
	if len(products) != 1 || products[0].ID != "lazada-HTML1" || products[0].Price != 15000000 {
		t.Fatalf("products = %+v", products)
	}
}

func TestParseProductsFallbackShapes(t *testing.T) {
	products, err := parseProducts([]byte(`{"products":[{"sku":"A1","title":"iPhone Case","priceShow":"Rp99.000"}]}`), "iphone")
	if err != nil {
		t.Fatalf("parseProducts error = %v", err)
	}
	if len(products) != 1 || products[0].Price != 99000 {
		t.Fatalf("products = %+v", products)
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
			"itemId":  "LAZ-" + strconv.Itoa(page) + "-" + strconv.Itoa(i),
			"name":    "iPhone " + strconv.Itoa(page) + " " + strconv.Itoa(i),
			"price":   14999000,
			"itemUrl": "//www.lazada.co.id/products/iphone.html",
			"image":   "https://lzd-img-global.slatic.net/image.jpg",
			"review":  1,
		})
	}
	payload, _ := json.Marshal(map[string]interface{}{"mods": map[string]interface{}{"listItems": products}})
	return string(payload)
}

const sampleResponse = `{
  "mods": {
    "listItems": [
      {
        "itemId": "123",
        "name": "Apple iPhone 15 128GB Garansi Resmi",
        "itemUrl": "//www.lazada.co.id/products/apple-iphone-15.html",
        "image": "https://lzd-img-global.slatic.net/image.jpg",
        "priceShow": "Rp14.999.000",
        "originalPriceShow": "Rp15.999.000",
        "discount": "6% Off",
        "ratingScore": "4.8",
        "review": "1234",
        "soldText": "1,2RB terjual",
        "location": "DKI Jakarta",
        "sellerName": "Apple Authorized Store",
        "isLazMall": true
      }
    ]
  }
}`
