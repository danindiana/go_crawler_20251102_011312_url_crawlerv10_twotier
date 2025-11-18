package tokenizer

import (
	"net/url"
	"strings"
	"sync/atomic"
)

// PathDecision indicates which tokenizer path to use
type PathDecision int

const (
	FastPath PathDecision = iota
	SlowPath
)

// Coordinator routes pages between fast and slow tokenization paths
type Coordinator struct {
	fastPath *FastPathTokenizer
	slowPath *SlowPathTokenizer

	// Routing metrics
	fastPathCount atomic.Uint64
	slowPathCount atomic.Uint64

	// Heuristics thresholds
	fastPathSizeLimit int // Bytes - pages under this go fast
	slowPathSizeLimit int // Bytes - pages over this go slow
}

// NewCoordinator creates a new two-tier coordinator
func NewCoordinator() *Coordinator {
	return &Coordinator{
		fastPath:          NewFastPathTokenizer(),
		slowPath:          NewSlowPathTokenizer(),
		fastPathSizeLimit: 100 * 1024, // 100 KB
		slowPathSizeLimit: 500 * 1024, // 500 KB
	}
}

// Decide determines which path to use based on URL and page characteristics
func (c *Coordinator) Decide(pageURL *url.URL, bodySize int) PathDecision {
	urlStr := pageURL.String()
	urlLower := strings.ToLower(urlStr)

	// FORCE SLOW PATH conditions (need full parsing)

	// 1. Large pages likely have important content
	if bodySize > c.slowPathSizeLimit {
		c.slowPathCount.Add(1)
		return SlowPath
	}

	// 2. Document repository URLs
	if strings.Contains(urlLower, "/document") ||
		strings.Contains(urlLower, "/paper") ||
		strings.Contains(urlLower, "/publication") ||
		strings.Contains(urlLower, "/research") ||
		strings.Contains(urlLower, "/library") {
		c.slowPathCount.Add(1)
		return SlowPath
	}

	// 3. Query parameters indicate dynamic content
	if pageURL.RawQuery != "" {
		c.slowPathCount.Add(1)
		return SlowPath
	}

	// FORCE FAST PATH conditions (link-heavy navigation)

	// 1. Small pages are usually navigation
	if bodySize < c.fastPathSizeLimit {
		c.fastPathCount.Add(1)
		return FastPath
	}

	// 2. Known navigation patterns
	if strings.Contains(urlLower, "/sitemap") ||
		strings.Contains(urlLower, "/archive") ||
		strings.Contains(urlLower, "/category") ||
		strings.Contains(urlLower, "/tag") ||
		strings.Contains(urlLower, "/index") ||
		strings.Contains(urlLower, "/list") {
		c.fastPathCount.Add(1)
		return FastPath
	}

	// 3. URL depth heuristic - shallow paths are often indexes
	pathParts := strings.Split(pageURL.Path, "/")
	if len(pathParts) <= 3 { // e.g., /section/ or /section/index
		c.fastPathCount.Add(1)
		return FastPath
	}

	// DEFAULT: Medium-sized content pages go to slow path for accuracy
	c.slowPathCount.Add(1)
	return SlowPath
}

// GetRoutingStats returns fast vs slow path usage
func (c *Coordinator) GetRoutingStats() (fastCount uint64, slowCount uint64, fastPercent float64) {
	fastCount = c.fastPathCount.Load()
	slowCount = c.slowPathCount.Load()

	total := fastCount + slowCount
	if total > 0 {
		fastPercent = float64(fastCount) / float64(total) * 100.0
	}

	return fastCount, slowCount, fastPercent
}

// GetFastPathStats returns fast-path tokenizer statistics
func (c *Coordinator) GetFastPathStats() (pages uint64, avgLatencyUs uint64, totalLinks uint64) {
	return c.fastPath.GetStats()
}

// GetSlowPathStats returns slow-path tokenizer statistics
func (c *Coordinator) GetSlowPathStats() (pages uint64, avgLatencyUs uint64, totalLinks uint64, totalDocs uint64) {
	return c.slowPath.GetStats()
}

// ProcessFastPath processes a page through the fast tokenizer
func (c *Coordinator) ProcessFastPath(htmlBytes []byte, baseURL *url.URL) *FastPathResult {
	return c.fastPath.ExtractLinks(htmlBytes, baseURL)
}

// ProcessSlowPath processes a page through the slow tokenizer
func (c *Coordinator) ProcessSlowPath(htmlBytes []byte, baseURL *url.URL, docExtensions []string) *SlowPathResult {
	return c.slowPath.AnalyzeDocument(htmlBytes, baseURL, docExtensions)
}

// SetFastPathSizeLimit adjusts the fast-path size threshold
func (c *Coordinator) SetFastPathSizeLimit(bytes int) {
	c.fastPathSizeLimit = bytes
}

// SetSlowPathSizeLimit adjusts the slow-path size threshold
func (c *Coordinator) SetSlowPathSizeLimit(bytes int) {
	c.slowPathSizeLimit = bytes
}

// ResetStats clears all metrics
func (c *Coordinator) ResetStats() {
	c.fastPath.ResetStats()
	c.slowPath.ResetStats()
	c.fastPathCount.Store(0)
	c.slowPathCount.Store(0)
}
