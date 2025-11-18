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
"github.com/jeb/url_crawler/utils"
)

// Crawler manages the web crawling process
type Crawler struct {
collector        *colly.Collector
visitedURLsMap   map[string]bool
mapMutex         *sync.RWMutex
firstRequestOnce sync.Once
startURL         string
logFilePath      string
downloadManager  *downloader.Manager
panicCount       int
panicMutex       sync.Mutex
}

// NewCrawler creates a new crawler instance
func NewCrawler(startURL, logFilePath string, downloadManager *downloader.Manager) *Crawler {
c := &Crawler{
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

// createCollector creates an ultra-aggressive collector with safety limits
func (c *Crawler) createCollector() *colly.Collector {
collector := colly.NewCollector(
colly.UserAgent(config.UserAgent),
colly.Async(true),
colly.IgnoreRobotsTxt(),
colly.MaxBodySize(5*1024*1024), // 5 MB limit to prevent pathological pages
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

// setupCallbacks configures the crawler callbacks
func (c *Crawler) setupCallbacks() {
c.collector.OnRequest(func(r *colly.Request) {
if r.URL.String() == c.startURL {
c.firstRequestOnce.Do(func() {
ctx := colly.NewContext()
ctx.Put("depth", "0")
r.Ctx = ctx
fmt.Printf("🚀 [0] Multi-NIC crawl started: %s\n", r.URL)
})
}
})

c.collector.OnResponse(func(r *colly.Response) {
// Minimal logging for performance
attempts, _, _, _, _ := c.downloadManager.GetStats()
if attempts < 50 {
depth := 0
if d := r.Ctx.Get("depth"); d != "" {
fmt.Sscanf(d, "%d", &depth)
}
if depth <= 1 {
fmt.Printf("✅ [%d] Response %d: %s\n", depth, r.StatusCode, r.Request.URL)
}
}
})

c.collector.OnError(func(r *colly.Response, err error) {
_, _, failed, _, _ := c.downloadManager.GetStats()
if failed < 20 {
fmt.Printf("❌ Crawl error: %v\n", err)
}
})

// MERGED: Single OnHTML handler with panic recovery
// Combines link discovery AND document detection in one pass
docExtensions := []string{".pdf"}

c.collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
// CRITICAL: Panic recovery to prevent stack overflow crashes
defer func() {
if r := recover(); r != nil {
c.panicMutex.Lock()
c.panicCount++
panicNum := c.panicCount
c.panicMutex.Unlock()

// Log panic with full details
log.Printf("🛑 PANIC #%d recovered in OnHTML handler\n", panicNum)
log.Printf("   URL: %s\n", e.Request.URL)
log.Printf("   Error: %v\n", r)
if panicNum <= 3 {
// Only log stack trace for first few panics to avoid spam
log.Printf("   Stack trace:\n%s\n", debug.Stack())
}

// Save problematic URL to a separate log
go func() {
f, err := os.OpenFile("panic_urls.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
if err == nil {
defer f.Close()
f.WriteString(fmt.Sprintf("%s\n", e.Request.URL))
}
}()
}
}()

href := e.Attr("href")
absURL := e.Request.AbsoluteURL(href)

parsed, err := url.Parse(absURL)
if err != nil || parsed.Host == "" {
return
}

currentDepth := 0
if d := e.Request.Ctx.Get("depth"); d != "" {
fmt.Sscanf(d, "%d", &currentDepth)
}

cleanURL := utils.NormalizeParsedURL(parsed)

// PART 1: Link Discovery (crawl new pages)
if currentDepth < config.MaxDepth {
if !c.hasVisited(cleanURL) {
c.saveVisitedURL(cleanURL)

newCtx := colly.NewContext()
newCtx.Put("depth", fmt.Sprintf("%d", currentDepth+1))
e.Request.Visit(absURL)
}
}

// PART 2: Document Detection (queue downloads)
if utils.IsDocumentURL(absURL, docExtensions) {
if !c.downloadManager.IsDownloadedOrPending(absURL) {
task := downloader.DownloadTask{
URL:      absURL,
Depth:    currentDepth,
Retry:    0,
Priority: false,
}

// Try to enqueue the task
if !c.downloadManager.EnqueueTask(task) {
// Queue full - try persistent enqueue
go c.downloadManager.PersistentEnqueue(task)
}
}
}
})
}

// hasVisited checks if a URL has been visited
func (c *Crawler) hasVisited(url string) bool {
c.mapMutex.RLock()
defer c.mapMutex.RUnlock()
_, exists := c.visitedURLsMap[url]
return exists
}

// saveVisitedURL marks a URL as visited
func (c *Crawler) saveVisitedURL(url string) {
c.mapMutex.Lock()
c.visitedURLsMap[url] = true
c.mapMutex.Unlock()

// Async logging for performance
go func() {
f, err := os.OpenFile(c.logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
if err != nil {
return
}
defer f.Close()
f.WriteString(url + "\n")
}()
}

// Start begins the crawling process
func (c *Crawler) Start() error {
return c.collector.Visit(c.startURL)
}

// Wait waits for the crawler to complete
func (c *Crawler) Wait() {
c.collector.Wait()

// Report panic statistics
if c.panicCount > 0 {
log.Printf("\n⚠️  Total panics recovered: %d\n", c.panicCount)
log.Printf("   Check panic_urls.txt for problematic pages\n")
}
}

// GetPanicCount returns the number of panics recovered
func (c *Crawler) GetPanicCount() int {
c.panicMutex.Lock()
defer c.panicMutex.Unlock()
return c.panicCount
}
