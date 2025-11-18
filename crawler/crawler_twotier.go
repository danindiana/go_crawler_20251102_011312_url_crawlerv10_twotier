package crawler

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime/debug"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/jeb/url_crawler/config"
	"github.com/jeb/url_crawler/downloader"
	"github.com/jeb/url_crawler/tokenizer"
	"github.com/jeb/url_crawler/utils"
)

// CrawlerTwoTier manages web crawling with two-tier tokenization
type CrawlerTwoTier struct {
	collector        *colly.Collector
	coordinator      *tokenizer.Coordinator
	visitedURLsMap   map[string]bool
	mapMutex         *sync.RWMutex
	firstRequestOnce sync.Once
	startURL         string
	logFilePath      string
	downloadManager  *downloader.Manager
	panicCount       int
	panicMutex       sync.Mutex
}

// NewCrawlerTwoTier creates a new two-tier crawler instance
func NewCrawlerTwoTier(startURL, logFilePath string, downloadManager *downloader.Manager) *CrawlerTwoTier {
	c := &CrawlerTwoTier{
		coordinator:     tokenizer.NewCoordinator(),
		visitedURLsMap:  make(map[string]bool),
		mapMutex:        &sync.RWMutex{},
		startURL:        startURL,
		logFilePath:     logFilePath,
		downloadManager: downloadManager,
		panicCount:      0,
	}

	c.collector = c.createCollector()
	c.setupCallbacks()

	return c
}

// createCollector creates collector with colly v2.2.0
func (c *CrawlerTwoTier) createCollector() *colly.Collector {
	collector := colly.NewCollector(
		colly.UserAgent(config.UserAgent),
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.MaxBodySize(5*1024*1024), // 5 MB limit
	)

	extensions.RandomUserAgent(collector)
	extensions.Referer(collector)
	collector.SetRequestTimeout(config.RequestTimeout)

	err := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: config.ConcurrentWorkers,
		Delay:       config.PoliteDelay,
		RandomDelay: 5,
	})

	if err != nil {
		fmt.Printf("❌ Failed to set crawl limits: %v\n", err)
	}

	cacheDir := ".colly_cache"
	os.RemoveAll(cacheDir)
	collector.CacheDir = cacheDir

	return collector
}

// setupCallbacks configures TWO-TIER tokenization callbacks
func (c *CrawlerTwoTier) setupCallbacks() {
	docExtensions := []string{".pdf"}

	c.collector.OnRequest(func(r *colly.Request) {
		if r.URL.String() == c.startURL {
			c.firstRequestOnce.Do(func() {
				ctx := colly.NewContext()
				ctx.Put("depth", "0")
				r.Ctx = ctx
				fmt.Printf("🚀🚀 [0] TWO-TIER Multi-NIC crawl started: %s\n", r.URL)
			})
		}
	})

	// TWO-TIER RESPONSE HANDLER - Routes to fast or slow path
	c.collector.OnResponse(func(r *colly.Response) {
		defer func() {
			if rec := recover(); rec != nil {
				c.panicMutex.Lock()
				c.panicCount++
				panicNum := c.panicCount
				c.panicMutex.Unlock()

				log.Printf("🛑 PANIC #%d in OnResponse: %v\n", panicNum, rec)
				if panicNum <= 3 {
					log.Printf("   Stack:\n%s\n", debug.Stack())
				}
			}
		}()

		currentDepth := 0
		if d := r.Ctx.Get("depth"); d != "" {
			fmt.Sscanf(d, "%d", &currentDepth)
		}

		// COORDINATOR DECISION: Fast or Slow path?
		decision := c.coordinator.Decide(r.Request.URL, len(r.Body))

		if decision == tokenizer.FastPath {
			// FAST PATH: Lightweight byte scanning
			result := c.coordinator.ProcessFastPath(r.Body, r.Request.URL)

			// Process extracted URLs
			for _, urlStr := range result.URLs {
				c.processDiscoveredURL(urlStr, currentDepth)
			}

			// Log first few fast-path results
			fastCount, _, _ := c.coordinator.GetRoutingStats()
			if fastCount <= 10 {
				fmt.Printf("⚡ FAST [%d] %s → %d links in %dμs\n",
					currentDepth, r.Request.URL, result.LinkCount, result.ProcessingUs)
			}

		} else {
			// SLOW PATH: Full DOM parsing + document detection
			result := c.coordinator.ProcessSlowPath(r.Body, r.Request.URL, docExtensions)

			// Process extracted URLs
			for _, urlStr := range result.URLs {
				c.processDiscoveredURL(urlStr, currentDepth)
			}

			// Process detected documents
			for _, doc := range result.Documents {
				if !c.downloadManager.IsDownloadedOrPending(doc.URL) {
					task := downloader.DownloadTask{
						URL:      doc.URL,
						Depth:    currentDepth,
						Retry:    0,
						Priority: false,
					}

					if !c.downloadManager.EnqueueTask(task) {
						go c.downloadManager.PersistentEnqueue(task)
					}
				}
			}

			// Log slow-path results
			_, slowCount, _ := c.coordinator.GetRoutingStats()
			if slowCount <= 10 {
				fmt.Printf("🐢 SLOW [%d] %s → %d links, %d docs in %dμs\n",
					currentDepth, r.Request.URL, result.LinkCount, result.DocCount, result.ProcessingUs)
			}
		}

		// Periodic stats logging
		attempts, _, _, _, _ := c.downloadManager.GetStats()
		if attempts > 0 && attempts%100 == 0 {
			c.logTwoTierStats()
		}
	})

	c.collector.OnError(func(r *colly.Response, err error) {
		_, _, failed, _, _ := c.downloadManager.GetStats()
		if failed < 20 {
			fmt.Printf("❌ Crawl error: %v\n", err)
		}
	})
}

