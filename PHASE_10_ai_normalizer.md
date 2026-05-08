# PHASE 10 — AI Normalizer for Product Groups

## Prerequisite
- Backend PHASE 1–9 sudah selesai dan `go build ./...` sukses.
- Tabel `runs` sudah ada.

## Objective
Menambahkan **AI-based product normalization & grouping** di atas hasil scraping, sehingga satu run bisa diubah menjadi sekumpulan `ProductGroup` yang siap dipakai untuk harga termurah dan komparasi.

## Scope
- Menambah tipe data untuk normalisasi dan grouping.
- Menambah kolom `normalized_json` di tabel `runs`.
- Membuat package `internal/ai` untuk normalizer (tanpa mengunci ke 1 provider tertentu).
- Menambahkan endpoint:
  - `POST /v1/runs/:id/normalize`
  - `GET  /v1/runs/:id/normalized`

> Catatan: di fase ini, implementasi call ke LLM boleh berupa stub/mock dulu jika API key belum siap. Fokus pada struktur kode, interface, dan flow.

---

## Step 10.1 — Update Schema Database

Tambahkan kolom baru di tabel `runs` untuk menyimpan hasil normalisasi:

**File:** `db/migrations/002_add_normalized_to_runs.sql`

```sql
ALTER TABLE runs
ADD COLUMN IF NOT EXISTS normalized_json JSONB;
```

Jalankan migration ini setelah build lewat command yang sesuai (misalnya lewat target `make migrate` atau manual psql).

---

## Step 10.2 — Tambah Tipe Data Normalized di Backend

**File baru:** `internal/ai/types.go`

```go
package ai

import "github.com/[username]/tokopedia-scraper/internal/scraper"

// NormalizedProduct merepresentasikan hasil ekstraksi atribut oleh AI
// untuk satu listing Product.
type NormalizedProduct struct {
    SourceProductID string   `json:"source_product_id"`
    Marketplace     string   `json:"marketplace"`
    URL             string   `json:"url"`

    Brand           string   `json:"brand"`
    Model           string   `json:"model"`
    Variant         string   `json:"variant"`
    CategoryPath    string   `json:"category_path"`
    ImportantSpecs  []string `json:"important_specs"`
    CanonicalKey    string   `json:"canonical_key"`
}

// GroupedItem adalah listing asli yang masuk ke dalam satu grup produk.
type GroupedItem struct {
    ProductID       string  `json:"product_id"`
    Marketplace     string  `json:"marketplace"`
    Name            string  `json:"name"`
    Price           int64   `json:"price"`
    OriginalPrice   int64   `json:"original_price"`
    DiscountPercent int     `json:"discount_percent"`
    Rating          float64 `json:"rating"`
    CountReview     int     `json:"count_review"`
    ShopName        string  `json:"shop_name"`
    IsOfficialStore bool    `json:"is_official_store"`
    URL             string  `json:"url"`
}

// ProductGroup merepresentasikan satu produk fisik (canonical) yang
digabung dari banyak listing di satu atau beberapa marketplace.
type ProductGroup struct {
    GroupID        string         `json:"group_id"`
    CanonicalName  string         `json:"canonical_name"`
    Brand          string         `json:"brand"`
    Model          string         `json:"model"`
    Variant        string         `json:"variant"`
    CategoryPath   string         `json:"category_path"`
    ImportantSpecs []string       `json:"important_specs"`

    Items          []GroupedItem  `json:"items"`
    MinPrice       int64          `json:"min_price"`
    MaxPrice       int64          `json:"max_price"`
    AvgPrice       float64        `json:"avg_price"`
    BestPriceID    string         `json:"best_price_id"`
}

// Helper untuk konversi Product -> GroupedItem
func FromProduct(p scraper.Product) GroupedItem {
    return GroupedItem{
        ProductID:       p.ID,
        Marketplace:     p.Marketplace,
        Name:            p.Name,
        Price:           p.Price,
        OriginalPrice:   p.OriginalPrice,
        DiscountPercent: p.DiscountPercent,
        Rating:          p.Rating,
        CountReview:     p.CountReview,
        ShopName:        p.ShopName,
        IsOfficialStore: p.IsOfficialStore,
        URL:             p.URL,
    }
}
```

> Pastikan import path `github.com/[username]/tokopedia-scraper` disesuaikan dengan module kamu.

---

## Step 10.3 — LLM Client Abstraction

**File baru:** `internal/ai/client.go`

Tujuan: membuat wrapper abstrak untuk panggilan ke LLM provider apapun (OpenAI/Groq/dll). Untuk fase ini, cukup definisikan interface dan skeleton implementation yang bisa diisi kemudian.

