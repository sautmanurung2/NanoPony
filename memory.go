package nanopony

import (
	"fmt"
	"runtime"
	"time"
)

// MemoryStats holds memory usage statistics
type MemoryStats struct {
	Alloc      uint64 // Memory currently allocated (bytes)
	TotalAlloc uint64 // Total memory allocated over time (bytes)
	Sys        uint64 // Total memory obtained from OS (bytes)
	NumGC      uint32 // Number of GC cycles
}

// GetMemoryStats returns current memory statistics
func GetMemoryStats() *MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return &MemoryStats{
		Alloc:      m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
	}
}

// FormatBytes converts bytes to human-readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// PrintMemoryStats prints current memory usage
func PrintMemoryStats() {
	stats := GetMemoryStats()
	fmt.Println("=== Memory Usage Statistics ===")
	fmt.Printf("Current Allocated Memory: %s\n", FormatBytes(stats.Alloc))
	fmt.Printf("Total Allocated Memory:   %s\n", FormatBytes(stats.TotalAlloc))
	fmt.Printf("Memory from OS:           %s\n", FormatBytes(stats.Sys))
	fmt.Printf("Number of GC Cycles:      %d\n", stats.NumGC)
	fmt.Printf("Number of Goroutines:     %d\n", runtime.NumGoroutine())
	fmt.Println("===============================")
}

// MonitorMemory starts a background goroutine that logs memory usage at intervals
// Returns a stop function to halt monitoring
func MonitorMemory(interval time.Duration) func() {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		fmt.Println("\n[Memory Monitor] Started monitoring...")

		for {
			select {
			case <-ticker.C:
				stats := GetMemoryStats()
				fmt.Printf("[Memory Monitor] Alloc: %s | TotalAlloc: %s | Sys: %s | GC: %d | Goroutines: %d\n",
					FormatBytes(stats.Alloc),
					FormatBytes(stats.TotalAlloc),
					FormatBytes(stats.Sys),
					stats.NumGC,
					runtime.NumGoroutine(),
				)
			case <-stop:
				fmt.Println("[Memory Monitor] Stopped monitoring")
				return
			}
		}
	}()

	return func() {
		close(stop)
	}
}
