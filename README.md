# NanoPony

[![Go Reference](https://pkg.go.dev/badge/github.com/sautmanurung2/nanopony.svg)](https://pkg.go.dev/github.com/sautmanurung2/nanopony)
[![Go Report Card](https://goreportcard.com/badge/github.com/sautmanurung2/nanopony)](https://goreportcard.com/report/github.com/sautmanurung2/nanopony)

**NanoPony** adalah framework Go untuk integrasi Kafka-Oracle dengan arsitektur yang clean, reusable, dan production-ready.

## 📚 Dokumentasi Lengkap

| Dokumen                      | Deskripsi                                           | Link                                                 |
| ---------------------------- | --------------------------------------------------- | ---------------------------------------------------- |
| **📖 Dokumentasi Lengkap**   | Panduan komprehensif semua komponen framework       | [DOKUMENTASI.md](DOKUMENTASI.md)                     |
| **🏗️ Arsitektur**            | Diagram arsitektur, pola desain, dan alur data      | [ARCHITECTURE.md](ARCHITECTURE.md)                   |
| **🔧 Panduan Testing**       | Cara menjalankan test, coverage, dan best practices | [TESTING_GUIDE.md](TESTING_GUIDE.md)                 |
| **📊 Benchmark Report**      | Hasil benchmark performa dan memory leak test       | [BENCHMARK_REPORT.md](BENCHMARK_REPORT.md)           |
| **⚙️ Worker Pool Deep Dive** | Penjelasan detail cara kerja worker pool            | [WORKER_POOL_EXPLAINED.md](WORKER_POOL_EXPLAINED.md) |

## Fitur Lengkap

### 🏗️ Core Framework & Builder Pattern

- ✅ **Fluent Builder Pattern** - Setup yang clean dan readable dengan method chaining
- ✅ **Lifecycle Management** - Start/Shutdown terkoordinasi untuk semua komponen
- ✅ **Dependency Injection** - Support untuk custom instances atau auto-create dari config
- ✅ **Graceful Shutdown** - Shutdown berurutan: Poller → Worker Pool → Cleanup
- ✅ **Concurrent Cleanup** - Cleanup functions dijalankan secara concurrent saat shutdown
- ✅ **Accessor Methods** - `GetDB()`, `GetProducer()`, `GetConfig()`, `GetWorkerPool()`, `GetPoller()`
- ✅ **Error Types** - Built-in errors: `ErrQueueFull`, `ErrConfigNotSet`, `ErrDatabaseNotSet`, dll

### ⚙️ Configuration System

- ✅ **Environment-based Config** - Konfigurasi via environment variables
- ✅ **Multi-environment Support** - Local, Staging, Production
- ✅ **Auto .env Loading** - Otomatis load dari `.env` file via `godotenv`
- ✅ **Config Validation** - Validasi environment dan Kafka models
- ✅ **Dynamic Variable Resolution** - Resolves env-specific variables (HOST_STAGING, HOST_PRODUCTION, dll)
- ✅ **Confluent Cloud Support** - SASL/TLS authentication untuk Confluent Cloud
- ✅ **Custom Operation Modes** - Dapat menambahkan validasi operation mode custom
- ✅ **BuildConfig Pattern** - Alternative builder dengan custom initializer functions
- ✅ **Dynamic Config** - Load environment variables arbitrary dengan prefix tertentu via `LoadDynamic()`

### 🧠 Memory Monitoring

- ✅ **Real-time Statistics** - Mengambil statistik alokasi memori dan jumlah goroutine
- ✅ **Human-readable Format** - Auto conversion bytes ke KB/MB/GB
- ✅ **Background Monitoring** - Monitoring memori secara periodik via background goroutine
- ✅ **Stop/Halt Control** - Stop monitoring kapan saja dengan return function

### 🗄️ Oracle Database

- ✅ **Connection Pooling** - Configurable: MaxIdleConns, MaxOpenConns, ConnMaxIdleTime, ConnMaxLifetime
- ✅ **Auto URL Building** - Oracle URL generation via `go_ora.BuildUrl`
- ✅ **Connection Health Check** - Auto ping saat inisialisasi
- ✅ **Query Interpolation** - Debug query dengan named parameter replacement
- ✅ **Transaction Support** - BeginTx, Commit, Auto-rollback on error
- ✅ **Safe Close** - Nil-safe database connection close
- ✅ **Sensible Defaults** - MaxIdleConns=10, MaxOpenConns=100, IdleTime=5min, Lifetime=60min


### 🚀 Worker Pool

- ✅ **Concurrent Job Processing** - Multiple workers untuk proses jobs secara parallel
- ✅ **Bounded Queue** - Backpressure dengan queue size limit
- ✅ **Non-blocking Submit** - Returns `ErrQueueFull` jika queue penuh
- ✅ **Submit Blocking** - Menunggu sampai queue tersedia (recommended untuk feedback backpressure)
- ✅ **Thread-safe** - sync.RWMutex untuk state management
- ✅ **Error Channel** - Monitoring job failures via `Errors()` channel
- ✅ **Graceful Shutdown** - Cancel context → Close channel → Wait workers → Close error channel
- ✅ **Idempotent Start/Stop** - Aman dipanggil berkali-kali
- ✅ **Job Metadata** - Job struct dengan ID (auto-prefix session), Data (any), Meta (map[string]any)
- ✅ **Overflow Protection** - Warning log jika error channel penuh

### 🔄 Poller

- ✅ **Periodic Data Fetching** - Polling data dengan interval configurable
- ✅ **Batch Processing** - BatchSize untuk kontrol jumlah data per poll
- ✅ **Retry Mechanism** - MaxRetries dan RetryDelay untuk error handling
- ✅ **Job Slot Semaphore** - JobSlotSize untuk rate limiting concurrent operations
- ✅ **Auto Metadata Injection** - ID (session-based), source, timestamp untuk setiap polled job
- ✅ **DataFetcher Interface** - Interface untuk custom data sources
- ✅ **Function Adapter** - DataFetcherFunc untuk kemudahan penggunaan
- ✅ **Empty Data Handling** - Graceful handling jika tidak ada data

### 📨 Kafka Producer

- ✅ **MessageProducer Interface** - Abstraksi untuk producing messages
- ✅ **Context Support** - ProduceWithContext untuk cancellation dan timeout
- ✅ **JSON Serialization** - Auto marshal messages ke JSON
- ✅ **Kafka-go Integration** - Menggunakan `segmentio/kafka-go`
- ✅ **Safe Close** - Proper resource cleanup

### 📥 Kafka Consumer

- ✅ **Reader-based Consumer** - Menggunakan kafka-go Reader
- ✅ **Consumer Groups** - Support untuk GroupID
- ✅ **Offset Management** - StartOffset config (FirstOffset, LastOffset)
- ✅ **MessageHandler** - Function handler untuk setiap message
- ✅ **Auto Commit** - Commit on success, skip on error (re-delivery)
- ✅ **Context Support** - ConsumeWithContext untuk graceful shutdown

### 🎯 Kafka Writer Configuration

- ✅ **Multiple Brokers** - Support untuk multiple Kafka brokers
- ✅ **Load Balancing** - Round-robin balancer (default)
- ✅ **Batch Timeout** - Configurable batch timeout
- ✅ **SASL/TLS Transport** - Untuk Confluent Cloud authentication
- ✅ **Auto Config Resolution** - Resolve brokers berdasarkan environment + model

### 📝 Logging System

- ✅ **Structured Logging** - Rich metadata: timestamps, duration, reference IDs, process names, dll
- ✅ **Multiple Output Modes** - Console, File, Elasticsearch, atau Hybrid (Configurable via `LOG_OUTPUT_MODE`)
- ✅ **Async Logging** - Buffered channel (cap: 1000) untuk performa tinggi tanpa blocking main process
- ✅ **Log File Rotation** - Via lumberjack: MaxSize=100MB, MaxBackups=3, MaxAge=28 days
- ✅ **Auto Date-based Indexing** - Elasticsearch index pattern: `{prefix}YYYYMMDD`
- ✅ **Elasticsearch Integration** - Auto-init client, test connectivity
- ✅ **Multiple Auth Methods** - Username/Password atau APIKey untuk Elasticsearch
- ✅ **Payload Processing** - Auto-convert map, string, []byte, any ke map[string]any
- ✅ **Request/Response Logging** - Structured RequestLog dan ResponseLog
- ✅ **Service Name Prefixing** - Auto prefix dengan "GO\_" dan "GO-Producer-"
- ✅ **Log Directory** - Defaults ke `./logs/orion-to-core-YYYY-MM-DD.log` (Auto-created)


### 🛡️ Concurrency & Safety

- ✅ **Context Propagation** - Throughout all components (worker pool, poller, producer, consumer)
- ✅ **Thread-safe Operations** - sync.RWMutex, sync.WaitGroup, sync.Once
- ✅ **Bounded Channels** - Backpressure via bounded channels
- ✅ **Non-blocking Operations** - Submit tanpa blocking
- ✅ **Concurrent Cleanup** - Cleanup functions dijalankan parallel
- ✅ **Mutex-safe Error Aggregation** - Error collection saat shutdown
- ✅ **Idempotent Methods** - Aman dipanggil berkali-kali
- ✅ **Singleton Patterns** - Config dan logger dengan sync.Once

### 🧪 Testing & Quality

- ✅ **Comprehensive Test Suite** - Unit tests untuk semua komponen utama
- ✅ **Race Detection** - Thread-safe verified dengan `go test -race`
- ✅ **Benchmark Tests** - Performance testing untuk semua critical paths
- ✅ **Memory Leak Testing** - Verified no memory leaks dalam long-running operations
- ✅ **Table-driven Tests** - Best practices untuk test coverage

### 📊 Performance Highlights

- ⚡ **Framework Creation**: 2.1 µs/op (Comparable to Fiber)
- ⚡ **Throughput**: ~39x faster than Fiber for internal processing
- ⚡ **Memory Efficiency**: 16 B/op with only 1 allocation per job
- ⚡ **Memory Leak Test**: NO LEAK DETECTED (Verified over 40+ cycles)
- ⚡ **Concurrent Safe**: 20 instances with memory growth only +50 KB

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
    // Inisialisasi konfigurasi
    config := nanopony.NewConfig()

    // Buat framework
    framework := nanopony.NewFramework().
        WithConfig(config).
        WithDatabase().
        WithKafkaWriter().
        WithProducer().
        WithWorkerPool(5, 100).
        WithPoller(nanopony.DefaultPollerConfig(), &myDataFetcher{})

    components := framework.Build()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    components.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
        log.Printf("Memproses job: %+v", job)
        return nil
    })

    // Tunggu interrupt
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    // Shutdown yang aman
    if err := components.Shutdown(ctx); err != nil {
        log.Printf("Error saat shutdown: %v", err)
    }
}

type myDataFetcher struct{}

func (f *myDataFetcher) Fetch() ([]interface{}, error) {
    // Ambil data dari database
    return []interface{}{"data1", "data2"}, nil
}


```

## Konfigurasi Environment

NanoPony menggunakan environment variables untuk konfigurasi:

### Environment Variables

| Variable                           | Deskripsi                  | Contoh                                                                    |
| ---------------------------------- | -------------------------- | ------------------------------------------------------------------------- |
| `GO_ENV`                           | Environment aplikasi       | `local`, `staging`, `production`                                          |
| `KAFKA_MODELS`                     | Model Kafka                | `kafka-localhost`, `kafka-staging`, `kafka-production`, `kafka-confluent` |
| `KAFKA_BROKERS_STAGING`            | Broker Kafka staging       | `broker1:9092,broker2:9092`                                               |
| `KAFKA_BROKERS_PRODUCTION`         | Broker Kafka production    | `broker1:9092,broker2:9092`                                               |
| `HOST_STAGING`                     | Host Oracle staging        | `oracle-staging.example.com`                                              |
| `PORT_STAGING`                     | Port Oracle staging        | `1521`                                                                    |
| `DATABASE_STAGING`                 | Database name staging      | `ORCL`                                                                    |
| `USERNAME_STAGING`                 | Username Oracle staging    | `user`                                                                    |
| `PASSWORD_STAGING`                 | Password Oracle staging    | `secret`                                                                  |
| `HOST_PRODUCTION`                  | Host Oracle production     | `oracle.example.com`                                                      |
| `PORT_PRODUCTION`                  | Port Oracle production     | `1521`                                                                    |
| `DATABASE_PRODUCTION`              | Database name production   | `ORCL`                                                                    |
| `USERNAME_PRODUCTION`              | Username Oracle production | `user`                                                                    |
| `PASSWORD_PRODUCTION`              | Password Oracle production | `secret`                                                                  |
| `API_KEY_KAFKA_CONFLUENT`          | API Key Confluent Cloud    | `xxx`                                                                     |
| `API_SECRET_KAFKA_CONFLUENT`       | API Secret Confluent Cloud | `xxx`                                                                     |
| `BOOTSTRAP_SERVER_KAFKA_CONFLUENT` | Bootstrap server Confluent | `pkc-xxx.us-east-1.aws.confluent.cloud:9092`                              |
| `LOG_OUTPUT_MODE`                  | Mode output log            | `console`, `file`, `elasticsearch`, `hybrid`                              |
| `LOG_FILE_PREFIX`                  | Prefix nama file log       | `any-prefix` (Default: `orion-to-core`)                                   |

### Contoh `.env` File

```env
GO_ENV=staging
KAFKA_MODELS=kafka-staging
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

// Load dynamic environment variables
config.LoadDynamic("CUSTOM_")
```

### Database (Oracle)

```go
// Dari konfigurasi
db, err := nanopony.NewOracleFromConfig(config)

// Atau dengan konfigurasi kustom
dbConfig := nanopony.DefaultDatabaseConfig()
dbConfig.Host = "localhost"
dbConfig.Port = "1521"
db, err := nanopony.NewOracleConnection(dbConfig)
```

### Kafka Producer

```go
writer := nanopony.NewKafkaWriterFromConfig(config)
producer := nanopony.NewKafkaProducer(writer)

// Kirim pesan
success, err := producer.Produce("topic-name", map[string]interface{}{
    "id":   1,
    "data": "hello",
})
```

### Worker Pool

```go
pool := nanopony.NewWorkerPool(5, 100)
pool.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
    // proses job
    fmt.Printf("Memproses: %+v\n", job.Data)
    return nil
})