```go
package ai

import (
    "context"
    "fmt"
)

// LLMClient mendefinisikan operasi yang dibutuhkan untuk AI layer.
type LLMClient interface {
    NormalizeProducts(ctx context.Context, itemsJSON []byte) ([]NormalizedProduct, error)
}

// DummyClient adalah implementasi sementara yang tidak memanggil API eksternal.
// Nanti bisa diganti dengan implementasi nyata (OpenAI, dsb.).
type DummyClient struct{}

func NewDummyClient() *DummyClient {
    return &DummyClient{}
}

func (c *DummyClient) NormalizeProducts(ctx context.Context, itemsJSON []byte) ([]NormalizedProduct, error) {
    // Implementasi awal: return error untuk mengingatkan bahwa AI sebenarnya belum dikonfigurasi.
    return nil, fmt.Errorf("LLM client not configured: please implement real provider or replace DummyClient")
}
```

> Di environment produksi, kamu bisa mengganti DummyClient dengan client yang memanggil API LLM sungguhan.

---

## Step 10.4 — Normalizer Logic (Chunking & Grouping)

**File baru:** `internal/ai/normalizer.go`

Buat dua fungsi utama:
- `NormalizeRun(ctx, repo, runID, llmClient) ([]ProductGroup, error)` — orchestration level run.
- `groupNormalized(products, normalized) ([]ProductGroup, error)` — pure function untuk grouping.

```go
package ai

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/[username]/tokopedia-scraper/internal/run"
    "github.com/[username]/tokopedia-scraper/internal/scraper"
)

const maxItemsPerChunk = 40

// NormalizeRun memuat hasil scraping dari run tertentu, memanggil LLM
// untuk normalisasi per chunk, lalu melakukan grouping menjadi ProductGroup.
func NormalizeRun(ctx context.Context, repo run.Repository, runID string, client LLMClient) ([]ProductGroup, error) {
    r, err := repo.GetByID(ctx, runID)
    if err != nil {
        return nil, fmt.Errorf("NormalizeRun: get run: %w", err)
    }
    if r.Status != run.StatusSucceeded {
        return nil, fmt.Errorf("NormalizeRun: run status must be SUCCEEDED, got %s", r.Status)
    }
    if len(r.ResultJSON) == 0 {
        return nil, fmt.Errorf("NormalizeRun: run has no result_json")
    }

    var products []scraper.Product
    if err := json.Unmarshal(r.ResultJSON, &products); err != nil {
        return nil, fmt.Errorf("NormalizeRun: unmarshal products: %w", err)
    }

    var allNormalized []NormalizedProduct
    for i := 0; i < len(products); i += maxItemsPerChunk {
        end := i + maxItemsPerChunk
        if end > len(products) {
            end = len(products)
        }
        chunk := products[i:end]

        // Siapkan JSON ringkas untuk dikirim ke LLM
        payload := struct {
            Items []scraper.Product `json:"items"`
        }{Items: chunk}

        payloadBytes, err := json.Marshal(payload)
        if err != nil {
            return nil, fmt.Errorf("NormalizeRun: marshal chunk: %w", err)
        }

        normalizedChunk, err := client.NormalizeProducts(ctx, payloadBytes)
        if err != nil {
            return nil, fmt.Errorf("NormalizeRun: LLM normalize chunk %d: %w", i/maxItemsPerChunk, err)
        }
        allNormalized = append(allNormalized, normalizedChunk...)
    }

    groups, err := groupNormalized(products, allNormalized)
    if err != nil {
        return nil, fmt.Errorf("NormalizeRun: grouping: %w", err)
    }

    // Simpan ke normalized_json di tabel runs
    groupsBytes, err := json.Marshal(groups)
    if err != nil {
        return nil, fmt.Errorf("NormalizeRun: marshal groups: %w", err)
    }
    if err := repo.SaveNormalized(ctx, runID, groupsBytes); err != nil {
        return nil, fmt.Errorf("NormalizeRun: save normalized: %w", err)
    }

    return groups, nil
}

// groupNormalized menggabungkan NormalizedProduct + Product menjadi ProductGroup.
func groupNormalized(products []scraper.Product, normalized []NormalizedProduct) ([]ProductGroup, error) {
    // Index Product by ID
    prodByID := make(map[string]scraper.Product, len(products))
    for _, p := range products {
        prodByID[p.ID] = p
    }

    // Group NormalizedProduct by CanonicalKey
    groupsMap := make(map[string]*ProductGroup)

    for _, n := range normalized {
        if n.CanonicalKey == "" {
            // Skip item tanpa canonical key
            continue
        }
        p, ok := prodByID[n.SourceProductID]
        if !ok {
            continue
        }

        g, ok := groupsMap[n.CanonicalKey]
        if !ok {
            g = &ProductGroup{
                GroupID:       n.CanonicalKey,
                CanonicalName: buildCanonicalName(n.Brand, n.Model, n.Variant),
                Brand:         n.Brand,
                Model:         n.Model,
                Variant:       n.Variant,
                CategoryPath:  n.CategoryPath,
                ImportantSpecs: append([]string{}, n.ImportantSpecs...),
            }
            groupsMap[n.CanonicalKey] = g
        }

        item := FromProduct(p)
        g.Items = append(g.Items, item)
    }

    // Hitung metrik harga
    var groups []ProductGroup
    for _, g := range groupsMap {
        var sum int64
        var bestPrice int64
        var bestID string
        for i, item := range g.Items {
            price := item.Price
            sum += price
            if i == 0 || price < bestPrice {
                bestPrice = price
                bestID = item.ProductID
            }
        }
        if len(g.Items) > 0 {
            g.MinPrice = g.Items[0].Price
            g.MaxPrice = g.Items[0].Price
            for _, item := range g.Items[1:] {
                if item.Price < g.MinPrice {
                    g.MinPrice = item.Price
                }
                if item.Price > g.MaxPrice {
                    g.MaxPrice = item.Price
                }
            }
            g.AvgPrice = float64(sum) / float64(len(g.Items))
        }
        g.BestPriceID = bestID
        groups = append(groups, *g)
    }

    return groups, nil
}

func buildCanonicalName(brand, model, variant string) string {
    // Sederhana dulu; bisa diperbaiki nanti.
    name := ""
    if brand != "" {
        name += brand
    }
    if model != "" {
        if name != "" {
            name += " "
        }
        name += model
    }
    if variant != "" {
        if name != "" {
            name += " "
        }
        name += variant
    }
    return name
}
```

