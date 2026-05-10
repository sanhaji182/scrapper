package proxy

import (
	"math/rand"
	"net/http"
	"net/url"
	"sync"
)

type Manager struct {
	proxies []string
	mu      sync.Mutex
	current int
}

func NewManager(proxies []string) *Manager {
	return &Manager{proxies: proxies}
}

func (m *Manager) GetProxy() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.proxies) == 0 {
		return ""
	}
	p := m.proxies[m.current%len(m.proxies)]
	m.current++
	return p
}

func (m *Manager) GetRandom() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.proxies) == 0 {
		return ""
	}
	return m.proxies[rand.Intn(len(m.proxies))]
}

func (m *Manager) Len() int {
	return len(m.proxies)
}

func (m *Manager) ProxyFunc() func(*http.Request) (*url.URL, error) {
	return func(*http.Request) (*url.URL, error) {
		proxyURL := m.GetProxy()
		if proxyURL == "" {
			return nil, nil
		}
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		return parsed, nil
	}
}
