package tokopedia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/proxy"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

const (
	Name               = "tokopedia"
	defaultEndpoint    = "https://gql.tokopedia.com/graphql/SearchProductV5Query"
	defaultRowsPerPage = 28
	defaultTimeoutSec  = 15
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 Edg/123.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.6367.82 Mobile Safari/537.36",
}

type Scraper struct {
	client       *http.Client
	proxyManager *proxy.Manager
	logger       *zap.Logger
	endpoint     string
	timeout      time.Duration
	delay        func(context.Context, time.Duration) error
	pageDelay    func() time.Duration
	retryDelay   time.Duration
}

type Option func(*Scraper)

func New(requestTimeoutSec int, proxyManager *proxy.Manager, logger *zap.Logger, options ...Option) *Scraper {
	if requestTimeoutSec <= 0 {
		requestTimeoutSec = defaultTimeoutSec
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	s := &Scraper{
		client:       &http.Client{},
		proxyManager: proxyManager,
		logger:       logger,
		endpoint:     defaultEndpoint,
		timeout:      time.Duration(requestTimeoutSec) * time.Second,
		delay:        sleepDelay,
		pageDelay:    randomPageDelay,
		retryDelay:   10 * time.Second,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func WithEndpoint(endpoint string) Option {
	return func(s *Scraper) {
		s.endpoint = endpoint
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(s *Scraper) {
		if client != nil {
			s.client = client
		}
	}
}

func WithDelay(delay func(context.Context, time.Duration) error) Option {
	return func(s *Scraper) {
		if delay != nil {
			s.delay = delay
		}
	}
}

func WithPageDelay(pageDelay func() time.Duration) Option {
	return func(s *Scraper) {
		if pageDelay != nil {
			s.pageDelay = pageDelay
		}
	}
}

func WithRetryDelay(delay time.Duration) Option {
	return func(s *Scraper) {
		s.retryDelay = delay
	}
}

func (s *Scraper) Name() string {
	return Name
}

func (s *Scraper) Search(ctx context.Context, opts scraper.SearchOptions) ([]scraper.Product, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("tokopedia.Search validate options: %w", err)
	}

	var products []scraper.Product
	invalidResponses := 0
	for page := 1; len(products) < opts.MaxItems; page++ {
		pageProducts, hasNextPage, err := s.fetchPage(ctx, opts, page)
		if err != nil {
			invalidResponses++
			if invalidResponses >= 2 {
				s.logger.Warn("tokopedia response invalid twice", zap.Error(err))
				return trimProducts(products, opts.MaxItems), nil
			}
			continue
		}
		invalidResponses = 0
		if len(pageProducts) == 0 {
			return trimProducts(products, opts.MaxItems), nil
		}
		products = append(products, pageProducts...)
		if len(products) >= opts.MaxItems || !hasNextPage {
			break
		}
		if err := s.delay(ctx, s.pageDelay()); err != nil {
			return trimProducts(products, opts.MaxItems), fmt.Errorf("tokopedia.Search page delay: %w", err)
		}
	}

	return trimProducts(products, opts.MaxItems), nil
}

func (s *Scraper) fetchPage(ctx context.Context, opts scraper.SearchOptions, page int) ([]scraper.Product, bool, error) {
	body, statusCode, err := s.doRequest(ctx, opts, page)
	if err != nil {
		return nil, false, err
	}
	if statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable {
		if err := s.delay(ctx, s.retryDelay); err != nil {
			return nil, false, fmt.Errorf("tokopedia.fetchPage retry delay: %w", err)
		}
		body, statusCode, err = s.doRequest(ctx, opts, page)
		if err != nil {
			return nil, false, err
		}
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, false, fmt.Errorf("tokopedia.fetchPage status %d", statusCode)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, false, fmt.Errorf("tokopedia.fetchPage empty response")
	}
	products, hasNextPage, err := parseProducts(body)
	if err != nil {
		return nil, false, err
	}
	return products, hasNextPage, nil
}

func (s *Scraper) doRequest(ctx context.Context, opts scraper.SearchOptions, page int) ([]byte, int, error) {
	requestCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	payload, err := json.Marshal(buildPayload(opts, page))
	if err != nil {
		return nil, 0, fmt.Errorf("tokopedia.doRequest marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("tokopedia.doRequest create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	req.Header.Set("Referer", "https://www.tokopedia.com/search?q="+url.QueryEscape(opts.Keyword))
	req.Header.Set("X-Source", "tokopedia-lite")
	req.Header.Set("X-Device", "default_v3")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("tokopedia.doRequest: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("tokopedia.doRequest read body: %w", err)
	}
	return body, resp.StatusCode, nil
}

func buildPayload(opts scraper.SearchOptions, page int) map[string]any {
	params := url.Values{}
	params.Set("device", "desktop")
	params.Set("navsource", "")
	params.Set("ob", sortOption(opts.SortBy))
	params.Set("page", strconv.Itoa(page))
	params.Set("q", opts.Keyword)
	params.Set("related", "true")
	params.Set("rows", strconv.Itoa(defaultRowsPerPage))
	params.Set("safe_search", "false")
	params.Set("scheme", "https")
	params.Set("shipping", "")
	params.Set("source", "search_product")
	params.Set("srp_component_id", "02.01.00.00")
	params.Set("st", "product")
	params.Set("start", strconv.Itoa((page-1)*defaultRowsPerPage))
	params.Set("topads_bucket", "true")
	params.Set("unique_id", "")
	params.Set("user_addressId", "")
	params.Set("user_cityId", "176")
	params.Set("user_districtId", "2274")
	params.Set("user_id", "")
	params.Set("user_lat", "")
	params.Set("user_long", "")
	params.Set("user_postCode", "")
	params.Set("user_warehouseId", "")
	params.Set("variants", "")
	params.Set("warehouses", "")
	if opts.MinPrice > 0 {
		params.Set("pmin", strconv.FormatInt(opts.MinPrice, 10))
	}
	if opts.MaxPrice > 0 {
		params.Set("pmax", strconv.FormatInt(opts.MaxPrice, 10))
	}

	return map[string]any{
		"operationName": "SearchProductV5Query",
		"variables": map[string]string{
			"params": params.Encode(),
		},
		"query": searchProductV5Query,
	}
}

const searchProductV5Query = `query SearchProductV5Query($params: String!) {
  searchProductV5(params: $params) {
    header {
      totalData
      responseCode
      keywordProcess
      keywordIntention
      componentID
      isQuerySafe
      additionalParams
      backendFilters
      meta { dynamicFields }
    }
    data {
      totalDataText
      products {
        oldID: id
        id: id_str_auto_
        name
        url
        applink
        mediaURL { image image300 videoCustom }
        shop { oldID: id id: id_str_auto_ name city url tier }
        badge { id title url }
        price { text number range original discountPercentage }
        freeShipping { url }
        labelGroups { id position title type url styles { key value } }
        category { id name breadcrumb gaKey }
        rating
        wishlist
        ads { id productClickURL productViewURL productWishlistURL tag }
        meta { parentID warehouseID isImageBlurred isPortrait }
      }
    }
  }
}`

func sortOption(sortBy string) string {
	switch sortBy {
	case scraper.SortByPriceAsc:
		return "3"
	case scraper.SortByPriceDesc:
		return "4"
	case scraper.SortByLatest:
		return "9"
	default:
		return "23"
	}
}

func trimProducts(products []scraper.Product, maxItems int) []scraper.Product {
	if len(products) <= maxItems {
		return products
	}
	return products[:maxItems]
}

func sleepDelay(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func randomPageDelay() time.Duration {
	return time.Duration(1500+rand.Intn(1501)) * time.Millisecond
}