> `SaveNormalized` akan ditambahkan di repository pada step berikutnya.

---

## Step 10.5 — Extend Run Repository untuk Normalized JSON

**File:** `internal/run/repository.go`

Tambahkan method baru di interface `Repository`:

```go
SaveNormalized(ctx context.Context, id string, normalizedJSON []byte) error
```

Dan implementasinya di `postgresRepository`:

```go
func (r *postgresRepository) SaveNormalized(ctx context.Context, id string, normalizedJSON []byte) error {
    _, err := r.pool.Exec(ctx, `
        UPDATE runs
        SET normalized_json = $1
        WHERE id = $2
    `, normalizedJSON, id)
    if err != nil {
        return fmt.Errorf("repository.SaveNormalized: %w", err)
    }
    return nil
}
```

Tambahkan juga field baru di struct `Run` (file `internal/run/model.go`):

```go
NormalizedJSON json.RawMessage `json:"normalized,omitempty" db:"normalized_json"`
```

Sesuaikan query `SELECT`/`INSERT` di repository agar menyertakan kolom ini.

---

## Step 10.6 — Endpoint HTTP untuk Normalisasi & Fetch Group

**File:** `internal/run/handler.go`

Tambahkan dua handler baru di struct `Handler`:

```go
func (h *Handler) RegisterRoutes(e *echo.Echo) {
    v1 := e.Group("/v1")
    // existing routes...
    v1.POST("/scrape/tokopedia/search", h.SubmitTokopediaSearch)
    v1.GET("/runs", h.ListRuns)
    v1.GET("/runs/:id", h.GetRun)
    v1.DELETE("/runs/:id", h.DeleteRun)

    // AI normalization
    v1.POST("/runs/:id/normalize", h.NormalizeRun)
    v1.GET("/runs/:id/normalized", h.GetNormalizedRun)
}
```

Implementasi handler:

```go
// POST /v1/runs/:id/normalize
func (h *Handler) NormalizeRun(c echo.Context) error {
    id := c.Param("id")
    ctx := c.Request().Context()

    // Untuk sementara gunakan DummyClient; nanti bisa di-inject LLMClient nyata.
    llmClient := ai.NewDummyClient()

    groups, err := ai.NormalizeRun(ctx, h.repo, id, llmClient)
    if err != nil {
        h.logger.Error("normalize run failed", zap.String("run_id", id), zap.Error(err))
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "run_id":   id,
        "groups":   groups,
        "group_cnt": len(groups),
    })
}

// GET /v1/runs/:id/normalized
func (h *Handler) GetNormalizedRun(c echo.Context) error {
    id := c.Param("id")
    ctx := c.Request().Context()

    r, err := h.repo.GetByID(ctx, id)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "run not found")
    }
    if len(r.NormalizedJSON) == 0 {
        return echo.NewHTTPError(http.StatusNotFound, "normalized result not found for this run")
    }

    var groups []ai.ProductGroup
    if err := json.Unmarshal(r.NormalizedJSON, &groups); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to decode normalized data")
    }

    return c.JSON(http.StatusOK, groups)
}
```

> Sesuaikan import: `internal/ai`, `encoding/json`, `go.uber.org/zap`.

---

## Step 10.7 — Verification

1. Jalankan migration untuk kolom `normalized_json`.
2. `go build ./...` harus sukses.
3. Jalankan job scraping seperti biasa untuk mendapatkan run dengan status SUCCEEDED.
4. Panggil:

```bash
curl -X POST http://localhost:8080/v1/runs/<RUN_ID>/normalize
```

Karena masih pakai `DummyClient`, seharusnya mendapat error yang jelas dari handler.
Ini menandakan flow normalisasi sudah siap, tinggal mengisi LLMClient nyata.

5. Setelah nanti LLMClient nyata diimplementasi, Anda bisa verifikasi:

```bash
curl http://localhost:8080/v1/runs/<RUN_ID>/normalized | jq .
```

Harus mengembalikan array `ProductGroup`.

Update checklist di AGENTS.md: Phase 10 ✅
