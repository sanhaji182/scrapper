# PHASE 13 — Real LLM Client (Multi-Provider)

## Prerequisite
- Phase 10 & 11 sudah selesai (AI Normalizer + AI Summary dengan DummyClient).
- `go build ./...` sukses.

## Objective
Mengganti `DummyClient` dengan implementasi nyata yang mendukung **multiple LLM provider**:
- **OpenAI** (`gpt-4.1-mini`, `o4-mini`, dll)
- **Groq** (`llama-3.3-70b`, `mixtral-8x7b`, dll)
- **Ollama** (self-hosted lokal: `llama3.2`, `qwen2.5`, dll)
- **Gemini** via OpenAI-compatible endpoint (`gemini-2.0-flash`, dll)

Semua provider menggunakan satu implementasi HTTP client karena formatnya kompatibel OpenAI.
Provider dipilih via env variable `AI_PROVIDER` tanpa ubah kode logic.

## Scope
- `internal/config/config.go` — tambah AI config fields
- `internal/ai/prompt.go` — template prompt sebagai constants/functions
- `internal/ai/openai_client.go` — OpenAI-compatible HTTP client (cover semua provider)
- `internal/ai/openai_client_test.go` — unit test dengan mock HTTP
- `cmd/api/main.go` + `cmd/worker/main.go` — wire LLM client nyata
- `.env.example` — tambah AI env variables

---

## Step 13.1 — Extend Config

**File:** `internal/config/config.go`

Tambahkan field berikut di struct `Config`:

```go
// AI Config
AIProvider      string // "openai" | "groq" | "ollama" | "gemini"
AIAPIKey        string // kosong untuk Ollama
AIModel         string // model name sesuai provider
AITimeoutSec    int    // default 30
AIMaxRetries    int    // default 2
```

Di fungsi `Load()`, tambahkan:

```go
cfg.AIProvider   = getEnv("AI_PROVIDER", "openai")
cfg.AIAPIKey     = getEnv("AI_API_KEY", "")
cfg.AIModel      = getEnv("AI_MODEL", "gpt-4.1-mini")
cfg.AITimeoutSec = getEnvInt("AI_TIMEOUT_SEC", 30)
cfg.AIMaxRetries = getEnvInt("AI_MAX_RETRIES", 2)
```

---

## Step 13.2 — Prompt Templates

**File baru:** `internal/ai/prompt.go`

Pisahkan semua prompt menjadi fungsi tersendiri agar gampang di-test dan di-edit.

```go
package ai

import "encoding/json"

// normalizeSystemPrompt adalah system prompt untuk normalisasi produk.
// Dikirim sebagai role "system" — konsisten di semua batch.
const normalizeSystemPrompt = `Kamu adalah sistem normalisasi katalog produk e-commerce Indonesia.
Tugasmu HANYA mengekstrak atribut dari data produk yang diberikan.
Aturan:
- Output HARUS berupa JSON valid. Tidak ada teks lain di luar JSON.
- Jangan mengarang brand, model, atau spesifikasi yang tidak ada di input.
- Jangan menambah produk baru.
- Jika suatu field tidak dapat ditentukan dari teks, isi dengan string kosong "".
- canonical_key harus berupa slug lowercase, gunakan tanda hubung "-" sebagai pemisah.`

// BuildNormalizeUserPrompt membentuk user prompt dengan data items yang akan dinormalisasi.
func BuildNormalizeUserPrompt(itemsJSON []byte) string {
    return `Ekstrak atribut dari setiap produk di array "items" berikut.

Kembalikan JSON dengan schema persis seperti ini:
{
  "normalized_items": [
    {
      "source_product_id": "string",
      "marketplace": "string",
      "url": "string",
      "brand": "string",
      "model": "string",
      "variant": "string",
      "category_path": "string",
      "important_specs": ["string"],
      "canonical_key": "string"
    }
  ]
}

