package run

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/ai"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

type QueueClient interface {
	EnqueueScrapeJob(ctx context.Context, runID string, opts scraper.SearchOptions) error
}

type aiSummaryRequest struct {
	Prompt string `json:"prompt"`
}

type Handler struct {
	repo   Repository
	queue  QueueClient
	logger *zap.Logger
}

func NewHandler(repo Repository, q QueueClient, logger *zap.Logger) *Handler {
	return &Handler{repo: repo, queue: q, logger: logger}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	v1 := e.Group("/v1")
	v1.POST("/scrape/tokopedia/search", h.SubmitTokopediaSearch)
	v1.GET("/runs", h.ListRuns)
	v1.GET("/runs/:id", h.GetRun)
	v1.DELETE("/runs/:id", h.DeleteRun)
	v1.POST("/runs/:id/normalize", h.NormalizeRun)
	v1.GET("/runs/:id/normalized", h.GetNormalizedRun)
	v1.POST("/runs/:id/ai-summary", h.GenerateAISummary)
	v1.GET("/runs/:id/ai-summary", h.GetAISummary)
}

func (h *Handler) SubmitTokopediaSearch(c echo.Context) error {
	var opts scraper.SearchOptions
	if err := c.Bind(&opts); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := opts.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	run, err := h.repo.Create(c.Request().Context(), "tokopedia", opts)
	if err != nil {
		h.logger.Error("failed to create run", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create job")
	}

	if err := h.queue.EnqueueScrapeJob(c.Request().Context(), run.ID, opts); err != nil {
		h.logger.Error("failed to enqueue job", zap.String("run_id", run.ID), zap.Error(err))
		_ = h.repo.UpdateStatus(c.Request().Context(), run.ID, StatusFailed, "failed to enqueue")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to enqueue job")
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"run_id":  run.ID,
		"status":  run.Status,
		"message": "Job submitted successfully",
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

	llmClient := ai.NewDummyClient()
	groups, err := ai.NormalizeRun(ctx, id, string(r.Status), r.ResultJSON, llmClient, h.repo.SaveNormalized)
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

	llmClient := ai.NewDummyClient()
	res, err := ai.SummarizeRun(ctx, id, r.NormalizedJSON, llmClient, req.Prompt, h.repo.SaveAISummary)
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
