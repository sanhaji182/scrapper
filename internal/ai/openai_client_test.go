package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
)

func TestOpenAICompatibleClientNormalizeSuccess(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", r.Header.Get("Authorization"))
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Fatalf("unexpected model: %s", req.Model)
		}
		if !strings.Contains(req.Messages[1].Content, `"items"`) {
			t.Fatalf("prompt missing items: %s", req.Messages[1].Content)
		}
		calls.Add(1)

		writeChatResponse(t, w, "```json\n{\"normalized_items\":[{\"source_product_id\":\"p1\",\"marketplace\":\"tokopedia\",\"url\":\"https://example.test/p1\",\"brand\":\"Apple\",\"model\":\"iPhone 15\",\"variant\":\"128GB\",\"category_path\":\"Handphone\",\"important_specs\":[\"128GB\"],\"canonical_key\":\"apple-iphone-15-128gb\"}]}\n```")
	}))
	defer server.Close()

	client := NewOpenAICompatibleClientDirect(server.URL, "test-key", "test-model", 0, zap.NewNop())
	items, err := client.NormalizeProducts(context.Background(), []byte(`{"items":[{"id":"p1"}]}`))
	if err != nil {
		t.Fatalf("NormalizeProducts error: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", calls.Load())
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].CanonicalKey != "apple-iphone-15-128gb" {
		t.Fatalf("unexpected canonical key: %s", items[0].CanonicalKey)
	}
}

func TestOpenAICompatibleClientSummarySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(req.Messages[1].Content, "Prioritaskan official store") {
			t.Fatalf("prompt missing instruction: %s", req.Messages[1].Content)
		}
		if !strings.Contains(req.Messages[1].Content, `"groups"`) {
			t.Fatalf("prompt missing groups: %s", req.Messages[1].Content)
		}

		writeChatResponse(t, w, `{"summary_text":"Harga stabil dan opsi official store menarik.","recommended_items":[{"group_id":"g1","product_id":"p1","reason":"Harga bagus dengan toko resmi."}]}`)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClientDirect(server.URL, "", "test-model", 0, zap.NewNop())
	result, err := client.SummarizeGroups(context.Background(), []byte(`{"groups":[{"group_id":"g1"}]}`), "Prioritaskan official store")
	if err != nil {
		t.Fatalf("SummarizeGroups error: %v", err)
	}
	if result.SummaryText != "Harga stabil dan opsi official store menarik." {
		t.Fatalf("unexpected summary: %s", result.SummaryText)
	}
	if len(result.RecommendedItems) != 1 || result.RecommendedItems[0].ProductID != "p1" {
		t.Fatalf("unexpected recommendations: %+v", result.RecommendedItems)
	}
}

func TestOpenAICompatibleClientRetriesInvalidJSON(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			writeChatResponse(t, w, `not json`)
			return
		}
		writeChatResponse(t, w, `{"normalized_items":[]}`)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClientDirect(server.URL, "", "test-model", 1, zap.NewNop())
	items, err := client.NormalizeProducts(context.Background(), []byte(`{"items":[]}`))
	if err != nil {
		t.Fatalf("NormalizeProducts error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty items, got %d", len(items))
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", calls.Load())
	}
}

func TestPromptBuildersContainSchemaAndData(t *testing.T) {
	normalizePrompt := BuildNormalizeUserPrompt([]byte(`{"items":[{"id":"p1"}]}`))
	if !strings.Contains(normalizePrompt, "normalized_items") || !strings.Contains(normalizePrompt, `"items"`) {
		t.Fatalf("normalize prompt missing schema/data: %s", normalizePrompt)
	}

	summaryPrompt := BuildSummarizeUserPrompt([]byte(`{"groups":[{"group_id":"g1"}]}`), "Instruksi khusus")
	if !strings.HasPrefix(summaryPrompt, "Instruksi khusus") {
		t.Fatalf("summary prompt missing instruction prefix: %s", summaryPrompt)
	}
	if !strings.Contains(summaryPrompt, "summary_text") || !strings.Contains(summaryPrompt, `"groups"`) {
		t.Fatalf("summary prompt missing schema/data: %s", summaryPrompt)
	}
}

func writeChatResponse(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	response := chatResponse{}
	response.Choices = append(response.Choices, struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}{})
	response.Choices[0].Message.Content = content
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
