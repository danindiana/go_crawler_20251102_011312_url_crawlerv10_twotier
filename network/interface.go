package network

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jeb/url_crawler/config"
)

// NetworkInterface represents a network interface with its configuration
type NetworkInterface struct {
	Name        string
	IP          string
	IsActive    bool
	Speed       string
	WorkerCount int
	Clients     []*http.Client
}

// DetectNetworkInterfaces discovers available network interfaces
func DetectNetworkInterfaces() ([]NetworkInterface, error) {
	fmt.Println("\n🔍 Detecting network interfaces...")

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var networkInterfaces []NetworkInterface

	for _, iface := range interfaces {
		// Skip loopback and virtual interfaces, but keep tun for VPN
		if iface.Flags&net.FlagLoopback != 0 ||
			strings.Contains(iface.Name, "vir") ||
			strings.Contains(iface.Name, "docker") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		// Get first valid IP
		var ip string
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ip = ipnet.IP.String()
					break
				}
			}
		}

		// Skip if no IP and interface is not active
		if ip == "" && !(iface.Flags&net.FlagUp != 0) {
			continue
		}

		// Determine if interface is active and get speed info
		isActive := iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagRunning != 0
		speed := getInterfaceSpeed(iface.Name)

		networkInterfaces = append(networkInterfaces, NetworkInterface{
			Name:     iface.Name,
			IP:       ip,
			IsActive: isActive,
			Speed:    speed,
		})

		status := "DOWN"
		if isActive {
			status = "UP"
		}

		fmt.Printf("🌐 Found: %s (%s) - %s - %s\n", iface.Name, ip, status, speed)
	}

	return networkInterfaces, nil
}

// getInterfaceSpeed attempts to determine interface speed
func getInterfaceSpeed(ifname string) string {
	// Try to read speed from /sys
	speedPath := fmt.Sprintf("/sys/class/net/%s/speed", ifname)
	if data, err := os.ReadFile(speedPath); err == nil {
		if speed, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			if speed >= 10000 {
				return fmt.Sprintf("%dGbE", speed/1000)
			} else if speed >= 1000 {
				return fmt.Sprintf("%dGbE", speed/1000)
			} else {
				return fmt.Sprintf("%dMbE", speed)
			}
		}
	}

	// Fallback: guess based on interface name
	if strings.Contains(ifname, "enp3s0f") {
		return "10GbE" // X540-AT2
	} else if strings.Contains(ifname, "enp9s0") {
		return "1GbE" // I211
	}

	return "Unknown"
}

// SelectNetworkInterfaces lets user choose which interfaces to use
func SelectNetworkInterfaces(networkInterfaces []NetworkInterface) []int {
	fmt.Println("\n🎯 Select network interfaces for crawling:")
	fmt.Println("Available interfaces:")

	activeCount := 0
	for i, iface := range networkInterfaces {
		status := "❌"
		if iface.IsActive {
			status = "✅"
			activeCount++
		}
		fmt.Printf("%d) %s %s (%s) - %s - %s\n",
			i+1, status, iface.Name, iface.IP, iface.Speed,
			map[bool]string{true: "ACTIVE", false: "INACTIVE"}[iface.IsActive])
	}

	if activeCount == 0 {
		fmt.Println("❌ No active interfaces found!")
		return nil
	}

	fmt.Printf("\nRecommendation: Use all active high-speed interfaces for maximum performance\n")
	fmt.Printf("Enter interface numbers (comma-separated, e.g., 1,2,3) or 'all' for all active: ")

	var input string
	fmt.Scanln(&input)

	if input == "all" {
		var selected []int
		for i, iface := range networkInterfaces {
			if iface.IsActive {
				selected = append(selected, i)
			}
		}
		return selected
	}

	// Parse comma-separated list
	parts := strings.Split(input, ",")
	var selected []int
	for _, part := range parts {
		if num, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			if num > 0 && num <= len(networkInterfaces) {
				idx := num - 1
				if networkInterfaces[idx].IsActive {
					selected = append(selected, idx)
				} else {
					fmt.Printf("⚠️ Interface %s is not active, skipping\n", networkInterfaces[idx].Name)
				}
			}
		}
	}

	return selected
}