Data produk:
` + string(itemsJSON)
}

// retryNormalizePrompt digunakan saat LLM mengembalikan JSON yang tidak valid.
const retryNormalizePrompt = `Output JSON kamu tidak valid atau tidak sesuai schema.
Perbaiki dan kembalikan HANYA JSON valid sesuai schema yang sudah diminta.
Jangan menambahkan penjelasan apapun.`

// summarizeSystemPrompt adalah system prompt untuk AI summary & recommendation.
const summarizeSystemPrompt = `Kamu adalah asisten belanja untuk pengguna di Indonesia.
Tugasmu memberikan ringkasan dan rekomendasi berdasarkan data produk yang diberikan.
Aturan:
- Hanya mengacu pada data di input. Jangan mengarang produk atau harga baru.
- Harga dalam format integer (Rupiah), bukan string.
- Output HARUS berupa JSON valid. Tidak ada teks lain di luar JSON.`

// BuildSummarizeUserPrompt membentuk user prompt untuk summary dengan data groups.
func BuildSummarizeUserPrompt(groupsJSON []byte, userInstruction string) string {
    if userInstruction == "" {
        userInstruction = `Berikan ringkasan singkat tentang pola harga dan hal yang perlu diwaspadai.
Pilih maksimal 5 rekomendasi terbaik untuk value for money di Indonesia.`
    }

    return userInstruction + `

Kembalikan JSON dengan schema persis seperti ini:
{
  "summary_text": "string (ringkasan dalam bahasa Indonesia)",
  "recommended_items": [
    {
      "group_id": "string",
      "product_id": "string",
      "reason": "string (1-2 kalimat)"
    }
  ]
}

Data produk groups:
` + string(groupsJSON)
}

// retrySummarizePrompt digunakan saat LLM mengembalikan JSON yang tidak valid.
const retrySummarizePrompt = `Output JSON kamu tidak valid atau tidak sesuai schema.
Perbaiki dan kembalikan HANYA JSON valid sesuai schema yang sudah diminta.
Jangan menambahkan penjelasan apapun.`
```

---

## Step 13.3 — OpenAI-Compatible Client

**File baru:** `internal/ai/openai_client.go`

Satu implementasi yang cover semua provider dengan URL base yang berbeda.

```go
package ai

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "go.uber.org/zap"
    "github.com/[username]/tokopedia-scraper/internal/config"
)

// Provider base URLs
const (
    BaseURLOpenAI = "https://api.openai.com/v1"
    BaseURLGroq   = "https://api.groq.com/openai/v1"
    BaseURLOllama = "http://localhost:11434/v1"
    BaseURLGemini = "https://generativelanguage.googleapis.com/v1beta/openai"
)

// chatMessage adalah satu pesan dalam conversation LLM.
type chatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// chatRequest adalah body request ke endpoint /chat/completions.
type chatRequest struct {
    Model          string            `json:"model"`
    Messages       []chatMessage     `json:"messages"`
    ResponseFormat map[string]string `json:"response_format,omitempty"`
    Temperature    float64           `json:"temperature"`
    MaxTokens      int               `json:"max_tokens,omitempty"`
}

// chatResponse adalah subset dari response /chat/completions.
type chatResponse struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"`
    } `json:"choices"`
    Error *struct {
        Message string `json:"message"`
        Type    string `json:"type"`
    } `json:"error,omitempty"`
}

// OpenAICompatibleClient adalah implementasi LLMClient yang kompatibel dengan
// semua provider yang menggunakan format API OpenAI (OpenAI, Groq, Ollama, Gemini).
type OpenAICompatibleClient struct {
    baseURL    string
    apiKey     string
    model      string
    maxRetries int
    httpClient *http.Client
    logger     *zap.Logger
}

// NewLLMClientFromConfig membuat LLMClient berdasarkan config provider.
func NewLLMClientFromConfig(cfg *config.Config, logger *zap.Logger) LLMClient {
    baseURL := BaseURLOpenAI
    switch cfg.AIProvider {
    case "groq":
        baseURL = BaseURLGroq
    case "ollama":
        baseURL = BaseURLOllama
    case "gemini":
        baseURL = BaseURLGemini
    }

    return &OpenAICompatibleClient{
        baseURL:    baseURL,
        apiKey:     cfg.AIAPIKey,
        model:      cfg.AIModel,
        maxRetries: cfg.AIMaxRetries,
        httpClient: &http.Client{
            Timeout: time.Duration(cfg.AITimeoutSec) * time.Second,
        },
        logger: logger,
    }
}

