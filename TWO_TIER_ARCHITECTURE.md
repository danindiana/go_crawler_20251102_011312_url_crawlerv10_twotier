# Two-Tier Tokenizer Architecture

## Overview
Implements Perplexity's two-tier HTML processing concept for ultra-high-throughput web crawling.

## Design Philosophy

### Problem Statement
Traditional web crawlers use a single HTML processing pipeline that treats all pages equally:
- Simple navigation pages (mostly links, minimal content) get full DOM parsing
- Document-heavy pages (PDFs, academic content) get same treatment
- 90% of pages are link-heavy navigation → wasting CPU on unnecessary parsing

### Solution: Two-Tier Processing
**Fast Path** (90% of pages):
- Byte-level href scanning (NO DOM parsing)
- Target: <50μs per page
- Extracts: URLs only
- Use case: Navigation pages, sitemaps, index pages

**Slow Path** (10% of pages):
- Full goquery/cascadia DOM parsing  
- Target: <500μs per page
- Extracts: URLs + metadata + document detection
- Use case: Content pages, document repositories

## Architecture Components

### 1. Fast-Path Tokenizer (`tokenizer/fastpath.go`)
```
Input: Raw HTML bytes
Process:
  - Scan for 'href=' patterns (case-insensitive)
  - Extract quoted/unquoted URLs
  - Skip anchors, javascript:, mailto:
  - Make absolute (lightweight - no full URL parsing)
Output: []string URLs
Metrics: Pages processed, avg latency, links extracted
```

**Key Optimizations:**
- NO regex (pure byte iteration)
- NO string allocations in hot loop
- NO DOM tree construction
- Inline URL validation

### 2. Slow-Path Tokenizer (`tokenizer/slowpath.go`)
```
Input: Raw HTML bytes
Process:
  - Full goquery DOM parsing
  - Extract title, meta description
  - Detect navigation elements
  - Calculate link density
  - Document detection (.pdf, etc.)
  - Extract link context (surrounding text)
Output: Slow PathResult{URLs, Documents, PageMetadata}
Metrics: Pages, latency, links, documents detected
```

**Comprehensive Features:**
- Document metadata (title, context, extension)
- Page quality metrics (link density)
- Structural analysis (has nav?)
- Accurate absolute URL resolution

### 3. Coordinator (`tokenizer/coordinator.go`)
```
Routes pages between fast/slow paths based on heuristics:

FAST PATH if:
  - Page size < 100KB
  - URL contains: /sitemap, /archive, /category, /tag
  - Link density > 50 links/KB (from previous crawls)
  - No query parameters
  
SLOW PATH if:
  - Page size > 100KB
  - URL contains: /document, /paper, /publication
  - Known document repository domain
  - Query parameters present (dynamic content)
  - First visit to domain (need metadata)
```

## Performance Gains

### Theoretical Improvements
- Fast-path: 10x faster than slow-path (50μs vs 500μs)
- If 90% fast-path: `0.9×50 + 0.1×500 = 95μs avg` vs `500μs baseline`
- **5.26x throughput improvement**

### Real-World Expectations
- Fast-path likely 50-150μs (depends on page size)
- Slow-path likely 300-800μs (goquery overhead)
- Conservative estimate: **3-4x throughput gain**
- Bonus: Lower CPU usage (less DOM parsing)

## Integration with Crawler

### Modified OnResponse Handler
```go
collector.OnResponse(func(r *colly.Response) {
    // Coordinator decides fast vs slow path
    decision := coordinator.Decide(r.Request.URL, len(r.Body))
    
    if decision == FastPath {
        result := fastTokenizer.ExtractLinks(r.Body, r.Request.URL)
        // Queue discovered URLs
        for _, url := range result.URLs {
            collector.Visit(url)
        }
    } else {
        result := slowTokenizer.AnalyzeDocument(r.Body, r.Request.URL, docExtensions)
        // Queue URLs + documents
        for _, url := range result.URLs {
            collector.Visit(url)
        }
        for _, doc := range result.Documents {
            downloadManager.Enqueue(doc.URL)
        }
    }
})
```

## Monitoring & Metrics

### Two-Tier Dashboard
```
=== TWO-TIER TOKENIZER STATS ===
Fast Path:  12,450 pages | Avg:  42μs | Links: 543,210
Slow Path:   1,380 pages | Avg: 387μs | Docs:    1,247
Routing:     90.0% fast | 10.0% slow
Throughput:  3.8x vs single-tier baseline
```

### Adaptive Routing
Coordinator learns from metrics:
- If fast-path avg > 100μs → tighten heuristics
- If slow-path ratio > 20% → expand fast-path rules
- Track per-domain patterns for smarter routing

## Testing Strategy

### Phase 1: Baseline (Current v9)
- Run with 20 workers, measure:
  - Pages/second
  - CPU usage
  - Avg processing time per page

### Phase 2: Two-Tier (v10)
- Same 20 workers, measure:
  - Fast/slow path split
  - Avg latency per path
  - Overall pages/second
  - CPU usage reduction

### Phase 3: Scale Up
- If two-tier shows gains, increase to 50-100 workers
- Monitor for channel deadlocks (should be rare with v2.2.0)
- Compare throughput: v9@20 vs v10@50 vs v10@100

## Expected Results

### Success Criteria
✅ Fast-path handles >85% of pages  
✅ Fast-path <100μs average  
✅ Slow-path <600μs average  
✅ Overall throughput >2x baseline  
✅ No document detection regressions  
✅ CPU usage reduction >30%

### Fallback Plan
If two-tier underperforms:
- Coordinator overhead too high → simplify heuristics
- Fast-path too slow → optimize byte scanning
- Slow-path too slow → reduce goquery features
- Worst case: Revert to v9 (colly v2.2.0 still beneficial)

## Future Enhancements

### Tier 0: Pre-Filter
Add ultra-fast pre-filter before byte scanning:
- Bloom filter for already-visited URLs
- Reject known-bad patterns (ads, tracking)
- Target: <10μs decision

### ML-Based Routing
Train classifier on:
- URL patterns
- Domain history
- Page size
- Response headers
Predict: Fast vs slow path with >95% accuracy

### Streaming Tokenizer
Process HTML incrementally:
- Start extracting links before full page loaded
- Abort slow-path if page too large
- Early exit for fast-path

---

**Implementation Date**: Nov 11, 2025  
**Based On**: Perplexity research (perplexity_answer.md)  
**Target Improvement**: 3-5x throughput at same concurrency