// ConfigureSelectedInterfaces sets up the selected network interfaces
func ConfigureSelectedInterfaces(networkInterfaces []NetworkInterface, selected []int) ([]NetworkInterface, error) {
	fmt.Println("\n⚙️ Configuring selected interfaces...")

	var activeInterfaces []NetworkInterface
	totalBandwidth := 0

	for _, idx := range selected {
		iface := networkInterfaces[idx]

		// Calculate worker distribution based on interface speed
		var workers int
		if strings.Contains(iface.Speed, "10G") {
			workers = 2000 // More workers for 10GbE
			totalBandwidth += 10000
		} else if strings.Contains(iface.Speed, "1G") {
			workers = 500 // Fewer workers for 1GbE
			totalBandwidth += 1000
		} else {
			workers = 200 // Conservative for unknown speed
			totalBandwidth += 100
		}

		iface.WorkerCount = workers
		activeInterfaces = append(activeInterfaces, iface)

		fmt.Printf("✅ %s (%s) - %s - %d workers\n",
			iface.Name, iface.IP, iface.Speed, workers)
	}

	fmt.Printf("🚀 Total bandwidth: %d Mbps across %d interfaces\n",
		totalBandwidth, len(activeInterfaces))

	return activeInterfaces, nil
}

// CreateInterfaceClient creates an HTTP client bound to a specific interface
func CreateInterfaceClient(iface NetworkInterface, numInterfaces int) *http.Client {
	// Create custom dialer that binds to specific interface
	localAddr, err := net.ResolveIPAddr("ip", iface.IP)
	if err != nil {
		fmt.Printf("⚠️ Warning: Could not resolve IP %s for %s\n", iface.IP, iface.Name)
		localAddr = nil
	}

	dialer := &net.Dialer{
		Timeout:   config.ConnectionTimeout,
		KeepAlive: config.KeepAliveTimeout,
	}

	// Bind to local interface IP if possible
	if localAddr != nil {
		dialer.LocalAddr = &net.TCPAddr{IP: localAddr.IP}
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          config.MaxConnectionsTotal / numInterfaces / 64,
		MaxIdleConnsPerHost:   config.MaxConnectionsPerHost / numInterfaces / 64,
		MaxConnsPerHost:       config.MaxConnectionsPerHost / numInterfaces / 64,
		IdleConnTimeout:       config.KeepAliveTimeout,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
		ForceAttemptHTTP2:     true,
	}

	return &http.Client{
		Timeout:   config.RequestTimeout,
		Transport: transport,
	}
}

// InitializeMultiNICSystem sets up queues and HTTP clients for each interface
func InitializeMultiNICSystem(networkInterfaces []NetworkInterface) []NetworkInterface {
	fmt.Println("\n🔧 Initializing multi-NIC system...")

	for i := range networkInterfaces {
		// Create HTTP clients for this interface
		clientCount := 64 // 64 clients per interface
		clients := make([]*http.Client, clientCount)

		for j := 0; j < clientCount; j++ {
			clients[j] = CreateInterfaceClient(networkInterfaces[i], len(networkInterfaces))
		}

		networkInterfaces[i].Clients = clients

		fmt.Printf("🌐 Interface %s: %d HTTP clients\n",
			networkInterfaces[i].Name, clientCount)
	}

	return networkInterfaces
}

// PrintNetworkStats displays network interface statistics
func PrintNetworkStats(networkInterfaces []NetworkInterface, downloadQueues []chan interface{}) {
	fmt.Printf("🌐 Network Status:\n")
	for i, iface := range networkInterfaces {
		queueLen := len(downloadQueues[i])
		queueCap := cap(downloadQueues[i])
		utilization := float64(queueLen) / float64(queueCap) * 100
		fmt.Printf("   %s (%s): Queue %d/%d (%.1f%%), %d clients\n",
			iface.Name, iface.Speed, queueLen, queueCap, utilization, len(iface.Clients))
	}
}
