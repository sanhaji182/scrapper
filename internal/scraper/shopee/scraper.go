package shopee

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sonick/tokopedia-scraper/internal/proxy"
	"github.com/sonick/tokopedia-scraper/internal/scraper"
)

const MarketplaceName = "shopee"

const defaultLimit = 60

type Scraper struct {
	client       *http.Client
	logger       *zap.Logger
	endpoint     string
	cookieHeader string
	sessionURL   string
	sessionMu    sync.Mutex
	sessionReady bool
}

type Option func(*Scraper)

func New(timeoutSec int, proxyMgr *proxy.Manager, logger *zap.Logger, options ...Option) *Scraper {
	transport := &http.Transport{}
	if proxyMgr != nil && proxyMgr.Len() > 0 {
		transport.Proxy = proxyMgr.ProxyFunc()
	}
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second, Transport: transport}
	attachCookieJar(client)
	scraper := &Scraper{
		client:     client,
		logger:     logger,
		endpoint:   "https://shopee.co.id/api/v4/search/search_items",
		sessionURL: "https://shopee.co.id/",
	}
	for _, option := range options {
		option(scraper)
	}
	return scraper
}

func NewWithClient(client *http.Client, endpoint string, logger *zap.Logger, options ...Option) *Scraper {
	attachCookieJar(client)
	scraper := &Scraper{client: client, endpoint: endpoint, sessionURL: sessionURLFromEndpoint(endpoint), logger: logger}
	for _, option := range options {
		option(scraper)
	}
	return scraper
}

func WithCookieHeader(cookieHeader string) Option {
	return func(s *Scraper) {
		s.cookieHeader = strings.TrimSpace(cookieHeader)
	}
}

func (s *Scraper) Name() string {
	return MarketplaceName
}

func (s *Scraper) Search(ctx context.Context, opts scraper.SearchOptions) ([]scraper.Product, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("shopee.Search validate: %w", err)
	}
	products := make([]scraper.Product, 0, opts.MaxItems)
	page := 0
	for len(products) < opts.MaxItems {
		items, err := s.fetchPage(ctx, opts, page)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		products = append(products, items...)
		if len(items) < defaultLimit {
			break
		}
		page++
	}
	products = scraper.FilterRelevantProducts(opts.Keyword, products)
	if len(products) > opts.MaxItems {
		products = products[:opts.MaxItems]
	}
	return products, nil
}

func (s *Scraper) fetchPage(ctx context.Context, opts scraper.SearchOptions, page int) ([]scraper.Product, error) {
	if s.cookieHeader == "" {
		if err := s.ensureSession(ctx, false); err != nil {
			return nil, fmt.Errorf("shopee.fetchPage session: %w", err)
		}
	}

	body, statusCode, err := s.doRequest(ctx, opts, page)
	if err != nil {
		return nil, fmt.Errorf("shopee.fetchPage request: %w", err)
	}
	if statusCode == http.StatusTooManyRequests || statusCode == http.StatusForbidden {
		s.logger.Warn("shopee api blocked request, trying seo html fallback", zap.Int("status", statusCode), zap.Bool("custom_cookie", s.cookieHeader != ""))
		seoProducts, seoErr := s.fetchSEOPage(ctx, opts)
		if seoErr == nil && len(seoProducts) > 0 {
			return seoProducts, nil
		}
		if s.cookieHeader != "" {
			return nil, fmt.Errorf("shopee.fetchPage status %d with SHOPEE_COOKIE_HEADER; cookie may be expired or IP/proxy mismatch; seo fallback: %w", statusCode, seoErr)
		}
		if sessionErr := s.ensureSession(ctx, true); sessionErr != nil {
			return nil, fmt.Errorf("shopee.fetchPage refresh session after status %d: %w", statusCode, sessionErr)
		}
		body, statusCode, err = s.doRequest(ctx, opts, page)
		if err != nil {
			return nil, fmt.Errorf("shopee.fetchPage retry request: %w", err)
		}
	}
	if statusCode == http.StatusTooManyRequests || statusCode == http.StatusForbidden {
		seoProducts, seoErr := s.fetchSEOPage(ctx, opts)
		if seoErr == nil && len(seoProducts) > 0 {
			return seoProducts, nil
		}
		return nil, fmt.Errorf("shopee.fetchPage status %d after anonymous session refresh; proxy may be required; seo fallback: %w", statusCode, seoErr)
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("shopee.fetchPage status %d", statusCode)
	}
	products, err := parseProducts(body, opts.Keyword)
	if err != nil {
		return nil, err
	}
	return products, nil
}

