// TWO-TIER TOKENIZER VERSION
package main

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/jeb/url_crawler/crawler"
	"github.com/jeb/url_crawler/downloader"
	"github.com/jeb/url_crawler/monitor"
	"github.com/jeb/url_crawler/network"
	"github.com/jeb/url_crawler/system"
)

func main() {
	// BEAST MODE SYSTEM CONFIGURATION
	monitor.SetupBeastMode()

	system.PrintSystemInfo(runtime.NumCPU())

	// Detect and configure network interfaces
	networkInterfaces, err := network.DetectNetworkInterfaces()
	if err != nil {
		fmt.Printf("❌ Failed to detect network interfaces: %v\n", err)
		return
	}

	// Let user select which interfaces to use
	selectedInterfaces := network.SelectNetworkInterfaces(networkInterfaces)
	if len(selectedInterfaces) == 0 {
		fmt.Println("❌ No network interfaces selected")
		return
	}

	// Configure selected interfaces
	networkInterfaces, err = network.ConfigureSelectedInterfaces(networkInterfaces, selectedInterfaces)
	if err != nil {
		fmt.Printf("❌ Failed to configure interfaces: %v\n", err)
		return
	}

	// Increase system limits
	system.IncreaseFileDescriptorLimit()
	system.OptimizeNetworkSettings()

	// Get user input
	var startURL, targetDir string
	fmt.Println("\nEnter the starting URL to crawl:")
	fmt.Scanln(&startURL)

	fmt.Println("Enter the target directory to save files:")
	fmt.Scanln(&targetDir)

	// URL validation
	parsedStart, err := url.Parse(startURL)
	if err != nil || parsedStart.Scheme == "" || parsedStart.Host == "" {
		fmt.Printf("❌ Invalid URL: %s\n", startURL)
		return
	}
	if parsedStart.Scheme != "http" && parsedStart.Scheme != "https" {
		parsedStart.Scheme = "https"
		startURL = parsedStart.String()
	}

	// Create target directory
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Printf("📁 Creating directory: %s\n", targetDir)
		err = os.MkdirAll(targetDir, 0755)
		if err != nil {
			fmt.Printf("❌ Failed to create directory: %v\n", err)
			return
		}
	}

	// Initialize log files
	timestamp := time.Now().Format("20060102_150405")
	logFilePath := fmt.Sprintf("visitedURLs_%s.txt", timestamp)
	downloadLogPath := fmt.Sprintf("downloads_%s.txt", timestamp)

	// Initialize multi-NIC system
	networkInterfaces = network.InitializeMultiNICSystem(networkInterfaces)

	// Create download manager
	downloadManager := downloader.NewManager(networkInterfaces, targetDir, downloadLogPath)

	// Start download workers
	downloadManager.StartWorkers()

	// Create shutdown channel
	shutdownChan := make(chan struct{})

	// Create and start monitor system
	monitorSystem := monitor.NewMonitor(downloadManager, networkInterfaces, shutdownChan)
	monitorSystem.StartMonitoring(16) // 16 concurrent scalers for ultra-fast response

	// Create crawler
	webCrawler := crawler.NewCrawlerTwoTier(startURL, logFilePath, downloadManager)

	// UNLEASH THE MULTI-NIC BEAST!
	monitor.PrintStartupInfo(startURL, targetDir, networkInterfaces)

	err = webCrawler.Start()
	if err != nil {
		fmt.Printf("❌ Failed to start crawl: %v\n", err)
		return
	}

	// Wait for crawling to complete
	webCrawler.Wait()

	// Shutdown sequence
	close(shutdownChan)
	monitorSystem.Wait()
	downloadManager.Shutdown()

	// Print final statistics
	monitor.PrintFinalStats(downloadManager, networkInterfaces)
}
