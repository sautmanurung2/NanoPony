# NanoPony

[![Go Reference](https://pkg.go.dev/badge/github.com/sautmanurung2/nanopony.svg)](https://pkg.go.dev/github.com/sautmanurung2/nanopony)
[![Go Report Card](https://goreportcard.com/badge/github.com/sautmanurung2/nanopony)](https://goreportcard.com/report/github.com/sautmanurung2/nanopony)

**NanoPony** adalah framework Go untuk integrasi Kafka-Oracle dengan arsitektur yang clean, reusable, dan production-ready.

## 📚 Dokumentasi Lengkap

| Dokumen | Deskripsi | Link |
|---------|-----------|------|
| **📖 Dokumentasi Lengkap** | Panduan komprehensif semua komponen framework | [DOKUMENTASI.md](DOKUMENTASI.md) |
| **🏗️ Arsitektur** | Diagram arsitektur, pola desain, dan alur data | [ARCHITECTURE.md](ARCHITECTURE.md) |
| **🔧 Panduan Testing** | Cara menjalankan test, coverage, dan best practices | [TESTING_GUIDE.md](TESTING_GUIDE.md) |
| **📊 Benchmark Report** | Hasil benchmark performa dan memory leak test | [BENCHMARK_REPORT.md](BENCHMARK_REPORT.md) |
| **⚙️ Worker Pool Deep Dive** | Penjelasan detail cara kerja worker pool | [WORKER_POOL_EXPLAINED.md](WORKER_POOL_EXPLAINED.md) |

## Fitur

- ✅ **Kafka Producer & Consumer** - Integrasi dengan Kafka menggunakan `kafka-go`
- ✅ **Oracle Database** - Koneksi Oracle menggunakan `go-ora` dengan connection pooling
- ✅ **Worker Pool** - Pool of workers untuk memproses jobs secara concurrent
- ✅ **Poller** - Polling data dengan interval yang dapat dikonfigurasi
- ✅ **Builder Pattern** - API yang clean dan mudah digunakan
- ✅ **Graceful Shutdown** - Shutdown yang aman untuk semua komponen
- ✅ **Environment-based Config** - Konfigurasi berbasis environment variables
- ✅ **Pipeline Processing** - Validator, Transformer, dan Processor pipeline
- ✅ **Transaction Support** - Database transaction helper
- ✅ **Comprehensive Tests** - Unit tests untuk semua komponen utama

## Instalasi

```bash
go get github.com/sautmanurung2/nanopony
```

## Quick Start

### 1. Import Package

```go
import "github.com/sautmanurung2/nanopony"
```

### 2. Inisialisasi Konfigurasi

```go
config := nanopony.NewConfig()
```

### 3. Buat Framework dengan Builder

```go
framework := nanopony.NewFramework().
    WithConfig(config).
    WithDatabase().
    WithKafkaWriter().
    WithProducer().
    WithWorkerPool(5, 100).
    WithPoller(nanopony.DefaultPollerConfig(), dataFetcher)

components := framework.Build()
```

### 4. Start Framework

```go
ctx := context.Background()
components.Start(ctx, jobHandler)
```

### 5. Graceful Shutdown

```go
components.Shutdown(ctx)
```

## Contoh Lengkap

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/sautmanurung2/nanopony"
)

func main() {
    // Initialize configuration
    config := nanopony.NewConfig()

    // Create framework
    framework := nanopony.NewFramework().
        WithConfig(config).
        WithDatabase().
        WithKafkaWriter().
        WithProducer().
        WithWorkerPool(5, 100).
        WithPoller(nanopony.DefaultPollerConfig(), &myDataFetcher{}).
        AddService(&myService{name: "MyService"})

    components := framework.Build()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    components.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
        log.Printf("Processing job: %+v", job)
        return nil
    })

    // Wait for interrupt
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    components.Shutdown(ctx)
}

type myDataFetcher struct{}

func (f *myDataFetcher) Fetch() ([]interface{}, error) {
    // Fetch data from database
    return []interface{}{"data1", "data2"}, nil
}

type myService struct {
    name string
}

func (s *myService) Initialize() error {
    log.Printf("Initializing %s", s.name)
    return nil
}

func (s *myService) Shutdown() error {
    log.Printf("Shutting down %s", s.name)
    return nil
}
```

## Konfigurasi Environment

NanoPony menggunakan environment variables untuk konfigurasi:

### Environment Variables

| Variable | Deskripsi | Contoh |
|----------|-----------|--------|
| `GO_ENV` | Environment aplikasi | `local`, `staging`, `production` |
| `KAFKA_MODELS` | Model Kafka | `kafka-localhost`, `kafka-staging`, `kafka-production`, `kafka-confluent` |
| `KAFKA_BROKERS_STAGING` | Broker Kafka staging | `broker1:9092,broker2:9092` |
| `KAFKA_BROKERS_PRODUCTION` | Broker Kafka production | `broker1:9092,broker2:9092` |
| `HOST_STAGING` | Host Oracle staging | `oracle-staging.example.com` |
| `PORT_STAGING` | Port Oracle staging | `1521` |
| `DATABASE_STAGING` | Database name staging | `ORCL` |
| `USERNAME_STAGING` | Username Oracle staging | `user` |
| `PASSWORD_STAGING` | Password Oracle staging | `secret` |
| `HOST_PRODUCTION` | Host Oracle production | `oracle.example.com` |
| `PORT_PRODUCTION` | Port Oracle production | `1521` |
| `DATABASE_PRODUCTION` | Database name production | `ORCL` |
| `USERNAME_PRODUCTION` | Username Oracle production | `user` |
| `PASSWORD_PRODUCTION` | Password Oracle production | `secret` |
| `API_KEY_KAFKA_CONFLUENT` | API Key Confluent Cloud | `xxx` |
| `API_SECRET_KAFKA_CONFLUENT` | API Secret Confluent Cloud | `xxx` |
| `BOOTSTRAP_SERVER_KAFKA_CONFLUENT` | Bootstrap server Confluent | `pkc-xxx.us-east-1.aws.confluent.cloud:9092` |

### Contoh `.env` File

```env
GO_ENV=staging
KAFKA-MODELS=kafka-staging
KAFKA_BROKERS_STAGING=broker1:9092,broker2:9092
HOST_STAGING=oracle-staging.example.com
PORT_STAGING=1521
DATABASE_STAGING=ORCL
USERNAME_STAGING=user
PASSWORD_STAGING=secret
```

## Komponen Framework

### Config

```go
// Default configuration from environment
config := nanopony.NewConfig()

// Or with custom initializers
config := nanopony.BuildConfig(func(c *nanopony.Config) {
    c.App.Env = "custom"
})
```

### Database (Oracle)

```go
// From config
db, err := nanopony.NewOracleFromConfig(config)

// Or with custom config
dbConfig := nanopony.DefaultDatabaseConfig()
dbConfig.Host = "localhost"
dbConfig.Port = "1521"
db, err := nanopony.NewOracleConnection(dbConfig)
```

### Kafka Producer

```go
writer := nanopony.NewKafkaWriterFromConfig(config)
producer := nanopony.NewKafkaProducer(writer)

// Produce message
success, err := producer.Produce("topic-name", map[string]interface{}{
    "id":   1,
    "data": "hello",
})
```

### Worker Pool

```go
pool := nanopony.NewWorkerPool(5, 100)
pool.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
    // process job
    fmt.Printf("Processing: %+v\n", job.Data)
    return nil
})

