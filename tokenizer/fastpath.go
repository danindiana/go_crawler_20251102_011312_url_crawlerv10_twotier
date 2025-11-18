package tokenizer

import (
	"bytes"
	"net/url"
	"sync/atomic"
	"time"
)

// FastPath provides ultra-low-latency URL extraction using regex-style scanning
// NO DOM PARSING - just byte-level scanning for href attributes
// Target: <50 microseconds per page for link-heavy HTML

type FastPathTokenizer struct {
	pagesProcessed atomic.Uint64
	totalLatencyUs atomic.Uint64
	linksExtracted atomic.Uint64
}

// FastPathResult contains extracted URLs without metadata
type FastPathResult struct {
	URLs         []string
	ProcessingUs uint64
	LinkCount    int
}

// NewFastPathTokenizer creates a new fast-path tokenizer
func NewFastPathTokenizer() *FastPathTokenizer {
	return &FastPathTokenizer{}
}

// ExtractLinks performs ultra-fast URL extraction via byte scanning
func (f *FastPathTokenizer) ExtractLinks(htmlBytes []byte, baseURL *url.URL) *FastPathResult {
	start := time.Now()

	var urls []string
	linkCount := 0

	// Fast byte-level scan for href attributes
	i := 0
	for i < len(htmlBytes)-6 {
		if matchesHref(htmlBytes[i:]) {
			i += 5

			quote := byte(0)
			if i < len(htmlBytes) {
				if htmlBytes[i] == '"' || htmlBytes[i] == '\'' {
					quote = htmlBytes[i]
					i++
				}
			}

			urlStart := i
			for i < len(htmlBytes) {
				if quote != 0 {
					if htmlBytes[i] == quote {
						break
					}
				} else {
					if htmlBytes[i] == ' ' || htmlBytes[i] == '>' {
						break
					}
				}
				i++
			}

			if i > urlStart {
				rawURL := string(htmlBytes[urlStart:i])

				if len(rawURL) > 0 && rawURL[0] != '#' &&
					!bytes.HasPrefix([]byte(rawURL), []byte("javascript:")) &&
					!bytes.HasPrefix([]byte(rawURL), []byte("mailto:")) {

					absURL := makeAbsolute(rawURL, baseURL)
					if absURL != "" {
						urls = append(urls, absURL)
						linkCount++
					}
				}
			}
		}
		i++
	}

	elapsedUs := uint64(time.Since(start).Microseconds())

	f.pagesProcessed.Add(1)
	f.totalLatencyUs.Add(elapsedUs)
	f.linksExtracted.Add(uint64(linkCount))

	return &FastPathResult{
		URLs:         urls,
		ProcessingUs: elapsedUs,
		LinkCount:    linkCount,
	}
}

func matchesHref(b []byte) bool {
	if len(b) < 5 {
		return false
	}
	return (b[0] == 'h' || b[0] == 'H') &&
		(b[1] == 'r' || b[1] == 'R') &&
		(b[2] == 'e' || b[2] == 'E') &&
		(b[3] == 'f' || b[3] == 'F') &&
		b[4] == '='
}

func makeAbsolute(rawURL string, base *url.URL) string {
	if len(rawURL) > 7 && (rawURL[0:7] == "http://" || rawURL[0:7] == "https:/") {
		return rawURL
	}

	if len(rawURL) > 2 && rawURL[0:2] == "//" {
		return base.Scheme + ":" + rawURL
	}

	if len(rawURL) > 0 && rawURL[0] == '/' {
		return base.Scheme + "://" + base.Host + rawURL
	}

	baseStr := base.String()
	if baseStr[len(baseStr)-1] == '/' {
		return baseStr + rawURL
	}
	return baseStr + "/" + rawURL
}

func (f *FastPathTokenizer) GetStats() (pages uint64, avgLatencyUs uint64, totalLinks uint64) {
	pages = f.pagesProcessed.Load()
	totalLatency := f.totalLatencyUs.Load()
	totalLinks = f.linksExtracted.Load()

	if pages > 0 {
		avgLatencyUs = totalLatency / pages
	}

	return pages, avgLatencyUs, totalLinks
}

func (f *FastPathTokenizer) ResetStats() {
	f.pagesProcessed.Store(0)
	f.totalLatencyUs.Store(0)
	f.linksExtracted.Store(0)
}