// NormalizeProducts mengirim batch items ke LLM dan mengembalikan hasil normalisasi.
func (c *OpenAICompatibleClient) NormalizeProducts(ctx context.Context, itemsJSON []byte) ([]NormalizedProduct, error) {
    userPrompt := BuildNormalizeUserPrompt(itemsJSON)

    content, err := c.callWithRetry(ctx, normalizeSystemPrompt, userPrompt, retryNormalizePrompt)
    if err != nil {
        return nil, fmt.Errorf("NormalizeProducts: call LLM: %w", err)
    }

    var result struct {
        NormalizedItems []NormalizedProduct `json:"normalized_items"`
    }
    if err := json.Unmarshal([]byte(content), &result); err != nil {
        return nil, fmt.Errorf("NormalizeProducts: parse response: %w (content: %.200s)", err, content)
    }

    return result.NormalizedItems, nil
}

// SummarizeGroups mengirim groups ke LLM dan mengembalikan summary & recommendations.
func (c *OpenAICompatibleClient) SummarizeGroups(ctx context.Context, groupsJSON []byte, userPrompt string) (*AISummaryResult, error) {
    fullUserPrompt := BuildSummarizeUserPrompt(groupsJSON, userPrompt)

    content, err := c.callWithRetry(ctx, summarizeSystemPrompt, fullUserPrompt, retrySummarizePrompt)
    if err != nil {
        return nil, fmt.Errorf("SummarizeGroups: call LLM: %w", err)
    }

    var result AISummaryResult
    if err := json.Unmarshal([]byte(content), &result); err != nil {
        return nil, fmt.Errorf("SummarizeGroups: parse response: %w (content: %.200s)", err, content)
    }
    if result.SummaryText == "" {
        return nil, fmt.Errorf("SummarizeGroups: empty summary_text in response")
    }

    return &result, nil
}

// callWithRetry memanggil LLM dan melakukan retry jika JSON tidak valid.
// Pada retry, mengganti user prompt dengan instruksi perbaikan JSON.
func (c *OpenAICompatibleClient) callWithRetry(ctx context.Context, systemPrompt, userPrompt, retryPrompt string) (string, error) {
    currentUserPrompt := userPrompt

    for attempt := 0; attempt <= c.maxRetries; attempt++ {
        content, err := c.call(ctx, systemPrompt, currentUserPrompt)
        if err != nil {
            // Rate limit: tunggu sebentar dan retry
            if isRateLimit(err) && attempt < c.maxRetries {
                wait := time.Duration(1<<uint(attempt)) * time.Second
                c.logger.Warn("rate limit hit, backing off",
                    zap.Duration("wait", wait),
                    zap.Int("attempt", attempt+1),
                )
                select {
                case <-time.After(wait):
                case <-ctx.Done():
                    return "", ctx.Err()
                }
                continue
            }
            return "", err
        }

        // Cek apakah output valid JSON
        if json.Valid([]byte(content)) {
            return content, nil
        }

        // JSON tidak valid: retry dengan prompt perbaikan
        if attempt < c.maxRetries {
            c.logger.Warn("LLM returned invalid JSON, retrying",
                zap.Int("attempt", attempt+1),
                zap.String("content_preview", truncate(content, 200)),
            )
            currentUserPrompt = retryPrompt + "

Output sebelumnya:
" + truncate(content, 500)
            continue
        }

        return "", fmt.Errorf("LLM returned invalid JSON after %d attempts", c.maxRetries+1)
    }

    return "", fmt.Errorf("exceeded max retries (%d)", c.maxRetries)
}

