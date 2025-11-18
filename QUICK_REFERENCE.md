# Quick Reference Guide

## Building the Application

```bash
# Build with default name
go build

# Build with custom name
go build -o url_crawler_modular

# Build with optimizations for production
go build -ldflags="-s -w" -o url_crawler_modular
```

## Running the Application

```bash
./url_crawler_modular
```

## Package Overview

| Package | Purpose | Key Files | Lines of Code (approx) |
|---------|---------|-----------|----------------------|
| `main` | Application entry point | main.go | 120 |
| `config` | Configuration constants | config.go | 40 |
| `network` | Network interface management | interface.go | 250 |
| `downloader` | Download task management | downloader.go | 350 |
| `crawler` | Web crawling logic | crawler.go | 180 |
| `monitor` | Performance monitoring | monitor.go | 280 |
| `system` | System optimization | system.go | 60 |
| `utils` | Utility functions | utils.go | 90 |

## Common Modifications

### Changing Maximum Depth
File: `config/config.go`
```go
MaxDepth = 13  // Change this value
```

### Adjusting Worker Counts
File: `config/config.go`
```go
InitialDownloadWorkers = 100  // Starting workers
MaxDownloadWorkers     = 800  // Maximum workers
```

### Modifying Scaling Behavior
File: `config/config.go`
```go
QueueGrowthThreshold = 0.4    // Trigger scaling at 40% full
ScaleUpAmount        = 300    // Add 300 workers per scale event
```

### Adding Document Types
File: `crawler/crawler.go`, in `setupCallbacks()` function
```go
docExtensions := []string{".pdf", ".doc", ".docx", ".zip"}
```

### Adjusting Buffer Size
File: `config/config.go`
```go
DownloadBufferSize = 32 * 1024 * 1024  // 32MB buffer
```

### Changing Timeouts
File: `config/config.go`
```go
RequestTimeout    = 60 * time.Second   // HTTP request timeout
ConnectionTimeout = 3 * time.Second    // Connection timeout
```

## Debugging

### Enable Verbose Logging
Modify monitoring intervals for more frequent updates:

File: `monitor/monitor.go`
```go
// Performance stats - change from 3s to 1s
ticker := time.NewTicker(1 * time.Second)

// Network stats - change from 15s to 5s  
ticker := time.NewTicker(5 * time.Second)
```

### Check Queue Status
The network monitor shows queue utilization per interface every 15 seconds.
Look for high utilization (>80%) to diagnose bottlenecks.

### Memory Issues
If memory usage grows too large:
1. Reduce `TargetMemoryUsageGB` in `config/config.go`
2. Reduce `MaxDownloadWorkers` 
3. Reduce `DownloadBufferSize`

### Connection Issues
If seeing many connection errors:
1. Reduce `MaxConnectionsTotal` in `config/config.go`
2. Increase `ConnectionTimeout`
3. Reduce `MaxDownloadWorkers`

## Performance Tuning Checklist

- [ ] File descriptor limit increased (`ulimit -n 65536`)
- [ ] Network kernel parameters tuned (see README.md)
- [ ] Multiple high-speed network interfaces available
- [ ] Sufficient RAM available (50GB+ recommended)
- [ ] Target website can handle the load
- [ ] Disk I/O is not a bottleneck (SSD recommended)

## Monitoring Key Metrics

### Performance Stats (every 3s)
- **workers**: Current active worker count
- **queued**: Items waiting in queues
- **dl/s**: Downloads per second (throughput)
- **Mbps**: Megabits per second (bandwidth)
- **success rate**: Percentage of successful downloads

### Network Stats (every 15s)
- **Queue utilization**: Percentage of queue capacity used
- Per-interface breakdown

### Memory Stats (every 20s)
- **Allocated**: Currently in use
- **System**: Total reserved from OS
- **GC count**: Number of garbage collections

## Troubleshooting

### "Too many open files" error
```bash
# Temporary fix
ulimit -n 65536

# Permanent fix (add to /etc/security/limits.conf)
* soft nofile 65536
* hard nofile 65536
```

### "Connection reset by peer" errors
- Reduce worker count
- Increase delay between requests
- Check if target site has rate limiting

### High memory usage
- Reduce buffer size
- Reduce max workers
- Increase GC frequency

### Low throughput despite high bandwidth
- Increase worker count
- Reduce delays
- Check if queue is empty (nothing to download)

## File Locations

| File | Purpose |
|------|---------|
| `visitedURLs_TIMESTAMP.txt` | Log of all visited URLs |
| `downloads_TIMESTAMP.txt` | Log of successfully downloaded files |
| `.colly_cache/` | Temporary cache (auto-cleaned) |
| Target directory | Downloaded documents |

## Code Organization Tips

### Adding a New Package
1. Create directory: `mkdir newpackage`
2. Create Go file: `touch newpackage/newpackage.go`
3. Define package: `package newpackage`
4. Import in main: `import "github.com/jeb/url_crawler/newpackage"`

### Adding New Configuration
1. Add constant to `config/config.go`
2. Use in relevant package: `config.YourNewSetting`

### Adding New Utility Function
1. Add to `utils/utils.go`
2. Export by capitalizing function name
3. Import where needed

## Dependencies

Main dependencies:
- `github.com/gocolly/colly`: Web scraping framework
- `golang.org/x/time/rate`: Rate limiting

To update dependencies:
```bash
go get -u ./...
go mod tidy
```

## Version History

- **v9.0 (Modular)**: Complete modularization into packages
- **v9.0 (Original)**: Monolithic multi-NIC beast mode implementation
