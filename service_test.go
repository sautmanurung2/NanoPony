package nanopony

import (
	"context"
	"errors"
	"testing"
)

func TestBaseService(t *testing.T) {
	service := NewBaseService("test-service")

	if service.ServiceName() != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", service.ServiceName())
	}

	if err := service.Initialize(); err != nil {
		t.Errorf("Expected no error from Initialize, got %v", err)
	}

	if err := service.Shutdown(); err != nil {
		t.Errorf("Expected no error from Shutdown, got %v", err)
	}
}

func TestBaseRepository(t *testing.T) {
	repo := NewBaseRepository(nil)

	if err := repo.Close(); err != nil {
		t.Errorf("Expected no error from Close, got %v", err)
	}
}

func TestProcessorFunc(t *testing.T) {
	processed := false
	processor := ProcessorFunc(func(data interface{}) error {
		processed = true
		return nil
	})

	err := processor.Process("test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !processed {
		t.Error("Expected processor to be called")
	}
}

func TestProcessorFuncWithContext(t *testing.T) {
	processed := false
	processor := ProcessorFunc(func(data interface{}) error {
		processed = true
		return nil
	})

	err := processor.ProcessWithContext(context.Background(), "test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !processed {
		t.Error("Expected processor to be called")
	}
}

func TestBatchProcessorFunc(t *testing.T) {
	processed := false
	processor := BatchProcessorFunc(func(data []interface{}) error {
		processed = true
		return nil
	})

	err := processor.ProcessBatch([]interface{}{"test"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !processed {
		t.Error("Expected processor to be called")
	}
}

func TestValidatorFunc(t *testing.T) {
	validator := ValidatorFunc(func(data interface{}) error {
		if data == nil {
			return errors.New("data cannot be nil")
		}
		return nil
	})

	err := validator.Validate("test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = validator.Validate(nil)
	if err == nil {
		t.Error("Expected error for nil data")
	}
}

func TestTransformerFunc(t *testing.T) {
	transformer := TransformerFunc(func(data interface{}) (interface{}, error) {
		return data.(string) + "-transformed", nil
	})

	result, err := transformer.Transform("test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "test-transformed" {
		t.Errorf("Expected 'test-transformed', got '%v'", result)
	}
}

func TestPipeline(t *testing.T) {
	validator := ValidatorFunc(func(data interface{}) error {
		if data == nil {
			return errors.New("data cannot be nil")
		}
		return nil
	})

	transformer := TransformerFunc(func(data interface{}) (interface{}, error) {
		return data.(string) + "-transformed", nil
	})

	processed := ""
	processor := ProcessorFunc(func(data interface{}) error {
		processed = data.(string)
		return nil
	})

	pipeline := NewPipeline(processor).
		AddValidator(validator).
		AddTransformer(transformer)

	err := pipeline.Process("test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if processed != "test-transformed" {
		t.Errorf("Expected 'test-transformed', got '%s'", processed)
	}
}

func TestPipelineValidationFailure(t *testing.T) {
	validator := ValidatorFunc(func(data interface{}) error {
		return errors.New("validation failed")
	})

	processor := ProcessorFunc(func(data interface{}) error {
		return nil
	})

	pipeline := NewPipeline(processor).
		AddValidator(validator)

	err := pipeline.Process("test")
	if err == nil {
		t.Error("Expected validation error")
	}
}

func TestPipelineTransformationFailure(t *testing.T) {
	transformer := TransformerFunc(func(data interface{}) (interface{}, error) {
		return nil, errors.New("transformation failed")
	})

	processor := ProcessorFunc(func(data interface{}) error {
		return nil
	})

	pipeline := NewPipeline(processor).
		AddTransformer(transformer)

	err := pipeline.Process("test")
	if err == nil {
		t.Error("Expected transformation error")
	}
}
