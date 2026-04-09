package nanopony

import (
	"context"
	"fmt"
)

// Service defines the interface for lifecycle-managed services.
// Services are initialized during framework Start() and shutdown during Shutdown().
//
// Example implementation:
//
//	type MyService struct {
//	    name string
//	}
//
//	func (s *MyService) Initialize() error {
//	    log.Printf("Initializing %s", s.name)
//	    return nil
//	}
//
//	func (s *MyService) Shutdown() error {
//	    log.Printf("Shutting down %s", s.name)
//	    return nil
//	}
type Service interface {
	// Initialize is called when the framework starts
	Initialize() error
	// Shutdown is called when the framework shuts down
	Shutdown() error
}

// BaseService provides common service functionality and implements the Service interface.
// Embed this struct in your custom services to get default no-op implementations.
//
// Example:
//
//	type MyService struct {
//	    *nanopony.BaseService
//	    // ... other fields
//	}
type BaseService struct {
	name string
}

// NewBaseService creates a new base service with the given name
func NewBaseService(name string) *BaseService {
	return &BaseService{
		name: name,
	}
}

// Initialize implements Service interface.
// Returns nil by default. Override this method in your service if initialization is needed.
func (s *BaseService) Initialize() error {
	return nil
}

// Shutdown implements Service interface.
// Returns nil by default. Override this method in your service if cleanup is needed.
func (s *BaseService) Shutdown() error {
	return nil
}

// ServiceName returns the service name
func (s *BaseService) ServiceName() string {
	return s.name
}

// Processor defines the interface for processing single data items.
// Implement this interface to create custom processors.
//
// Example using function adapter:
//
//	processor := ProcessorFunc(func(data any) error {
//	    log.Printf("Processing: %v", data)
//	    return nil
//	})
type Processor interface {
	// Process processes a single data item
	Process(data any) error
	// ProcessWithContext processes a single data item with context support
	ProcessWithContext(ctx context.Context, data any) error
}

// ProcessorFunc is a function adapter that allows regular functions
// to implement the Processor interface.
//
// Note: This adapter ignores the context in ProcessWithContext.
// If you need context support, implement the Processor interface directly.
type ProcessorFunc func(any) error

// Process implements Processor
func (f ProcessorFunc) Process(data any) error {
	return f(data)
}

// ProcessWithContext implements Processor.
// Note: Context is currently ignored. This may change in future versions.
func (f ProcessorFunc) ProcessWithContext(ctx context.Context, data any) error {
	// TODO: Add context support in future versions
	return f(data)
}

// BatchProcessor defines the interface for processing batches of data
//
// Example using function adapter:
//
//	batchProcessor := BatchProcessorFunc(func(data []any) error {
//	    log.Printf("Processing batch: %d items", len(data))
//	    return nil
//	})
type BatchProcessor interface {
	// ProcessBatch processes a batch of data items
	ProcessBatch(data []any) error
	// ProcessBatchWithContext processes a batch with context support
	ProcessBatchWithContext(ctx context.Context, data []any) error
}

// BatchProcessorFunc is a function adapter that allows regular functions
// to implement the BatchProcessor interface.
type BatchProcessorFunc func([]any) error

// ProcessBatch implements BatchProcessor
func (f BatchProcessorFunc) ProcessBatch(data []any) error {
	return f(data)
}

// ProcessBatchWithContext implements BatchProcessor.
// Note: Context is currently ignored. This may change in future versions.
func (f BatchProcessorFunc) ProcessBatchWithContext(ctx context.Context, data []any) error {
	// TODO: Add context support in future versions
	return f(data)
}

// Validator defines the interface for validating data before processing.
// Validators are executed in the order they are added to the pipeline.
//
// Example using function adapter:
//
//	validator := ValidatorFunc(func(data any) error {
//	    if data == nil {
//	        return errors.New("data cannot be nil")
//	    }
//	    return nil
//	})
type Validator interface {
	// Validate checks if the data is valid. Returns an error if validation fails.
	Validate(data any) error
}

// ValidatorFunc is a function adapter that allows regular functions
// to implement the Validator interface.
type ValidatorFunc func(any) error

// Validate implements Validator
func (f ValidatorFunc) Validate(data any) error {
	return f(data)
}

// Transformer defines the interface for transforming data during processing.
// Transformers are executed in the order they are added to the pipeline.
//
// Example using function adapter:
//
//	transformer := TransformerFunc(func(data any) (any, error) {
//	    return strings.ToUpper(data.(string)), nil
//	})
type Transformer interface {
	// Transform converts the input data to a new form. Returns the transformed data or an error.
	Transform(data any) (any, error)
}

// TransformerFunc is a function adapter that allows regular functions
// to implement the Transformer interface.
type TransformerFunc func(any) (any, error)

// Transform implements Transformer
func (f TransformerFunc) Transform(data any) (any, error) {
	return f(data)
}

// Pipeline represents a processing pipeline with validation and transformation.
// Data flows through validators -> transformers -> processor in sequence.
//
// Example:
//
//	pipeline := NewPipeline(processor).
//	    AddValidator(validateNotNull).
//	    AddValidator(validateFormat).
//	    AddTransformer(transformToDomain).
//	    AddTransformer(enrichWithData)
//
//	err := pipeline.Process(rawData)
type Pipeline struct {
	validators   []Validator
	transformers []Transformer
	processor    Processor
}

// NewPipeline creates a new processing pipeline with the given processor.
// Add validators and transformers using AddValidator and AddTransformer methods.
func NewPipeline(processor Processor) *Pipeline {
	return &Pipeline{
		validators:   make([]Validator, 0),
		transformers: make([]Transformer, 0),
		processor:    processor,
	}
}

// AddValidator adds a validator to the pipeline.
// Validators are executed in the order they are added.
// If any validator fails, processing stops and returns the error.
func (p *Pipeline) AddValidator(validator Validator) *Pipeline {
	p.validators = append(p.validators, validator)
	return p
}

// AddTransformer adds a transformer to the pipeline.
// Transformers are executed in the order they are added.
// Each transformer receives the output of the previous one.
func (p *Pipeline) AddTransformer(transformer Transformer) *Pipeline {
	p.transformers = append(p.transformers, transformer)
	return p
}

// Process executes the pipeline on the given data.
//
// Execution order:
// 1. Validators (fail-fast: stops on first error)
// 2. Transformers (chain: data flows through each)
// 3. Processor (final processing)
//
// Example:
//
//	err := pipeline.Process(data)
//	if err != nil {
//	    log.Printf("Pipeline error: %v", err)
//	}
func (p *Pipeline) Process(data any) error {
	// Step 1: Run validators (fail-fast strategy)
	for _, validator := range p.validators {
		if err := validator.Validate(data); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Step 2: Run transformers (chain pattern)
	result := data
	for _, transformer := range p.transformers {
		var err error
		result, err = transformer.Transform(result)
		if err != nil {
			return fmt.Errorf("transformation failed: %w", err)
		}
	}

	// Step 3: Process the final result
	return p.processor.Process(result)
}
