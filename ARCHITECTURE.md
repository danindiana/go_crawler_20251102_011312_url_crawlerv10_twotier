# URL Crawler Architecture

## Package Dependency Graph

```
                    ┌─────────────┐
                    │   main.go   │
                    │ (entry point)│
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
         ▼                 ▼                 ▼
    ┌────────┐      ┌──────────┐     ┌──────────┐
    │ system │      │ network  │     │  config  │
    └────────┘      └─────┬────┘     └────┬─────┘
                          │                │
                          │                │ (used by all)
         ┌────────────────┼────────────────┘
         │                │
         ▼                ▼
    ┌──────────┐    ┌──────────┐
    │downloader│◄───│ crawler  │
    └────┬─────┘    └──────────┘
         │                │
         │                │
         ▼                ▼
    ┌─────────┐      ┌────────┐
    │ monitor │      │ utils  │
    └─────────┘      └────────┘
```

## Data Flow

```
1. User Input → main.go
2. Network Detection → network.DetectNetworkInterfaces()
3. Interface Selection → network.SelectNetworkInterfaces()
4. System Setup → system.IncreaseFileDescriptorLimit()
5. Download Manager Init → downloader.NewManager()
6. Monitor Start → monitor.StartMonitoring()
7. Crawler Start → crawler.NewCrawler() → crawler.Start()
8. Crawling Loop:
   ├─ Discover Links → crawler (OnHTML)
   ├─ Find Documents → crawler (OnHTML)
   ├─ Queue Downloads → downloader.EnqueueTask()
   ├─ Worker Processing → downloader.multiNICDownloadWorker()
   ├─ Auto-Scaling → monitor.checkAndScaleMultiNIC()
   └─ Statistics → monitor.printPerformanceStats()
9. Shutdown → monitor.Wait() → downloader.Shutdown()
10. Final Stats → monitor.PrintFinalStats()
```

## Module Interactions

### Main → All Packages
- Orchestrates the entire application lifecycle
- Initializes all major components
- Manages shutdown sequence

### Network Package
- Used by: main, downloader, monitor
- Detects and configures network interfaces
- Creates HTTP clients bound to specific interfaces

### Downloader Package
- Used by: main, crawler, monitor
- Manages download queue and workers
- Tracks download state (pending, completed, failed)
- Handles retries and rate limiting

### Crawler Package
- Used by: main
- Uses: downloader (to queue documents)
- Manages URL crawling and link discovery
- Detects downloadable documents

### Monitor Package
- Used by: main
- Uses: downloader, network
- Auto-scales workers based on queue utilization
- Displays performance statistics
- Manages memory and resource monitoring

### Utils Package
- Used by: downloader, monitor, crawler
- Provides common utilities
- No dependencies on other packages

### Config Package
- Used by: All packages
- Central configuration constants
- No dependencies on other packages

### System Package
- Used by: main
- System optimization functions
- No dependencies on application packages

## Key Design Patterns

### 1. Manager Pattern
- `downloader.Manager`: Encapsulates download system state and operations
- `monitor.Monitor`: Encapsulates monitoring state and operations

### 2. Factory Pattern
- `network.CreateInterfaceClient()`: Creates configured HTTP clients
- `crawler.NewCrawler()`: Creates configured crawler instance

### 3. Worker Pool Pattern
- `downloader.multiNICDownloadWorker()`: Multiple workers process download queue
- Dynamic scaling based on load

### 4. Observer Pattern
- `crawler` callbacks: React to crawl events
- `monitor` tickers: Periodic system observation

### 5. Singleton-like Pattern
- Global shutdown channel coordinated across packages
- Single download manager instance

## Benefits of This Architecture

1. **Separation of Concerns**: Each package has a clear, single responsibility
2. **Testability**: Packages can be tested in isolation
3. **Reusability**: Packages can be used in other projects
4. **Maintainability**: Changes are localized to specific packages
5. **Scalability**: Easy to add new features or modify existing ones
6. **Readability**: Clear module boundaries and dependencies
