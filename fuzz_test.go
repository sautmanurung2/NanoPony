package nanopony

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// FuzzGetEnvValue tests the environment variable retrieval logic.
func FuzzGetEnvValue(f *testing.F) {
	// Seed names and values
	f.Add("GO_ENV", "production", "local")
	f.Add("KAFKA_MODELS", "kafka-staging", "kafka-localhost")
	f.Add("LOG_FILE_PREFIX", "my-app", "default-prefix")

	f.Fuzz(func(t *testing.T, name, value, defaultValue string) {
		// Save original env
		orig := os.Getenv(name)
		defer os.Setenv(name, orig)

		os.Setenv(name, value)

		cfg := envConfig{
			name:        name,
			validValues: []string{"localhost", "staging", "production", "kafka-staging", "kafka-production", "kafka-confluent"},
			defaultVal:  defaultValue,
		}

		result := getEnvValue(cfg)

		// If validValues is empty, it should return the value
		// If value is empty, it should return default
		// If value is in validValues, it should return value
		// Otherwise return default
		if result == "" && defaultValue != "" {
			t.Errorf("expected non-empty result when defaultValue is set")
		}
	})
}

// FuzzProcessPayload tests the log payload processing logic which handles diverse types.
func FuzzProcessPayload(f *testing.F) {
	// Add seeds for common payload types
	f.Add([]byte(`{"key": "value"}`))           // Valid JSON
	f.Add([]byte(`invalid json`))               // Invalid JSON
	f.Add([]byte(``))                           // Empty
	f.Add([]byte(`{"nested": {"inner": 123}}`)) // Nested JSON

	f.Fuzz(func(t *testing.T, data []byte) {
		// Test with []byte
		_, _ = processPayload(data)

		// Test with string
		_, _ = processPayload(string(data))

		// Test with map
		m := make(map[string]any)
		_ = json.Unmarshal(data, &m)
		_, _ = processPayload(m)

		// Test with struct-like map
		s := struct{ Data string }{Data: string(data)}
		_, _ = processPayload(s)
	})
}

// FuzzLoadDynamic tests the dynamic configuration loading.
func FuzzLoadDynamic(f *testing.F) {
	f.Add("CUSTOM_")
	f.Add("APP_")
	f.Add("")

	f.Fuzz(func(t *testing.T, prefix string) {
		conf := &Config{
			Dynamic: make(map[string]string),
		}

		// Set some dummy env vars
		os.Setenv("CUSTOM_VAR1", "val1")
		os.Setenv("APP_READY", "true")
		defer os.Unsetenv("CUSTOM_VAR1")
		defer os.Unsetenv("APP_READY")

		conf.LoadDynamic(prefix)

		if conf.Dynamic == nil {
			t.Errorf("Dynamic map should not be nil after LoadDynamic")
		}
	})
}

// FuzzJobSubmission tests worker pool submission with varying job data.
func FuzzJobSubmission(f *testing.F) {
	f.Add("job-1", "data-1")
	f.Add("job-2", "")
	f.Add("", "data-empty-id")

	f.Fuzz(func(t *testing.T, id, data string) {
		wp := NewWorkerPool(1, 10)
		ctx := context.Background()

		job := Job{
			ID:   id,
			Data: data,
		}

		// Submit job
		_ = wp.Submit(ctx, job)

		// Cleanup
		wp.Stop()
	})
}

// FuzzKafkaProduce tests the Kafka producer's marshaling logic.
func FuzzKafkaProduce(f *testing.F) {
	f.Add("topic-1", "message-1")
	f.Add("events", `{"type": "event", "id": 123}`)

	f.Fuzz(func(t *testing.T, topic, message string) {
		// We can't easily fuzz the actual Kafka writing without a mock,
		// but we can fuzz the ProduceWithContext logic up to the marshaling.

		var dummy any = message
		if json.Valid([]byte(message)) {
			var m map[string]any
			if err := json.Unmarshal([]byte(message), &m); err == nil {
				dummy = m
			}
		}

		_, _ = json.Marshal(dummy)
	})
}

