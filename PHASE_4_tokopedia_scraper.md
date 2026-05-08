# PHASE 4 — Tokopedia Scraper

## Prerequisite
Phase 1-3 selesai. `go build ./...` sukses.

## Objective
Implementasi scraper Tokopedia menggunakan HTTP POST ke internal GraphQL API.
Ini adalah fase paling kritikal — kerjakan dengan teliti.

## Scope
- `internal/proxy/manager.go` (dibuat dulu, dipakai scraper)
- `internal/scraper/tokopedia/scraper.go`
- `internal/scraper/tokopedia/parser.go`
- `internal/scraper/tokopedia/scraper_test.go`

---

## Step 4.1 — File: `internal/proxy/manager.go`

```go
package proxy

import (
    "math/rand"
    "sync"
)

type Manager struct {
    proxies []string
    mu      sync.Mutex
    current int
}

func NewManager(proxies []string) *Manager {
    return &Manager{proxies: proxies}
}

// GetProxy return proxy berikutnya (round-robin), atau "" jika tidak ada proxy
func (m *Manager) GetProxy() string {
    m.mu.Lock()
    defer m.mu.Unlock()
    if len(m.proxies) == 0 {
        return ""
    }
    p := m.proxies[m.current%len(m.proxies)]
    m.current++
    return p
}

// GetRandom return proxy random dari pool
func (m *Manager) GetRandom() string {
    m.mu.Lock()
    defer m.mu.Unlock()
    if len(m.proxies) == 0 {
        return ""
    }
    return m.proxies[rand.Intn(len(m.proxies))]
}

func (m *Manager) Len() int {
    return len(m.proxies)
}
```

---

## Step 4.2 — File: `internal/scraper/tokopedia/scraper.go`

Implement TokopediaScraper dengan spesifikasi:

### Endpoint & Protocol
- **URL:** `https://gql.tokopedia.com/`
- **Method:** POST
- **Protocol:** HTTP/2 (pakai standard `http.Client`)

### Sort mapping (`ob` parameter)
```
price_asc  → ob=3
price_desc → ob=4
latest     → ob=9
relevancy  → ob=23 (default)
```

### Request headers wajib
```
Content-Type:    application/json
User-Agent:      [random dari pool di bawah]
Referer:         https://www.tokopedia.com/search?q={keyword}
X-Source:        tokopedia-lite
X-Device:        default_v3
Accept:          */*
Accept-Language: id-ID,id;q=0.9,en-US;q=0.8
```

### User-Agent Pool (minimal 8 UA)
```
Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36
Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 Edg/123.0.0.0
Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36
Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15
Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36
Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0
Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Mobile/15E148 Safari/604.1
Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.6367.82 Mobile Safari/537.36
```

### Pagination
- `rows=28` per halaman (default Tokopedia search)
- Loop page 1, 2, 3... sampai `MaxItems` terpenuhi atau `hasNextPage = false` atau hasil kosong
- Delay **1500–3000ms random** antar halaman (bukan antar request, tapi antar page)

### Error handling & Retry
- Status 429 atau 503: tunggu 10 detik, retry sekali
- Response body kosong / JSON invalid 2x berturut: return hasil yang sudah ada + log warning
- Timeout per request: pakai `RequestTimeoutSec` dari config (default 15 detik)

### GraphQL Payload
```json
[
  {
    "operationName": "SearchProductQueryV4",
    "variables": {
      "params": "q={keyword}&ob={ob}&page={page}&rows={rows}&price_min={min}&price_max={max}&start={start}"
    },
    "query": "fragment ProductHighlight on searchProductV5Product { id name url imageURL:imageUrl price originalPrice:slashedPrice discountedPercentage ratingAverage countReview countSold shop { id name city goldMerchant officialStore } badges { title imageURL } }"
  }
]
```

> ⚠️ Query fragment di atas adalah approximation. Saat implementasi, tangkap real response dari browser DevTools
> di `Network > gql.tokopedia.com > Preview` dan sesuaikan field names jika berbeda.
> Prioritaskan response yang actual dari browser daripada hardcode query yang mungkin sudah berubah.

---

## Step 4.3 — File: `internal/scraper/tokopedia/parser.go`

Buat fungsi untuk parse response JSON Tokopedia ke `[]scraper.Product`.

Struktur response yang umum dari Tokopedia GraphQL search:
```json
[
  {
    "data": {
      "searchProductV5": {
        "data": {
          "products": [
            {
              "id": "...",
              "name": "...",
              "url": "...",
              "imageUrl": "...",
              "price": "Rp14.999.000",
              "slashedPrice": "Rp15.999.000",
              "discountedPercentage": 6,
              "ratingAverage": "4.8",
              "countReview": 1234,
              "countSold": "1,2rb",
              "shop": {
                "id": "...",
                "name": "iBox Official",
                "city": "Jakarta",
                "goldMerchant": false,
                "officialStore": true
              }
            }
          ],
          "isQuerySafe": true
        }
      }
    }
  }
]
```

Handle edge cases di parser:
- `price` adalah string format "Rp14.999.000" → harus di-parse ke int64 (strip "Rp", ".", ",")
- `ratingAverage` adalah string "4.8" → parse ke float64
- `countSold` bisa "1,2rb" atau "123" → parse ke int best-effort (boleh return 0 jika gagal)
- Field kosong / null → gunakan zero value, jangan panic

---

## Step 4.4 — File: `internal/scraper/tokopedia/scraper_test.go`

Buat test menggunakan `httptest.NewServer` untuk mock Tokopedia endpoint.

Test cases yang wajib ada:
1. `TestSearch_Success` — keyword valid, single page, return produk dengan semua field
2. `TestSearch_PaginationLoop` — MaxItems=60, verify fetch page 1 dan 2 (mock 2 halaman)
3. `TestSearch_RateLimit` — mock return 429, verify retry dengan delay
4. `TestSearch_EmptyResponse` — mock return empty products, return nil error
5. `TestParsePrice` — unit test parse "Rp14.999.000" → 14999000
6. `TestParseRating` — unit test parse "4.8" → 4.8 float64

---

## Verification

```bash
go build ./...
go test ./internal/scraper/... -v
```

Semua test harus PASS. Update checklist AGENTS.md: Phase 4 ✅
