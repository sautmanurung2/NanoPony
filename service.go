package nanopony

import (
	"context"
	"fmt"
)

// Service defines the base interface for all services
type Service interface {
	Initialize() error
	Shutdown() error
}

// BaseService provides common service functionality
type BaseService struct {
	name string
}

// NewBaseService creates a new base service
func NewBaseService(name string) *BaseService {
	return &BaseService{
		name: name,
	}
}

// Initialize implements Service interface
func (s *BaseService) Initialize() error {
	return nil
}

// Shutdown implements Service interface
func (s *BaseService) Shutdown() error {
	return nil
}

// ServiceName returns the service name
func (s *BaseService) ServiceName() string {
	return s.name
}

// Processor defines the interface for processing data
type Processor interface {
	Process(data interface{}) error
	ProcessWithContext(ctx context.Context, data interface{}) error
}

// ProcessorFunc is a function adapter for Processor
type ProcessorFunc func(interface{}) error

// Process implements Processor
func (f ProcessorFunc) Process(data interface{}) error {
	return f(data)
}

// ProcessWithContext implements Processor
func (f ProcessorFunc) ProcessWithContext(ctx context.Context, data interface{}) error {
	return f(data)
}

// BatchProcessor defines the interface for processing batches
type BatchProcessor interface {
	ProcessBatch(data []interface{}) error
	ProcessBatchWithContext(ctx context.Context, data []interface{}) error
}

// BatchProcessorFunc is a function adapter for BatchProcessor
type BatchProcessorFunc func([]interface{}) error

// ProcessBatch implements BatchProcessor
func (f BatchProcessorFunc) ProcessBatch(data []interface{}) error {
	return f(data)
}

// ProcessBatchWithContext implements BatchProcessor
func (f BatchProcessorFunc) ProcessBatchWithContext(ctx context.Context, data []interface{}) error {
	return f(data)
}

// Validator defines the interface for validating data
type Validator interface {
	Validate(data interface{}) error
}

// ValidatorFunc is a function adapter for Validator
type ValidatorFunc func(interface{}) error

// Validate implements Validator
func (f ValidatorFunc) Validate(data interface{}) error {
	return f(data)
}

// Transformer defines the interface for transforming data
type Transformer interface {
	Transform(data interface{}) (interface{}, error)
}

// TransformerFunc is a function adapter for Transformer
type TransformerFunc func(interface{}) (interface{}, error)

// Transform implements Transformer
func (f TransformerFunc) Transform(data interface{}) (interface{}, error) {
	return f(data)
}

// Pipeline represents a processing pipeline with validation and transformation
type Pipeline struct {
	validators  []Validator
	transformers []Transformer
	processor    Processor
}

// NewPipeline creates a new processing pipeline
func NewPipeline(processor Processor) *Pipeline {
	return &Pipeline{
		validators:  make([]Validator, 0),
		transformers: make([]Transformer, 0),
		processor:   processor,
	}
}

// AddValidator adds a validator to the pipeline
func (p *Pipeline) AddValidator(validator Validator) *Pipeline {
	p.validators = append(p.validators, validator)
	return p
}

// AddTransformer adds a transformer to the pipeline
func (p *Pipeline) AddTransformer(transformer Transformer) *Pipeline {
	p.transformers = append(p.transformers, transformer)
	return p
}

// Process executes the pipeline on the given data
func (p *Pipeline) Process(data interface{}) error {
	// Run validators
	for _, validator := range p.validators {
		if err := validator.Validate(data); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Run transformers
	result := data
	for _, transformer := range p.transformers {
		var err error
		result, err = transformer.Transform(result)
		if err != nil {
			return fmt.Errorf("transformation failed: %w", err)
		}
	}

	// Process the result
	return p.processor.Process(result)
}