// call mengirim satu request ke endpoint /chat/completions.
func (c *OpenAICompatibleClient) call(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
    reqBody := chatRequest{
        Model: c.model,
        Messages: []chatMessage{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: userPrompt},
        },
        Temperature: 0.1,
    }

    // Aktifkan JSON mode untuk provider yang mendukung
    // (OpenAI, Groq). Untuk Ollama/Gemini bisa di-skip atau diabaikan.
    if c.baseURL == BaseURLOpenAI || c.baseURL == BaseURLGroq {
        reqBody.ResponseFormat = map[string]string{"type": "json_object"}
    }

    bodyBytes, err := json.Marshal(reqBody)
    if err != nil {
        return "", fmt.Errorf("marshal request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
    if err != nil {
        return "", fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    if c.apiKey != "" {
        req.Header.Set("Authorization", "Bearer "+c.apiKey)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("http do: %w", err)
    }
    defer resp.Body.Close()

    respBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("read body: %w", err)
    }

    if resp.StatusCode == http.StatusTooManyRequests {
        return "", fmt.Errorf("rate_limit: status 429")
    }
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("unexpected status %d: %.300s", resp.StatusCode, respBytes)
    }

    var chatResp chatResponse
    if err := json.Unmarshal(respBytes, &chatResp); err != nil {
        return "", fmt.Errorf("parse response: %w", err)
    }
    if chatResp.Error != nil {
        return "", fmt.Errorf("LLM error [%s]: %s", chatResp.Error.Type, chatResp.Error.Message)
    }
    if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
        return "", fmt.Errorf("empty choices in response")
    }

    return chatResp.Choices[0].Message.Content, nil
}

// isRateLimit cek apakah error adalah rate limit.
func isRateLimit(err error) bool {
    if err == nil {
        return false
    }
    return len(err.Error()) >= 10 && err.Error()[:10] == "rate_limit"
}

func truncate(s string, n int) string {
    if len(s) <= n {
        return s
    }
    return s[:n] + "..."
}
```

---

## Step 13.4 — Unit Test dengan Mock HTTP

**File baru:** `internal/ai/openai_client_test.go`

Test tanpa hit API nyata, pakai `httptest.NewServer`.

```go
package ai_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "go.uber.org/zap"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/[username]/tokopedia-scraper/internal/ai"
)

// mockLLMServer membuat mock HTTP server yang selalu return JSON tertentu.
func mockLLMServer(t *testing.T, statusCode int, responseBody string) *httptest.Server {
    t.Helper()
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(statusCode)
        w.Write([]byte(responseBody))
    }))
}

// buildTestClient membuat OpenAICompatibleClient yang mengarah ke mock server.
func buildTestClient(t *testing.T, mockURL string) ai.LLMClient {
    t.Helper()
    return ai.NewOpenAICompatibleClientDirect(mockURL, "", "test-model", 1, zap.NewNop())
}

// --- NormalizeProducts Tests ---

func TestNormalizeProducts_Success(t *testing.T) {
    mockResponse := `{
      "choices": [{
        "message": {
          "content": "{"normalized_items":[{"source_product_id":"prod-1","marketplace":"tokopedia","url":"https://tokopedia.com/...","brand":"Apple","model":"iPhone 15","variant":"128GB","category_path":"Electronics > Smartphones","important_specs":["A16 Bionic","6.1-inch OLED"],"canonical_key":"apple-iphone-15-128gb"}]}"
        }
      }]
    }`

    srv := mockLLMServer(t, http.StatusOK, mockResponse)
    defer srv.Close()

    client := buildTestClient(t, srv.URL)
    items := []byte(`{"items":[{"id":"prod-1","name":"Apple iPhone 15 128GB","price":14999000}]}`)

    result, err := client.NormalizeProducts(context.Background(), items)
    require.NoError(t, err)
    require.Len(t, result, 1)
    assert.Equal(t, "Apple", result[0].Brand)
    assert.Equal(t, "iPhone 15", result[0].Model)
    assert.Equal(t, "128GB", result[0].Variant)
    assert.Equal(t, "apple-iphone-15-128gb", result[0].CanonicalKey)
}

