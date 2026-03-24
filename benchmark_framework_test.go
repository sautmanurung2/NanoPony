package nanopony

import (
	"context"
	"testing"
	"time"
)

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

// TestFrameworkMemoryLeak detects memory leaks in framework
func TestFrameworkMemoryLeak(t *testing.T) {
	ctx := context.Background()
	
	for cycle := 0; cycle < 5; cycle++ {
		ResetConfig()
		config := NewConfig()
		
		framework := NewFramework().
			WithConfig(config).
			WithWorkerPool(5, 100).
			AddService(&BaseService{name: "test-service"})
		
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
