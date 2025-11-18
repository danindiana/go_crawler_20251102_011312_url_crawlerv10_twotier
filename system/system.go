package system

import (
	"fmt"
	"syscall"

	"github.com/jeb/url_crawler/config"
)

// IncreaseFileDescriptorLimit increases system file descriptor limits
func IncreaseFileDescriptorLimit() {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Printf("⚠️ Could not get file descriptor limit: %v\n", err)
		return
	}

	fmt.Printf("📁 Current FD limit: %d\n", rLimit.Cur)

	rLimit.Cur = rLimit.Max
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Printf("⚠️ Could not increase FD limit: %v\n", err)
	} else {
		fmt.Printf("📁 Increased FD limit to: %d\n", rLimit.Cur)
	}
}

// OptimizeNetworkSettings displays recommended network optimizations
func OptimizeNetworkSettings() {
	fmt.Println("🔧 Optimizing network settings...")

	// These would require root privileges, so we'll just report what should be done
	optimizations := []string{
		"net.core.rmem_max = 134217728",
		"net.core.wmem_max = 134217728",
		"net.ipv4.tcp_rmem = 4096 87380 134217728",
		"net.ipv4.tcp_wmem = 4096 65536 134217728",
		"net.core.netdev_max_backlog = 30000",
		"net.core.netdev_budget = 600",
		"net.ipv4.tcp_congestion_control = bbr",
	}

	fmt.Println("💡 For optimal performance, run as root:")
	for _, opt := range optimizations {
		fmt.Printf("   sysctl -w %s\n", opt)
	}
}

// PrintSystemInfo displays system configuration information
func PrintSystemInfo(numCPU int) {
	fmt.Printf("🔥🔥🔥 MULTI-NIC BEAST MODE ACTIVATED! 🔥🔥🔥\n")
	fmt.Printf("🖥️ System: AMD Ryzen 9 5950X (%d cores) with 128GB RAM\n", numCPU)
	fmt.Printf("⚡ GOMAXPROCS: %d\n", numCPU*4)
	fmt.Printf("💾 Memory target: %dGB\n", config.TargetMemoryUsageGB)
}
