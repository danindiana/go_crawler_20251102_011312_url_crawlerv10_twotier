package monitor

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/jeb/url_crawler/config"
	"github.com/jeb/url_crawler/downloader"
	"github.com/jeb/url_crawler/network"
	"github.com/jeb/url_crawler/utils"
)

// Monitor manages system monitoring and auto-scaling
type Monitor struct {
	downloadManager   *downloader.Manager
	networkInterfaces []network.NetworkInterface
	shutdownChan      chan struct{}
	wg                sync.WaitGroup
}

// NewMonitor creates a new monitor instance
func NewMonitor(downloadManager *downloader.Manager, networkInterfaces []network.NetworkInterface, shutdownChan chan struct{}) *Monitor {
	return &Monitor{
		downloadManager:   downloadManager,
		networkInterfaces: networkInterfaces,
		shutdownChan:      shutdownChan,
	}
}

// StartMonitoring starts all monitoring goroutines
func (m *Monitor) StartMonitoring(scalerCount int) {
	// Start multiple scalers for ultra-fast response
	for i := 0; i < scalerCount; i++ {
		m.wg.Add(1)
		go m.downloadScaler()
	}

	m.wg.Add(1)
	go m.performanceMonitor()

	m.wg.Add(1)
	go m.memoryMonitor()

	m.wg.Add(1)
	go m.networkMonitor()
}

// Wait waits for all monitoring goroutines to complete
func (m *Monitor) Wait() {
	m.wg.Wait()
}

// downloadScaler automatically scales workers based on queue utilization
func (m *Monitor) downloadScaler() {
	defer m.wg.Done()
	ticker := time.NewTicker(config.ScaleCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.shutdownChan:
			return
		case <-ticker.C:
			m.checkAndScaleMultiNIC()
		}
	}
}

// checkAndScaleMultiNIC checks queue utilization and scales workers
func (m *Monitor) checkAndScaleMultiNIC() {
	totalQueued, totalCapacity := m.downloadManager.GetQueueStatus()
	utilization := float64(totalQueued) / float64(totalCapacity)
	currentWorkers := m.downloadManager.GetActiveWorkers()

	if utilization > config.QueueGrowthThreshold && currentWorkers < config.MaxDownloadWorkers {
		// Determine scale amount based on utilization
		scaleAmount := config.ScaleUpAmount
		if utilization > 0.8 {
			scaleAmount = config.ScaleUpAmount * 4 // Quad scaling when critically full
		} else if utilization > 0.6 {
			scaleAmount = config.ScaleUpAmount * 2 // Double scaling when very full
		}

		newWorkersTotal := min(scaleAmount, config.MaxDownloadWorkers-int(currentWorkers))
		if newWorkersTotal > 0 {
			m.downloadManager.AddWorkers(newWorkersTotal)
			fmt.Printf("📈 Multi-NIC scaled: +%d workers across %d interfaces (util: %.1f%%)\n",
				newWorkersTotal, len(m.networkInterfaces), utilization*100)
		}
	}
}

// ForceScaleUp performs emergency scaling
func (m *Monitor) ForceScaleUp() {
	currentWorkers := m.downloadManager.GetActiveWorkers()
	if currentWorkers < config.MaxDownloadWorkers {
		newWorkersTotal := min(config.ScaleUpAmount*3, config.MaxDownloadWorkers-int(currentWorkers))
		if newWorkersTotal > 0 {
			m.downloadManager.AddWorkers(newWorkersTotal)
			fmt.Printf("🚀 EMERGENCY Multi-NIC scale: +%d workers (now %d)\n",
				newWorkersTotal, currentWorkers+int64(newWorkersTotal))
		}
	}
}

// performanceMonitor displays performance statistics
func (m *Monitor) performanceMonitor() {
	defer m.wg.Done()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.shutdownChan:
			return
		case <-ticker.C:
			m.printPerformanceStats()
		}
	}
}

// printPerformanceStats prints current performance statistics
func (m *Monitor) printPerformanceStats() {
	attempts, success, failed, bytes, elapsed := m.downloadManager.GetStats()
	workers := m.downloadManager.GetActiveWorkers()
	totalQueued, _ := m.downloadManager.GetQueueStatus()

	if attempts > 0 {
		successRate := float64(success) / float64(attempts) * 100
		throughput := float64(success) / elapsed.Seconds()
		mbps := float64(bytes) * 8 / elapsed.Seconds() / 1024 / 1024 // Mbps

		fmt.Printf("🔥 MULTI-NIC: %d workers, %d queued | %d attempts, %d success, %d failed (%.1f%%) | %.1f dl/s, %.1f Mbps | %s\n",
			workers, totalQueued, attempts, success, failed, successRate, throughput, mbps, utils.FormatBytes(bytes))
	}
}

