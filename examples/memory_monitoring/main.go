// Example: Memory monitoring with NanoPony framework
// Run: go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sautmanurung2/nanopony"
)

func main() {
	// Set environment variables (in production, use .env file)
	os.Setenv("GO_ENV", "local")
	os.Setenv("KAFKA-MODELS", "kafka-localhost")
	os.Setenv("KAFKA_BROKERS_STAGING", "localhost:9092")
	os.Setenv("HOST_STAGING", "localhost")
	os.Setenv("PORT_STAGING", "1521")
	os.Setenv("DATABASE_STAGING", "ORCL")
	os.Setenv("USERNAME_STAGING", "user")
	os.Setenv("PASSWORD_STAGING", "password")

	// Initialize configuration
	config := nanopony.NewConfig()

	// Create framework with builder pattern
	framework := nanopony.NewFramework().
		WithConfig(config).
		WithWorkerPool(5, 100).
		WithPoller(nanopony.DefaultPollerConfig(), &exampleDataFetcher{})

	// Build components
	components := framework.Build()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start memory monitoring (logs every 5 seconds)
	log.Println("Starting memory monitor...")
	stopMonitor := nanopony.MonitorMemory(5 * time.Second)
	defer func() {
		log.Println("Stopping memory monitor...")
		stopMonitor()
	}()

	// Print initial memory stats
	nanopony.PrintMemoryStats()

	// Start framework
	components.Start(ctx, exampleJobHandler)

	log.Println("NanoPony framework started with memory monitoring...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	// Print final memory stats
	log.Println("Final memory statistics:")
	nanopony.PrintMemoryStats()

	// Graceful shutdown
	if err := components.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Framework stopped")
}

// exampleDataFetcher implements nanopony.DataFetcher interface
type exampleDataFetcher struct{}

func (f *exampleDataFetcher) Fetch() ([]interface{}, error) {
	// Fetch data from database or other source
	return []interface{}{
		map[string]interface{}{"id": 1, "data": "example"},
	}, nil
}

// exampleJobHandler handles jobs from the worker pool
func exampleJobHandler(ctx context.Context, job nanopony.Job) error {
	fmt.Printf("Processing job: %+v\n", job)
	time.Sleep(100 * time.Millisecond) // Simulate work
	return nil
}
