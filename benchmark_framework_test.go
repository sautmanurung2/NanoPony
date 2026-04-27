package nanopony

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

// ==================== BENCHMARK TESTS ====================

// BenchmarkFrameworkCreation tests memory allocation for framework creation
func BenchmarkFrameworkCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		framework := NewFramework()
		if framework == nil {
			b.Fatal("Expected framework to be created")
		}
	}
}

// BenchmarkFrameworkBuild tests memory allocation for framework build
func BenchmarkFrameworkBuild(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ResetConfig()
		config := NewConfig()

		framework := NewFramework().
			WithConfig(config).
			WithWorkerPool(5, 100)

		components := framework.Build()
		if components == nil {
			b.Fatal("Expected components to be built")
		}
	}
}

// BenchmarkFrameworkWithConfig tests chaining WithConfig
func BenchmarkFrameworkWithConfig(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ResetConfig()
		config := NewConfig()

		framework := NewFramework().WithConfig(config)
		if framework.config == nil {
			b.Fatal("Expected config to be set")
		}
	}
}

// BenchmarkFrameworkWithDatabaseFromInstance tests setting DB connection
func BenchmarkFrameworkWithDatabaseFromInstance(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		framework := NewFramework().WithDatabaseFromInstance(nil)
		if framework.db != nil {
			b.Fatal("Expected db to be nil")
		}
	}
}

// BenchmarkFrameworkWithKafkaWriterFromInstance tests setting Kafka writer
func BenchmarkFrameworkWithKafkaWriterFromInstance(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		framework := NewFramework().WithKafkaWriterFromInstance(nil)
		if framework.kafkaWriter != nil {
			b.Fatal("Expected kafkaWriter to be nil")
		}
	}
}

// BenchmarkFrameworkWithProducerFromInstance tests setting producer
func BenchmarkFrameworkWithProducerFromInstance(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		framework := NewFramework().WithProducerFromInstance(nil)
		if framework.producer != nil {
			b.Fatal("Expected producer to be nil")
		}
	}
}

// BenchmarkFrameworkWithWorkerPoolFromInstance tests setting worker pool
func BenchmarkFrameworkWithWorkerPoolFromInstance(b *testing.B) {
	b.ReportAllocs()

	pool := NewWorkerPool(5, 100)

	for i := 0; i < b.N; i++ {
		framework := NewFramework().WithWorkerPoolFromInstance(pool)
		if framework.workerPool == nil {
			b.Fatal("Expected workerPool to be set")
		}
	}
}

// BenchmarkFrameworkAddCleanup tests adding cleanup functions
func BenchmarkFrameworkAddCleanup(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		framework := NewFramework()
		for j := 0; j < 10; j++ {
			framework.AddCleanup(func() error { return nil })
		}
	}
}

// BenchmarkFrameworkCompleteSetup tests full framework setup
func BenchmarkFrameworkCompleteSetup(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ResetConfig()
		config := NewConfig()

		framework := NewFramework().
			WithConfig(config).
			WithWorkerPool(5, 100)

		components := framework.Build()
		if components == nil {
			b.Fatal("Expected components to be built")
		}
	}
}

// BenchmarkFrameworkStartStop tests start and stop cycle
func BenchmarkFrameworkStartStop(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ResetConfig()
		config := NewConfig()

		framework := NewFramework().
			WithConfig(config).
			WithWorkerPool(2, 50)

		components := framework.Build()

		b.StartTimer()
		components.Start(ctx, func(ctx context.Context, job Job) error {
			return nil
		})
		b.StopTimer()

		time.Sleep(1 * time.Millisecond)

		b.StartTimer()
		components.Shutdown(ctx)
		b.StopTimer()
	}
}

// BenchmarkFrameworkShutdown tests shutdown performance
func BenchmarkFrameworkShutdown(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ResetConfig()
		config := NewConfig()

		framework := NewFramework().
			WithConfig(config).
			WithWorkerPool(2, 50).
			AddCleanup(func() error { return nil })

		components := framework.Build()
		components.Start(ctx, func(ctx context.Context, job Job) error {
			return nil
		})

		time.Sleep(1 * time.Millisecond)

		b.StartTimer()
		components.Shutdown(ctx)
		b.StopTimer()
	}
}

// BenchmarkFrameworkWithMultipleCleanup tests multiple cleanup functions
func BenchmarkFrameworkWithMultipleCleanup(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ResetConfig()
		config := NewConfig()

		framework := NewFramework().
			WithConfig(config).
			WithWorkerPool(2, 50)

		for j := 0; j < 5; j++ {
			framework.AddCleanup(func() error { return nil })
		}

		components := framework.Build()
		components.Start(ctx, func(ctx context.Context, job Job) error {
			return nil
		})

		time.Sleep(1 * time.Millisecond)

		b.StartTimer()
		components.Shutdown(ctx)
		b.StopTimer()
	}
}

