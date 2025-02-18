package provider

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/21state/celestia-snapshot-finder/internal/config"
)

type DebugLogger func(format string, a ...interface{})

type Manager struct {
	providers []config.Provider
	client    *http.Client
	debugLog  DebugLogger
}

func NewManager(providers []config.Provider, debugLog DebugLogger) *Manager {
	return &Manager{
		providers: providers,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		debugLog: debugLog,
	}
}

func (m *Manager) FilterProviders(nodeType, snapshotType, chainID string) []ProviderInfo {
	var result []ProviderInfo
	targetType := nodeType + "-" + snapshotType

	for _, p := range m.providers {
		for _, s := range p.Snapshots {
			if s.Type == targetType && (chainID == "" || s.ChainID == chainID) {
				result = append(result, ProviderInfo{
					Name:        p.Name,
					URL:         s.URL,
					MetadataURL: s.MetadataURL,
				})
			}
		}
	}

	return result
}

func (m *Manager) CheckHealth(providers []ProviderInfo) []ProviderInfo {
	var healthy []ProviderInfo
	for _, p := range providers {
		m.debugLog("Checking health for snapshot %s (%s)", p.Name, p.URL)
		isHealthy, err := m.isHealthy(p.URL, &p)
		if !isHealthy {
			m.debugLog("Snapshot %s health check failed: %v", p.Name, err)
			continue
		}
		m.debugLog("Snapshot %s passed health check", p.Name)
		healthy = append(healthy, p)
	}
	return healthy
}

func (m *Manager) isHealthy(url string, info *ProviderInfo) (bool, error) {
	start := time.Now()
	resp, err := m.client.Head(url)
	if err != nil {
		return false, fmt.Errorf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	m.debugLog("  Response time: %v", latency)
	m.debugLog("  Status code: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		m.debugLog("  Warning: Content-Length header not provided")
		info.Size = 0
	} else {
		m.debugLog("  Content-Length: %s", contentLength)
		size, err := strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			m.debugLog("  Warning: Failed to parse Content-Length: %v", err)
			info.Size = 0
		} else {
			info.Size = size
		}
	}

	contentType := resp.Header.Get("Content-Type")
	m.debugLog("  Content-Type: %s", contentType)

	acceptRanges := resp.Header.Get("Accept-Ranges")
	if acceptRanges == "bytes" {
		m.debugLog("  Resume capability: supported (Accept-Ranges: bytes)")
	} else {
		m.debugLog("  Resume capability: not supported")
	}

	return true, nil
}

type ProviderInfo struct {
	Name         string
	URL          string
	MetadataURL  string
	Speed        float64
	Size         int64
	DownloadTime float64
}

func (p ProviderInfo) String() string {
	speed := ""
	if p.Speed > 0 {
		speed = fmt.Sprintf(" (%.2f MB/s)", p.Speed)
	}
	return fmt.Sprintf("%s%s", p.Name, speed)
}
