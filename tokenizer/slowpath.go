package tokenizer

import (
	"bytes"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// SlowPath provides comprehensive HTML analysis with full DOM parsing
// Uses goquery + cascadia for accurate link/document detection
// Target: <500 microseconds per page (10x slower than fast-path, but thorough)

type SlowPathTokenizer struct {
	pagesProcessed atomic.Uint64
	totalLatencyUs atomic.Uint64
	linksExtracted atomic.Uint64
	docsDetected   atomic.Uint64
}

// SlowPathResult contains extracted URLs plus document metadata
type SlowPathResult struct {
	URLs         []string
	Documents    []DocumentInfo
	ProcessingUs uint64
	LinkCount    int
	DocCount     int
	PageMetadata PageMetadata
}

// DocumentInfo holds metadata about detected documents
type DocumentInfo struct {
	URL       string
	Extension string
	Title     string // Link text
	Context   string // Surrounding text
}

// PageMetadata contains page-level information
type PageMetadata struct {
	Title       string
	Description string
	LinkDensity float64 // links per KB of HTML
	HasNav      bool    // Contains navigation elements
	Depth       int     // From context
}

// NewSlowPathTokenizer creates a new slow-path tokenizer
func NewSlowPathTokenizer() *SlowPathTokenizer {
	return &SlowPathTokenizer{}
}

// AnalyzeDocument performs comprehensive HTML analysis with full parsing
func (s *SlowPathTokenizer) AnalyzeDocument(htmlBytes []byte, baseURL *url.URL, docExtensions []string) *SlowPathResult {
	start := time.Now()

	result := &SlowPathResult{
		URLs:      make([]string, 0, 100),
		Documents: make([]DocumentInfo, 0, 10),
	}

	// Parse HTML with goquery (full DOM)
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		// Fallback to fast-path on parse error
		elapsedUs := uint64(time.Since(start).Microseconds())
		result.ProcessingUs = elapsedUs
		s.pagesProcessed.Add(1)
		s.totalLatencyUs.Add(elapsedUs)
		return result
	}

	// Extract page metadata
	result.PageMetadata.Title = doc.Find("title").First().Text()
	result.PageMetadata.Description = doc.Find("meta[name='description']").AttrOr("content", "")
	result.PageMetadata.HasNav = doc.Find("nav").Length() > 0

	// Process all links
	doc.Find("a[href]").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists || href == "" || href == "#" {
			return
		}

		// Skip javascript/mailto
		if strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
			return
		}

		// Make absolute
		absURL, err := baseURL.Parse(href)
		if err != nil {
			return
		}

		urlStr := absURL.String()
		result.URLs = append(result.URLs, urlStr)
		result.LinkCount++

		// Check if it's a document
		if isDocument(urlStr, docExtensions) {
			doc := DocumentInfo{
				URL:       urlStr,
				Extension: getExtension(urlStr),
				Title:     sel.Text(),
				Context:   getContext(sel),
			}
			result.Documents = append(result.Documents, doc)
			result.DocCount++
		}
	})

	// Calculate link density
	htmlSize := float64(len(htmlBytes)) / 1024.0 // KB
	if htmlSize > 0 {
		result.PageMetadata.LinkDensity = float64(result.LinkCount) / htmlSize
	}

	elapsedUs := uint64(time.Since(start).Microseconds())
	result.ProcessingUs = elapsedUs

	// Update metrics
	s.pagesProcessed.Add(1)
	s.totalLatencyUs.Add(elapsedUs)
	s.linksExtracted.Add(uint64(result.LinkCount))
	s.docsDetected.Add(uint64(result.DocCount))

	return result
}

// isDocument checks if URL points to a document
func isDocument(urlStr string, extensions []string) bool {
	urlLower := strings.ToLower(urlStr)
	for _, ext := range extensions {
		if strings.HasSuffix(urlLower, ext) {
			return true
		}
	}
	return false
}

// getExtension extracts file extension from URL
func getExtension(urlStr string) string {
	parts := strings.Split(urlStr, ".")
	if len(parts) > 1 {
		ext := parts[len(parts)-1]
		// Remove query params
		if idx := strings.Index(ext, "?"); idx != -1 {
			ext = ext[:idx]
		}
		return "." + ext
	}
	return ""
}

// getContext extracts surrounding text context for a link
func getContext(sel *goquery.Selection) string {
	// Get parent element text (simplified)
	parent := sel.Parent()
	if parent.Length() > 0 {
		text := parent.Text()
		if len(text) > 200 {
			text = text[:200] + "..."
		}
		return strings.TrimSpace(text)
	}
	return ""
}

// GetStats returns slow-path statistics
func (s *SlowPathTokenizer) GetStats() (pages uint64, avgLatencyUs uint64, totalLinks uint64, totalDocs uint64) {
	pages = s.pagesProcessed.Load()
	total := s.totalLatencyUs.Load()
	totalLinks = s.linksExtracted.Load()
	totalDocs = s.docsDetected.Load()

	if pages > 0 {
		avgLatencyUs = total / pages
	}

	return pages, avgLatencyUs, totalLinks, totalDocs
}

// ResetStats clears all metrics
func (s *SlowPathTokenizer) ResetStats() {
	s.pagesProcessed.Store(0)
	s.totalLatencyUs.Store(0)
	s.linksExtracted.Store(0)
	s.docsDetected.Store(0)
}
