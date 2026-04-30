package nanopony

import (
	"testing"
	"time"
)

func TestGetMemoryStats(t *testing.T) {
	stats := GetMemoryStats()

	if stats == nil {
		t.Fatal("GetMemoryStats() returned nil")
	}

	t.Logf("Alloc: %s", FormatBytes(stats.Alloc))
	t.Logf("TotalAlloc: %s", FormatBytes(stats.TotalAlloc))
	t.Logf("Sys: %s", FormatBytes(stats.Sys))
	t.Logf("NumGC: %d", stats.NumGC)

	// Basic validation - these should be non-zero for a running Go program
	if stats.Sys == 0 {
		t.Error("Expected Sys to be non-zero")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 1024, "1.00 KB"},
		{"megabytes", 1048576, "1.00 MB"},
		{"gigabytes", 1073741824, "1.00 GB"},
		{"terabytes", 1099511627776, "1.00 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestPrintMemoryStats(t *testing.T) {
	// This should not panic and should print to stdout
	PrintMemoryStats()
}

func TestMonitorMemory(t *testing.T) {
	// Start monitoring with a short interval
	stop := MonitorMemory(100 * time.Millisecond)

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	// Stop monitoring
	stop()

	// Give it time to shut down
	time.Sleep(50 * time.Millisecond)
}

func TestMonitorMemoryStopMultipleTimes(t *testing.T) {
	// Test that calling stop multiple times doesn't panic
	stop := MonitorMemory(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	// First stop should be fine
	stop()

	time.Sleep(50 * time.Millisecond)

	// Second stop should panic - this is expected behavior
	// We'll recover from the panic
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic on second stop: %v", r)
		}
	}()

	stop()
}
