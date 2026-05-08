# PHASE 11 — AI Summary & Recommendations

## Prerequisite
- PHASE 10 (AI Normalizer) sudah selesai.
- `normalized_json` sudah terisi untuk beberapa run (bisa diisi manual/stub dulu).

## Objective
Menambahkan fitur **AI summary & recommendation** di atas `ProductGroup`.
Hasilnya bisa ditampilkan di UI sebagai insight yang human-friendly.

## Scope
- Menambah tipe data `AISummaryResult`.
- Menambah kolom `ai_summary_json` di tabel `runs`.
- Menambah method tambahan di `LLMClient`.
- Menambah endpoint:
  - `POST /v1/runs/:id/ai-summary`
  - `GET  /v1/runs/:id/ai-summary`

---

## Step 11.1 — Update Schema Database

**File:** `db/migrations/003_add_ai_summary_to_runs.sql`

```sql
ALTER TABLE runs
ADD COLUMN IF NOT EXISTS ai_summary_json JSONB;
```

Jalankan migration ini setelah build.

---

## Step 11.2 — Tambah Tipe Data AISummaryResult

**File:** `internal/ai/types.go`

Tambahkan di bawah tipe `ProductGroup`:

```go
// AISummaryResult merepresentasikan hasil ringkasan & rekomendasi dari AI.
type AISummaryResult struct {
    SummaryText      string             `json:"summary_text"`
    RecommendedItems []RecommendedItem  `json:"recommended_items"`
}

type RecommendedItem struct {
    GroupID   string `json:"group_id"`
    ProductID string `json:"product_id"`
    Reason    string `json:"reason"`
}
```

---

## Step 11.3 — Extend LLMClient untuk Summary

**File:** `internal/ai/client.go`

Update interface `LLMClient`:

```go
type LLMClient interface {
    NormalizeProducts(ctx context.Context, itemsJSON []byte) ([]NormalizedProduct, error)
    SummarizeGroups(ctx context.Context, groupsJSON []byte, userPrompt string) (*AISummaryResult, error)
}
```

Untuk `DummyClient`, implementasikan stub:

```go
func (c *DummyClient) SummarizeGroups(ctx context.Context, groupsJSON []byte, userPrompt string) (*AISummaryResult, error) {
    return nil, fmt.Errorf("LLM client not configured: SummarizeGroups not implemented")
}
```

---

## Step 11.4 — Summary Logic di Backend

**File baru:** `internal/ai/summary.go`

```go
package ai

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/[username]/tokopedia-scraper/internal/run"
)

// SummarizeRun memuat normalized_json untuk suatu run, lalu memanggil LLM
// untuk menghasilkan ringkasan & rekomendasi.
func SummarizeRun(ctx context.Context, repo run.Repository, runID string, client LLMClient, userPrompt string) (*AISummaryResult, error) {
    r, err := repo.GetByID(ctx, runID)
    if err != nil {
        return nil, fmt.Errorf("SummarizeRun: get run: %w", err)
    }
    if len(r.NormalizedJSON) == 0 {
        return nil, fmt.Errorf("SummarizeRun: normalized_json is empty; run NormalizeRun first")
    }

    // Potong data jika perlu (misalnya hanya kirim N group teratas)
    var groups []ProductGroup
    if err := json.Unmarshal(r.NormalizedJSON, &groups); err != nil {
        return nil, fmt.Errorf("SummarizeRun: unmarshal normalized_json: %w", err)
    }

    // Simple: kirim semua groups, nanti bisa dioptimasi
    payload := struct {
        Groups []ProductGroup `json:"groups"`
    }{Groups: groups}

    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("SummarizeRun: marshal groups: %w", err)
    }

    if userPrompt == "" {
        userPrompt = defaultSummaryPrompt()
    }

    result, err := client.SummarizeGroups(ctx, payloadBytes, userPrompt)
    if err != nil {
        return nil, fmt.Errorf("SummarizeRun: LLM summarize: %w", err)
    }

    // Simpan ke ai_summary_json
    resBytes, err := json.Marshal(result)
    if err != nil {
        return nil, fmt.Errorf("SummarizeRun: marshal result: %w", err)
    }
    if err := repo.SaveAISummary(ctx, runID, resBytes); err != nil {
        return nil, fmt.Errorf("SummarizeRun: save ai summary: %w", err)
    }

    return result, nil
}

func defaultSummaryPrompt() string {
    return `Kamu adalah asisten belanja untuk pengguna di Indonesia.