// Submit job
pool.Submit(ctx, nanopony.Job{
    ID:   "job-1",
    Data: map[string]interface{}{"key": "value"},
})

// Hentikan pool
pool.Stop()
```

> 📖 **Deep Dive:** Baca [WORKER_POOL_EXPLAINED.md](WORKER_POOL_EXPLAINED.md) untuk penjelasan detail tentang cara kerja worker pool, blocking behavior, dan optimasi.

### Poller

```go
dataFetcher := nanopony.DataFetcherFunc(func() ([]interface{}, error) {
    // ambil data dari database
    return []interface{}{data1, data2}, nil
})

pollerConfig := nanopony.DefaultPollerConfig()
pollerConfig.Interval = 5 * time.Second

poller := nanopony.NewPoller(pollerConfig, workerPool, dataFetcher)
poller.Start()
```

### Memory Monitoring

```go
// Print memory summary
nanopony.PrintMemoryStats()

// Start background monitor
stop := nanopony.MonitorMemory(5 * time.Second)
defer stop()
```


## 📊 Performa & Benchmark

Framework NanoPony telah melalui pengujian performa menyeluruh:

- ✅ **High Throughput**: ~39x lebih cepat daripada Fiber untuk pemrosesan internal.
- ✅ **Ultra Efficient**: Hanya mengonsumsi **16 byte** dengan **1 alokasi** per job.
- ✅ **Micro-second Setup**: Inisialisasi framework hanya membutuhkan waktu **2.1 µs**.
- ✅ **Memory Leak Test**: Lolos uji 40+ siklus setup-shutdown tanpa kebocoran.
- ✅ **Multi-Framework**: Terbukti lebih efisien dibandingkan Fiber, Echo, dan Iris untuk job processing.

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

## Menjalankan Contoh

```bash
cd examples
go run main.go
```

## Best Practices

1. **Selalu gunakan Graceful Shutdown** untuk memastikan semua resources dilepaskan dengan benar
2. **Gunakan Builder Pattern** untuk setup yang clean dan readable
3. **Gunakan pola arsitektur layer** untuk code yang terstruktur
4. **Gunakan Context** untuk cancellation dan timeout
5. **Handle errors** dengan proper error handling
6. **Reuse WorkerPool** - jangan buat/hapus secara terus-menerus, buat sekali di startup
7. **Monitor error channel** - gunakan `range` pada `pool.Errors()` untuk melacak kegagalan job

> 📖 **Arsitektur:** Baca [ARCHITECTURE.md](ARCHITECTURE.md) untuk diagram arsitektur lengkap dan pola desain.
>
> 📖 **Dokumentasi:** Baca [DOKUMENTASI.md](DOKUMENTASI.md) untuk panduan penggunaan setiap komponen.
>
> 📖 **Contoh Lainnya:** Lihat folder `/examples` untuk implementasi:
>
> - `dynamic_config`: Penggunaan konfigurasi dinamis.
> - `memory_monitoring`: Integrasi monitoring memori.
> - `layered_separation`: Pemisahan layer repository dan service yang clean.

## Project Structure

```
├── job.go                 # Definisi unit kerja (Job) & Handler
├── worker.go              # Logic Worker Pool & Concurrency
├── poller.go              # Logic Data Poller & Rate Limiting
├── database.go            # Koneksi Oracle DB & pooling
├── kafka.go               # Wrapper kafka-go reader/writer
├── producer.go            # Logic Kafka producer & consumer
├── framework.go           # Main builder & lifecycle management
├── config.go              # Konfigurasi sistem (Structs)
├── config_init.go         # Inisialisasi environment vars (Logic)
├── logger.go              # Structured logging (Public API)
├── logger_internal.go     # Internal logging machinery & state
├── memory.go              # Memory monitoring utilities
├── *_test.go              # Unit tests & benchmarks
├── go.mod                 # Go module definition
├── ARCHITECTURE.md        # 📐 Arsitektur & pola desain
├── BENCHMARK_REPORT.md    # 📊 Hasil benchmark & memory test
├── TESTING_GUIDE.md       # 🧪 Panduan testing
├── CONTRIBUTING.md        # 🤝 Panduan kontribusi project
├── DOKUMENTASI.md         # 📖 Dokumentasi lengkap komponen
├── WORKER_POOL_EXPLAINED.md # ⚙️ Deep dive worker pool
├── src/                   # Source tambahan (logs, etc)
└── examples/              # Koleksi contoh aplikasi
    ├── main.go
    ├── dynamic_config/
    ├── memory_monitoring/
    └── layered_separation/
```

## Persyaratan

- Go 1.25.1 atau lebih baru (v0.0.30)
- Oracle Database (opsional, untuk fitur database)
- Kafka Broker (opsional, untuk fitur messaging)

## Dependensi

- [kafka-go](https://github.com/segmentio/kafka-go) - Kafka client
- [go-ora](https://github.com/sijms/go-ora) - Oracle driver
- [godotenv](https://github.com/joho/godotenv) - Environment variables loader

## Lisensi

MIT License
