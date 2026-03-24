package nanopony

import (
	"os"
	"testing"
)

func TestResetConfig(t *testing.T) {
	// Set initial config
	_ = NewConfig()
	if appConfig == nil {
		t.Fatal("Expected config to be initialized")
	}

	// Reset config
	ResetConfig()
	if appConfig != nil {
		t.Error("Expected config to be reset")
	}
}

func TestBuildConfig(t *testing.T) {
	ResetConfig()

	config := BuildConfig(func(c *Config) {
		c.App.Env = "custom"
	})

	if config == nil {
		t.Fatal("Expected config to be created")
	}
}

func TestGetEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		cfg      envConfig
		envValue string
		expected string
	}{
		{
			name: "valid value",
			cfg: envConfig{
				name:        "TEST_ENV",
				validValues: []string{"staging", "production"},
				defaultVal:  "local",
			},
			envValue: "staging",
			expected: "staging",
		},
		{
			name: "invalid value returns default",
			cfg: envConfig{
				name:        "TEST_ENV",
				validValues: []string{"staging", "production"},
				defaultVal:  "local",
			},
			envValue: "invalid",
			expected: "local",
		},
		{
			name: "no valid values",
			cfg: envConfig{
				name:        "TEST_ENV",
				validValues: []string{},
				defaultVal:  "default",
			},
			envValue: "any",
			expected: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.cfg.name, tt.envValue)
			defer os.Unsetenv(tt.cfg.name)

			result := getEnvValue(tt.cfg)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestWithOperationValidValues(t *testing.T) {
	customValues := []string{"custom1", "custom2"}
	WithOperationValidValues(customValues)

	// Reset to default
	WithOperationValidValues([]string{})
}
