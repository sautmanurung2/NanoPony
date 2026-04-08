# Dokumentasi Framework NanoPony

**NanoPony** adalah framework Go untuk integrasi Kafka-Oracle dengan arsitektur yang clean, reusable, dan production-ready. Framework ini menyediakan komponen-komponen siap pakai untuk membangun sistem data pipeline yang scalable.

---

## Daftar Isi

1. [Arsitektur Keseluruhan](#arsitektur-keseluruhan)
2. [Layer Konfigurasi](#1-layer-konfigurasi)
3. [Layer Database](#2-layer-database)
4. [Layer Kafka](#3-layer-kafka)
5. [Worker Pool](#4-worker-pool)
6. [Poller](#5-poller)
7. [Processing Pipeline](#6-processing-pipeline)
8. [Repository & Service](#7-repository--service)
9. [Framework Builder](#8-framework-builder)
10. [Proto File Compiler](#proto-file-compiler)
11. [Alur Penggunaan Tipikal](#alur-penggunaan-tipikal)
12. [Best Practices](#best-practices)

---

## Arsitektur Keseluruhan

```
┌─────────────────────────────────────────────┐
│          Framework Builder                   │
│    (Fluent API untuk wiring komponen)        │
└─────────────────────────────────────────────┘
                     ↓
┌──────────┐  ┌───────────┐  ┌──────────────┐
│ Config   │  │ Database  │  │ Kafka Writer │
│ (Env)    │  │ (Oracle)  │  │ (Producer)   │
└──────────┘  └───────────┘  └──────────────┘
                     ↓
┌──────────────────────────────────────────┐
│           Poller                         │
│   (Mengambil data secara periodik)       │
│         ↓                                │
│   Submit Jobs ke Worker Pool             │
└──────────────────────────────────────────┘
                     ↓
┌──────────────────────────────────────────┐
│          Worker Pool                     │
│   (Pemrosesan job secara concurrent)     │
│         ↓                                │
│   Memanggil JobHandler untuk tiap job    │
└──────────────────────────────────────────┘
                     ↓
┌──────────────────────────────────────────┐
│        Processing Pipeline               │
│   Validator → Transformer → Processor    │
└──────────────────────────────────────────┘
```

---

## 1. Layer Konfigurasi

**File**: `config.go` + `config_init.go`

### Tujuan

Manajemen konfigurasi terpusat berbasis environment variables.

### Proses Inisialisasi

1. **`NewConfig()`** - Memuat file `.env` dan menginisialisasi konfigurasi singleton
2. **`initApp()`** - Membaca `GO_ENV` (local/staging/production)
3. **`initKafka()`** - Menentukan model Kafka yang akan digunakan
4. **`initOracle()`** - Memuat kredensial Oracle berdasarkan environment
5. **`initOperation()`** - Mengatur mode operasi

### Environment Variables

| Variable | Deskripsi | Contoh |
|----------|-----------|--------|
| `GO_ENV` | Environment aplikasi | `local`, `staging`, `production` |
| `KAFKA-MODELS` | Model Kafka yang digunakan | `kafka-localhost`, `kafka-staging`, `kafka-production`, `kafka-confluent` |
| `KAFKA_BROKERS_STAGING` | Broker Kafka staging | `broker1:9092,broker2:9092` |
| `KAFKA_BROKERS_PRODUCTION` | Broker Kafka production | `broker1:9092,broker2:9092` |
| `HOST_STAGING` | Host Oracle staging | `oracle-staging.example.com` |
| `PORT_STAGING` | Port Oracle staging | `1521` |
| `DATABASE_STAGING` | Nama database staging | `ORCL` |
| `USERNAME_STAGING` | Username Oracle staging | `user` |
| `PASSWORD_STAGING` | Password Oracle staging | `secret` |
| `HOST_PRODUCTION` | Host Oracle production | `oracle.example.com` |
| `PORT_PRODUCTION` | Port Oracle production | `1521` |
| `DATABASE_PRODUCTION` | Nama database production | `ORCL` |
| `USERNAME_PRODUCTION` | Username Oracle production | `user` |
| `PASSWORD_PRODUCTION` | Password Oracle production | `secret` |
| `API_KEY_KAFKA_CONFLUENT` | API Key Confluent Cloud | `xxx` |
| `API_SECRET_KAFKA_CONFLUENT` | API Secret Confluent Cloud | `xxx` |
| `BOOTSTRAP_SERVER_KAFKA_CONFLUENT` | Bootstrap server Confluent | `pkc-xxx.us-east-1.aws.confluent.cloud:9092` |

### Fitur Utama

- **Singleton Pattern**: Konfigurasi hanya diinisialisasi sekali
- **`ResetConfig()`**: Reset konfigurasi untuk testing
- **Validasi Otomatis**: Memvalidasi environment variables terhadap nilai yang diperbolehkan
- **Dukungan Confluent Cloud**: Otomatis menggunakan SASL authentication jika model adalah `kafka-confluent`

### Contoh Penggunaan

```go
// Cara 1: Default configuration
config := nanopony.NewConfig()

// Cara 2: Custom configuration
config := nanopony.BuildConfig(func(c *nanopony.Config) {
    c.App.Env = "custom"
})
```

---

## 2. Layer Database

**File**: `database.go`

### Tujuan

Manajemen koneksi Oracle database dengan connection pooling.

### Proses Kerja

1. **`NewOracleConnection()`** - Membuat koneksi Oracle
   - Membangun connection URL menggunakan `go_ora.BuildUrl()`
   - Mengkonfigurasi connection pool:
     - `MaxIdleConns`: 10 (default) - Jumlah koneksi idle maksimum
     - `MaxOpenConns`: 100 (default) - Jumlah koneksi terbuka maksimum
     - `ConnMaxIdleTime`: 5 menit - Waktu maksimum koneksi idle
     - `ConnMaxLifetime`: 60 menit - Waktu maksimum lifetime koneksi
   - Melakukan ping untuk memverifikasi koneksi

2. **`NewOracleFromConfig()`** - Wrapper yang mengambil konfigurasi DB dari `Config`

3. **`CloseDB()`** - Menutup koneksi database dengan aman

### Contoh Penggunaan

```go
// Dari config
db, err := nanopony.NewOracleFromConfig(config)

// Atau dengan custom config
dbConfig := nanopony.DefaultDatabaseConfig()
dbConfig.Host = "localhost"
dbConfig.Port = "1521"
db, err := nanopony.NewOracleConnection(dbConfig)
```

---

## 3. Layer Kafka

**File**: `kafka.go`

### Tujuan

Producer dan consumer Kafka dengan dukungan standard Kafka dan Confluent Cloud.

### Sisi Producer

#### Proses Kerja

1. **`NewKafkaWriterFromConfig()`** - Membuat Kafka writer
   - Jika `kafka-confluent`: Menggunakan SASL/TLS transport dengan API key/secret
   - Jika tidak: Koneksi standard dengan round-robin balancer

2. **`NewKafkaProducer()`** - Wrapper writer dalam producer interface

3. **`ProduceWithContext()`** - Mengirim pesan JSON ke topic Kafka
   - Marshal message ke JSON
   - Buat Kafka message dengan topic dan value
   - Tulis message menggunakan writer

#### Contoh Penggunaan

```go
writer := nanopony.NewKafkaWriterFromConfig(config)
producer := nanopony.NewKafkaProducer(writer)

success, err := producer.Produce("topic-name", map[string]interface{}{
    "id":   1,
    "data": "hello",
})
```

### Sisi Consumer

#### Proses Kerja

1. **`NewKafkaConsumer()`** - Membuat Kafka reader dengan consumer group
2. **`ConsumeWithContext()`** - Loop membaca pesan:
   - Baca message dari Kafka
   - Panggil handler dengan message value
   - Commit message setelah berhasil diproses
   - Support graceful shutdown via context cancellation

#### Contoh Penggunaan

```go
consumer := nanopony.NewKafkaConsumer(nanopony.KafkaConsumerConfig{
    Brokers:     []string{"localhost:9092"},
    Topic:       "my-topic",
    GroupID:     "my-group",
    StartOffset: kafka.LastOffset,
})

err := consumer.ConsumeWithContext(ctx, func(message []byte) error {
    fmt.Printf("Received: %s\n", message)
    return nil
})
```

---

## 4. Worker Pool

**File**: `worker.go`

### Tujuan

Pemrosesan job secara concurrent dengan jumlah worker dan ukuran queue yang dapat dikonfigurasi.

### Proses Kerja

#### 1. Inisialisasi

```go
pool := nanopony.NewWorkerPool(5, 100)  // 5 workers, queue 100
```

Membuat:
- Buffered channel untuk job (`queueSize`)
- Error channel untuk tracking error
- Context untuk cancellation

#### 2. Start Worker Pool

```go
pool.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
    // Proses job
    return nil
})
```

- Mem-spawn N goroutine worker
- Setiap worker mendengarkan `jobChan`
- Memproses job via `JobHandler` callback
- Mengirim error ke `errChan` (non-blocking)

#### 3. Submit Job

```go
err := pool.Submit(ctx, nanopony.Job{
    ID:   "job-1",
    Data: map[string]interface{}{"key": "value"},
})
```

- Non-blocking submission
- Mengembalikan `ErrQueueFull` jika queue penuh

#### 4. Stop Worker Pool

```go
pool.Stop()
```

Shutdown yang graceful:
1. Cancel context
2. Tutup job channel
3. Tunggu semua worker selesai
4. Tutup error channel

### Desain Utama

Workers adalah goroutine yang blocking pada channel receive, membuat sistem ini efisien dan scalable.

---

## 5. Poller

**File**: `worker.go`

### Tujuan

Mengambil data secara periodik dan submit sebagai job ke worker pool.

### Proses Kerja

#### 1. Inisialisasi

```go
dataFetcher := nanopony.DataFetcherFunc(func() ([]interface{}, error) {
    // Ambil data dari database
    return []interface{}{data1, data2}, nil
})

pollerConfig := nanopony.DefaultPollerConfig()
// Interval: 1 detik, MaxRetries: 3, BatchSize: 100

poller := nanopony.NewPoller(pollerConfig, workerPool, dataFetcher)
```

Membuat:
- Job slots untuk membatasi concurrent polling (semaphore)
- Link ke worker pool untuk job submission
- Context untuk cancellation

#### 2. Start Polling

```go
poller.Start()
```

Memulai goroutine polling yang menggunakan ticker dengan interval yang dikonfigurasi.

#### 3. Main Polling Loop (`poll()`)

- Menunggu ticker atau cancellation
- Memanggil `pollOnce()` pada setiap tick

#### 4. Single Poll Iteration (`pollOnce()`)

1. Cek apakah job slot tersedia (rate limiting)
2. Fetch data via `DataFetcher.Fetch()`
3. Buat `Job` untuk setiap data item
4. Submit jobs ke worker pool
5. Release slot ketika selesai

### Desain Utama

Job slots berfungsi sebagai semaphore untuk mencegah worker pool kewalahan.

### Contoh Penggunaan

```go
pollerConfig := nanopony.DefaultPollerConfig()
pollerConfig.Interval = 5 * time.Second

poller := nanopony.NewPoller(pollerConfig, workerPool, dataFetcher)
poller.Start()

// Nanti...
poller.Stop()
```

---

## 6. Processing Pipeline

**File**: `service.go`

### Tujuan

Chain data melalui validators, transformers, dan processors.

### Komponen Pipeline

1. **Validator** - Memvalidasi data
2. **Transformer** - Mengubah/men-transform data
3. **Processor** - Memproses data final

### Proses Kerja

#### 1. Buat Pipeline

```go
pipeline := nanopony.NewPipeline(processor).
    AddValidator(validator).
    AddTransformer(transformer)
```

#### 2. Eksekusi Pipeline (`Process()`)

Pipeline mengeksekusi dalam urutan:

```
Data → [Validator 1] → [Validator 2] → ... → 
       [Transformer 1] → [Transformer 2] → ... → 
       [Processor]
```

- **Step 1**: Jalankan semua validator secara sequence (fail fast on error)
- **Step 2**: Jalankan semua transformer secara sequence (data mengalir melalui masing-masing)
- **Step 3**: Panggil processor final dengan data yang sudah di-transform

### Function Adapter Pattern

Memungkinkan penggunaan fungsi biasa sebagai implementasi interface:

```go
validator := nanopony.ValidatorFunc(func(data interface{}) error {
    if data == nil {
        return errors.New("data tidak boleh nil")
    }
    return nil
})

transformer := nanopony.TransformerFunc(func(data interface{}) (interface{}, error) {
    return data.(string) + "-transformed", nil
})

processor := nanopony.ProcessorFunc(func(data interface{}) error {
    fmt.Printf("Memproses: %v\n", data)
    return nil
})
```

### Contoh Lengkap

```go
pipeline := nanopony.NewPipeline(processor).
    AddValidator(validator).
    AddTransformer(transformer)

err := pipeline.Process("test")
if err != nil {
    log.Printf("Pipeline error: %v", err)
}
```

### Batch Processing

Untuk memproses batch data:

```go
batchProcessor := nanopony.BatchProcessorFunc(func(data []interface{}) error {
    fmt.Printf("Memproses batch: %d items\n", len(data))
    return nil
})

err := batchProcessor.ProcessBatch([]interface{}{"item1", "item2"})
```

---

## 7. Repository & Service

### Repository (`repository.go`)

#### Tujuan

Layer abstraksi untuk akses data.

#### Komponen

1. **`Repository` Interface** - Base interface dengan method `Close()`
2. **`BaseRepository`** - Wrapper `*sql.DB` untuk operasi DB umum
3. **`TransactionExecutor`** - Helper untuk operasi transaction

#### Transaction Support

```go
executor := nanopony.NewTransactionExecutor(db)

err := executor.WithTransaction(func(tx *sql.Tx) error {
    // Lakukan operasi database dalam transaksi
    _, err := tx.Exec("INSERT INTO table VALUES (?)", value)
    return err
})
```

- Otomatis rollback jika error
- Otomatis commit jika sukses

### Service (`service.go`)

#### Tujuan

Layer business logic dengan lifecycle management.

#### Komponen

1. **`Service` Interface** - Lifecycle methods:
   - `Initialize()` - Dipanggil saat startup
   - `Shutdown()` - Dipanggil saat shutdown

2. **`BaseService`** - Service dasar dengan tracking nama

#### Lifecycle

- Framework memanggil `Initialize()` pada startup
- Framework memanggil `Shutdown()` pada teardown

---

## 8. Framework Builder

**File**: `framework.go`

### Tujuan

Fluent builder API untuk wiring semua komponen bersama-sama.

### Build Phase (Fase Pembangunan)

```
NewFramework()
  ↓
WithConfig()          ← Set konfigurasi
  ↓
WithDatabase()        ← Buat koneksi Oracle
  ↓
WithKafkaWriter()     ← Buat Kafka writer
  ↓
WithProducer()        ← Buat Kafka producer (butuh writer)
  ↓
WithWorkerPool()      ← Buat worker pool
  ↓
WithPoller()          ← Buat poller (butuh worker pool)
  ↓
AddRepository()       ← Opsional: tambahkan repositories
  ↓
AddService()          ← Opsional: tambahkan services
  ↓
Build()               ← Kembalikan FrameworkComponents
```

### Runtime Phase (Fase Eksekusi)

#### `Start()` - Memulai Komponen

1. Starts worker pool dengan job handler
2. Starts poller
3. Initialize semua services

#### `Shutdown()` - Graceful Teardown

Urutan shutdown:
1. Stop poller
2. Stop worker pool
3. Shutdown semua services
4. Close semua repositories
5. Jalankan cleanup functions secara concurrent

### Fitur Desain Utama

- **Builder Pattern**: Method `With*()` yang fluent mengembalikan `*Framework` untuk chaining
- **Dependency Injection**: Komponen dilewatkan secara eksplisit (`WithDatabase()`, dll)
- **Cleanup Registration**: `AddCleanup()` mendaftarkan fungsi yang dipanggil saat shutdown
- **Panic on Misconfiguration**: Kesalahan konfigur menyebabkan panic langsung

---

## Alur Penggunaan Tipikal

### Contoh Lengkap

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
    // 1. Load konfigurasi
    config := nanopony.NewConfig()

    // 2. Buat data fetcher
    dataFetcher := nanopony.DataFetcherFunc(func() ([]interface{}, error) {
        // Ambil data dari database
        return []interface{}{"data1", "data2"}, nil
    })

    // 3. Build framework
    components := nanopony.NewFramework().
        WithConfig(config).
        WithDatabase().
        WithKafkaWriter().
        WithProducer().
        WithWorkerPool(5, 100).
        WithPoller(nanopony.DefaultPollerConfig(), dataFetcher).
        AddService(&myService{name: "MyService"}).
        Build()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 4. Start processing
    components.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
        log.Printf("Memproses job: %+v", job)
        
        // Kirim ke Kafka
        components.Producer.Produce("my-topic", job.Data)
        
        return nil
    })

    // 5. Tunggu interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Println("Menerima signal shutdown...")

    // 6. Graceful shutdown
    components.Shutdown(ctx)
    log.Println("Shutdown selesai")
}

type myService struct {
    name string
}

func (s *myService) Initialize() error {
    log.Printf("Inisialisasi %s", s.name)
    return nil
}

func (s *myService) Shutdown() error {
    log.Printf("Shutdown %s", s.name)
    return nil
}
```

---

## Best Practices

### 1. Selalu Gunakan Graceful Shutdown

```go
defer func() {
    components.Shutdown(ctx)
}()
```

Memastikan semua resources dilepaskan dengan benar dan data tidak hilang.

### 2. Gunakan Builder Pattern untuk Setup yang Clean

```go
// ✅ Good
framework := nanopony.NewFramework().
    WithConfig(config).
    WithDatabase().
    WithWorkerPool(5, 100).
    Build()

// ❌ Bad: Manual setup yang verbose
pool := nanopony.NewWorkerPool(5, 100)
producer := nanopony.NewKafkaProducer(writer)
// ... manual wiring yang panjang
```

### 3. Implementasikan Interface Repository dan Service

```go
type UserRepository struct {
    *nanopony.BaseRepository
}

func (r *UserRepository) GetUsers() ([]User, error) {
    rows, err := r.DB.Query("SELECT * FROM users")
    // ...
}

type UserService struct {
    *nanopony.BaseService
    repo *UserRepository
}
```

### 4. Gunakan Context untuk Cancellation dan Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := pool.Submit(ctx, job)
```

### 5. Handle Errors dengan Proper

```go
errChan := workerPool.Errors()
go func() {
    for err := range errChan {
        log.Printf("Job error: %v", err)
    }
}()
```

### 6. Gunakan Pipeline untuk Complex Data Processing

```go
pipeline := nanopony.NewPipeline(processor).
    AddValidator(validateNotNull).
    AddValidator(validateFormat).
    AddTransformer(transformToDomain).
    AddTransformer(enrichWithData)

err := pipeline.Process(rawData)
```

### 7. Konfigurasi Connection Pool Sesuai Kebutuhan

```go
dbConfig := nanopony.DefaultDatabaseConfig()
dbConfig.MaxIdleConns = 20   // Sesuaikan dengan workload
dbConfig.MaxOpenConns = 200  // Sesuaikan dengan capacity database
```

### 8. Gunakan Dependency Injection untuk Testing

Lihat contoh lengkap di `examples/complete_with_layers/` yang menunjukkan:
- Interface-based design untuk loose coupling
- Dependency injection untuk easy mocking
- Repository pattern untuk data access
- Service layer untuk business logic

---

## Proto File Compiler

NanoPony menyediakan CLI tool untuk mengompilasi file `.proto` menjadi kode Go (`.pb.go`).

### Instalasi

```bash
go install github.com/sautmanurung2/nanopony/cmd/nanopony@latest
```

### Penggunaan Dasar

```bash
# Kompilasi proto file
nanopony --proto service.proto

# Dengan output directory
nanopony --proto service.proto --output ./gen

# Dengan import path
nanopony --proto service.proto -I ./protos --output ./gen
```

### Contoh Proto File

```protobuf
syntax = "proto3";

package user;

option go_package = "./pb;userpb";

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}

service UserService {
  rpc GetUser (GetUserRequest) returns (GetUserResponse);
}

message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  User user = 1;
}
```

### Requirements

1. **protoc** - Protocol Buffers Compiler
   ```bash
   # Ubuntu/Debian
   sudo apt install protobuf-compiler
   
   # macOS
   brew install protobuf
   ```

2. **protoc-gen-go** - Go plugin
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   ```

3. **protoc-gen-go-grpc** (opsional, untuk gRPC)
   ```bash
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

### Dokumentasi Lengkap

Lihat [PROTO_COMPILER_GUIDE.md](PROTO_COMPILER_GUIDE.md) untuk panduan lengkap.

---

## Ringkalan

NanoPony adalah framework yang menyediakan:

✅ **Kafka Producer & Consumer** - Integrasi Kafka dengan `kafka-go`  
✅ **Oracle Database** - Koneksi Oracle dengan connection pooling  
✅ **Worker Pool** - Pool of workers untuk memproses jobs secara concurrent  
✅ **Poller** - Polling data dengan interval yang dapat dikonfigurasi  
✅ **Builder Pattern** - API yang clean dan mudah digunakan  
✅ **Graceful Shutdown** - Shutdown yang aman untuk semua komponen  
✅ **Environment-based Config** - Konfigurasi berbasis environment variables  
✅ **Pipeline Processing** - Validator, Transformer, dan Processor pipeline  
✅ **Transaction Support** - Database transaction helper  
✅ **Comprehensive Tests** - Unit tests untuk semua komponen utama  
✅ **Repository & Service Layer** - Interface-based design untuk clean architecture  

Framework ini dirancang untuk membangun sistem data pipeline yang robust, scalable, dan mudah di-maintain.

---

## Contoh Penggunaan

Framework ini menyediakan dua contoh:

1. **Basic Example** (`examples/main.go`) - Contoh dasar penggunaan framework
2. **Complete Example** (`examples/complete_with_layers/`) - Contoh lengkap dengan:
   - Interface-based design
   - Repository pattern untuk data access
   - Service layer untuk business logic
   - Dependency injection
   - Transaction management
   - Pipeline processing
   - Mock implementations untuk testing

Mulai dari `examples/complete_with_layers/` untuk melihat arsitektur production-ready.