// Submit job
pool.Submit(ctx, nanopony.Job{
    ID:   "job-1",
    Data: map[string]interface{}{"key": "value"},
})

// Stop pool
pool.Stop()
```

> 📖 **Deep Dive:** Baca [WORKER_POOL_EXPLAINED.md](WORKER_POOL_EXPLAINED.md) untuk penjelasan detail tentang cara kerja worker pool, blocking behavior, dan optimasi.

### Poller

```go
dataFetcher := nanopony.DataFetcherFunc(func() ([]interface{}, error) {
    // fetch data from database
    return []interface{}{data1, data2}, nil
})

pollerConfig := nanopony.DefaultPollerConfig()
pollerConfig.Interval = 5 * time.Second

poller := nanopony.NewPoller(pollerConfig, workerPool, dataFetcher)
poller.Start()
```

### Pipeline (Validator + Transformer + Processor)

```go
validator := nanopony.ValidatorFunc(func(data interface{}) error {
    if data == nil {
        return errors.New("data cannot be nil")
    }
    return nil
})

transformer := nanopony.TransformerFunc(func(data interface{}) (interface{}, error) {
    return data.(string) + "-transformed", nil
})

processor := nanopony.ProcessorFunc(func(data interface{}) error {
    fmt.Printf("Processing: %v\n", data)
    return nil
})

