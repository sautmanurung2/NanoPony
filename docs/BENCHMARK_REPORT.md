> **Bahasa:** [Bahasa Indonesia](BENCHMARK_REPORT.md) | [English](BENCHMARK_REPORT_EN.md)

# NanoPony: Benchmark & Performance Report

*Dokumen ini menyajikan analisis performa, efisiensi memori, dan perbandingan framework NanoPony terbaru (Juni 2026).*

---

## 🚀 Ringkasan Performa (Update Juni 2026)

Setelah serangkaian optimasi pada *hot path* sistem, NanoPony mencapai efisiensi yang sangat tinggi untuk pemrosesan *job* internal:

| Metrik | Nilai | Catatan |
| :--- | :--- | :--- |
| **Throughput (Worker Pool)** | **~438.5 ns/op** | Performa tinggi berkat arsitektur sharded worker pool |
| **Alokasi Memori** | **2 allocs/op** | Sangat efisien dalam penggunaan heap (16 B/op) |
| **Status Memory Leak** | **✅ LOLOS** | Pertumbuhan memori negatif (-58 KB) setelah 50 siklus |

---

## 📊 Hasil Benchmark (vs Framework Lain)

NanoPony dirancang untuk memproses *job* (background task) secara efisien tanpa overhead stack HTTP. Berikut perbandingannya dengan framework web populer:

| Framework | Throughput (Job Processing) | Status |
| :--- | :--- | :--- |
| **NanoPony** | **~1.48 µs/op** | 🚀 **Juara Umum** |
| **Iris** | ~3.10 µs/op | 🥈 Cepat |
| **Echo** | ~3.17 µs/op | 🥉 Cepat |
| **Fiber** | ~19.87 µs/op | 🐢 Lambat (di skenario ini) |

> **Analisis:** NanoPony unggul karena fokus pada *internal job processing*. Arsitektur *sharded worker pool* kami secara drastis mengurangi *lock contention* dibandingkan framework yang didesain untuk request-response HTTP.

---

## ⚙️ Detail Optimasi (Pembaruan Juni 2026)

Untuk mencapai throughput saat ini, kami menerapkan optimasi berikut:

### 1. Sharded Worker Pool
Kami membagi `WorkerPool` tunggal menjadi beberapa independen *shard*.
- **Manfaat**: Secara drastis mengurangi *lock contention* pada mutex pool, terutama di bawah beban tinggi dengan banyak goroutine.

### 2. Efisiensi Hot Path (Poller ID)
Penggantian `fmt.Sprintf` dengan `strings.Builder` dan `strconv.AppendInt`.
- **Manfaat**: Menghilangkan alokasi *heap* yang tidak perlu pada setiap pembuatan ID job di Poller.

### 3. Job Lifecycle Management (sync.Pool)
Penggunaan `sync.Pool` untuk mendaur ulang objek `Job`.
- **Manfaat**: Mengurangi beban *Garbage Collector* secara signifikan dengan penggunaan kembali alokasi memori yang sudah ada.

### 4. Zero-Allocation Field Access
Akses field pada `FrameworkComponents` dioptimalkan untuk nol alokasi (0 allocs/op).

---

## 🔍 Hasil Uji Stabilitas Memori

| Komponen | Penjelasan | Status |
| :--- | :--- | :--- |
| **Core Lifecycle** | 50 siklus setup-shutdown | ✅ Stabil (-58 KB growth) |
| **WorkerPool** | Stress test 1.000 jobs | ✅ Stabil (+66 KB growth) |
| **Concurrent** | 20 instance simultan | ✅ Stabil (+42 KB growth) |
| **Poller** | Long running (2 detik) | ✅ Stabil (100% diproses) |

---

## 🎯 Kesimpulan Final

Framework NanoPony adalah solusi *high-performance* untuk integrasi Kafka-Oracle. Dengan arsitektur modular dan optimasi tingkat lanjut pada manajemen *lifecycle* job, NanoPony memberikan efisiensi yang sangat baik bagi sistem *background processing* yang membutuhkan *throughput* tinggi dengan *memory footprint* minimal.

*Laporan dihasilkan pada 3 Juni 2026. Data valid untuk v0.0.59.*
