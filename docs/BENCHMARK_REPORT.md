> **Bahasa:** [Bahasa Indonesia](BENCHMARK_REPORT.md) | [English](BENCHMARK_REPORT_EN.md)

# NanoPony: Benchmark & Performance Report

*Dokumen ini menyajikan analisis performa, efisiensi memori, dan perbandingan framework NanoPony terbaru (Mei 2026).*

---

## 🚀 Ringkasan Performa (Update Mei 2026)

Setelah serangkaian optimasi pada *hot path* sistem, NanoPony kini mencapai efisiensi yang lebih tinggi:

| Metrik | Nilai | Catatan |
| :--- | :--- | :--- |
| **Throughput** | **~390.2 ns/op** | Peningkatan efisiensi dari optimasi sharding |
| **Alokasi Memori** | **3 allocs/op** | Sangat stabil untuk pemrosesan volume tinggi |
| **Status Memory Leak** | **✅ LOLOS** | Tidak ada kebocoran setelah 40+ siklus |

---

## 📊 Hasil Benchmark (vs Framework Lain)

NanoPony dirancang untuk memproses *job* (background task) secara efisien. Berikut perbandingannya dengan framework populer lainnya:

| Framework | Throughput (Job Processing) | Status |
| :--- | :--- | :--- |
| **NanoPony** | **~390 ns/op** | 🚀 **Juara Umum** |
| **Echo** | ~2.502 ns/op | 🥈 Cepat |
| **Iris** | ~2.827 ns/op | 🥉 Cepat |
| **Fiber** | ~10.769 ns/op | 🐢 Lambat (di skenario ini) |

> **Analisis:** NanoPony unggul karena tidak memiliki overhead HTTP stack yang berat. Arsitektur *sharded worker pool* kami secara drastis mengurangi *lock contention* yang biasanya menghambat skalabilitas.

---

## ⚙️ Detail Optimasi (Pembaruan Mei 2026)

Untuk mencapai throughput saat ini, kami menerapkan optimasi berikut:

### 1. Sharded Worker Pool
Kami membagi `WorkerPool` tunggal menjadi beberapa independen *shard*.
- **Manfaat**: Secara drastis mengurangi *lock contention* pada mutex pool, terutama di bawah beban tinggi dengan banyak goroutine.

### 2. Efisiensi Hot Path (Poller ID)
Penggantian `fmt.Sprintf` dengan `strings.Builder` dan `strconv.FormatInt`.
- **Manfaat**: Menghilangkan alokasi *heap* yang tidak perlu pada setiap pembuatan ID job.

### 3. Logging System (Deep-Copy Manual)
Penggantian siklus JSON `Marshal/Unmarshal` dengan rekursi manual untuk *deep-copy* payload `map[string]any`.
- **Manfaat**: Pengurangan drastis penggunaan CPU dan alokasi memori pada sistem logging terstruktur.

### 4. Job Lifecycle Management
Perubahan strategi pembersihan map pada `Job[T].Release` dengan pembuatan ulang map (`make`).
- **Manfaat**: Lebih efisien untuk *Garbage Collector* dibandingkan menghapus kunci satu per satu (*key-by-key deletion*).

---

## 🔍 Hasil Uji Stabilitas Memori

| Komponen | Penjelasan | Status |
| :--- | :--- | :--- |
| **Core Lifecycle** | 40 siklus setup-shutdown | ✅ Stabil (+19 KB) |
| **WorkerPool** | Stress test 1.000+ jobs | ✅ Stabil (+6 KB) |
| **Poller** | 10 siklus polling | ✅ Stabil (+2 KB) |
| **Concurrent** | 20 instance simultan | ✅ Stabil (+50 KB) |

---

## 🎯 Kesimpulan Final

Framework NanoPony adalah solusi *high-performance* untuk integrasi Kafka-Oracle. Dengan arsitektur modular dan optimasi tingkat lanjut, NanoPony memberikan efisiensi yang sangat baik bagi sistem *background processing* yang membutuhkan *throughput* tinggi dengan *memory footprint* minimal.

*Laporan dihasilkan pada 29 Mei 2026. Data valid untuk v0.0.59.*