// BenchmarkFrameworkGetters tests direct field access performance
func BenchmarkFrameworkGetters(b *testing.B) {
	b.ReportAllocs()
	ResetConfig()
	config := NewConfig()
	pool := NewWorkerPool(5, 100)

	components := &FrameworkComponents{
		Config:     config,
		DB:         nil,
		WorkerPool: pool,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = components.Config
		_ = components.DB
		_ = components.WorkerPool
		_ = components.Poller
		_ = components.Producer
	}
}

// ==================== MEMORY LEAK TESTS ====================

// TestFrameworkMemoryLeak detects memory leaks in framework
func TestFrameworkMemoryLeak(t *testing.T) {
	ctx := context.Background()

	for cycle := 0; cycle < 5; cycle++ {
		ResetConfig()
		config := NewConfig()

		framework := NewFramework().
			WithConfig(config).
			WithWorkerPool(5, 100)

		components := framework.Build()

		components.Start(ctx, func(ctx context.Context, job Job) error {
			return nil
		})

		time.Sleep(50 * time.Millisecond)

		if err := components.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown error: %v", err)
		}

		t.Logf("Cycle %d completed", cycle)
	}
}

// TestFrameworkMemoryLeakDetailed provides detailed memory tracking
func TestFrameworkMemoryLeakDetailed(t *testing.T) {
	ctx := context.Background()

	// Force GC before test
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialMem := m1.Alloc
	t.Logf("Initial memory: %d KB", initialMem/1024)

	// Run multiple cycles to detect leaks
	cycles := 50
	for cycle := 0; cycle < cycles; cycle++ {
		ResetConfig()
		config := NewConfig()

		framework := NewFramework().
			WithConfig(config).
			WithWorkerPool(5, 200)

		for i := 0; i < 3; i++ {
			framework.AddCleanup(func() error { return nil })
		}

		components := framework.Build()

		components.Start(ctx, func(ctx context.Context, job Job) error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})

		time.Sleep(20 * time.Millisecond)

		if err := components.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown error on cycle %d: %v", cycle, err)
		}

		if cycle%10 == 0 {
			runtime.GC()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			t.Logf("Cycle %d - Memory: %d KB", cycle, m.Alloc/1024)
		}
	}

	// Force GC after test
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalMem := m2.Alloc
	t.Logf("Final memory: %d KB", finalMem/1024)

	memGrowth := int64(finalMem) - int64(initialMem)
	t.Logf("Memory growth: %d KB", memGrowth/1024)

	// Allow max 1 MB growth for 50 cycles
	if memGrowth > 1*1024*1024 {
		t.Errorf("Potential memory leak detected: %d KB growth over %d cycles",
			memGrowth/1024, cycles)
	}
}

// TestFrameworkWorkerPoolMemoryLeak tests worker pool memory in framework context
func TestFrameworkWorkerPoolMemoryLeak(t *testing.T) {
	ctx := context.Background()

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialMem := m1.Alloc

	// Submit many jobs through framework
	ResetConfig()
	config := NewConfig()

	framework := NewFramework().
		WithConfig(config).
		WithWorkerPool(10, 500)

	components := framework.Build()

	components.Start(ctx, func(ctx context.Context, job Job) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	})

	// Submit 1000 jobs
	for i := 0; i < 1000; i++ {
		components.WorkerPool.Submit(ctx, Job{
			ID:   fmt.Sprintf("job-%d", i),
			Data: map[string]interface{}{"index": i, "data": make([]byte, 100)},
		})
	}

	time.Sleep(500 * time.Millisecond)

	if err := components.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown error: %v", err)
	}

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalMem := m2.Alloc

	memGrowth := int64(finalMem) - int64(initialMem)
	t.Logf("Memory growth after 1000 jobs: %d KB", memGrowth/1024)

	if memGrowth > 500*1024 {
		t.Errorf("Potential memory leak: %d KB growth", memGrowth/1024)
	}
}

// TestFrameworkConcurrentMemoryLeak tests concurrent framework usage
func TestFrameworkConcurrentMemoryLeak(t *testing.T) {
	ctx := context.Background()

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialMem := m1.Alloc

	// Run multiple frameworks concurrently
	// Use BuildConfig() to avoid race condition on config singleton
	concurrent := 20
	done := make(chan bool, concurrent)

	for i := 0; i < concurrent; i++ {
		go func(idx int) {
			// Use BuildConfig() to create an isolated config (no singleton race)
			config := BuildConfig()

			framework := NewFramework().
				WithConfig(config).
				WithWorkerPool(2, 100)

			components := framework.Build()

			components.Start(ctx, func(ctx context.Context, job Job) error {
				time.Sleep(1 * time.Millisecond)
				return nil
			})

			time.Sleep(50 * time.Millisecond)
			components.Shutdown(ctx)
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < concurrent; i++ {
		<-done
	}

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalMem := m2.Alloc

	memGrowth := int64(finalMem) - int64(initialMem)
	t.Logf("Memory growth from %d concurrent frameworks: %d KB", concurrent, memGrowth/1024)

	if memGrowth > 1*1024*1024 {
		t.Errorf("Potential memory leak: %d KB growth", memGrowth/1024)
	}
}