// memoryMonitor monitors memory usage and triggers GC when needed
func (m *Monitor) memoryMonitor() {
	defer m.wg.Done()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.shutdownChan:
			return
		case <-ticker.C:
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			allocGB := float64(memStats.Alloc) / 1024 / 1024 / 1024
			sysGB := float64(memStats.Sys) / 1024 / 1024 / 1024

			fmt.Printf("🧠 Memory: %.1fGB allocated, %.1fGB system (target: %dGB), GC: %d\n",
				allocGB, sysGB, config.TargetMemoryUsageGB, memStats.NumGC)

			if allocGB > float64(config.TargetMemoryUsageGB)*0.95 {
				fmt.Printf("🧹 Triggering GC (approaching %dGB limit)\n", config.TargetMemoryUsageGB)
				runtime.GC()
			}
		}
	}
}

// networkMonitor displays network interface statistics
func (m *Monitor) networkMonitor() {
	defer m.wg.Done()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.shutdownChan:
			return
		case <-ticker.C:
			m.printNetworkStats()
		}
	}
}

// printNetworkStats prints network interface statistics
func (m *Monitor) printNetworkStats() {
	fmt.Printf("🌐 Network Status:\n")
	downloadQueues := m.downloadManager.GetDownloadQueues()
	for i, iface := range m.networkInterfaces {
		queueLen := len(downloadQueues[i])
		queueCap := cap(downloadQueues[i])
		utilization := float64(queueLen) / float64(queueCap) * 100
		fmt.Printf("   %s (%s): Queue %d/%d (%.1f%%), %d clients\n",
			iface.Name, iface.Speed, queueLen, queueCap, utilization, len(iface.Clients))
	}
}

// PrintStartupInfo displays startup information
func PrintStartupInfo(startURL, targetDir string, networkInterfaces []network.NetworkInterface) {
	fmt.Printf("\n🔥🔥🔥 MULTI-NIC BEAST UNLEASHED! 🔥🔥🔥\n")
	fmt.Printf("🎯 Target: %s (max depth %d)\n", startURL, config.MaxDepth)
	fmt.Printf("📁 Output: %s\n", targetDir)
	fmt.Printf("👥 Workers: %d initial → %d max\n", config.InitialDownloadWorkers, config.MaxDownloadWorkers)
	fmt.Printf("🌐 Interfaces: %d active\n", len(networkInterfaces))
	for _, iface := range networkInterfaces {
		fmt.Printf("   • %s (%s) - %s - %d workers\n",
			iface.Name, iface.IP, iface.Speed, iface.WorkerCount)
	}
	fmt.Printf("⚡ Crawl delay: %v (INSANE MODE)\n", config.PoliteDelay)
	fmt.Printf("💾 Buffer size: %dMB per download\n", config.DownloadBufferSize/1024/1024)
	fmt.Printf("📦 Total queue capacity: %d items\n\n", config.MaxQueueSize)
}

// PrintFinalStats displays final statistics
func PrintFinalStats(downloadManager *downloader.Manager, networkInterfaces []network.NetworkInterface) {
	attempts, success, failed, bytes, elapsed := downloadManager.GetStats()

	fmt.Printf("\n🔥🔥🔥 MULTI-NIC BEAST MODE COMPLETE! 🔥🔥🔥\n")
	fmt.Printf("⏱️ Total time: %v\n", elapsed)
	fmt.Printf("📊 Downloads: %d attempts, %d success, %d failed\n", attempts, success, failed)
	fmt.Printf("💾 Data downloaded: %s\n", utils.FormatBytes(bytes))
	fmt.Printf("⚡ Average throughput: %.2f downloads/sec\n", float64(success)/elapsed.Seconds())
	fmt.Printf("🌐 Average bandwidth: %.2f Mbps\n", float64(bytes)*8/elapsed.Seconds()/1024/1024)
	fmt.Printf("💪 Peak workers: %d across %d interfaces\n", downloadManager.GetActiveWorkers(), len(networkInterfaces))
	fmt.Printf("🧠 Final memory: %s\n", utils.FormatMemory(utils.GetMemStats()))

	fmt.Printf("\n🌐 Per-Interface Stats:\n")
	for _, iface := range networkInterfaces {
		fmt.Printf("   %s (%s): %s - %d workers configured\n",
			iface.Name, iface.IP, iface.Speed, iface.WorkerCount)
	}
}

// SetupBeastMode configures system for maximum performance
func SetupBeastMode() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 4) // Even more OS threads for networking

	// Optimize GC for high throughput
	runtime.GC()
	debug := os.Getenv("GODEBUG")
	if debug == "" {
		os.Setenv("GODEBUG", fmt.Sprintf("gctrace=0,gcpacertarget=%d", config.GCTargetPercent))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
