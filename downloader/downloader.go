package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jeb/url_crawler/config"
	"github.com/jeb/url_crawler/network"
	"github.com/jeb/url_crawler/utils"
	"golang.org/x/time/rate"
)

// DownloadTask represents a download task
type DownloadTask struct {
	URL         string
	Depth       int
	Retry       int
	Priority    bool
	InterfaceID int // Which network interface to use
}

// Manager manages the download system
type Manager struct {
	networkInterfaces     []network.NetworkInterface
	downloadQueues        []chan DownloadTask
	priorityQueue         chan DownloadTask
	downloadLimiter       *rate.Limiter
	downloadWG            sync.WaitGroup
	activeWorkers         int64
	shutdownChan          chan struct{}
	currentInterfaceIndex int64

	// State management
	downloadedFiles  map[string]bool
	pendingDownloads map[string]bool
	failedDownloads  map[string]int
	mapMutex         *sync.RWMutex

	// File paths
	targetDir       string
	downloadLogPath string

	// Statistics
	stats struct {
		downloadAttempts int64
		downloadSuccess  int64
		downloadFailed   int64
		bytesDownloaded  int64
		startTime        time.Time
	}
}

// NewManager creates a new download manager
func NewManager(networkInterfaces []network.NetworkInterface, targetDir, downloadLogPath string) *Manager {
	m := &Manager{
		networkInterfaces: networkInterfaces,
		downloadQueues:    make([]chan DownloadTask, len(networkInterfaces)),
		priorityQueue:     make(chan DownloadTask, config.MaxQueueSize),
		downloadedFiles:   make(map[string]bool),
		pendingDownloads:  make(map[string]bool),
		failedDownloads:   make(map[string]int),
		mapMutex:          &sync.RWMutex{},
		targetDir:         targetDir,
		downloadLogPath:   downloadLogPath,
		shutdownChan:      make(chan struct{}),
	}

	// Initialize queues for each interface
	for i := range networkInterfaces {
		queueSize := config.MaxQueueSize / len(networkInterfaces)
		m.downloadQueues[i] = make(chan DownloadTask, queueSize)
	}

	// Ultra-permissive rate limiting
	m.downloadLimiter = rate.NewLimiter(rate.Every(10*time.Microsecond), config.MaxDownloadWorkers*3)

	m.stats.startTime = time.Now()

	return m
}

// StartWorkers starts download workers distributed across interfaces
func (m *Manager) StartWorkers() {
	fmt.Println("\n👥 Starting multi-NIC workers...")

	totalWorkers := 0
	for i, iface := range m.networkInterfaces {
		workers := min(iface.WorkerCount, config.InitialDownloadWorkers/len(m.networkInterfaces)+100)
		for j := 0; j < workers; j++ {
			m.downloadWG.Add(1)
			go m.multiNICDownloadWorker(i, j%len(iface.Clients))
			atomic.AddInt64(&m.activeWorkers, 1)
			totalWorkers++
		}

		fmt.Printf("🚀 %s: Started %d workers\n", iface.Name, workers)
	}

	fmt.Printf("💪 Total workers started: %d\n", totalWorkers)
}

// multiNICDownloadWorker processes downloads on a specific network interface
func (m *Manager) multiNICDownloadWorker(interfaceID, clientIndex int) {
	defer m.downloadWG.Done()
	defer atomic.AddInt64(&m.activeWorkers, -1)

	iface := m.networkInterfaces[interfaceID]
	client := iface.Clients[clientIndex]
	workerName := fmt.Sprintf("%s-W%d", iface.Name, clientIndex)

	for {
		var task DownloadTask
		var ok bool

		// Check priority queue first
		select {
		case task, ok = <-m.priorityQueue:
			if !ok {
				// Priority queue closed
			} else {
				goto processTask
			}
		default:
			// Priority queue empty
		}

		// Check interface-specific queue
		select {
		case task, ok = <-m.downloadQueues[interfaceID]:
			if !ok {
				// Interface queue closed
				return
			}
		default:
			// No work available, sleep briefly
			time.Sleep(1 * time.Millisecond)
			continue
		}

	processTask:
		// Rate limiting
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		m.downloadLimiter.Wait(ctx)
		cancel()

		atomic.AddInt64(&m.stats.downloadAttempts, 1)

		err := m.downloadDocument(task.URL, client, workerName)
		if err != nil {
			atomic.AddInt64(&m.stats.downloadFailed, 1)

			if task.Retry < config.MaxRetries {
				task.Retry++
				task.Priority = true
				task.InterfaceID = interfaceID

				go func(t DownloadTask) {
					time.Sleep(config.RetryBackoff * time.Duration(t.Retry))
					select {
					case m.priorityQueue <- t:
						// Successfully re-queued
					default:
						m.markDownloadFailed(t.URL)
					}
				}(task)
			} else {
				m.markDownloadFailed(task.URL)
			}
		} else {
			atomic.AddInt64(&m.stats.downloadSuccess, 1)
			m.markDownloadCompleted(task.URL)
		}
	}
}

