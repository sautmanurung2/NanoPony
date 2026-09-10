> **Bahasa:** [Bahasa Indonesia](BENCHMARK_REPORT.md) | [English](BENCHMARK_REPORT_EN.md)

# NanoPony: Benchmark & Performance Report

*Dokumen ini menyajikan analisis performa, efisiensi memori, dan perbandingan framework NanoPony terbaru (Juni 2026).*

---

## 🚀 Ringkasan Performa (Pembaruan September 2026)

Setelah serangkaian optimasi pada *hot path* sistem dan implementasi subsistem Kafka native murni Golang standard library, NanoPony mencapai efisiensi yang sangat tinggi untuk pemrosesan *job* internal dan *event streaming*:

| Metrik | Nilai | Catatan |
| :--- | :--- | :--- |
| **Throughput (Worker Pool)** | **~438.5 ns/op** | Performa tinggi berkat arsitektur sharded worker pool |
| **Alokasi Memori** | **2 allocs/op** | Sangat efisien dalam penggunaan heap (16 B/op) |
| **Status Memory Leak** | **✅ LOLOS** | Pertumbuhan memori negatif (-58 KB) setelah 50 siklus |
| **Kafka Wire Protocol** | **Zero-Third-Party** | 100% Go stdlib (`net`, `crypto/tls`, `encoding/binary`, `hash/crc32`) |
| **Test Coverage & Safety** | **85.7% (0 Race)** | Terverifikasi via `-race`, in-memory TCP mock server tanpa Kafka cluster eksternal |

---

## 📊 Hasil Benchmark Multi-Framework (Pembaruan Terkini)

Pengujian benchmark dijalankan secara *head-to-head* pada lingkungan Linux (13th Gen Intel Core i7-13700H) menggunakan `go test -tags benchmark -bench=. -benchmem` di folder `benchmark/`.

### 1. Throughput Pemrosesan Job
NanoPony dirancang khusus untuk memproses *event* dan *background job* tanpa overhead stack HTTP:

| Framework | Kecepatan (Throughput) | Memori (B/op) | Alokasi (Allocs/op) | Peringkat & Status |
| :--- | :--- | :--- | :--- | :--- |
| **NanoPony** | **~3.94 µs/op** (3,943 ns/op) | **98 B/op** | **2 allocs/op** | 🚀 **Juara 1 (Paling Cepat & Paling Hemat)** |
| **Echo** | ~6.00 µs/op (5,999 ns/op) | 5,312 B/op | 13 allocs/op | 🥈 Cepat |
| **Iris** | ~6.86 µs/op (6,860 ns/op) | 5,312 B/op | 13 allocs/op | 🥉 Cepat |
| **Fiber** | ~26.11 µs/op (26,112 ns/op) | 5,533 B/op | 21 allocs/op | 🐢 Lambat (~6.6x lebih lambat) |

### 2. Setup Overhead & Idle Memory

| Framework | Setup Overhead | Setup Memori | Idle Memory Footprint |
| :--- | :--- | :--- | :--- |
| **NanoPony** | 689.4 µs/op | 4,037 B/op (17 allocs) | **14 KB** (Worker Pool aktif) |
| **Fiber** | 4.7 µs/op | 2,912 B/op (12 allocs) | **2 KB** |
| **Echo** | 1,538.2 µs/op | 3,632 B/op (53 allocs) | **0 KB** |
| **Iris** | 216.3 µs/op | 21,560 B/op (221 allocs) | **14 KB** |

> **Analisis:**
> - **Throughput Unggul**: NanoPony memproses job **6.6x lebih cepat dibandingkan Fiber**, **1.5x lebih cepat dari Echo**, dan **1.7x lebih cepat dari Iris**.
> - **Efisiensi Alokasi Memori (54x Lebih Hemat)**: Berkat daur ulang `sync.Pool` dan eliminasi library *third-party*, NanoPony hanya membutuhkan **98 B/op** dan **2 alokasi** per job. Bandingkan dengan framework web konvensional yang membuang **5.300 - 5.500 B/op** dan 13 - 21 alokasi per siklus.
> - **Setup Bertanggung Jawab**: Waktu inisialisasi NanoPony dialokasikan di awal untuk mem-booting goroutine worker pool, bounded channels, dan verifikasi native wire protocol, sehingga pada fase eksekusi tidak terjadi *lock contention* maupun relokasi buffer.

---

## ⚙️ Detail Optimasi (Pembaruan Juni - September 2026)

Untuk mencapai throughput saat ini dan menghilangkan overhead library eksternal, kami menerapkan optimasi berikut:

### 1. Sharded Worker Pool
Kami membagi `WorkerPool` tunggal menjadi beberapa independen *shard*.
- **Manfaat**: Secara drastis mengurangi *lock contention* pada mutex pool, terutama di bawah beban tinggi dengan banyak goroutine.

### 2. Efisiensi Hot Path (Poller ID)
Penggantian `fmt.Sprintf` dengan `strings.Builder` dan `strconv.AppendInt`.
- **Manfaat**: Menghilangkan alokasi *heap* yang tidak perlu pada setiap pembuatan ID job di Poller.

### 3. Job Lifecycle Management (sync.Pool & Atomic CAS)
Penggunaan `sync.Pool` untuk mendaur ulang objek `Job` dilengkapi pagar `inUse atomic.Bool` via Compare-And-Swap (CAS).
- **Manfaat**: Mengurangi beban *Garbage Collector* secara drastis serta mencegah double-release / data race pada skenario konkurensi tinggi.

### 4. Zero-Allocation Field Access
Akses field pada `FrameworkComponents` dioptimalkan untuk nol alokasi (0 allocs/op).

### 5. Native Pure Go Kafka Wire Protocol (September 2026)
Penggantian library pihak ketiga (`github.com/segmentio/kafka-go`, `klauspost/compress`, `pierrec/lz4`) dengan binary wire protocol murni Go standard library (`net`, `crypto/tls`, `encoding/binary`, `hash/crc32`).
- **Manfaat**: Menghapus beban dependensi eksternal, memangkas binary footprint, menghindari alokasi refleksi berlebih, dan mendukung partisi terdistribusi (RoundRobin, LeastBytes, Hash/Murmur2) dengan performa soket TCP/TLS murni.

---

## 🔍 Hasil Uji Stabilitas Memori & Keamanan Konkurensi

| Komponen | Penjelasan | Status |
| :--- | :--- | :--- |
| **Core Lifecycle** | 50 siklus setup-shutdown | ✅ Stabil (-58 KB growth) |
| **WorkerPool** | Stress test 1.000 jobs | ✅ Stabil (+66 KB growth) |
| **Concurrent** | 20 instance simultan | ✅ Stabil (+42 KB growth) |
| **Poller** | Long running (2 detik) | ✅ Stabil (100% diproses) |
| **Native Kafka Protocol** | Mock in-memory TCP socket test (Produce/Fetch/ListOffsets/Metadata/SASL) | ✅ Stabil (0 leak, 0 race, 85.7% coverage) |

---

## 🎯 Kesimpulan Final

Framework NanoPony adalah solusi *high-performance* untuk integrasi Kafka-Oracle. Dengan implementasi Kafka native tanpa library third-party dan arsitektur modular dengan optimasi tingkat lanjut pada manajemen *lifecycle* job, NanoPony memberikan efisiensi yang sangat baik bagi sistem *background processing* yang membutuhkan *throughput* tinggi dengan *memory footprint* minimal dan dependensi yang bersih.

*Laporan diperbarui pada 10 September 2026. Data valid untuk v0.0.60 (Pure Native Kafka).*
