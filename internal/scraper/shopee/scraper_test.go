package shopee

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

func TestSearchSuccessUsesAnonymousSession(t *testing.T) {
	var sessionRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			sessionRequests.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "SPC_F", Value: "anon-session", Path: "/"})
			_, _ = w.Write([]byte("ok"))
		case "/api/v4/search/search_items":
			if r.URL.Query().Get("keyword") != "iphone" {
				t.Fatalf("keyword = %s", r.URL.Query().Get("keyword"))
			}
			if cookie, err := r.Cookie("SPC_F"); err != nil || cookie.Value != "anon-session" {
				t.Fatalf("anonymous cookie missing, cookie = %v, err = %v", cookie, err)
			}
			_, _ = w.Write([]byte(sampleResponse))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	shopeeScraper := NewWithClient(server.Client(), server.URL+"/api/v4/search/search_items", zaptest.NewLogger(t))
	products, err := shopeeScraper.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 10})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if sessionRequests.Load() != 1 {
		t.Fatalf("session requests = %d", sessionRequests.Load())
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d", len(products))
	}
	product := products[0]
	if product.Marketplace != MarketplaceName {
		t.Fatalf("marketplace = %s", product.Marketplace)
	}
	if !strings.Contains(product.URL, "shopee.co.id/product/123/456") {
		t.Fatalf("url = %s", product.URL)
	}
	if product.Price != 12500000 {
		t.Fatalf("price = %d", product.Price)
	}
}

func TestSearchReturnsSEOFallbackAfterForbidden(t *testing.T) {
	var searchRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.SetCookie(w, &http.Cookie{Name: "SPC_F", Value: "anon-session", Path: "/"})
			_, _ = w.Write([]byte("ok"))
		case "/api/v4/search/search_items":
			searchRequests.Add(1)
			w.WriteHeader(http.StatusForbidden)
		case "/search":
			_, _ = w.Write([]byte(sampleSEOHTML))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	shopeeScraper := NewWithClient(server.Client(), server.URL+"/api/v4/search/search_items", zaptest.NewLogger(t))
	products, err := shopeeScraper.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 10})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if searchRequests.Load() != 1 {
		t.Fatalf("search requests = %d", searchRequests.Load())
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d", len(products))
	}
}

func TestSearchFallsBackToSEOHTMLAfterForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte("ok"))
		case "/api/v4/search/search_items":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":90309999}`))
		case "/search":
			if got := r.Header.Get("User-Agent"); got != "facebookexternalhit/1.1" {
				t.Fatalf("seo user-agent = %q", got)
			}
			_, _ = w.Write([]byte(sampleSEOHTML))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	shopeeScraper := NewWithClient(server.Client(), server.URL+"/api/v4/search/search_items", zaptest.NewLogger(t))
	products, err := shopeeScraper.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 10})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d", len(products))
	}
	if products[0].Name != "iPhone 15 128GB | 256GB [RESMI] [SEGEL]" {
		t.Fatalf("name = %q", products[0].Name)
	}
	if products[0].Price != 12189000 {
		t.Fatalf("price = %d", products[0].Price)
	}
}

func TestSearchUsesConfiguredCookieHeaderWithoutSessionBootstrap(t *testing.T) {
	var sessionRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			sessionRequests.Add(1)
			_, _ = w.Write([]byte("ok"))
		case "/api/v4/search/search_items":
			if got := r.Header.Get("Cookie"); got != "SPC_F=manual; csrftoken=abc" {
				t.Fatalf("cookie header = %q", got)
			}
			_, _ = w.Write([]byte(sampleResponse))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	shopeeScraper := NewWithClient(server.Client(), server.URL+"/api/v4/search/search_items", zaptest.NewLogger(t), WithCookieHeader("SPC_F=manual; csrftoken=abc"))
	products, err := shopeeScraper.Search(context.Background(), scraper.SearchOptions{Keyword: "iphone", MaxItems: 10})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if sessionRequests.Load() != 0 {
		t.Fatalf("session requests = %d", sessionRequests.Load())
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d", len(products))
	}
}

func TestParseProducts(t *testing.T) {
	products, err := parseProducts([]byte(sampleResponse), "iphone")
	if err != nil {
		t.Fatalf("parseProducts error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("len(products) = %d", len(products))
	}
	if products[0].DiscountPercent != 16 {
		t.Fatalf("discount = %d", products[0].DiscountPercent)
	}
	if products[0].Sold != 1200 {
		t.Fatalf("sold = %d", products[0].Sold)
	}
}

const sampleResponse = `{
  "items": [
    {
      "item_basic": {
        "itemid": 456,
        "shopid": 123,
        "name": "Apple iPhone 15 128GB Garansi Resmi",
        "price": 1250000000000,
        "price_before_discount": 1500000000000,
        "item_rating": {"rating_star": 4.9, "rating_count": [0, 0, 0, 1, 5, 90]},
        "sold": "1,2RB+ terjual",
        "image": "abc123",
        "shop_location": "KOTA JAKARTA SELATAN",
        "shop_name": "Apple Authorized Store",
        "is_official_shop": true
      }
    }
  ]
}`

const sampleSEOHTML = `<div role="group" aria-label="Product card"><a aria-label="View product: iPhone 15 128GB | 256GB [RESMI] [SEGEL]" href="/iPhone-15-128GB-256GB-RESMI-SEGEL--i.30846303.24642171933?extraParams=x"><img src="https://down-id.img.susercontent.com/file/id-abc" alt="iPhone 15"/><span class="font-medium mr-px text-xs/sp14">Rp</span><span class="truncate text-base/5 font-medium">12.189.000</span><span aria-label="-30%"></span><div>5.0</div></div></div></div><div class="flex items-center space-x-1 max-w-full"><span aria-label="location-Jakarta Pusat"></span><span>Jakarta Pusat</span></div></a></div>`
