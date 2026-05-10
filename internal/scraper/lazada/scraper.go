package lazada

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/proxy"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

const (
	MarketplaceName     = "lazada"
	defaultEndpoint     = "https://www.lazada.co.id/catalog/"
	defaultItemsPerPage = 40
	defaultTimeoutSec   = 15
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0",
}

type Scraper struct {
	client       *http.Client
	logger       *zap.Logger
	endpoint     string
	cookieHeader string
	delay        func(context.Context, time.Duration) error
	pageDelay    func() time.Duration
}

type Option func(*Scraper)

func New(timeoutSec int, proxyMgr *proxy.Manager, logger *zap.Logger, options ...Option) *Scraper {
	if timeoutSec <= 0 {
		timeoutSec = defaultTimeoutSec
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	transport := &http.Transport{}
	if proxyMgr != nil && proxyMgr.Len() > 0 {
		transport.Proxy = proxyMgr.ProxyFunc()
	}
	s := &Scraper{
		client:    &http.Client{Timeout: time.Duration(timeoutSec) * time.Second, Transport: transport},
		logger:    logger,
		endpoint:  defaultEndpoint,
		delay:     sleepDelay,
		pageDelay: randomPageDelay,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func NewWithClient(client *http.Client, endpoint string, logger *zap.Logger, options ...Option) *Scraper {
	if logger == nil {
		logger = zap.NewNop()
	}
	s := &Scraper{client: client, endpoint: endpoint, logger: logger, delay: sleepDelay, pageDelay: randomPageDelay}
	for _, option := range options {
		option(s)
	}
	return s
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

func (s *Scraper) Name() string {
	return MarketplaceName
}

func (s *Scraper) SearchWithCookie(ctx context.Context, opts scraper.SearchOptions, cookieHeader string) ([]scraper.Product, error) {
	clone := *s
	clone.cookieHeader = strings.TrimSpace(cookieHeader)
	return clone.Search(ctx, opts)
}

func (s *Scraper) Search(ctx context.Context, opts scraper.SearchOptions) ([]scraper.Product, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("lazada.Search validate: %w", err)
	}

	products := make([]scraper.Product, 0, opts.MaxItems)
	for page := 1; len(products) < opts.MaxItems; page++ {
		items, err := s.fetchPage(ctx, opts, page)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		products = append(products, scraper.FilterRelevantProducts(opts.Keyword, items)...)
		if len(items) < defaultItemsPerPage || len(products) >= opts.MaxItems {
			break
		}
		if err := s.delay(ctx, s.pageDelay()); err != nil {
			return trimProducts(products, opts.MaxItems), fmt.Errorf("lazada.Search page delay: %w", err)
		}
	}
	return trimProducts(scraper.FilterRelevantProducts(opts.Keyword, products), opts.MaxItems), nil
}

func (s *Scraper) fetchPage(ctx context.Context, opts scraper.SearchOptions, page int) ([]scraper.Product, error) {
	requestURL, err := s.buildURL(opts, page)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("lazada.fetchPage create request: %w", err)
	}
	applyHeaders(request, opts.Keyword)
	if s.cookieHeader != "" {
		request.Header.Set("Cookie", s.cookieHeader)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("lazada.fetchPage request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("lazada.fetchPage read body: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("lazada.fetchPage status %d", response.StatusCode)
	}
	products, err := parseProducts(body, opts.Keyword)
	if err != nil {
		return nil, err
	}
	return products, nil
}

func (s *Scraper) buildURL(opts scraper.SearchOptions, page int) (string, error) {
	parsed, err := url.Parse(s.endpoint)
	if err != nil {
		return "", fmt.Errorf("lazada.buildURL parse endpoint: %w", err)
	}
	query := parsed.Query()
	query.Set("ajax", "true")
	query.Set("q", opts.Keyword)
	query.Set("page", strconv.Itoa(page))
	sort := mapSort(opts.SortBy)
	if sort != "" {
		query.Set("sort", sort)
	}
	if opts.MinPrice > 0 || opts.MaxPrice > 0 {
		query.Set("price", priceRange(opts.MinPrice, opts.MaxPrice))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func mapSort(sortBy string) string {
	switch sortBy {
	case scraper.SortByPriceAsc:
		return "priceasc"
	case scraper.SortByPriceDesc:
		return "pricedesc"
	case scraper.SortByLatest:
		return "newest"
	default:
		return ""
	}
}

func priceRange(minPrice, maxPrice int64) string {
	min := ""
	max := ""
	if minPrice > 0 {
		min = strconv.FormatInt(minPrice, 10)
	}
	if maxPrice > 0 {
		max = strconv.FormatInt(maxPrice, 10)
	}
	return min + "-" + max
}

func applyHeaders(request *http.Request, keyword string) {
	request.Header.Set("Accept", "application/json, text/plain, */*")
	request.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8")
	request.Header.Set("Referer", "https://www.lazada.co.id/catalog/?q="+url.QueryEscape(keyword))
	request.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
}

func trimProducts(products []scraper.Product, maxItems int) []scraper.Product {
	if len(products) > maxItems {
		return products[:maxItems]
	}
	return products
}

func randomPageDelay() time.Duration {
	return time.Duration(1000+rand.Intn(2001)) * time.Millisecond
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