func (s *Scraper) fetchSEOPage(ctx context.Context, opts scraper.SearchOptions) ([]scraper.Product, error) {
	searchURL := strings.TrimRight(s.sessionURL, "/") + "/search?keyword=" + url.QueryEscape(opts.Keyword)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("shopee.fetchSEOPage create request: %w", err)
	}
	applySEOHeaders(request)
	if s.cookieHeader != "" {
		request.Header.Set("Cookie", s.cookieHeader)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("shopee.fetchSEOPage request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("shopee.fetchSEOPage read body: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("shopee.fetchSEOPage status %d", response.StatusCode)
	}
	products := parseSEOProducts(body, opts.Keyword)
	if len(products) == 0 {
		return nil, fmt.Errorf("shopee.fetchSEOPage no products found")
	}
	if len(products) > opts.MaxItems {
		products = products[:opts.MaxItems]
	}
	return products, nil
}

func applySEOHeaders(request *http.Request) {
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")
	request.Header.Set("User-Agent", "facebookexternalhit/1.1")
}

func (s *Scraper) ensureSession(ctx context.Context, force bool) error {
	s.sessionMu.Lock()
	if s.sessionReady && !force {
		s.sessionMu.Unlock()
		return nil
	}
	s.sessionMu.Unlock()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.sessionURL, nil)
	if err != nil {
		return fmt.Errorf("shopee.ensureSession create request: %w", err)
	}
	applyBrowserHeaders(request, "https://shopee.co.id/")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	request.Header.Set("Sec-Fetch-Dest", "document")
	request.Header.Set("Sec-Fetch-Mode", "navigate")
	request.Header.Set("Sec-Fetch-Site", "none")
	request.Header.Set("Upgrade-Insecure-Requests", "1")

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("shopee.ensureSession request: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode == http.StatusForbidden {
		return fmt.Errorf("shopee.ensureSession status %d", response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return fmt.Errorf("shopee.ensureSession status %d", response.StatusCode)
	}

	s.sessionMu.Lock()
	s.sessionReady = true
	s.sessionMu.Unlock()
	return nil
}

func applyBrowserHeaders(request *http.Request, referer string) {
	request.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Pragma", "no-cache")
	request.Header.Set("Referer", referer)
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	request.Header.Set("Sec-Fetch-User", "?1")
}

func attachCookieJar(client *http.Client) {
	if client == nil || client.Jar != nil {
		return
	}
	jar, err := cookiejar.New(nil)
	if err == nil {
		client.Jar = jar
	}
}

func sessionURLFromEndpoint(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "https://shopee.co.id/"
	}
	return parsed.Scheme + "://" + parsed.Host + "/"
}

func (s *Scraper) doRequest(ctx context.Context, opts scraper.SearchOptions, page int) ([]byte, int, error) {
	endpoint, err := url.Parse(s.endpoint)
	if err != nil {
		return nil, 0, fmt.Errorf("shopee.doRequest parse url: %w", err)
	}
	query := endpoint.Query()
	query.Set("by", sortBy(opts.SortBy))
	query.Set("keyword", opts.Keyword)
	query.Set("limit", strconv.Itoa(defaultLimit))
	query.Set("newest", strconv.Itoa(page*defaultLimit))
	query.Set("order", sortOrder(opts.SortBy))
	query.Set("page_type", "search")
	query.Set("scenario", "PAGE_GLOBAL_SEARCH")
	query.Set("version", "2")
	if opts.MinPrice > 0 {
		query.Set("price_min", strconv.FormatInt(opts.MinPrice, 10))
	}
	if opts.MaxPrice > 0 {
		query.Set("price_max", strconv.FormatInt(opts.MaxPrice, 10))
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("shopee.doRequest create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	applyBrowserHeaders(req, "https://shopee.co.id/search?keyword="+url.QueryEscape(opts.Keyword))
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if s.cookieHeader != "" {
		req.Header.Set("Cookie", s.cookieHeader)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("shopee.doRequest: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("shopee.doRequest read body: %w", err)
	}
	return body, resp.StatusCode, nil
}

func sortBy(sort string) string {
	switch sort {
	case scraper.SortByPriceAsc, scraper.SortByPriceDesc:
		return "price"
	case scraper.SortByLatest:
		return "ctime"
	default:
		return "relevancy"
	}
}

func sortOrder(sort string) string {
	if sort == scraper.SortByPriceAsc {
		return "asc"
	}
	return "desc"
}

func (s *Scraper) SearchWithCookie(ctx context.Context, opts scraper.SearchOptions, cookieHeader string) ([]scraper.Product, error) {
	clone := *s
	clone.cookieHeader = strings.TrimSpace(cookieHeader)
	return clone.Search(ctx, opts)
}
