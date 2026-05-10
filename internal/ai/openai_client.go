package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/config"
)

const (
	BaseURLOpenAI = "https://api.openai.com/v1"
	BaseURLGroq   = "https://api.groq.com/openai/v1"
	BaseURLOllama = "http://localhost:11434/v1"
	BaseURLGemini = "https://generativelanguage.googleapis.com/v1beta/openai"

	defaultMaxTokens = 4096
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
	Temperature    float64           `json:"temperature"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
}

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

type OpenAICompatibleClient struct {
	baseURL    string
	apiKey     string
	model      string
	maxRetries int
	httpClient *http.Client
	logger     *zap.Logger
}

func NewLLMClientFromConfig(cfg *config.Config, logger *zap.Logger) LLMClient {
	baseURL := BaseURLOpenAI
	switch strings.ToLower(cfg.AIProvider) {
	case "groq":
		baseURL = BaseURLGroq
	case "ollama":
		baseURL = BaseURLOllama
	case "gemini":
		baseURL = BaseURLGemini
	}

	if logger == nil {
		logger = zap.NewNop()
	}
	return &OpenAICompatibleClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     cfg.AIAPIKey,
		model:      cfg.AIModel,
		maxRetries: cfg.AIMaxRetries,
		httpClient: &http.Client{Timeout: time.Duration(cfg.AITimeoutSec) * time.Second},
		logger:     logger,
	}
}

func NewOpenAICompatibleClientDirect(baseURL, apiKey, model string, maxRetries int, logger *zap.Logger) LLMClient {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OpenAICompatibleClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		maxRetries: maxRetries,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
	}
}

func (c *OpenAICompatibleClient) NormalizeProducts(ctx context.Context, itemsJSON []byte) ([]NormalizedProduct, error) {
	var lastErr error
	messages := []chatMessage{
		{Role: "system", Content: normalizeSystemPrompt},
		{Role: "user", Content: BuildNormalizeUserPrompt(itemsJSON)},
	}

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		content, err := c.chatCompletions(ctx, messages)
		if err != nil {
			lastErr = err
		} else {
			items, err := parseNormalizedProducts(content)
			if err == nil {
				return items, nil
			}
			lastErr = err
		}
		messages = append(messages, chatMessage{Role: "user", Content: retryNormalizePrompt})
		c.logger.Warn("retrying normalize request", zap.Int("attempt", attempt+1), zap.Error(lastErr))
	}

	return nil, fmt.Errorf("NormalizeProducts: %w", lastErr)
}

func (c *OpenAICompatibleClient) SummarizeGroups(ctx context.Context, groupsJSON []byte, userPrompt string) (*AISummaryResult, error) {
	var lastErr error
	messages := []chatMessage{
		{Role: "system", Content: summarizeSystemPrompt},
		{Role: "user", Content: BuildSummarizeUserPrompt(groupsJSON, userPrompt)},
	}

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		content, err := c.chatCompletions(ctx, messages)
		if err != nil {
			lastErr = err
		} else {
			result, err := parseSummaryResult(content)
			if err == nil {
				return result, nil
			}
			lastErr = err
		}
		messages = append(messages, chatMessage{Role: "user", Content: retrySummarizePrompt})
		c.logger.Warn("retrying summary request", zap.Int("attempt", attempt+1), zap.Error(lastErr))
	}

	return nil, fmt.Errorf("SummarizeGroups: %w", lastErr)
}

func (c *OpenAICompatibleClient) chatCompletions(ctx context.Context, messages []chatMessage) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:          c.model,
		Messages:       messages,
		ResponseFormat: map[string]string{"type": "json_object"},
		Temperature:    0,
		MaxTokens:      defaultMaxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send chat request: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read chat response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("chat request failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(resBody)))
	}

	var decoded chatResponse
	if err := json.Unmarshal(resBody, &decoded); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	if decoded.Error != nil {
		return "", fmt.Errorf("chat response error: %s", decoded.Error.Message)
	}
	if len(decoded.Choices) == 0 || decoded.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("chat response has no content")
	}

	return decoded.Choices[0].Message.Content, nil
}

func parseNormalizedProducts(content string) ([]NormalizedProduct, error) {
	var wrapper struct {
		Items []NormalizedProduct `json:"normalized_items"`
	}
	if err := json.Unmarshal(extractJSON(content), &wrapper); err != nil {
		return nil, fmt.Errorf("decode normalized products: %w", err)
	}
	if wrapper.Items == nil {
		return nil, fmt.Errorf("decode normalized products: missing normalized_items")
	}
	return wrapper.Items, nil
}

func parseSummaryResult(content string) (*AISummaryResult, error) {
	var result AISummaryResult
	if err := json.Unmarshal(extractJSON(content), &result); err != nil {
		return nil, fmt.Errorf("decode summary result: %w", err)
	}
	if result.SummaryText == "" {
		return nil, fmt.Errorf("decode summary result: missing summary_text")
	}
	if result.RecommendedItems == nil {
		result.RecommendedItems = []RecommendedItem{}
	}
	return &result, nil
}

func extractJSON(content string) []byte {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```JSON")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(strings.TrimSpace(trimmed), "```")
	trimmed = strings.TrimSpace(trimmed)

	start := strings.IndexFunc(trimmed, func(r rune) bool { return r == '{' || r == '[' })
	if start < 0 {
		return []byte(trimmed)
	}
	end := strings.LastIndexFunc(trimmed, func(r rune) bool { return r == '}' || r == ']' })
	if end < start {
		return []byte(strings.TrimFunc(trimmed[start:], unicode.IsSpace))
	}
	return []byte(trimmed[start : end+1])
}