// downloadDocument downloads a document using the specified HTTP client
func (m *Manager) downloadDocument(docURL string, client *http.Client, workerName string) error {
	req, err := http.NewRequestWithContext(context.Background(), "GET", docURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", config.UserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	filename := utils.ExtractFilename(docURL, resp.Header)
	path := filepath.Join(m.targetDir, filename)

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	// Use massive buffer optimized for 10GbE
	buf := make([]byte, config.DownloadBufferSize)
	written, err := io.CopyBuffer(out, resp.Body, buf)

	if err == nil {
		atomic.AddInt64(&m.stats.bytesDownloaded, written)
	}

	return err
}

// EnqueueTask adds a task to the download queue
func (m *Manager) EnqueueTask(task DownloadTask) bool {
	if m.IsDownloadedOrPending(task.URL) {
		return false
	}

	// Load-balanced interface selection
	interfaceID := int(atomic.AddInt64(&m.currentInterfaceIndex, 1)) % len(m.networkInterfaces)
	task.InterfaceID = interfaceID

	// Try interface-specific queue
	select {
	case m.downloadQueues[interfaceID] <- task:
		m.markPendingDownload(task.URL)
		return true
	default:
		// Queue full, try priority queue
		select {
		case m.priorityQueue <- task:
			m.markPendingDownload(task.URL)
			return true
		default:
			// Both queues full
			return false
		}
	}
}

// PersistentEnqueue tries persistently to enqueue a task
func (m *Manager) PersistentEnqueue(task DownloadTask) {
	maxAttempts := 50
	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(time.Duration(attempt*50) * time.Millisecond)

		// Try priority queue first
		select {
		case m.priorityQueue <- task:
			m.markPendingDownload(task.URL)
			return
		default:
			// Try interface-specific queues
			for i := range m.downloadQueues {
				select {
				case m.downloadQueues[i] <- task:
					m.markPendingDownload(task.URL)
					return
				default:
					continue
				}
			}
		}
	}
	fmt.Printf("❌ [%d] Multi-NIC dropped after %d attempts: %s\n", task.Depth, maxAttempts, task.URL)
}

// IsDownloadedOrPending checks if a URL has been downloaded or is pending
func (m *Manager) IsDownloadedOrPending(url string) bool {
	m.mapMutex.RLock()
	defer m.mapMutex.RUnlock()
	_, downloaded := m.downloadedFiles[url]
	_, pending := m.pendingDownloads[url]
	return downloaded || pending
}

// markPendingDownload marks a URL as pending download
func (m *Manager) markPendingDownload(url string) {
	m.mapMutex.Lock()
	m.pendingDownloads[url] = true
	m.mapMutex.Unlock()
}

// markDownloadCompleted marks a download as completed
func (m *Manager) markDownloadCompleted(url string) {
	m.mapMutex.Lock()
	delete(m.pendingDownloads, url)
	m.downloadedFiles[url] = true
	m.mapMutex.Unlock()

	// Async logging to avoid blocking worker
	go func() {
		f, err := os.OpenFile(m.downloadLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()
		f.WriteString(url + "\n")
	}()
}

// markDownloadFailed marks a download as failed
func (m *Manager) markDownloadFailed(url string) {
	m.mapMutex.Lock()
	delete(m.pendingDownloads, url)
	m.failedDownloads[url]++
	m.mapMutex.Unlock()
}

// GetStats returns current download statistics
func (m *Manager) GetStats() (attempts, success, failed, bytes int64, elapsed time.Duration) {
	attempts = atomic.LoadInt64(&m.stats.downloadAttempts)
	success = atomic.LoadInt64(&m.stats.downloadSuccess)
	failed = atomic.LoadInt64(&m.stats.downloadFailed)
	bytes = atomic.LoadInt64(&m.stats.bytesDownloaded)
	elapsed = time.Since(m.stats.startTime)
	return
}

// GetActiveWorkers returns the number of active workers
func (m *Manager) GetActiveWorkers() int64 {
	return atomic.LoadInt64(&m.activeWorkers)
}

// GetQueueStatus returns queue length and capacity information
func (m *Manager) GetQueueStatus() (totalQueued, totalCapacity int) {
	totalQueued = len(m.priorityQueue)
	totalCapacity = cap(m.priorityQueue)

	for _, queue := range m.downloadQueues {
		totalQueued += len(queue)
		totalCapacity += cap(queue)
	}

	return
}

// GetDownloadQueues returns the download queues (for monitoring)
func (m *Manager) GetDownloadQueues() []chan DownloadTask {
	return m.downloadQueues
}

// GetPriorityQueue returns the priority queue
func (m *Manager) GetPriorityQueue() chan DownloadTask {
	return m.priorityQueue
}

// AddWorkers adds new workers to the system
func (m *Manager) AddWorkers(count int) {
	if count <= 0 {
		return
	}

	// Distribute workers across interfaces based on their capacity
	workersPerInterface := count / len(m.networkInterfaces)
	remainder := count % len(m.networkInterfaces)

	for i, iface := range m.networkInterfaces {
		workers := workersPerInterface
		if i < remainder {
			workers++ // Distribute remainder
		}

		for j := 0; j < workers; j++ {
			m.downloadWG.Add(1)
			go m.multiNICDownloadWorker(i, j%len(iface.Clients))
			atomic.AddInt64(&m.activeWorkers, 1)
		}
	}
}

// Shutdown gracefully shuts down the download manager
func (m *Manager) Shutdown() {
	close(m.shutdownChan)
	close(m.priorityQueue)
	for _, queue := range m.downloadQueues {
		close(queue)
	}
	m.downloadWG.Wait()
}

// Wait waits for all downloads to complete
func (m *Manager) Wait() {
	m.downloadWG.Wait()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
