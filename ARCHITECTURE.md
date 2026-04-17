# Framework NanoPony - Dokumentasi Arsitektur

## 📋 Daftar Isi

1. [Ringkasan](#ringkasan)
2. [Diagram Arsitektur](#diagram-arsitektur)
3. [Komponen Inti](#komponen-inti)
   - [Layer Konfigurasi](#1-layer-konfigurasi)
   - [Layer Database](#2-layer-database)
   - [Layer Kafka](#3-layer-kafka)
   - [Producer & Consumer](#4-producer--consumer)
   - [Worker Pool](#5-worker-pool)
   - [Poller](#6-poller)
   - [Framework Builder](#9-framework-builder)
   - [Logger](#10-logger)
4. [Diagram Alur Data](#diagram-alur-data)
5. [Pola Desain](#pola-desain)
6. [Referensi Konfigurasi](#referensi-konfigurasi)
7. [Referensi Interface](#referensi-interface)
8. [Panduan Cepat](#panduan-cepat)

---

## Ringkasan

**NanoPony** adalah framework Go (`github.com/sautmanurung2/nanopony`) yang menyediakan platform integrasi Kafka-Oracle yang komprehensif dengan kemampuan worker pool dan polling. Framework ini dirancang menggunakan pola builder yang fluent untuk menghubungkan koneksi database, Kafka producer/consumer, worker pool konkuren, dan data poller.

**Fitur Utama:**
- ✅ **Kafka Producer & Consumer** - Integrasi dengan Kafka menggunakan `kafka-go`
- ✅ **Oracle Database** - Koneksi menggunakan `go-ora` dengan connection pooling
- ✅ **Worker Pool** - Pemrosesan job konkuren dengan queue bounded
- ✅ **Poller** - Pengambilan data periodik dengan interval yang dapat dikonfigurasi
- ✅ **Builder Pattern** - API yang clean dan fluent
- ✅ **Graceful Shutdown** - Teardown yang aman untuk semua komponen
- ✅ **Konfigurasi Berbasis Environment** - Konfigurasi melalui environment variables
- ✅ **Logging Terstruktur** - Rotasi file dan integrasi Elasticsearch

**Dependensi Inti:**
- `segmentio/kafka-go` - Kafka client
- `sijms/go-ora/v2` - Oracle driver
- `elastic/go-elasticsearch/v8` - Elasticsearch client
- `joho/godotenv` - Environment variables loader
- `natefinch/lumberjack` - Rotasi log

---

## Diagram Arsitektur

```
┌─────────────────────────────────────────────────────────────┐
│                    Framework Builder                         │
│  (framework.go - Fluent API, mengorkestrasi semua komponen)  │
└───────────────┬───────────────────────────────┬─────────────┘
                │                               │
                ▼                               ▼
┌───────────────────────┐         ┌─────────────────────────────┐
│   Layer Konfigurasi    │         │   FrameworkComponents       │
│  config.go             │         │   (menyimpan semua artifak) │
│  config_init.go        │         │   Start() / Shutdown()      │
│  (env-based, singleton)│         │   Getters untuk DB, Producer│
└───────────┬───────────┘         └─────────────────────────────┘
            │
    ┌───────┼───────────┬──────────────────────┐
    ▼       ▼           ▼                      ▼
┌────────┐ ┌──────────┐ ┌──────────────┐ ┌──────────────┐
│ Oracle │ │ Standard │ │ Confluent    │ │ Elastic      │
│  DB    │ │ Kafka    │ │ Cloud Kafka  │ │ Search       │
│database│ │ kafka.go │ │ (SASL/TLS)   │ │ logger.go    │
│  .go   │ │          │ │              │ │              │
└───┬────┘ └────┬─────┘ └──────┬───────┘ └──────────────┘
    │           │              │
    ▼           ▼              ▼
┌──────────────────────────────────────────────────────────┐
│                  Layer Akses Data                         │
│  producer.go  - KafkaProducer, KafkaConsumer              │
└──────────────────────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────────────────────┐
│                 Layer Pemrosesan                          │
│  worker.go  - WorkerPool, Poller, DataFetcher             │
└──────────────────────────────────────────────────────────┘
```

---

## Komponen Inti

### 1. Layer Konfigurasi

**File:** `config.go`, `config_init.go`

**Tujuan:** Singleton konfigurasi terpusat yang digerakkan oleh environment.

**Struct Utama:**
- `Config` - Container tingkat atas yang menampung `AppConfig`, `OracleConfig`, `KafkaConfig`, `KafkaConfluentConfig`, `ElasticSearchConfig`
- `envConfig` - Helper validasi dengan `validValues` dan `defaultVal`
- `oracleEnv` - Pemetaan nama environment variable (staging vs production)

**Fungsi Utama:**
- `NewConfig()` - Inisialisasi singleton; memuat `.env`, menjalankan semua fungsi `init*`
- `BuildConfig(initFuncs ...func(*Config))` - Memungkinkan callback inisialisasi kustom
- `ResetConfig()` - Reset singleton (untuk testing)
- `getEnvValue()` - Membaca env var dengan validasi terhadap nilai yang diperbolehkan
- `getOracleEnv()` - Mengembalikan prefix nama env var staging atau production
- `getKafkaBrokers()` - Menyelesaikan alamat broker berdasarkan kombinasi env + kafka-model
- `initKafka()` - Rute ke `initKafkaConfluent()` ketika model adalah `kafka-confluent`

**Pola Desain:** Singleton + Strategy (routing berbasis env). Konfigurasi divalidasi saat load; nilai env yang tidak valid kembali ke default.

---

### 2. Layer Database

**File:** `database.go`

**Tujuan:** Manajemen koneksi database Oracle dengan connection pooling.

**Struct Utama:** `DatabaseConfig`
```go
type DatabaseConfig struct {
    Host            string
    Port            string
    Database        string
    Username        string
    Password        string
    MaxIdleConns    int
    MaxOpenConns    int
    ConnIdleTime    time.Duration
    ConnMaxLifetime time.Duration
}
```

**Fungsi Utama:**
- `NewOracleConnection(config DatabaseConfig)` - Membangun Oracle URL via `go_ora.BuildUrl`, membuka `*sql.DB`, mengkonfigurasi pool, ping untuk verifikasi
- `NewOracleFromConfig(conf *Config)` - Wrapper yang mengekstrak konfigurasi Oracle dari `Config`
- `CloseDB(db *sql.DB)` - Close yang nil-safe
- `InterpolateQuery(query, args...)` - Mengganti param `:Named` dengan nilai terformat (berguna untuk debug logging)
- `LogInterpolatedQuery(query, args...)` - Log SQL yang telah diinterpolasi

**Pengaturan Pool Default:**
- `MaxIdleConns`: 10
- `MaxOpenConns`: 100
- `ConnIdleTime`: 5 menit
- `ConnMaxLifetime`: 60 menit

**Pola Desain:** Factory + Wrapper. Variabel `oracleDB` di package-level berfungsi sebagai singleton accessor via `GetOracleDB()`.

---

### 3. Layer Kafka

**File:** `kafka.go`

**Tujuan:** Pembuatan Kafka writer yang mendukung Kafka standar dan Confluent Cloud (SASL/TLS).

**Struct Utama:** `KafkaWriterConfig`
```go
type KafkaWriterConfig struct {
    Brokers      []string
    Balancer     kafka.Balancer
    BatchTimeout time.Duration
    Transport    kafka.RoundTripper
}
```

**Fungsi Utama:**
- `NewKafkaWriter(config)` - Membuat `*kafka.Writer` dengan round-robin balancer dan batch timeout 1μs
- `NewKafkaWriterFromConfig(conf)` - Cabang pada `kafka-confluent`: membuat transport SASL dengan TLS 1.2+, atau menggunakan broker polos
- `createSASLTransport(apiKey, apiSecret)` - Membuat `kafka.Transport` dengan TLS + `plain.Mechanism`
- `CloseKafkaWriter(writer)` - Close yang nil-safe

---

### 4. Producer & Consumer

**File:** `producer.go`

**Tujuan:** Interface produksi dan konsumsi pesan dengan serialisasi JSON.

**Interface Utama:**
- `MessageProducer`
  ```go
  type MessageProducer interface {
      Produce(topic string, message interface{}) (bool, error)
      ProduceWithContext(ctx context.Context, topic string, message interface{}) (bool, error)
      Close() error
  }
  ```

**Struct Utama:**
- `KafkaProducer` - Membungkus `*kafka.Writer`, marshals pesan ke JSON
- `KafkaConsumer` - Membungkus `*kafka.Reader`, membaca pesan dalam loop, commit offset

**Fungsi Utama:**
- `NewKafkaProducer(writer)` - Membuat producer dari writer
- `ProduceWithContext(ctx, topic, message)` - JSON marshal, membuat `kafka.Message`, menulis via `WriteMessages`
- `NewKafkaConsumer(config)` - Membuat consumer dengan `kafka.ReaderConfig`
- `ConsumeWithContext(ctx, handler)` - Loop blocking: `ReadMessage` → `handler` → `CommitMessages`

**Pola Desain:** Interface + Implementasi (Strategy). Interface `MessageProducer` memungkinkan penukaran implementasi untuk testing.

---

### 5. Worker Pool

**File:** `worker.go` (bagian WorkerPool)

**Tujuan:** Pemrosesan job konkuren dengan goroutine pool ukuran tetap dan queue bounded.

**Struct Utama:**
- `Job`
  ```go
  type Job struct {
      ID   string
      Data any
      Meta map[string]any
  }
  ```
- `JobHandler` - `func(ctx context.Context, job Job) error`
- `WorkerPool` - Mengelola `numWorkers`, `jobChan` buffered, `errChan`, `sync.WaitGroup`, context, flag `running` yang dilindungi mutex

**Fungsi Utama:**
- `NewWorkerPool(numWorkers, queueSize)` - Membuat channel buffered, context yang dapat dibatalkan
- `Start(ctx, handler)` - Spawn N goroutine worker; dilindungi mutex terhadap double-start
- `worker(ctx, id)` - Select loop: pembatalan context atau konsumsi job; error dikirim ke `errChan` secara non-blocking
- `Submit(ctx, job)` - Submit non-blocking; mengembalikan `ErrQueueFull` jika channel penuh
- `Stop()` - Dilindungi mutex; batal context, tutup `jobChan`, tunggu via `wg.Wait()`, tutup `errChan`
- `Errors()` - Mengembalikan channel error read-only untuk monitoring eksternal

**Pola Desain:** Worker Pool (goroutine pool bounded dengan komunikasi berbasis channel). Thread-safe via `sync.RWMutex`.

---

### 6. Poller

**File:** `worker.go` (bagian Poller)

**Tujuan:** Pengambilan data periodik dengan submit job yang di-rate-limit ke worker pool.

**Struct Utama:**
- `PollerConfig`
  ```go
  type PollerConfig struct {
      Interval    time.Duration
      MaxRetries  int
      RetryDelay  time.Duration
      BatchSize   int
      JobSlotSize int
  }
  ```
- `Poller` - Menampung config, job slots (channel semaphore), referensi worker pool, data fetcher, context
- Interface `DataFetcher` - `Fetch() ([]any, error)`
- `DataFetcherFunc` - Function adapter yang mengimplementasikan `DataFetcher`

**Fungsi Utama:**
- `NewPoller(config, workerPool, dataFetcher)` - Mengisi sebelumnya channel `jobSlots` dengan token `JobSlotSize`
- `Start()` - Memulai goroutine dengan `time.NewTicker`
- `poll()` - Loop ticker; memanggil `pollOnce()`
- `pollOnce()` - Akuisisi job slot (non-blocking), fetch data, submit setiap item sebagai `Job` ke worker pool
- `releaseSlot()` - Mengembalikan token ke channel job slots
- `Stop()` - Batal context, tunggu goroutine keluar

**Pola Desain:** Polling berbasis timer + Rate limiting semaphore (channel jobSlots). Poller TIDAK melepaskan slot sampai setelah data di-fetch; jika data kosong atau fetch gagal, slot segera dilepaskan. Job di-submit ke worker pool, yang menangani backpressure via queue-nya sendiri.

---

### 7. Framework Builder

**File:** `framework.go`

**Tujuan:** Builder fluent yang menghubungkan semua komponen bersama dengan manajemen dependency.

**Struct Utama:** `Framework` - Menampung semua referensi komponen, fungsi cleanup, dan flag `built`.

**Method Builder:** (semua mengembalikan `*Framework` untuk chaining)

| Method | Dependency | Membuat atau Menerima |
|--------|-----------|-------------------|
| `WithConfig(config)` | Tidak ada | Menerima `*Config` |
| `WithDatabase()` | Config diperlukan | Membuat Oracle dari config |
| `WithDatabaseFromConnection(db)` | Tidak ada | Menerima `*sql.DB` yang ada |
| `WithKafkaWriter()` | Config diperlukan | Membuat writer dari config |
| `WithKafkaWriterFromInstance(writer)` | Tidak ada | Menerima `*kafka.Writer` yang ada |
| `WithProducer()` | KafkaWriter diperlukan | Membuat producer |
| `WithProducerFromInstance(producer)` | Tidak ada | Menerima `*KafkaProducer` yang ada |
| `WithWorkerPool(n, size)` | Tidak ada | Membuat worker pool |
| `WithWorkerPoolFromInstance(pool)` | Tidak ada | Menerima pool yang ada |
| `WithPoller(config, fetcher)` | WorkerPool diperlukan | Membuat poller |
| `WithPollerFromInstance(poller)` | Tidak ada | Menerima poller yang ada |
| `AddCleanup(fn)` | Tidak ada | Mendaftarkan fungsi cleanup |

**`Build()`** mengembalikan `FrameworkComponents` - struct yang mengekspos semua komponen yang telah di-wire ditambah slice privat untuk fungsi cleanup. Panic pada double-build.

**`FrameworkComponents.Start(ctx, handler)`:**
1. Memulai WorkerPool dengan handler
2. Memulai Poller

**`FrameworkComponents.Shutdown(ctx)` (teardown berurutan):**
1. Hentikan Poller
2. Hentikan WorkerPool
3. Jalankan semua fungsi cleanup secara konkuren (goroutine + WaitGroup)
4. Kembalikan error pertama yang ditemui

**Pola Desain:** Builder + Dependency Injection. Setiap method `With*` memvalidasi prasyarat dan panic pada dependency yang hilang (misalnya `WithProducer()` panic jika `kafkaWriter` nil).

---

### 8. Logger

**File:** `logger.go`

**Tujuan:** Logging terstruktur dengan rotasi file, output console, dan integrasi Elasticsearch.

**Struct Utama:** `LoggerEntry` - Log terstruktur kaya dengan timestamp, ID referensi, nama proses, payload request, dan detail response.

**Fungsi Utama:**
- `NewLogger(serviceName, userLogin, referenceId, ...)` - Membuat entri log, menginisialisasi writer file log (lumberjack untuk rotasi)
- `LoggingData(level, payload, response)` - Memproses payload, rute ke console/Elasticsearch/hybrid berdasarkan env var `LOG_OUTPUT_MODE`
- `SendToFile(level, response)` - Menulis log JSON ke file terrotasi
- `SendToElasticSearch(level, payload, response)` - Mengindeks log ke Elasticsearch dengan nama index berakhiran tanggal
- `InitElasticSearch()` - Menginisialisasi client Elasticsearch dari config

**Pola Desain:** Lazy init singleton (`sync.Once` untuk writer file log), Strategy (routing mode output).

---

## Diagram Alur Data

### Alur Poll-to-Process

```
   Ticker (setiap Interval)
        │
        ▼
   pollOnce()
        │
        ├── Akuisisi job slot (semaphore)
        │
        ▼
   DataFetcher.Fetch()  ──►  []any  (item data mentah)
        │
        ▼
   Untuk setiap item:
        │
        ├── Buat Job{Data: item}
        │
        ▼
   WorkerPool.Submit(job)  ──►  jobChan (buffered)
                                      │
                                      ▼
                              Goroutine worker
                                      │
                                      ▼
                              JobHandler(ctx, job)
                                      │
                                      │
                                      ▼
                          Kafka Produce / Database Query
```

### Alur Lifecycle Framework

```
   NewFramework()
        │
        ▼
   WithConfig ──► WithDatabase ──► WithKafkaWriter ──► WithProducer
        │              │                  │                 │
        ▼              ▼                  ▼                 ▼
   WithWorkerPool ──► WithPoller
        │
        ▼
   Build() ──► FrameworkComponents
        │
        ▼
   Start(ctx, handler)
        │
        ├── WorkerPool.Start(ctx, handler)
        ├── Poller.Start()
        │
        │  ... berjalan ...
        │
        ▼
   Shutdown(ctx)
        │
        ├── Poller.Stop()
        ├── WorkerPool.Stop()
        └── Fungsi cleanup (konkuren)
```

---

## Pola Desain

| Pola | Dimana Digunakan | Deskripsi |
|---------|-----------|-------------|
| **Builder** | `Framework` - chain `With*()` fluent, `Build()` menghasilkan `FrameworkComponents` | Konstruksi langkah demi langkah dengan API fluent |
| **Singleton** | `Config` (via global `appConfig`), writer file `LoggerEntry` (`sync.Once`) | Instance tunggal di seluruh aplikasi |
| **Factory** | `NewOracleConnection`, `NewKafkaWriter`, `NewWorkerPool`, `NewPoller` | Pembuatan objek tanpa mengekspos logika instansiasi |
| **Strategy** | Routing mode output di Logger, routing model Kafka di config | Algoritma yang dapat dipertukarkan saat runtime |
| **Worker Pool** | `WorkerPool` - goroutine pool bounded dengan channel | Gunakan ulang goroutine untuk memproses banyak tugas |
| **Semaphore** | Channel `jobSlots` Poller untuk rate limiting poll konkuren | Kontrol akses ke resource |
| **Dependency Injection** | Semua method `With*FromInstance()` memungkinkan injeksi komponen yang telah dibuat sebelumnya | Injeksi dependency daripada membuatnya |

---

## Referensi Konfigurasi

### Environment Variables

| Variabel | Deskripsi | Contoh |
|----------|-----------|--------|
| `GO_ENV` | Environment aplikasi | `local`, `staging`, `production` |
| `KAFKA-MODELS` | Model Kafka yang digunakan | `kafka-localhost`, `kafka-staging`, `kafka-production`, `kafka-confluent` |
| `KAFKA_BROKERS_STAGING` | Broker Kafka staging | `broker1:9092,broker2:9092` |
| `KAFKA_BROKERS_PRODUCTION` | Broker Kafka production | `broker1:9092,broker2:9092` |
| `HOST_STAGING` | Host Oracle staging | `oracle-staging.example.com` |
| `PORT_STAGING` | Port Oracle staging | `1521` |
| `DATABASE_STAGING` | Database Oracle staging | `ORCL` |
| `USERNAME_STAGING` | Username Oracle staging | `user` |
| `PASSWORD_STAGING` | Password Oracle staging | `secret` |
| `HOST_PRODUCTION` | Host Oracle production | `oracle.example.com` |
| `PORT_PRODUCTION` | Port Oracle production | `1521` |
| `DATABASE_PRODUCTION` | Database Oracle production | `ORCL` |
| `USERNAME_PRODUCTION` | Username Oracle production | `user` |
| `PASSWORD_PRODUCTION` | Password Oracle production | `secret` |
| `API_KEY_KAFKA_CONFLUENT` | API Key Confluent Cloud | `xxx` |
| `API_SECRET_KAFKA_CONFLUENT` | API Secret Confluent Cloud | `xxx` |
| `BOOTSTRAP_SERVER_KAFKA_CONFLUENT` | Bootstrap server Confluent | `pkc-xxx.us-east-1.aws.confluent.cloud:9092` |
| `ELASTIC_HOST` | Host Elasticsearch | `localhost:9200` |
| `ELASTIC_USERNAME` | Username Elasticsearch | `elastic` |
| `ELASTIC_PASSWORD` | Password Elasticsearch | `secret` |
| `ELASTIC_INDEX_DATA` | Nama index Elasticsearch | `nanopony-logs` |
| `LOG_OUTPUT_MODE` | Mode output logger | `console`, `file`, `elasticsearch`, `hybrid` |

### Struktur Konfigurasi

```
Config
├── AppConfig
│   ├── Env            (GO_ENV: local/staging/production)
│   ├── KafkaModels    (KAFKA-MODELS: kafka-localhost/kafka-staging/kafka-production/kafka-confluent)
│   └── Operation      (OPERATION)
├── OracleConfig
│   ├── Username       (USERNAME_STAGING / USERNAME_PRODUCTION)
│   ├── Password       (PASSWORD_STAGING / PASSWORD_PRODUCTION)
│   ├── Host           (HOST_STAGING / HOST_PRODUCTION)
│   ├── Port           (PORT_STAGING / PORT_PRODUCTION)
│   └── DatabaseName   (DATABASE_STAGING / DATABASE_PRODUCTION)
├── KafkaConfig
│   └── Brokers        (KAFKA_BROKERS_STAGING / KAFKA_BROKERS_PRODUCTION)
├── KafkaConfluentConfig
│   ├── ApiKey         (API_KEY_KAFKA_CONFLUENT)
│   ├── ApiSecret      (API_SECRET_KAFKA_CONFLUENT)
│   ├── Resource       (RESOURCE_KAFKA_CONFLUENT)
│   └── BootstrapServers (BOOTSTRAP_SERVER_KAFKA_CONFLUENT)
└── ElasticSearchConfig
    ├── ElasticHost    (ELASTIC_HOST)
    ├── ElasticUsername (ELASTIC_USERNAME)
    ├── ElasticPassword (ELASTIC_PASSWORD)
    ├── ElasticIndexName (ELASTIC_INDEX_DATA)
    ├── ElasticApiKey  (ELASTIC_API_KEY)
    └── ElasticPrefixIndex (ELASTIC_PREFIX_INDEX)
```

---

## Referensi Interface

| Interface | Method | Tujuan |
|-----------|---------|---------|
| `DataFetcher` | `Fetch() ([]any, error)` | Sumber data abstrak untuk poller |
| `JobHandler` | `func(ctx, Job) error` | Callback pemrosesan job |
| `MessageProducer` | `Produce`, `ProduceWithContext`, `Close` | Abstraksi Kafka producer |

---

## Panduan Cepat

### 1. Inisialisasi Konfigurasi

```go
config := nanopony.NewConfig()
```

### 2. Build Framework

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

### 3. Mulai Framework

```go
ctx := context.Background()
components.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
    log.Printf("Memproses job: %+v", job)
    return nil
})
```

### 4. Graceful Shutdown

```go
components.Shutdown(ctx)
```

### Contoh Lengkap

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/sautmanurung2/nanopony"
)

func main() {
    // Inisialisasi konfigurasi
    config := nanopony.NewConfig()

    // Buat data fetcher
    dataFetcher := nanopony.DataFetcherFunc(func() ([]interface{}, error) {
        return []interface{}{"data1", "data2"}, nil
    })

    // Build framework
    components := nanopony.NewFramework().
        WithConfig(config).
        WithDatabase().
        WithKafkaWriter().
        WithProducer().
        WithWorkerPool(5, 100).
        WithPoller(nanopony.DefaultPollerConfig(), dataFetcher).
        Build()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Mulai pemrosesan
    components.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
        log.Printf("Memproses job: %+v", job)

        // Kirim ke Kafka
        components.Producer.Produce("my-topic", job.Data)

        return nil
    })

    // Tunggu interrupt
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Println("Menerima signal shutdown...")

    // Graceful shutdown
    components.Shutdown(ctx)
    log.Println("Shutdown selesai")
}

```

---

## Peta File

| File | Tujuan |
|------|---------|
| `config.go` | Definisi struct config, NewConfig, BuildConfig, ResetConfig |
| `config_init.go` | Parsing environment variable, fungsi init* |
| `database.go` | Koneksi Oracle, pooling, interpolasi query |
| `kafka.go` | Pembuatan Kafka writer, SASL/TLS untuk Confluent |
| `producer.go` | KafkaProducer, KafkaConsumer, interface MessageProducer |
| `worker.go` | WorkerPool, Poller, DataFetcher |
| `framework.go` | Framework builder, FrameworkComponents, Start/Shutdown |
| `logger.go` | Logging terstruktur, rotasi file, Elasticsearch |

---

## Best Practices

1. **Selalu gunakan Graceful Shutdown** - Memastikan semua resource dilepaskan dengan benar
2. **Gunakan Builder Pattern untuk setup yang clean** - Method chaining meningkatkan keterbacaan
3. **Gunakan pola arsitektur layer** - Meningkatkan pemisahan logic
4. **Gunakan Context untuk cancellation dan timeout** - Memungkinkan manajemen lifecycle yang benar
5. **Handle error dengan benar** - Monitor channel error dari worker pool
6. **Konfigurasi connection pool sesuai kebutuhan** - Tune berdasarkan workload
7. **Gunakan dependency injection untuk testing** - Injeksi mock via method `With*FromInstance()`

---

## Ringkasan

NanoPony menyediakan:

✅ **Kafka Producer & Consumer** - Integrasi Kafka dengan `kafka-go`  
✅ **Oracle Database** - Koneksi Oracle dengan connection pooling  
✅ **Worker Pool** - Pemrosesan job konkuren dengan queue bounded  
✅ **Poller** - Pengambilan data periodik dengan interval yang dapat dikonfigurasi  
✅ **Builder Pattern** - API yang clean dan fluent  
✅ **Graceful Shutdown** - Teardown yang aman untuk semua komponen  
✅ **Konfigurasi Berbasis Environment** - Konfigurasi via environment variables  
✅ **Logging Terstruktur** - Rotasi file dan integrasi Elasticsearch  

Framework ini dirancang untuk membangun sistem data pipeline yang robust, scalable, dan mudah di-maintain.