pipeline := nanopony.NewPipeline(processor).
    AddValidator(validator).
    AddTransformer(transformer)

err := pipeline.Process("test")
```

### Transaction Support

```go
executor := nanopony.NewTransactionExecutor(db)

err := executor.WithTransaction(func(tx *sql.Tx) error {
    // perform database operations within transaction
    _, err := tx.Exec("INSERT INTO table VALUES (?)", value)
    return err
})
```

## 📊 Performa & Benchmark

Framework NanoPony telah melalui pengujian performa menyeluruh:

- ✅ **Memory Leak Test**: 8 test cycles - **NO LEAK DETECTED**
- ✅ **Framework Creation**: 0.25 ns/op, 0 B/op (ultra fast)
- ✅ **WorkerPool Submit Parallel**: 924.8 ns/op, 7 B/op (efficient)
- ✅ **Pipeline Process Parallel**: 28.48 ns/op (sangat cepat)
- ✅ **Concurrent Safe**: 20 instances dengan memory growth hanya +23 KB

> 📖 **Detail:** Baca [BENCHMARK_REPORT.md](BENCHMARK_REPORT.md) untuk hasil lengkap.

## 🧪 Testing

Framework ini dilengkapi comprehensive test suite:

```bash
# Semua test
go test ./... -v

# Dengan race detection
go test -race ./... -v

# Benchmark
go test -bench=. -benchmem

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

> 📖 **Panduan:** Baca [TESTING_GUIDE.md](TESTING_GUIDE.md) untuk cara menjalankan test, coverage goals, dan debugging tips.

## Running Examples

```bash
cd examples
go run main.go
```

## Best Practices

1. **Selalu gunakan Graceful Shutdown** untuk memastikan semua resources dilepaskan dengan benar
2. **Gunakan Builder Pattern** untuk setup yang clean dan readable
3. **Implementasikan interface** `Repository` dan `Service` untuk code yang terstruktur
4. **Gunakan Context** untuk cancellation dan timeout
5. **Handle errors** dengan proper error handling
6. **Use Pipeline** untuk complex data processing dengan validation dan transformation
7. **Reuse WorkerPool** - jangan create/destroy频繁, buat sekali di startup
8. **Monitor error channel** - range over `pool.Errors()` untuk tracking job failures

> 📖 **Arsitektur:** Baca [ARCHITECTURE.md](ARCHITECTURE.md) untuk diagram arsitektur lengkap dan pola desain.
> 
> 📖 **Dokumentasi:** Baca [DOKUMENTASI.md](DOKUMENTASI.md) untuk panduan penggunaan setiap komponen.

## Project Structure

```
NanoPony/
├── config.go              # Configuration structures
├── config_init.go         # Configuration initialization
├── database.go            # Oracle database connection
├── kafka.go               # Kafka writer/reader
├── producer.go            # Kafka producer & consumer
├── worker.go              # Worker pool & poller
├── service.go             # Service, Pipeline, Processor
├── repository.go          # Repository base interface
├── framework.go           # Main framework builder
├── logger.go              # Structured logging with rotation
├── *_test.go              # Unit tests & benchmarks
├── README.md              # This file
├── go.mod                 # Go module definition
├── ARCHITECTURE.md        # 📐 Arsitektur & pola desain
├── BENCHMARK_REPORT.md    # 📊 Hasil benchmark & memory test
├── TESTING_GUIDE.md       # 🧪 Panduan testing
├── DOKUMENTASI.md         # 📖 Dokumentasi lengkap komponen
├── WORKER_POOL_EXPLAINED.md # ⚙️ Deep dive worker pool
└── examples/              # Example application
    ├── main.go            # Complete example
    └── go.mod
```

## Requirements

- Go 1.25.1 atau lebih baru
- Oracle Database (opsional, untuk fitur database)
- Kafka Broker (opsional, untuk fitur messaging)

## Dependencies

- [kafka-go](https://github.com/segmentio/kafka-go) - Kafka client
- [go-ora](https://github.com/sijms/go-ora) - Oracle driver
- [godotenv](https://github.com/joho/godotenv) - Environment variables loader

## License

MIT License
