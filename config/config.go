package config

import "time"

// MULTI-NIC BEAST MODE CONFIGURATION (Conservative for colly v1.2.0 stability)
const (
	MaxDepth       = 13               // Max crawl depth
	RequestTimeout = 60 * time.Second // Longer timeout for large files

	// CRITICAL FIX #2: Further reduced from 50 to 20
	// Even 50 workers overwhelms colly v1.2.0's internal channel buffers
	// Colly v1.2.0 appears to have ~10-20 item response queue
	ConcurrentWorkers = 20 // Ultra-safe limit for colly v1.2.0

	PoliteDelay = 30 * time.Millisecond // Aggressive crawling
	UserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0 Safari/537.36"

	// MULTI-NIC download configuration - UNCHANGED (this works fine)
	InitialDownloadWorkers = 100                    // Start with 100 workers
	MaxDownloadWorkers     = 800                    // Scale up to 800 concurrent downloads
	QueueGrowthThreshold   = 0.4                    // Scale at 40% full
	ScaleCheckInterval     = 500 * time.Millisecond // Check twice per second
	ScaleUpAmount          = 300                    // Add 300 workers at a time
	MaxQueueSize           = 50000                  // 50K item queue

	// Multi-NIC network beast mode
	MaxConnectionsTotal   = 12000             // 12K total connections across all NICs
	MaxConnectionsPerHost = 1200              // 1.2K per host
	ConnectionTimeout     = 3 * time.Second   // Ultra-fast connection establishment
	KeepAliveTimeout      = 300 * time.Second // 5-minute keep-alive

	// Hardware-optimized settings
	DownloadBufferSize = 32 * 1024 * 1024       // 32MB buffer for 10GbE
	MaxRetries         = 3                      // Fewer retries for speed
	RetryBackoff       = 300 * time.Millisecond // Very fast retry

	// Memory settings
	TargetMemoryUsageGB = 50  // Use up to 50GB of RAM
	GCTargetPercent     = 100 // Less frequent GC
)
