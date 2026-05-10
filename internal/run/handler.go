package run

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/ai"
	"github.com/sonick/tokopedia-scraper/internal/config"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

type QueueClient interface {
	EnqueueScrapeJob(ctx context.Context, runID string, opts scraper.SearchOptions) error
	EnqueueMarketplaceScrapeJob(ctx context.Context, runID, marketplace string, opts scraper.SearchOptions) error
	EnqueueMarketplaceScrapeJobWithCookie(ctx context.Context, runID, marketplace string, opts scraper.SearchOptions, cookieHeader string) error
}

type aiSummaryRequest struct {
	Prompt string `json:"prompt"`
}

type Handler struct {
	repo               Repository
	queue              QueueClient
	logger             *zap.Logger
	llmClient          ai.LLMClient
	aiRuntime          *ai.RuntimeClient
	marketplaceRuntime *config.MarketplaceRuntimeSettings
}

func NewHandler(repo Repository, q QueueClient, logger *zap.Logger, llmClient ai.LLMClient) *Handler {
	return &Handler{repo: repo, queue: q, logger: logger, llmClient: llmClient}
}

func NewHandlerWithAISettings(repo Repository, q QueueClient, logger *zap.Logger, aiRuntime *ai.RuntimeClient) *Handler {
	return &Handler{repo: repo, queue: q, logger: logger, llmClient: aiRuntime, aiRuntime: aiRuntime}
}

func NewHandlerWithRuntimeSettings(repo Repository, q QueueClient, logger *zap.Logger, aiRuntime *ai.RuntimeClient, marketplaceRuntime *config.MarketplaceRuntimeSettings) *Handler {
	return &Handler{repo: repo, queue: q, logger: logger, llmClient: aiRuntime, aiRuntime: aiRuntime, marketplaceRuntime: marketplaceRuntime}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	v1 := e.Group("/v1")
	v1.POST("/scrape/tokopedia/search", h.SubmitTokopediaSearch)
	v1.POST("/scrape/shopee/search", h.SubmitShopeeSearch)
	v1.POST("/scrape/blibli/search", h.SubmitBlibliSearch)
	v1.POST("/scrape/lazada/search", h.SubmitLazadaSearch)
	v1.POST("/scrape/:marketplace/search", h.SubmitMarketplaceSearch)
	v1.GET("/runs", h.ListRuns)
	v1.GET("/runs/:id", h.GetRun)
	v1.DELETE("/runs/:id", h.DeleteRun)
	v1.POST("/runs/:id/normalize", h.NormalizeRun)
	v1.GET("/runs/:id/normalized", h.GetNormalizedRun)
	v1.POST("/runs/:id/ai-summary", h.GenerateAISummary)
	v1.GET("/runs/:id/ai-summary", h.GetAISummary)
	v1.GET("/ai/settings", h.GetAISettings)
	v1.PUT("/ai/settings", h.UpdateAISettings)
	v1.POST("/ai/test", h.TestAISettings)
	v1.GET("/ai/status", h.GetAIStatus)
	v1.GET("/marketplace/settings", h.GetMarketplaceSettings)
	v1.PUT("/marketplace/settings", h.UpdateMarketplaceSettings)
}

func (h *Handler) GetMarketplaceSettings(c echo.Context) error {
	if h.marketplaceRuntime == nil {
		return c.JSON(http.StatusOK, config.MarketplaceSettings{})
	}
	return c.JSON(http.StatusOK, h.marketplaceRuntime.Settings(true))
}

func (h *Handler) UpdateMarketplaceSettings(c echo.Context) error {
	if h.marketplaceRuntime == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "marketplace settings are not available")
	}
	var req config.MarketplaceSettings
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	h.marketplaceRuntime.Update(req)
	return c.JSON(http.StatusOK, h.marketplaceRuntime.Settings(true))
}

func (h *Handler) GetAISettings(c echo.Context) error {
	if h.aiRuntime == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "ai settings are not available")
	}
	return c.JSON(http.StatusOK, h.aiRuntime.Settings(true))
}

func (h *Handler) GetAIStatus(c echo.Context) error {
	if h.aiRuntime == nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"ready":   false,
			"message": "AI runtime belum tersedia.",
		})
	}
	settings := h.aiRuntime.Settings(true)
	message := "AI siap dipakai."
	if !settings.Configured {
		message = "AI belum aktif. Isi API key atau pilih Ollama lokal di Pengaturan AI."
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"ready":    settings.Configured,
		"message":  message,
		"provider": settings.Provider,
		"model":    settings.Model,
	})
}

