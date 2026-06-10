package nanopony

import (
	"testing"
	"time"
)

func TestGetOrDefault(t *testing.T) {
	// Int
	if GetOrDefault(10, 20) != 10 {
		t.Error("Expected 10")
	}
	if GetOrDefault(0, 20) != 20 {
		t.Error("Expected 20")
	}
	if GetOrDefault(-1, 20) != 20 {
		t.Error("Expected 20")
	}

	// Float
	if GetOrDefault(10.5, 20.5) != 10.5 {
		t.Error("Expected 10.5")
	}
	if GetOrDefault(0.0, 20.5) != 20.5 {
		t.Error("Expected 20.5")
	}

	// Duration (int64)
	if GetOrDefault(time.Second, 5*time.Second) != time.Second {
		t.Error("Expected 1s")
	}
	if GetOrDefault(time.Duration(0), 5*time.Second) != 5*time.Second {
		t.Error("Expected 5s")
	}
}