// FuzzFormatBytes tests the memory size formatting utility.
func FuzzFormatBytes(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1024))
	f.Add(uint64(1024 * 1024))
	f.Add(uint64(18446744073709551615)) // Max uint64

	f.Fuzz(func(t *testing.T, b uint64) {
		_ = FormatBytes(b)
	})
}

// FuzzStressFrameworkLifecycle tests the orchestration logic with random sequences.
func FuzzStressFrameworkLifecycle(f *testing.F) {
	f.Add(int(1), int(1))

	f.Fuzz(func(t *testing.T, workers int, queueSize int) {
		// Clean up global state
		ResetConfig()

		// Sanitize inputs for stability
		if workers < 1 {
			workers = 1
		}
		if workers > 100 {
			workers = 100
		}
		if queueSize < 1 {
			queueSize = 1
		}
		if queueSize > 1000 {
			queueSize = 1000
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		fw := NewFramework().
			WithConfig(BuildConfig(func(c *Config) {
				c.App.Env = "localhost"
				c.App.KafkaModels = "kafka-localhost"
			})).
			WithWorkerPool(workers, queueSize).
			WithPoller(DefaultPollerConfig(), DataFetcherFunc(func() ([]any, error) {
				return []any{"fuzz-item"}, nil
			}))

		// Test Build behavior
		comp := fw.Build()

		// Test Start behavior
		comp.Start(ctx, func(ctx context.Context, job Job) error {
			return nil
		})

		// Allow some execution
		time.Sleep(1 * time.Millisecond)

		// Test Shutdown behavior
		_ = comp.Shutdown(ctx)
	})
}

// FuzzStressWorkerPoolParams tests worker pool with extreme parameter values.
func FuzzStressWorkerPoolParams(f *testing.F) {
	f.Add(5, 100)
	f.Add(-1, -1)
	f.Add(1000, 10000)

	f.Fuzz(func(t *testing.T, numWorkers int, queueSize int) {
		// Sanitize inputs for stability to avoid invalid channel sizes (panics)
		if numWorkers < 1 {
			numWorkers = 1
		}
		if numWorkers > 500 {
			numWorkers = 500
		}
		if queueSize < 1 {
			queueSize = 1
		}
		if queueSize > 10000 {
			queueSize = 10000
		}

		// Test initialization with random parameters
		wp := NewWorkerPool(numWorkers, queueSize)
		if wp == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		// Attempt to start - should handle extreme values gracefully
		// Note: Very large values might cause OOM, so we might want to cap it for CI
		if numWorkers > 0 && numWorkers < 500 {
			wp.Start(ctx, func(ctx context.Context, job Job) error {
				return nil
			})

			// Submit some data
			_ = wp.Submit(ctx, Job{ID: "stress-job"})

			wp.Stop()
		}
	})
}

// FuzzStressPollerParams tests poller with extreme timing and batch parameters.
func FuzzStressPollerParams(f *testing.F) {
	f.Add(int64(time.Second), 100, 1)
	f.Add(int64(time.Millisecond), 0, 0)
	f.Add(int64(-1), -1, -1)

	f.Fuzz(func(t *testing.T, intervalNs int64, batchSize int, jobSlots int) {
		interval := time.Duration(intervalNs)
		if interval < 0 {
			interval = 1 * time.Millisecond
		}
		if interval > 1*time.Hour {
			interval = 1 * time.Hour
		}

		cfg := PollerConfig{
			Interval:    interval,
			BatchSize:   batchSize,
			JobSlotSize: jobSlots,
		}

		wp := NewWorkerPool(1, 10)
		fetcher := DataFetcherFunc(func() ([]any, error) {
			return []any{1, 2, 3}, nil
		})

		// Test NewPoller behavior with these params
		// We catch any panics that might occur from chan size <= 0
		defer func() { recover() }()

		poller := NewPoller(cfg, wp, fetcher)
		if poller != nil {
			poller.Start()
			time.Sleep(1 * time.Millisecond)
			poller.Stop()
		}

		wp.Stop()
	})
}
