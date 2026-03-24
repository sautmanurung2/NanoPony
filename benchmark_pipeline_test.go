package nanopony

import (
	"errors"
	"testing"
)

var errDataNil = errors.New("data cannot be nil")

// BenchmarkPipelineCreation tests memory allocation for pipeline creation
func BenchmarkPipelineCreation(b *testing.B) {
	b.ReportAllocs()
	
	processor := ProcessorFunc(func(data interface{}) error {
		return nil
	})
	
	for i := 0; i < b.N; i++ {
		pipeline := NewPipeline(processor)
		if pipeline == nil {
			b.Fatal("Expected pipeline to be created")
		}
	}
}

// BenchmarkPipelineProcess tests memory allocation for pipeline processing
func BenchmarkPipelineProcess(b *testing.B) {
	validator := ValidatorFunc(func(data interface{}) error {
		if data == nil {
			return errDataNil
		}
		return nil
	})
	
	transformer := TransformerFunc(func(data interface{}) (interface{}, error) {
		return data.(string) + "-transformed", nil
	})
	
	processor := ProcessorFunc(func(data interface{}) error {
		return nil
	})
	
	pipeline := NewPipeline(processor).
		AddValidator(validator).
		AddTransformer(transformer)
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		pipeline.Process("test-data")
	}
}

// BenchmarkPipelineProcessParallel tests concurrent pipeline processing
func BenchmarkPipelineProcessParallel(b *testing.B) {
	validator := ValidatorFunc(func(data interface{}) error {
		return nil
	})
	
	transformer := TransformerFunc(func(data interface{}) (interface{}, error) {
		return data.(string) + "-transformed", nil
	})
	
	processor := ProcessorFunc(func(data interface{}) error {
		return nil
	})
	
	pipeline := NewPipeline(processor).
		AddValidator(validator).
		AddTransformer(transformer)
	
	b.ReportAllocs()
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pipeline.Process("test-data")
		}
	})
}

// TestPipelineMemoryLeak detects memory leaks in pipeline
func TestPipelineMemoryLeak(t *testing.T) {
	for cycle := 0; cycle < 10; cycle++ {
		validator := ValidatorFunc(func(data interface{}) error {
			if data == nil {
				return errDataNil
			}
			return nil
		})
		
		transformer := TransformerFunc(func(data interface{}) (interface{}, error) {
			return data.(string) + "-transformed", nil
		})
		
		processor := ProcessorFunc(func(data interface{}) error {
			return nil
		})
		
		pipeline := NewPipeline(processor).
			AddValidator(validator).
			AddTransformer(transformer)
		
		for i := 0; i < 100; i++ {
			pipeline.Process("test-data")
		}
		
		t.Logf("Cycle %d completed", cycle)
	}
}
