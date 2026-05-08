# PHASE 2 — Core Structs & Interfaces

## Prerequisite
Phase 1 selesai. `go build ./...` sudah sukses.

## Objective
Definisikan semua kontrak data (struct + interface) yang akan dipakai seluruh sistem.
Ini adalah "sumber kebenaran" — file lain mengikuti definisi di sini.

## Scope
- `internal/scraper/interface.go`
- `internal/run/model.go`

Jangan implement logic apapun di fase ini.

---

## Step 2.1 — File: `internal/scraper/interface.go`

```go
package scraper

import "context"

// SortBy values yang valid
const (
    SortByRelevancy = "relevancy"
    SortByPriceAsc  = "price_asc"
    SortByPriceDesc = "price_desc"
    SortByLatest    = "latest"
)

// Product adalah representasi unified produk dari marketplace manapun
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

// SearchOptions adalah parameter pencarian yang dikirim user
type SearchOptions struct {
    Keyword  string `json:"keyword"`
    MaxItems int    `json:"max_items"`
    SortBy   string `json:"sort_by"`
    MinPrice int64  `json:"min_price"`
    MaxPrice int64  `json:"max_price"`
}

// Validate memastikan SearchOptions valid sebelum dieksekusi
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

// MarketplaceScraper adalah interface yang diimplementasikan tiap marketplace
type MarketplaceScraper interface {
    Search(ctx context.Context, opts SearchOptions) ([]Product, error)
    Name() string
}
```

> ⚠️ Tambahkan import `"fmt"` di package scraper.

---

## Step 2.2 — File: `internal/run/model.go`

```go
package run

import (
    "encoding/json"
    "time"
)

// Status enum untuk lifecycle sebuah run
type Status string

const (
    StatusQueued    Status = "QUEUED"
    StatusRunning   Status = "RUNNING"
    StatusSucceeded Status = "SUCCEEDED"
    StatusFailed    Status = "FAILED"
    StatusTimedOut  Status = "TIMED_OUT"
)

// Run merepresentasikan satu eksekusi scraping job
type Run struct {
    ID           string          `json:"id"                    db:"id"`
    Status       Status          `json:"status"                db:"status"`
    Marketplace  string          `json:"marketplace"           db:"marketplace"`
    InputJSON    json.RawMessage `json:"input"                 db:"input_json"`
    ResultJSON   json.RawMessage `json:"result,omitempty"      db:"result_json"`
    ErrorMessage string          `json:"error_message,omitempty" db:"error_message"`
    ItemCount    int             `json:"item_count"            db:"item_count"`
    CreatedAt    time.Time       `json:"created_at"            db:"created_at"`
    StartedAt    *time.Time      `json:"started_at,omitempty"  db:"started_at"`
    FinishedAt   *time.Time      `json:"finished_at,omitempty" db:"finished_at"`
}

// IsTerminal returns true jika run sudah selesai (tidak bisa berubah lagi)
func (r *Run) IsTerminal() bool {
    return r.Status == StatusSucceeded ||
        r.Status == StatusFailed ||
        r.Status == StatusTimedOut
}
```

---

## Verification

```bash
go build ./...
```
Harus sukses. Update checklist di AGENTS.md: Phase 2 ✅
