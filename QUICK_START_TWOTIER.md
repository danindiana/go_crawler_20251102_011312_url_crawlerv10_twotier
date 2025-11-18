# Two-Tier Crawler - Quick Start Guide

## 🚀 Run the Two-Tier Crawler

```bash
cd /home/jeb/programs/go_crawler/20251102_011312_url_crawlerv10_twotier
./url_crawler_twotier
```

## 📊 What You'll See

### Startup
```
🔥🔥🔥 MULTI-NIC BEAST MODE ACTIVATED! 🔥🔥🔥
🚀🚀 [0] TWO-TIER Multi-NIC crawl started: https://example.com
```

### Fast-Path Processing (first 10 pages)
```
⚡ FAST [0] https://example.com/ → 87 links in 42μs
⚡ FAST [1] https://example.com/sitemap → 234 links in 38μs
⚡ FAST [1] https://example.com/archive → 156 links in 51μs
```

### Slow-Path Processing (first 10 pages)
```
🐢 SLOW [2] https://example.com/research/paper.html → 23 links, 3 docs in 412μs
🐢 SLOW [2] https://example.com/publications → 45 links, 12 docs in 387μs
```

### Periodic Stats (every 100 downloads)
```
╔══════════════════════════════════════════════════════════╗
║         TWO-TIER TOKENIZER PERFORMANCE STATS           ║
╠══════════════════════════════════════════════════════════╣
║ FAST PATH:   8,456 pages | Avg:   45μs | Links: 367,892 ║
║ SLOW PATH:     894 pages | Avg:  398μs | Docs:      823 ║
║ ROUTING:      90.4% fast |   9.6% slow                  ║
╚══════════════════════════════════════════════════════════╝
```

## 🎛️ Tuning Parameters

### Adjust Concurrency
```bash
# Edit config/config.go
ConcurrentWorkers = 50  # Increase from 20
go build -o url_crawler_twotier
./url_crawler_twotier
```

### Adjust Routing Thresholds
Fast-path size limit: 100KB (default)  
Slow-path size limit: 500KB (default)

To change, edit `tokenizer/coordinator.go`:
```go
fastPathSizeLimit: 150 * 1024, // Increase to 150KB
slowPathSizeLimit: 400 * 1024, // Decrease to 400KB
```

## 📈 Performance Comparison

### Measure v9 Baseline
```bash
cd ../20251102_011312_url_crawlerv9
./url_crawler_v2
# Note: Pages/sec, CPU%, runtime
```

### Measure v10 Two-Tier
```bash
cd ../20251102_011312_url_crawlerv10_twotier
./url_crawler_twotier
# Note: Pages/sec, CPU%, fast/slow split
```

### Expected Improvement
- **Throughput**: 3-5x more pages/second
- **CPU Usage**: 30-50% reduction
- **Latency**: Average processing time <100μs (was ~500μs)

## 🐛 Troubleshooting

### Issue: Low fast-path percentage (<80%)
**Solution**: URLs not matching fast-path patterns. Check coordinator heuristics.

### Issue: Fast-path too slow (>100μs)
**Cause**: Large pages with many links.  
**Solution**: Lower fastPathSizeLimit to 50KB.

### Issue: Slow-path missing documents
**Cause**: Document URLs routed to fast-path.  
**Solution**: Add more slow-path URL patterns in coordinator.

### Issue: High panic count
**Cause**: Malformed HTML in slow-path DOM parsing.  
**Solution**: Already handled with defer/recover. Check `panic_urls.txt`.

## 🔄 Rollback to Single-Tier

If two-tier causes issues:
```bash
cp crawler/crawler_singletier.go.backup crawler/crawler.go
cp main_singletier.go.backup main.go
sed -i 's/NewCrawlerTwoTier/NewCrawler/' main.go
go build -o url_crawler_modular
./url_crawler_modular
```

## 📁 Output Files

```
visitedURLs_TIMESTAMP.txt  - All crawled URLs
downloads_TIMESTAMP.txt    - Downloaded documents  
panic_urls.txt             - Pages that caused panics (if any)
```

## 🎯 Quick Checks

✅ **Is it working?**
- Look for "⚡ FAST" and "🐢 SLOW" messages
- Stats dashboard should show 85-95% fast-path
- No crashes or excessive panics

✅ **Is it faster?**
- Compare pages/second vs v9 baseline
- Check CPU usage (should be lower)
- Fast-path avg <100μs, slow-path <600μs

✅ **Is it accurate?**
- Documents still being detected
- No obvious missing links
- Download count similar to v9

---

**Binary**: `url_crawler_twotier` (15MB)  
**Full Docs**: `TWO_TIER_IMPLEMENTATION.md`  
**Architecture**: `TWO_TIER_ARCHITECTURE.md`
