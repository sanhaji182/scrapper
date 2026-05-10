package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/config"
)

type Settings struct {
	Provider   string `json:"provider"`
	APIKey     string `json:"api_key,omitempty"`
	Model      string `json:"model"`
	TimeoutSec int    `json:"timeout_sec"`
	MaxRetries int    `json:"max_retries"`
	Configured bool   `json:"configured"`
}

type RuntimeClient struct {
	mu       sync.RWMutex
	settings Settings
	client   LLMClient
	logger   *zap.Logger
}

func NewRuntimeClientFromConfig(cfg *config.Config, logger *zap.Logger) *RuntimeClient {
	if logger == nil {
		logger = zap.NewNop()
	}
	settings := Settings{
		Provider:   cfg.AIProvider,
		APIKey:     cfg.AIAPIKey,
		Model:      cfg.AIModel,
		TimeoutSec: cfg.AITimeoutSec,
		MaxRetries: cfg.AIMaxRetries,
	}
	r := &RuntimeClient{logger: logger}
	r.Update(settings)
	return r
}

func (r *RuntimeClient) Settings(maskKey bool) Settings {
	r.mu.RLock()
	defer r.mu.RUnlock()
	settings := r.settings
	settings.Configured = settings.Provider == "ollama" || settings.APIKey != ""
	if maskKey && settings.APIKey != "" {
		settings.APIKey = maskAPIKey(settings.APIKey)
	}
	return settings
}

func (r *RuntimeClient) Update(settings Settings) {
	settings.Provider = normalizeProvider(settings.Provider)
	if settings.Model == "" {
		settings.Model = defaultModel(settings.Provider)
	}
	if settings.TimeoutSec <= 0 {
		settings.TimeoutSec = 30
	}
	if settings.MaxRetries < 0 {
		settings.MaxRetries = 0
	}

	cfg := &config.Config{
		AIProvider:   settings.Provider,
		AIAPIKey:     settings.APIKey,
		AIModel:      settings.Model,
		AITimeoutSec: settings.TimeoutSec,
		AIMaxRetries: settings.MaxRetries,
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings = settings
	r.client = NewLLMClientFromConfig(cfg, r.logger)
}

func (r *RuntimeClient) NormalizeProducts(ctx context.Context, itemsJSON []byte) ([]NormalizedProduct, error) {
	client, err := r.currentClient()
	if err != nil {
		return nil, err
	}
	return client.NormalizeProducts(ctx, itemsJSON)
}

func (r *RuntimeClient) SummarizeGroups(ctx context.Context, groupsJSON []byte, userPrompt string) (*AISummaryResult, error) {
	client, err := r.currentClient()
	if err != nil {
		return nil, err
	}
	return client.SummarizeGroups(ctx, groupsJSON, userPrompt)
}

func (r *RuntimeClient) currentClient() (LLMClient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.settings.Provider != "ollama" && r.settings.APIKey == "" {
		return nil, fmt.Errorf("AI API key is required for provider %s", r.settings.Provider)
	}
	return r.client, nil
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "groq", "ollama", "gemini":
		return strings.ToLower(strings.TrimSpace(provider))
	default:
		return "openai"
	}
}

func defaultModel(provider string) string {
	switch provider {
	case "groq":
		return "llama-3.3-70b-versatile"
	case "ollama":
		return "llama3.2"
	case "gemini":
		return "gemini-2.0-flash"
	default:
		return "gpt-4.1-mini"
	}
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "••••"
	}
	return key[:4] + "••••" + key[len(key)-4:]
}
