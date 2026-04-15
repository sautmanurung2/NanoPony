package nanopony

import (
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