// processDiscoveredURL handles a newly discovered URL
func (c *CrawlerTwoTier) processDiscoveredURL(urlStr string, currentDepth int) {
	parsed, err := url.Parse(urlStr)
	if err != nil || parsed.Host == "" {
		return
	}

	cleanURL := utils.NormalizeParsedURL(parsed)

	if currentDepth < config.MaxDepth {
		if !c.hasVisited(cleanURL) {
			c.saveVisitedURL(cleanURL)

			newCtx := colly.NewContext()
			newCtx.Put("depth", fmt.Sprintf("%d", currentDepth+1))
			c.collector.Request("GET", urlStr, nil, newCtx, nil)
		}
	}
}

// logTwoTierStats prints two-tier performance metrics
func (c *CrawlerTwoTier) logTwoTierStats() {
	_, _, fastPercent := c.coordinator.GetRoutingStats()
	fastPages, fastAvgUs, fastLinks := c.coordinator.GetFastPathStats()
	slowPages, slowAvgUs, _, slowDocs := c.coordinator.GetSlowPathStats()

	fmt.Printf("\n╔══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║         TWO-TIER TOKENIZER PERFORMANCE STATS           ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ FAST PATH:  %6d pages | Avg: %4dμs | Links: %7d ║\n",
		fastPages, fastAvgUs, fastLinks)
	fmt.Printf("║ SLOW PATH:  %6d pages | Avg: %4dμs | Docs:  %7d ║\n",
		slowPages, slowAvgUs, slowDocs)
	fmt.Printf("║ ROUTING:    %5.1f%% fast | %5.1f%% slow              ║\n",
		fastPercent, 100.0-fastPercent)
	fmt.Printf("╚══════════════════════════════════════════════════════════╝\n\n")
}

// hasVisited checks if URL was visited
func (c *CrawlerTwoTier) hasVisited(url string) bool {
	c.mapMutex.RLock()
	defer c.mapMutex.RUnlock()
	_, exists := c.visitedURLsMap[url]
	return exists
}

// saveVisitedURL marks URL as visited
func (c *CrawlerTwoTier) saveVisitedURL(url string) {
	c.mapMutex.Lock()
	c.visitedURLsMap[url] = true
	c.mapMutex.Unlock()

	go func() {
		f, err := os.OpenFile(c.logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()
		f.WriteString(url + "\n")
	}()
}

// Start begins crawling
func (c *CrawlerTwoTier) Start() error {
	return c.collector.Visit(c.startURL)
}

// Wait waits for completion
func (c *CrawlerTwoTier) Wait() {
	c.collector.Wait()

	// Final stats
	c.logTwoTierStats()

	if c.panicCount > 0 {
		log.Printf("\n⚠️  Total panics recovered: %d\n", c.panicCount)
	}
}

// GetPanicCount returns panic count
func (c *CrawlerTwoTier) GetPanicCount() int {
	c.panicMutex.Lock()
	defer c.panicMutex.Unlock()
	return c.panicCount
}