func TestNormalizeProducts_InvalidJSONRetry(t *testing.T) {
    callCount := 0
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        w.Header().Set("Content-Type", "application/json")
        if callCount == 1 {
            // First call: return invalid JSON content
            w.Write([]byte(`{"choices":[{"message":{"content":"here are the products: {invalid json}"}}]}`))
            return
        }
        // Second call (retry): return valid JSON
        w.Write([]byte(`{"choices":[{"message":{"content":"{"normalized_items":[]}"}}]}`))
    }))
    defer srv.Close()

    client := buildTestClient(t, srv.URL)
    _, err := client.NormalizeProducts(context.Background(), []byte(`{"items":[]}`))
    require.NoError(t, err)
    assert.Equal(t, 2, callCount, "should retry once on invalid JSON")
}

func TestNormalizeProducts_RateLimit(t *testing.T) {
    srv := mockLLMServer(t, http.StatusTooManyRequests, `{"error":{"type":"rate_limit_exceeded","message":"rate limit"}}`)
    defer srv.Close()

    client := buildTestClient(t, srv.URL)
    _, err := client.NormalizeProducts(context.Background(), []byte(`{"items":[]}`))
    require.Error(t, err)
}

// --- SummarizeGroups Tests ---

func TestSummarizeGroups_Success(t *testing.T) {
    mockContent := `{"summary_text":"Harga iPhone 15 berkisar 14-16 juta. Beli dari toko resmi untuk garansi.","recommended_items":[{"group_id":"apple-iphone-15-128gb","product_id":"prod-1","reason":"Harga terbaik dari toko resmi dengan rating tinggi."}]}`

    mockResponse := `{"choices":[{"message":{"content":"` + mockContent + `"}}]}`
    srv := mockLLMServer(t, http.StatusOK, mockResponse)
    defer srv.Close()

    client := buildTestClient(t, srv.URL)
    groups := []byte(`{"groups":[]}`)

    result, err := client.SummarizeGroups(context.Background(), groups, "")
    require.NoError(t, err)
    assert.NotEmpty(t, result.SummaryText)
    assert.Len(t, result.RecommendedItems, 1)
    assert.Equal(t, "apple-iphone-15-128gb", result.RecommendedItems[0].GroupID)
}

// --- Prompt Tests (pure functions, no HTTP) ---

func TestBuildNormalizeUserPrompt_ContainsItems(t *testing.T) {
    items := []byte(`{"items":[{"id":"1","name":"Test Product"}]}`)
    prompt := ai.BuildNormalizeUserPrompt(items)
    assert.Contains(t, prompt, "Test Product")
    assert.Contains(t, prompt, "normalized_items")
}

func TestBuildSummarizeUserPrompt_DefaultPrompt(t *testing.T) {
    groups := []byte(`{"groups":[]}`)
    prompt := ai.BuildSummarizeUserPrompt(groups, "")
    assert.Contains(t, prompt, "summary_text")
    assert.Contains(t, prompt, "recommended_items")
}
```

> Kamu perlu menambahkan constructor `NewOpenAICompatibleClientDirect` di `openai_client.go` untuk keperluan testing yang menerima baseURL langsung (tanpa config):

```go
// NewOpenAICompatibleClientDirect adalah constructor untuk testing.
func NewOpenAICompatibleClientDirect(baseURL, apiKey, model string, maxRetries int, logger *zap.Logger) LLMClient {
    return &OpenAICompatibleClient{
        baseURL:    baseURL,
        apiKey:     apiKey,
        model:      model,
        maxRetries: maxRetries,
        httpClient: &http.Client{Timeout: 10 * time.Second},
        logger:     logger,
    }
}
```

---

## Step 13.5 — Update Entry Points

Ganti `DummyClient` di handler dengan client nyata berdasarkan config.

**File:** `cmd/api/main.go` dan `cmd/worker/main.go`

Di kedua file ini, setelah load config, tambahkan:

```go
llmClient := ai.NewLLMClientFromConfig(cfg, logger)
```

Lalu inject `llmClient` ke handler dan worker yang membutuhkannya.

**File:** `internal/run/handler.go`

Update struct `Handler` untuk menyimpan `LLMClient`:

```go
type Handler struct {
    repo      Repository
    queue     *queue.Client
    logger    *zap.Logger
    llmClient ai.LLMClient   // tambahkan ini
}

