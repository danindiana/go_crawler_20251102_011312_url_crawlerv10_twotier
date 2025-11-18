# Go Web Crawler Stack Overflow Research Query

## Problem Context

We have a high-performance multi-NIC web crawler built in Go using the following stack:
- **colly** v1.2.0 (web scraping framework)
- **goquery** v1.8.1 (HTML parsing and selection)
- **cascadia** v1.3.2 (CSS selector engine)

### Current Architecture
- Ultra-aggressive crawling with high concurrency (massive parallel workers)
- Multiple `OnHTML("a[href]", ...)` callbacks for link discovery and document detection
- Async operation with goroutine-based download queue management
- Multi-NIC network interface management for distributed load
- No current panic recovery or HTML complexity limits

### Observed Error
```
github.com/andybalholm/cascadia.Selector.matchAllInto(0xc09ba63e20, 0xc0d4ea0770, {0x0, 0x0, 0x0})
    /home/jeb/go/pkg/mod/github.com/andybalholm/cascadia@v1.3.2/selector.go:132 +0xcd
github.com/andybalholm/cascadia.Selector.matchAllInto(0xc09ba63e20, 0xc0baececb0, {0x0, 0x0, 0x0})
    /home/jeb/go/pkg/mod/github.com/andybalholm/cascadia@v1.3.2/selector.go:132 +0xcd
[recursive calls continue...]
github.com/gocolly/colly.(*Collector).handleOnHTML(0xc00013d6c0, 0xc000bc8840)
    /home/jeb/go/pkg/mod/github.com/gocolly/colly@v1.2.0/colly.go:948 +0x111
```

This is a **stack overflow in cascadia's selector matching** occurring during HTML parsing in the colly OnHTML callback handlers.

## Research Questions

### 1. Root Cause Analysis
**Q1.1:** What are the known issues with `cascadia` v1.3.2 regarding stack overflow in `matchAllInto()` when processing deeply nested HTML DOM structures?

**Q1.2:** Are there specific HTML patterns (deeply nested divs, tables, SVG elements, etc.) that are known to trigger recursive depth issues in cascadia's CSS selector matching algorithm?

**Q1.3:** What is the typical Go goroutine stack size, and at what DOM nesting depth would cascadia typically exhaust the stack?

### 2. Library Version Updates
**Q2.1:** Have newer versions of cascadia (post v1.3.2) addressed stack overflow issues in selector matching? What's the current stable version and its changelog regarding recursion limits?

**Q2.2:** Are there known issues or fixes in colly versions after v1.2.0 that improve HTML parsing stability or add safety mechanisms?

**Q2.3:** Should we consider migrating from cascadia to alternative CSS selector libraries like `go-css` or custom XPath solutions?

### 3. Architectural Solutions

**Q3.1:** What are the Go best practices for implementing panic recovery in high-concurrency web scraping applications? Specifically:
- Should we use `defer recover()` in each OnHTML callback?
- Should we wrap the entire colly callback registration?
- How to properly log and continue after recovering from panics in goroutines?

**Q3.2:** How can we limit HTML parsing complexity **before** it reaches cascadia? Options to research:
- Pre-filtering HTML documents by size (what's a reasonable max?)
- Using response content-length headers to reject large pages
- Streaming HTML parsing with early termination
- DOM depth analysis before full parsing

**Q3.3:** What's the best way to handle duplicate or multiple OnHTML handlers in colly?
- Should they be merged into a single handler with conditional logic?
- Is there a performance penalty for multiple handlers on the same selector?
- Can handler order cause race conditions or unexpected behavior?

### 4. Stack Management Solutions

**Q4.1:** Can we increase the goroutine stack size for specific crawler goroutines? If so:
- What's the syntax/API for setting custom stack sizes in Go?
- What are the memory implications for high-concurrency crawlers?
- Is this considered a good practice or a code smell?

**Q4.2:** Are there tail-recursion optimizations or iterative alternatives to cascadia's recursive matching algorithm that we could implement or configure?

**Q4.3:** Should we implement a custom HTML selector with explicit recursion depth limits? What are examples of production-grade implementations?

### 5. Error Handling & Resilience

**Q5.1:** What's the industry-standard approach for making colly-based crawlers resilient to malformed/pathological HTML? Research:
- HTML sanitization libraries compatible with colly
- Pre-processing HTML to remove problematic structures
- Timeout mechanisms for individual HTML parsing operations

**Q5.2:** How do production web scrapers (Common Crawl, Archive.org, commercial scrapers) handle extreme HTML edge cases?

**Q5.3:** What monitoring/telemetry should we add to detect problematic pages before they cause crashes?
- HTML complexity metrics (nesting depth, node count, document size)
- Parsing time budgets per page
- Automatic blacklisting of problematic domains/patterns

### 6. Performance vs. Stability Trade-offs

**Q6.1:** If we implement the following safety measures, what's the expected performance impact?
- Panic recovery on every HTML callback
- HTML complexity pre-checks
- Document size limits (e.g., rejecting pages > 10MB)
- Parsing timeouts

**Q6.2:** Are there "fast path" optimizations we can use for simple HTML while keeping safety measures for complex pages?

**Q6.3:** Should we consider a two-tier architecture: fast crawling for standard pages + isolated worker pool for complex/risky pages?

### 7. Alternative Approaches

**Q7.1:** Should we switch from CSS selectors to XPath for more predictable performance characteristics?

**Q7.2:** Are there streaming HTML parsers for Go that can extract links without building a full DOM tree?

**Q7.3:** Would using a headless browser approach (chromedp, playwright-go) be more stable for complex modern web pages, despite the performance cost?

## Code-Specific Questions

**Q8.1:** In our current implementation, we have two separate `OnHTML("a[href]", ...)` handlers - one for link discovery and one for document detection. What's the correct way to structure this in colly to avoid redundant DOM traversals?

**Q8.2:** We're using `extensions.RandomUserAgent(collector)` - could user agent variations cause different HTML responses that affect parsing stability?

**Q8.3:** Our crawler ignores robots.txt and uses aggressive parallelism - are there specific types of websites (SPAs, infinite scroll, dynamically generated content) that are more likely to produce pathological HTML?

## Implementation Priorities

Based on the research findings, please provide:
1. **Quick Wins** - Immediate fixes that can be implemented in < 30 minutes
2. **Medium-term Solutions** - Architectural improvements for the next iteration
3. **Long-term Considerations** - Fundamental redesigns to consider if issues persist

## Success Criteria

The solution should:
- ✅ Eliminate stack overflow crashes from HTML parsing
- ✅ Maintain high-throughput crawling performance (thousands of concurrent workers)
- ✅ Gracefully handle and log problematic pages without stopping the crawler
- ✅ Provide visibility into HTML complexity issues for debugging
- ✅ Be production-ready for 24/7 operation

## Additional Context

- Target deployment: Linux systems (Ubuntu/Debian)
- Typical crawl targets: Academic sites, documentation sites, general web
- Current performance: Processing hundreds of pages per second across multiple NICs
- Memory constraints: Running on systems with 16-64GB RAM
- Go version: (please research compatibility with latest Go 1.21+ features)

---

**Request to Perplexity:** Please provide detailed research on each question above, with:
- Code examples where applicable
- Links to relevant GitHub issues, blog posts, or documentation
- Performance benchmarks if available
- Recommended libraries/versions with justification
- Real-world case studies of similar issues and their solutions

Priority should be given to solutions that maintain the high-performance nature of the crawler while adding necessary safety mechanisms.
