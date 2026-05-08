package run

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

type QueueClient interface {
	EnqueueScrapeJob(ctx context.Context, runID string, opts scraper.SearchOptions) error
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
