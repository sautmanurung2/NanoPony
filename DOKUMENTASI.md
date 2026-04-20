# Dokumentasi Framework NanoPony (v0.0.30 - Sync April 2026)

**NanoPony** adalah framework Go untuk integrasi Kafka-Oracle dengan arsitektur yang clean, reusable, dan production-ready. Framework ini menyediakan komponen-komponen siap pakai untuk membangun sistem data pipeline yang scalable dengan minimal boilerplate.

---

## Daftar Isi

1. [Arsitektur Keseluruhan](#arsitektur-keseluruhan)
2. [Layer Konfigurasi](#1-layer-konfigurasi)
3. [Layer Database (Oracle)](#2-layer-database-oracle)
4. [Layer Kafka](#3-layer-kafka)
5. [Worker Pool](#4-worker-pool)
6. [Poller](#5-poller)
8. [Framework Builder & Lifecycle](#7-framework-builder--lifecycle)
9. [Best Practices](#best-practices)

---

## Arsitektur Keseluruhan

NanoPony menggunakan pola **Builder** untuk menyusun komponen-komponen aplikasi secara deklaratif. Hasil akhir dari proses pembangunan adalah `FrameworkComponents` yang mengelola siklus hidup (*lifecycle*) seluruh layanan.

```
┌───────────────────────────────────────────────┐
│           Framework Builder (Fluent API)      │
│  (WithConfig -> WithDatabase -> WithPoller)   │
└───────────────────────────────────────────────┘
                     ↓
┌───────────────────────────────────────────────┐
│            FrameworkComponents                │
│ (DB, Producer, WorkerPool, Poller, Cleanup)   │
└───────────────────────────────────────────────┘
          /          |          \
┌──────────┐   ┌────────────┐   ┌────────────┐
│  Poller  │──>│ Worker Pool│──>│ Job Handler│
└──────────┘   └────────────┘   └────────────┘
     ^               |                |
     |        (Submit Job)       (Business Logic)
  (Fetch)            |                |
     |               v                v
┌──────────┐   ┌────────────┐   ┌────────────┐
│  Oracle  │   │ Kafka/ES   │   │ Business   │
└──────────┘   └────────────┘   └────────────┘
```

---

## 1. Layer Konfigurasi

**File**: `config.go`, `config_init.go`

### Deskripsi
Manajemen konfigurasi terpusat berbasis environment variables dengan dukungan sinkronisasi singleton dan validasi otomatis.

### Environment Variables Utama

| Variable | Deskripsi | Nilai yang Diperbolehkan |
|----------|-----------|-------------------------|
| `GO_ENV` | Mode Environment | `local`, `staging`, `production` |
| `KAFKA_MODELS` | Konfigurasi Kafka | `kafka-localhost`, `kafka-staging`, `kafka-production`, `kafka-confluent` |
| `OPERATION` | Mode Operasi | Bebas (kustom per aplikasi) |
| `LOG_FILE_PREFIX` | Prefix Nama File Log | Bebas (Default: `orion-to-core`) |

### Fitur Lanjutan

#### A. Dynamic Config
Memungkinkan pemuatan variabel environment kustom tanpa harus mengubah struct `Config`.
```go
config := nanopony.NewConfig()
config.LoadDynamic("CUSTOM_") // Memuat semua env var berawalan CUSTOM_
apiUrl := config.Dynamic["CUSTOM_API_URL"]
```

#### B. ElasticSearch Config
Mendukung konfigurasi Elasticsearch out-of-the-box.
- `ELASTIC_HOST`, `ELASTIC_USERNAME`, `ELASTIC_PASSWORD`
- `ELASTIC_INDEX_DATA`, `ELASTIC_API_KEY`, `ELASTIC_PREFIX_INDEX`

#### C. Validation & Reset
- **`Validate()`**: Memastikan variabel wajib (`GO_ENV`, `KAFKA_MODELS`) telah diatur sebelum framework dijalankan.
- **`ResetConfig()`**: Digunakan dalam unit testing untuk mereset *global state* konfigurasi.

---

## 2. Layer Database (Oracle)

**File**: `database.go`

### Connection Pooling
Secara default, NanoPony mengatur batas koneksi yang optimal untuk menjaga performa:
- `MaxIdleConns`: 2
- `MaxOpenConns`: 20
- `ConnIdleTime`: 5 Menit
- `ConnMaxLifetime`: 60 Menit

### Inisialisasi
```go
// Cara Aman (Direkomendasikan)
fw, err := nanopony.NewFramework().
    WithConfig(config).
    WithDatabaseSafe() // Mengembalikan error jika koneksi gagal

// Menggunakan Instance yang Sudah Ada
fw.WithDatabaseFromInstance(myExistingDB)
```

### Utilitas Debugging (SQL Interpolation)
NanoPony menyediakan alat untuk mempermudah logging query yang memiliki parameter.
> [!WARNING]
> Gunakan hanya untuk logging/debugging. Jangan jalankan hasil interpolasi langsung ke DB untuk menghindari SQL Injection.

```go
nanopony.LogInterpolatedQuery(
    "SELECT * FROM users WHERE id = :id",
    sql.Named("id", 123),
)
// Output: [SQL Query] SELECT * FROM users WHERE id = 123
```

---

## 3. Layer Kafka

**File**: `kafka.go`

### Optimasi Otomatis
Saat Anda memanggil `.Build()` pada framework, NanoPony secara otomatis menyesuaikan `BatchSize` pada Kafka Writer agar sesuai dengan jumlah worker di Worker Pool. Hal ini memastikan throughput yang seimbang antara pemrosesan data dan pengiriman pesan.

### Mode Confluent Cloud
Jika `KAFKA_MODELS=kafka-confluent`, NanoPony otomatis mengaktifkan autentikasi SASL/PLAIN dan TLS menggunakan:
- `API_KEY_KAFKA_CONFLUENT`
- `API_SECRET_KAFKA_CONFLUENT`
- `BOOTSTRAP_SERVER_KAFKA_CONFLUENT`

---

## 4. Worker Pool

**File**: `job.go`, `worker.go`

### Konsep Utama
Worker Pool mengelola kumpulan goroutine yang memproses `Job` secara konkuren. `job.go` menangani definisi data, sementara `worker.go` menangani orkestrasi pemrosesan.

### Job Metadata
Field `Meta` pada struct `Job` memungkinkan Anda menyisipkan data tambahan seperti *source*, *retry count*, atau *trace ID*.
```go
job := nanopony.Job{
    ID:   "job-123",
    Data: payload,
    Meta: map[string]any{"source": "poller-01"},
}
```

### Mekanisme Backpressure (SubmitBlocking)
Sangat direkomendasikan untuk produksi. Jika antrean penuh, pemanggil akan "menunggu" sampai ada ruang tersedia, sehingga mencegah hilangnya data (*data loss*).
```go
err := pool.SubmitBlocking(ctx, job)
```

---

## 5. Poller

**File**: `poller.go`

### Mekanisme Slot (Semaphore)
Poller menggunakan `JobSlotSize` untuk membatasi berapa banyak operasi polling yang boleh berjalan secara bersamaan. Jika slot penuh (misalnya karena polling sebelumnya masih berlangsung), polling berikutnya akan dilewati (*skipped*).

### Keunikan ID Pekerjaan
Setiap pekerjaan yang dihasilkan oleh Poller memiliki ID format: `poll-[SessionID]-[Counter]`.
- **SessionID**: Berbasis timestamp saat aplikasi dijalankan.
- **Counter**: Atomic counter yang terus meningkat.
Ini menjamin tidak ada duplikasi ID antar proses.

---

## 6. Framework Builder & Lifecycle

**File**: `framework.go`

### Tahapan Pembangunan (Fluent API)
```go
components := nanopony.NewFramework().
    WithConfig(config).
    WithDatabase().
    WithKafkaWriter().
    WithProducer().
    WithWorkerPool(5, 100).
    WithPoller(pollerConfig, fetcher).
    AddCleanup(myCleanupFunc).
    Build()
```

### Tahapan Eksekusi
1. **`Start(ctx, handler)`**:
   - Menjalankan Worker Pool.
   - Menjalankan Poller.
2. **`Shutdown(ctx)`**:
   - Menghentikan Poller terlebih dahulu.
   - Menghentikan Worker Pool (menunggu tugas selesai).
   - Menutup koneksi DB dan Kafka.
   - Menjalankan fungsi `cleanup` kustom secara konkuren.

---

## Best Practices

### 1. Gunakan SubmitBlocking
Hindari `Submit` biasa jika Anda tidak ingin mengabaikan data saat beban sistem sedang tinggi. `SubmitBlocking` memberikan mekanisme *backpressure* alami.

### 2. Implementasikan Graceful Shutdown
Selalu pastikan `components.Shutdown(ctx)` dipanggil, idealnya menggunakan sinyal OS (`SIGINT`, `SIGTERM`).
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
<-sigChan
components.Shutdown(ctx)
```

### 3. Pantau Error Channel
Jangan biarkan error di Worker Pool menghilang begitu saja. Selalu dengarkan channel `Errors()`.
```go
go func() {
    for err := range components.WorkerPool.Errors() {
        log.Printf("Worker Alert: %v", err)
    }
}()
```

### 4. Manfaatkan Dynamic Config
Gunakan `DynamicConfig` untuk pengaturan yang bersifat opsional atau modul kustom sehingga framework tetap bersih dari dependensi yang tidak diperlukan oleh semua tim.

---

## Ringkasan Fitur Utama

✅ **Auto-Scale Kafka Batching** - Berdasarkan kapasitas worker pool.  
✅ **Oracle Pool Optimization** - Pengaturan default yang aman untuk produksi.  
✅ **Dynamic Configuration** - Fleksibel untuk variabel environment kustom.  
✅ **Safe-Fail Validation** - Validasi konfigurasi di awal aplikasi berjalan.  
✅ **Aggregated Shutdown Errors** - Melaporkan semua masalah saat proses shutdown berakhir.  
✅ **SQL Debugger** - Utilitas interpolasi query untuk logging yang mudah dibaca.  
✅ **Multi-Framework Speed** - ~39x lebih cepat dari Fiber untuk internal jobs.