Tugasmu:
1. Baca data "groups" yang berisi kelompok produk dengan harga dan rating.
2. Berikan ringkasan singkat dalam bahasa Indonesia tentang:
   - pola harga (misalnya range harga, brand mahal/murah),
   - hal yang perlu diwaspadai (harga terlalu murah, rating jelek).
3. Pilih maksimal 5 rekomendasi terbaik untuk value for money.

Untuk setiap rekomendasi, sertakan:
- group_id
- product_id
- reason (1-2 kalimat)

Jawab HANYA dalam format JSON dengan struktur:
{
  "summary_text": "...",
  "recommended_items": [
    {"group_id": "...", "product_id": "...", "reason": "..."}
  ]
}
`}
```

---

## Step 11.5 — Extend Run Repository untuk AI Summary

**File:** `internal/run/repository.go`

Di interface `Repository`, tambah method:

```go
SaveAISummary(ctx context.Context, id string, summaryJSON []byte) error
```

Implementasi:

```go
func (r *postgresRepository) SaveAISummary(ctx context.Context, id string, summaryJSON []byte) error {
    _, err := r.pool.Exec(ctx, `
        UPDATE runs
        SET ai_summary_json = $1
        WHERE id = $2
    `, summaryJSON, id)
    if err != nil {
        return fmt.Errorf("repository.SaveAISummary: %w", err)
    }
    return nil
}
```

Tambahkan field baru di struct `Run` (file `internal/run/model.go`):

```go
AISummaryJSON json.RawMessage `json:"ai_summary,omitempty" db:"ai_summary_json"`
```

Update query SELECT agar field ini ikut dibaca.

---

## Step 11.6 — Endpoint HTTP untuk AI Summary

**File:** `internal/run/handler.go`

Tambahkan route baru:

```go
func (h *Handler) RegisterRoutes(e *echo.Echo) {
    v1 := e.Group("/v1")
    // existing routes...

    v1.POST("/runs/:id/normalize", h.NormalizeRun)
    v1.GET("/runs/:id/normalized", h.GetNormalizedRun)

    // AI summary
    v1.POST("/runs/:id/ai-summary", h.GenerateAISummary)
    v1.GET("/runs/:id/ai-summary", h.GetAISummary)
}
```

Tambahkan struct request body simple (di file yang sama atau file baru):

```go
type aiSummaryRequest struct {
    Prompt string `json:"prompt"`
}
```

Implementasi handler:

```go
// POST /v1/runs/:id/ai-summary
func (h *Handler) GenerateAISummary(c echo.Context) error {
    id := c.Param("id")
    ctx := c.Request().Context()

    var req aiSummaryRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
    }

    llmClient := ai.NewDummyClient() // ganti dengan client nyata nanti

    res, err := ai.SummarizeRun(ctx, h.repo, id, llmClient, req.Prompt)
    if err != nil {
        h.logger.Error("ai summary failed", zap.String("run_id", id), zap.Error(err))
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    return c.JSON(http.StatusOK, res)
}

// GET /v1/runs/:id/ai-summary
func (h *Handler) GetAISummary(c echo.Context) error {
    id := c.Param("id")
    ctx := c.Request().Context()

    r, err := h.repo.GetByID(ctx, id)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "run not found")
    }
    if len(r.AISummaryJSON) == 0 {
        return echo.NewHTTPError(http.StatusNotFound, "ai summary not found for this run")
    }

    var summary ai.AISummaryResult
    if err := json.Unmarshal(r.AISummaryJSON, &summary); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to decode ai summary")
    }

    return c.JSON(http.StatusOK, summary)
}
```

Sesuaikan import: `internal/ai`, `encoding/json`, `go.uber.org/zap`.

---

## Step 11.7 — Verification

1. Jalankan migration untuk kolom `ai_summary_json`.
2. `go build ./...` harus sukses.
3. Pastikan setidaknya satu run sudah memiliki `normalized_json` (bisa isi manual dengan JSON dummy ProductGroup).
4. Panggil:

```bash
curl -X POST http://localhost:8080/v1/runs/<RUN_ID>/ai-summary   -H "Content-Type: application/json"   -d '{"prompt":"Ringkas pilihan terbaik untuk budget 10-12 juta"}'
```

Karena masih memakai `DummyClient`, seharusnya mengembalikan error yang jelas.
Ini menandakan flow summary sudah tersambung; nanti tinggal mengganti DummyClient dengan implementasi LLM yang sebenarnya.

5. Setelah LLM nyata terpasang, verifikasi:

```bash
curl http://localhost:8080/v1/runs/<RUN_ID>/ai-summary | jq .
```

Harus mengembalikan `summary_text` dan `recommended_items`.

Update checklist di AGENTS.md: Phase 11 ✅