func NewHandler(repo Repository, q *queue.Client, logger *zap.Logger, llmClient ai.LLMClient) *Handler {
    return &Handler{repo: repo, queue: q, logger: logger, llmClient: llmClient}
}
```

Update method `NormalizeRun` dan `GenerateAISummary` untuk menggunakan `h.llmClient` alih-alih `ai.NewDummyClient()`:

```go
// Sebelum (Phase 10/11):
llmClient := ai.NewDummyClient()

// Sesudah (Phase 13):
// h.llmClient sudah diinject di constructor, tinggal pakai:
groups, err := ai.NormalizeRun(ctx, h.repo, id, h.llmClient)
```

---

## Step 13.6 — Update `.env.example`

Tambahkan section AI di `.env.example`:

```env
# AI Provider Configuration
# Options: openai | groq | ollama | gemini
AI_PROVIDER=openai

# API Key (kosongkan untuk Ollama)
AI_API_KEY=sk-...

# Model yang digunakan per provider:
#   openai  → gpt-4.1-mini, o4-mini, gpt-4.1
#   groq    → llama-3.3-70b-versatile, mixtral-8x7b-32768
#   ollama  → llama3.2, qwen2.5, gemma3
#   gemini  → gemini-2.0-flash, gemini-1.5-pro
AI_MODEL=gpt-4.1-mini

# Timeout per request ke LLM (detik)
AI_TIMEOUT_SEC=30

# Jumlah retry jika JSON invalid atau rate limit
AI_MAX_RETRIES=2
```

---

## Step 13.7 — Verification

1. `go build ./...` dan `go test ./internal/ai/... -v` harus sukses.
2. Isi `.env` dengan salah satu provider nyata:

```env
# Contoh pakai Groq (gratis, daftar di console.groq.com)
AI_PROVIDER=groq
AI_API_KEY=gsk_...
AI_MODEL=llama-3.3-70b-versatile

# Atau pakai Ollama (lokal, gratis, install ollama dulu)
AI_PROVIDER=ollama
AI_API_KEY=
AI_MODEL=llama3.2
```

3. Pastikan ada run dengan status SUCCEEDED. Lalu jalankan:

```bash
# Normalisasi
curl -X POST http://localhost:8080/v1/runs/<RUN_ID>/normalize | jq .

# Lihat groups
curl http://localhost:8080/v1/runs/<RUN_ID>/normalized | jq .[0]

# Generate summary
curl -X POST http://localhost:8080/v1/runs/<RUN_ID>/ai-summary   -H "Content-Type: application/json"   -d '"prompt": "Rekomendasi laptop gaming terbaik budget 15-20 juta"}' | jq .

# Lihat summary
curl http://localhost:8080/v1/runs/<RUN_ID>/ai-summary | jq .
```

4. Verifikasi hasil:
   - `normalized` harus return array `ProductGroup` dengan `brand`, `model`, dan `canonical_key` yang benar.
   - `ai-summary` harus return `summary_text` dalam bahasa Indonesia dan `recommended_items` yang merujuk ke `group_id`/`product_id` yang ada di data.

Update checklist di AGENTS.md: Phase 13 ✅

---

## Provider Quick-Start Reference

| Provider | Daftar | Base URL | Model Rekomendasi |
|---|---|---|---|
| OpenAI | platform.openai.com | `https://api.openai.com/v1` | `gpt-4.1-mini` |
| Groq | console.groq.com | `https://api.groq.com/openai/v1` | `llama-3.3-70b-versatile` |
| Ollama | ollama.com (local) | `http://localhost:11434/v1` | `llama3.2` / `qwen2.5` |
| Gemini | aistudio.google.com | `https://generativelanguage.googleapis.com/v1beta/openai` | `gemini-2.0-flash` |
