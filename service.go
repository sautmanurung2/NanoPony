package nanopony

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

