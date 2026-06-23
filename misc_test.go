package nanopony

import (
	"errors"
	"testing"
)

func TestJobReleaseNil(t *testing.T) {
	var j *Job
	j.Release() // Should not panic
}

func TestNewKafkaWriterErrors(t *testing.T) {
	ResetConfig()
	config := BuildConfig(func(c *Config) {
		c.App.KafkaModels = "invalid"
	})

	// Should return default writer instead of panicking or failing
	w := NewKafkaWriterFromConfigRoundRobin(config)
	if w == nil {
		t.Error("Expected default writer")
	}
}

func TestFrameworkSafeBuildersSuccess(t *testing.T) {
	ResetConfig()
	config := BuildConfig(func(c *Config) {
		c.App.Env = "local"
		c.App.KafkaModels = "kafka-localhost"
		c.App.Operation = "test"
	})

	fw := NewFramework().WithConfig(config)

	// WithKafkaWriterSafeRoundRobin should succeed (just creates struct)
	_, err := fw.WithKafkaWriterSafeRoundRobin()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// WithProducerSafe should succeed
	_, err = fw.WithProducerSafe()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestKafkaConsumerCloseNil(t *testing.T) {
	c := &KafkaConsumer{}
	if err := c.Close(); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestPollerStopIdempotent(t *testing.T) {
	p := &Poller{}
	p.Stop()
	p.Stop()
}

func TestWorkerPoolReportErrorFull(t *testing.T) {
	pool := NewWorkerPool(1, 1, 1)
	shard := pool.shards[0]

	// Fill errChan
	shard.errChan <- errors.New("first")

	// Should not block
	pool.reportError(shard, "job1", 0, errors.New("second"))
}

func TestWorkerPoolNumShardsCoverage(t *testing.T) {
	pool := NewWorkerPool(1, 1, 1)
	if pool.NumShards() != 1 {
		t.Errorf("Expected 1 shard")
	}
}
