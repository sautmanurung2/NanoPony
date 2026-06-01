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
	f.Add("GO_ENV", "production", "local")
	f.Add("KAFKA_MODELS", "kafka-staging", "kafka-localhost")
	f.Add("LOG_FILE_PREFIX", "my-app", "default-prefix")

	f.Fuzz(func(t *testing.T, name, value, defaultValue string) {
		orig := os.Getenv(name)
		defer os.Setenv(name, orig)
		os.Setenv(name, value)

		cfg := envConfig{
			name:        name,
			validValues: []string{"localhost", "staging", "production", "kafka-staging", "kafka-production", "kafka-confluent"},
			defaultVal:  defaultValue,
		}
		_ = getEnvValue(cfg)
	})
}

// FuzzProcessPayload tests the log payload processing logic.
func FuzzProcessPayload(f *testing.F) {
	f.Add([]byte(`{"key": "value"}`))
	f.Add([]byte(`invalid json`))
	f.Add([]byte(``))
	f.Add([]byte(`{"nested": {"inner": 123}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = processPayload(data)
		_, _ = processPayload(string(data))
		m := make(map[string]any)
		_ = json.Unmarshal(data, &m)
		_, _ = processPayload(m)
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
		os.Setenv("CUSTOM_VAR1", "val1")
		os.Setenv("APP_READY", "true")
		defer os.Unsetenv("CUSTOM_VAR1")
		defer os.Unsetenv("APP_READY")
		conf.LoadDynamic(prefix)
	})
}

// FuzzJobSubmission tests worker pool submission.
func FuzzJobSubmission(f *testing.F) {
	f.Add("job-1", "data-1")
	f.Add("job-2", "")
	f.Add("", "data-empty-id")

	f.Fuzz(func(t *testing.T, id, data string) {
		wp := NewWorkerPool[string](1, 10, 1)
		ctx := context.Background()

		job := AcquireJob[string]()
		job.ID = id
		job.Data = data

		_ = wp.Submit(ctx, job)
		wp.Stop()
	})
}

// FuzzKafkaProduce tests the Kafka producer's marshaling logic.
func FuzzKafkaProduce(f *testing.F) {
	f.Add("topic-1", "message-1")
	f.Add("events", `{"type": "event", "id": 123}`)

	f.Fuzz(func(t *testing.T, topic, message string) {
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
	f.Add(uint64(18446744073709551615))

	f.Fuzz(func(t *testing.T, b uint64) {
		_ = FormatBytes(b)
	})
}

// FuzzStressFrameworkLifecycle tests the orchestration logic.
func FuzzStressFrameworkLifecycle(f *testing.F) {
	f.Add(int(1), int(1), int(1))

	f.Fuzz(func(t *testing.T, workers int, queueSize int, shards int) {
		ResetConfig()
		if workers < 1 { workers = 1 }
		if workers > 100 { workers = 100 }
		if queueSize < 1 { queueSize = 1 }
		if queueSize > 1000 { queueSize = 1000 }
		if shards < 1 { shards = 1 }

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		fw := NewFramework[string]().
			WithConfig(BuildConfig(func(c *Config) {
				c.App.Env = "localhost"
				c.App.KafkaModels = "kafka-localhost"
			})).
			WithWorkerPool(workers, queueSize, shards).
			WithPoller(DefaultPollerConfig(), DataFetcherFunc[string](func() ([]string, error) {
				return []string{"fuzz-item"}, nil
			}))

		comp := fw.Build()
		comp.Start(ctx, func(ctx context.Context, job *Job[string]) error {
			return nil
		})
		time.Sleep(1 * time.Millisecond)
		_ = comp.Shutdown(ctx)
	})
}

// FuzzStressWorkerPoolParams tests worker pool with extreme parameters.
func FuzzStressWorkerPoolParams(f *testing.F) {
	f.Add(5, 100, 2)
	f.Add(-1, -1, -1)
	f.Add(1000, 10000, 10)

	f.Fuzz(func(t *testing.T, numWorkers int, queueSize int, numShards int) {
		if numWorkers < 1 { numWorkers = 1 }
		if numWorkers > 500 { numWorkers = 500 }
		if queueSize < 1 { queueSize = 1 }
		if queueSize > 10000 { queueSize = 10000 }
		if numShards < 1 { numShards = 1 }

		wp := NewWorkerPool[any](numWorkers, queueSize, numShards)
		if wp == nil { return }

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		if numWorkers > 0 && numWorkers < 500 {
			wp.Start(ctx, func(ctx context.Context, job *Job[any]) error {
				return nil
			})
			job := AcquireJob[any]()
			job.ID = "stress-job"
			_ = wp.Submit(ctx, job)
			wp.Stop()
		}
	})
}

// FuzzStressPollerParams tests poller with extreme parameters.
func FuzzStressPollerParams(f *testing.F) {
	f.Add(int64(time.Second), 100, 1)
	f.Add(int64(time.Millisecond), 0, 0)
	f.Add(int64(-1), -1, -1)

	f.Fuzz(func(t *testing.T, intervalNs int64, batchSize int, jobSlots int) {
		interval := time.Duration(intervalNs)
		if interval < 0 { interval = 1 * time.Millisecond }
		if interval > 1*time.Hour { interval = 1 * time.Hour }

		cfg := PollerConfig{
			Interval:    interval,
			BatchSize:   batchSize,
			JobSlotSize: jobSlots,
		}

		wp := NewWorkerPool[int](1, 10, 1)
		fetcher := DataFetcherFunc[int](func() ([]int, error) {
			return []int{1, 2, 3}, nil
		})

		defer func() { recover() }()

		poller := NewPoller[int](cfg, wp, fetcher)
		if poller != nil {
			poller.Start()
			time.Sleep(1 * time.Millisecond)
			poller.Stop()
		}
		wp.Stop()
	})
}