func (h *Handler) UpdateAISettings(c echo.Context) error {
	if h.aiRuntime == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "ai settings are not available")
	}
	var req ai.Settings
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	current := h.aiRuntime.Settings(false)
	if req.APIKey == "" || strings.Contains(req.APIKey, "•") {
		req.APIKey = current.APIKey
	}
	h.aiRuntime.Update(req)
	return c.JSON(http.StatusOK, h.aiRuntime.Settings(true))
}

func (h *Handler) TestAISettings(c echo.Context) error {
	if h.aiRuntime == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "ai settings are not available")
	}
	_, err := h.aiRuntime.SummarizeGroups(c.Request().Context(), []byte(`{"groups":[]}`), "Jawab ringkas untuk test koneksi. Jangan rekomendasikan produk.")
	if err != nil {
		h.logger.Warn("ai test failed", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) SubmitTokopediaSearch(c echo.Context) error {
	return h.submitSearch(c, "tokopedia")
}

func (h *Handler) SubmitShopeeSearch(c echo.Context) error {
	return h.submitSearch(c, "shopee")
}

func (h *Handler) SubmitBlibliSearch(c echo.Context) error {
	return h.submitSearch(c, "blibli")
}

func (h *Handler) SubmitLazadaSearch(c echo.Context) error {
	return h.submitSearch(c, "lazada")
}

func (h *Handler) SubmitMarketplaceSearch(c echo.Context) error {
	return h.submitSearch(c, c.Param("marketplace"))
}

func (h *Handler) submitSearch(c echo.Context, marketplace string) error {
	marketplace = strings.ToLower(strings.TrimSpace(marketplace))
	if marketplace != "tokopedia" && marketplace != "shopee" && marketplace != "blibli" && marketplace != "lazada" {
		return echo.NewHTTPError(http.StatusBadRequest, "marketplace must be one of: tokopedia, shopee, blibli, lazada")
	}

	var opts scraper.SearchOptions
	if err := c.Bind(&opts); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := opts.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	run, err := h.repo.Create(c.Request().Context(), marketplace, opts)
	if err != nil {
		h.logger.Error("failed to create run", zap.String("marketplace", marketplace), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create job")
	}

	cookieHeader := ""
	if h.marketplaceRuntime != nil {
		cookieHeader = h.marketplaceRuntime.CookieFor(marketplace)
	}
	if err := h.queue.EnqueueMarketplaceScrapeJobWithCookie(c.Request().Context(), run.ID, marketplace, opts, cookieHeader); err != nil {
		h.logger.Error("failed to enqueue job", zap.String("run_id", run.ID), zap.String("marketplace", marketplace), zap.Error(err))
		_ = h.repo.UpdateStatus(c.Request().Context(), run.ID, StatusFailed, "failed to enqueue")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to enqueue job")
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"run_id":      run.ID,
		"status":      run.Status,
		"marketplace": marketplace,
		"message":     "Job submitted successfully",
	})
}

func (h *Handler) GetRun(c echo.Context) error {
	id := c.Param("id")
	run, err := h.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "run not found")
	}

	if run.Status != StatusSucceeded {
		run.ResultJSON = nil
	}
	return c.JSON(http.StatusOK, run)
}

func (h *Handler) ListRuns(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	runs, total, err := h.repo.List(c.Request().Context(), limit, offset)
	if err != nil {
		h.logger.Error("failed to list runs", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch runs")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"runs":   runs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) DeleteRun(c echo.Context) error {
	id := c.Param("id")
	if err := h.repo.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "run not found")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) NormalizeRun(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	r, err := h.repo.GetForNormalization(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "run not found")
	}

	groups, err := ai.NormalizeRun(ctx, id, string(r.Status), r.ResultJSON, h.llmClient, h.repo.SaveNormalized)
	if err != nil {
		h.logger.Error("normalize run failed", zap.String("run_id", id), zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"run_id":    id,
		"groups":    groups,
		"group_cnt": len(groups),
	})
}

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

func (h *Handler) GenerateAISummary(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	var req aiSummaryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	r, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "run not found")
	}

	res, err := ai.SummarizeRun(ctx, id, r.NormalizedJSON, h.llmClient, req.Prompt, h.repo.SaveAISummary)
	if err != nil {
		h.logger.Error("ai summary failed", zap.String("run_id", id), zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, res)
}

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
