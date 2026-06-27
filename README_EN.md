# NanoPony

[![Go Reference](https://pkg.go.dev/badge/github.com/sautmanurung2/nanopony.svg)](https://pkg.go.dev/github.com/sautmanurung2/nanopony)
[![Go Report Card](https://goreportcard.com/badge/github.com/sautmanurung2/nanopony)](https://goreportcard.com/report/github.com/sautmanurung2/nanopony)
<a href="https://octocounts.com/?q=https%3A%2F%2Fgithub.com%2Fsautmanurung2%2FNanoPony.git&ref=main">
  <img 
    src="https://api.octocounts.com/badge/sautmanurung2/NanoPony/branch/main"
    height="24"
  />
</a>

**NanoPony** is a high-performance Go framework designed for clean, reusable, and production-ready integration between Kafka and Oracle. It provides ready-to-use components to build scalable data pipelines with minimal boilerplate, featuring high memory efficiency through `sync.Pool` and a sharded worker pool architecture.

## 📚 Documentation

| Document | Description | Link |
| :--- | :--- | :--- |
| **📖 Full Documentation** | Comprehensive guide to all framework components | [DOKUMENTASI.md](docs/DOKUMENTASI.md) |
| **🏗️ Architecture** | Architecture diagrams, design patterns, and data flow | [ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| **🔧 Testing Guide** | How to run tests, coverage, and best practices | [TESTING_GUIDE.md](docs/TESTING_GUIDE.md) |
| **📊 Benchmark Report** | Performance benchmarks and memory leak tests | [BENCHMARK_REPORT.md](docs/BENCHMARK_REPORT.md) |
| **⚙️ Worker Pool Deep Dive** | Detailed explanation of worker pool mechanics | [WORKER_POOL_EXPLAINED.md](docs/WORKER_POOL_EXPLAINED.md) |

## Key Features

### 🏗️ Core Framework & Builder Pattern
- ✅ **Fluent Builder Pattern**: Clean, readable setup with method chaining.
- ✅ **Lifecycle Management**: Coordinated Start/Shutdown for all components.
- ✅ **Dependency Injection**: Supports custom instances or auto-creation from configuration.
- ✅ **Graceful Shutdown**: Sequential teardown (Poller → Worker Pool → Cleanup).
- ✅ **Concurrent Cleanup**: Cleanup functions run concurrently on shutdown.

### ⚙️ Configuration System
- ✅ **Environment-based**: Configuration via environment variables.
- ✅ **Multi-environment**: Local, Staging, Production support.
- ✅ **Auto .env Loading**: Automatic loading via `godotenv`.
- ✅ **Dynamic Resolution**: Supports env-specific variables (e.g., HOST_STAGING, HOST_PRODUCTION).
- ✅ **Confluent Cloud**: Built-in SASL/TLS authentication.

### 🚀 High-Performance Worker Pool
- ✅ **Shared Queue with Elastic Buffer**: Eliminates Head-of-Line blocking and deadlock on queue spikes.
- ✅ **Ultra-Efficient**: Uses `sync.Pool` to recycle `Job` objects, reducing GC pressure.
- ✅ **100% Reliable Submit**: Eliminates `ErrQueueFull` using Elastic Buffer mechanism and exponential backoff.
- ✅ **Backpressure**: Supports `SubmitBlocking` to prevent data loss.
- ✅ **Thread-safe**: Robust state management using `sync.RWMutex`.

### 🔄 Data Processing (Poller & Producers)
- ✅ **Batch Processing**: Configurable batch sizes for efficient data fetching.
- ✅ **Rate Limiting**: JobSlot semaphore system for controlled concurrency.
- ✅ **Interface-based**: Easy to implement custom data sources via `DataFetcher`.
- ✅ **Kafka Integration**: Native integration with `kafka-go`.

### 📝 Performance Highlights
- ⚡ **Throughput**: Significantly optimized via sharded worker pools.
- ⚡ **Memory Efficiency**: Minimal allocation per job.
- ⚡ **Stability**: No memory leaks verified over extended operations.

## Installation

```bash
go get github.com/sautmanurung2/nanopony
```

## Quick Start

```go
import "github.com/sautmanurung2/nanopony"

// 1. Initialize config
config := nanopony.NewConfig()

// 2. Build framework
components := nanopony.NewFramework().
    WithConfig(config).
    WithDatabase().
    WithKafkaWriter().
    WithProducer().
    WithWorkerPool(5, 100, 3).
    WithPoller(nanopony.DefaultPollerConfig(), dataFetcher).
    Build()

// 3. Start processing
ctx := context.Background()
components.Start(ctx, jobHandler)

// 4. Graceful shutdown
defer components.Shutdown(ctx)
```

## Contributing

We welcome contributions! Please see our [CONTRIBUTING.md](docs/CONTRIBUTING.md) and [CLA.md](docs/CLA.md) for details.

## License

Apache License 2.0
