package tokopedia

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/proxy"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

func TestSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content type = %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("User-Agent") == "" {
			t.Fatal("user-agent is empty")
		}
		if r.Header.Get("X-Source") != "tokopedia-lite" {
			t.Fatalf("x-source = %s", r.Header.Get("X-Source"))
		}
		if r.Header.Get("X-Device") != "default_v3" {
			t.Fatalf("x-device = %s", r.Header.Get("X-Device"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON(1, false)))
	}))
	defer server.Close()

	s := newTestScraper(server.URL)
	products, err := s.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 10})

	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d, want 1", len(products))
	}
	product := products[0]
	assertEqual(t, "id", "product-1", product.ID)
	assertEqual(t, "name", "iPhone 1", product.Name)
	assertEqual(t, "price", int64(14999000), product.Price)
	assertEqual(t, "original price", int64(15999000), product.OriginalPrice)
	assertEqual(t, "discount", 6, product.DiscountPercent)
	assertFloat(t, "rating", 4.8, product.Rating)
	assertEqual(t, "review", 1234, product.CountReview)
	assertEqual(t, "sold", 1200, product.Sold)
	assertEqual(t, "url", "https://www.tokopedia.com/shop/product-1", product.URL)
	assertEqual(t, "image", "https://images.tokopedia.net/product-1.jpg", product.ImageURL)
	assertEqual(t, "shop", "iBox Official", product.ShopName)
	assertEqual(t, "city", "Jakarta", product.ShopCity)
	assertEqual(t, "official", true, product.IsOfficialStore)
	assertEqual(t, "marketplace", "tokopedia", product.Marketplace)
}

func TestSearch_PaginationLoop(t *testing.T) {
	var pages []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := requestPage(t, r)
		pages = append(pages, page)
		_, _ = w.Write([]byte(responseJSON(page, page == 1)))
	}))
	defer server.Close()

	s := newTestScraper(server.URL)
	products, err := s.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 60})

	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}
	if len(pages) != 2 || pages[0] != 1 || pages[1] != 2 {
		t.Fatalf("pages = %v, want [1 2]", pages)
	}
}

func TestSearch_RateLimit(t *testing.T) {
	var requests int32
	var delayed time.Duration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(responseJSON(1, false)))
	}))
	defer server.Close()

	s := New(15, proxy.NewManager(nil), zap.NewNop(),
		WithEndpoint(server.URL),
		WithRetryDelay(10*time.Second),
		WithPageDelay(func() time.Duration { return 0 }),
		WithDelay(func(ctx context.Context, delay time.Duration) error {
			delayed = delay
			return nil
		}),
	)
	products, err := s.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 1})

	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d, want 1", len(products))
	}
	if atomic.LoadInt32(&requests) != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if delayed != 10*time.Second {
		t.Fatalf("delay = %s, want 10s", delayed)
	}
}

func TestSearch_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"data":{"searchProductV5":{"data":{"products":[],"hasNextPage":false}}}}]`))
	}))
	defer server.Close()

	s := newTestScraper(server.URL)
	products, err := s.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 10})

	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(products) != 0 {
		t.Fatalf("len(products) = %d, want 0", len(products))
	}
}

func TestParsePrice(t *testing.T) {
	if got := parsePrice("Rp14.999.000"); got != 14999000 {
		t.Fatalf("parsePrice = %d, want 14999000", got)
	}
}

func TestParseRating(t *testing.T) {
	assertFloat(t, "rating", 4.8, parseRating("4.8"))
}

func newTestScraper(endpoint string) *Scraper {
	return New(15, proxy.NewManager(nil), zap.NewNop(),
		WithEndpoint(endpoint),
		WithPageDelay(func() time.Duration { return 0 }),
		WithDelay(func(ctx context.Context, delay time.Duration) error { return nil }),
	)
}

func requestPage(t *testing.T, r *http.Request) int {
	t.Helper()
	var payload struct {
		Variables struct {
			Params string `json:"params"`
		} `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Variables.Params == "" {
		t.Fatal("empty params")
	}
	params, err := url.ParseQuery(payload.Variables.Params)
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}
	page, err := strconv.Atoi(params.Get("page"))
	if err != nil {
		t.Fatalf("parse page: %v", err)
	}
	return page
}

func responseJSON(index int, hasNextPage bool) string {
	response := `[
  {
    "data": {
      "searchProductV5": {
        "data": {
          "products": [
            {
              "id": "product-INDEX",
              "name": "iPhone INDEX",
              "url": "https://www.tokopedia.com/shop/product-INDEX",
              "imageUrl": "https://images.tokopedia.net/product-INDEX.jpg",
              "price": "Rp14.999.000",
              "slashedPrice": "Rp15.999.000",
              "discountedPercentage": 6,
              "ratingAverage": "4.8",
              "countReview": 1234,
              "countSold": "1,2rb",
              "shop": {
                "id": "shop-INDEX",
                "name": "iBox Official",
                "city": "Jakarta",
                "goldMerchant": false,
                "officialStore": true
              }
            }
          ],
          "hasNextPage": HAS_NEXT,
          "isQuerySafe": true
        }
      }
    }
  }
]`
	response = strings.ReplaceAll(response, `\"`, `"`)
	response = strings.ReplaceAll(response, "INDEX", strconv.Itoa(index))
	response = strings.ReplaceAll(response, "HAS_NEXT", strconv.FormatBool(hasNextPage))
	return response
}

func assertEqual[T comparable](t *testing.T, field string, want, got T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", field, got, want)
	}
}

func assertFloat(t *testing.T, field string, want, got float64) {
	t.Helper()
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("%s = %v, want %v", field, got, want)
	}
}
