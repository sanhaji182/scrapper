package config

import "strings"

type MarketplaceSettings struct {
	ShopeeCookieHeader string `json:"shopee_cookie_header,omitempty"`
	LazadaCookieHeader string `json:"lazada_cookie_header,omitempty"`
}

type MarketplaceRuntimeSettings struct {
	settings MarketplaceSettings
}

func NewMarketplaceRuntimeSettings(cfg *Config) *MarketplaceRuntimeSettings {
	return &MarketplaceRuntimeSettings{settings: MarketplaceSettings{ShopeeCookieHeader: cfg.ShopeeCookieHeader}}
}

func (r *MarketplaceRuntimeSettings) Settings(mask bool) MarketplaceSettings {
	settings := r.settings
	if mask {
		settings.ShopeeCookieHeader = maskSecret(settings.ShopeeCookieHeader)
		settings.LazadaCookieHeader = maskSecret(settings.LazadaCookieHeader)
	}
	return settings
}

func (r *MarketplaceRuntimeSettings) Update(settings MarketplaceSettings) {
	if settings.ShopeeCookieHeader != "" && !strings.Contains(settings.ShopeeCookieHeader, "•") {
		r.settings.ShopeeCookieHeader = strings.TrimSpace(settings.ShopeeCookieHeader)
	}
	if settings.LazadaCookieHeader != "" && !strings.Contains(settings.LazadaCookieHeader, "•") {
		r.settings.LazadaCookieHeader = strings.TrimSpace(settings.LazadaCookieHeader)
	}
}

func (r *MarketplaceRuntimeSettings) CookieFor(marketplace string) string {
	switch strings.ToLower(strings.TrimSpace(marketplace)) {
	case "shopee":
		return r.settings.ShopeeCookieHeader
	case "lazada":
		return r.settings.LazadaCookieHeader
	default:
		return ""
	}
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 12 {
		return "••••"
	}
	return value[:6] + "••••" + value[len(value)-6:]
}
