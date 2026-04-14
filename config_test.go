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

func TestLoadDynamic(t *testing.T) {
	ResetConfig()

	// Set custom environment variables
	os.Setenv("CUSTOM_API_URL", "https://api.example.com")
	os.Setenv("CUSTOM_TIMEOUT", "30s")
	os.Setenv("CUSTOM_RETRY_COUNT", "3")
	os.Setenv("OTHER_VAR", "should-not-load")
	defer func() {
		os.Unsetenv("CUSTOM_API_URL")
		os.Unsetenv("CUSTOM_TIMEOUT")
		os.Unsetenv("CUSTOM_RETRY_COUNT")
		os.Unsetenv("OTHER_VAR")
	}()

	config := BuildConfig()
	config.LoadDynamic("CUSTOM_")

	if len(config.Dynamic) != 3 {
		t.Errorf("Expected 3 dynamic config items, got %d", len(config.Dynamic))
	}

	if config.Dynamic["CUSTOM_API_URL"] != "https://api.example.com" {
		t.Errorf("Expected CUSTOM_API_URL 'https://api.example.com', got '%s'", config.Dynamic["CUSTOM_API_URL"])
	}

	if config.Dynamic["CUSTOM_TIMEOUT"] != "30s" {
		t.Errorf("Expected CUSTOM_TIMEOUT '30s', got '%s'", config.Dynamic["CUSTOM_TIMEOUT"])
	}

	if config.Dynamic["CUSTOM_RETRY_COUNT"] != "3" {
		t.Errorf("Expected CUSTOM_RETRY_COUNT '3', got '%s'", config.Dynamic["CUSTOM_RETRY_COUNT"])
	}

	// OTHER_VAR should not be loaded
	if _, exists := config.Dynamic["OTHER_VAR"]; exists {
		t.Error("OTHER_VAR should not be loaded with CUSTOM_ prefix")
	}
}

func TestLoadDynamicEmptyPrefix(t *testing.T) {
	ResetConfig()

	// Set a unique environment variable
	os.Setenv("TEST_DYNAMIC_UNIQUE", "test-value")
	defer os.Unsetenv("TEST_DYNAMIC_UNIQUE")

	config := BuildConfig()
	config.LoadDynamic("")

	// Should load all environment variables
	if len(config.Dynamic) == 0 {
		t.Error("Expected dynamic config to be populated with all env vars")
	}

	if config.Dynamic["TEST_DYNAMIC_UNIQUE"] != "test-value" {
		t.Errorf("Expected TEST_DYNAMIC_UNIQUE 'test-value', got '%s'", config.Dynamic["TEST_DYNAMIC_UNIQUE"])
	}
}

func TestLoadDynamicNilMap(t *testing.T) {
	ResetConfig()

	config := &Config{}
	config.LoadDynamic("TEST_")

	if config.Dynamic == nil {
		t.Error("Expected Dynamic map to be initialized")
	}
}

func TestGetEnvByPrefix(t *testing.T) {
	// Set test environment variables
	os.Setenv("PREFIX_VAR1", "value1")
	os.Setenv("PREFIX_VAR2", "value2")
	os.Setenv("OTHER_VAR", "other")
	defer func() {
		os.Unsetenv("PREFIX_VAR1")
		os.Unsetenv("PREFIX_VAR2")
		os.Unsetenv("OTHER_VAR")
	}()

	result := getEnvByPrefix("PREFIX_")

	if len(result) != 2 {
		t.Errorf("Expected 2 variables with PREFIX_, got %d", len(result))
	}

	if result["PREFIX_VAR1"] != "value1" {
		t.Errorf("Expected PREFIX_VAR1 'value1', got '%s'", result["PREFIX_VAR1"])
	}

	if result["PREFIX_VAR2"] != "value2" {
		t.Errorf("Expected PREFIX_VAR2 'value2', got '%s'", result["PREFIX_VAR2"])
	}

	// OTHER_VAR should not be in results
	if _, exists := result["OTHER_VAR"]; exists {
		t.Error("OTHER_VAR should not be in results with PREFIX_ filter")
	}
}

func TestGetEnvByPrefixEmpty(t *testing.T) {
	// Set a unique environment variable
	os.Setenv("TEST_UNIQUE_PREFIX", "unique-value")
	defer os.Unsetenv("TEST_UNIQUE_PREFIX")

	result := getEnvByPrefix("")

	// Should contain the unique variable
	if result["TEST_UNIQUE_PREFIX"] != "unique-value" {
		t.Errorf("Expected TEST_UNIQUE_PREFIX 'unique-value', got '%s'", result["TEST_UNIQUE_PREFIX"])
	}
}
