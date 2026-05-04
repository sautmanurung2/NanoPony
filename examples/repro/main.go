package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/sautmanurung2/nanopony"
)

func main() {
	// Initialize config
	nanopony.NewConfig()

	fmt.Println("Starting memory stress test...")
	printMem("Initial")

	// Stress the logger
	go func() {
		logger := nanopony.NewLogger("StressTest", "admin", "123", "456", "System", "Process", "Entity", "Add")
		
		// Large payload
		payload := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			payload[fmt.Sprintf("key_%d", i)] = "This is a relatively large string value to increase memory usage during marshaling and cloning operations in the logging pipeline."
		}

		for i := 0; i < 1000000; i++ {
			logger.LoggingData("info", payload, nanopony.ResponseLog{
				Status:  "success",
				Message: fmt.Sprintf("Log message %d", i),
			})
			if i%50000 == 0 {
				fmt.Printf("Logged %d entries...\n", i)
			}
		}
	}()

	// Monitor for a bit
	for i := 0; i < 60; i++ {
		time.Sleep(1 * time.Second)
		printMem(fmt.Sprintf("After %ds", i+1))
	}
}

func printMem(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("[%s] Alloc: %v MB, Sys: %v MB, NumGC: %v, Goroutines: %v\n",
		label, m.Alloc/1024/1024, m.Sys/1024/1024, m.NumGC, runtime.NumGoroutine())
}
