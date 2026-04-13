# Arsitektur Worker Pool

## Ringkasan

Dokumen ini menjelaskan arsitektur worker pool di framework NanoPony, mencakup cara worker memproses job, manajemen queue, dan karakteristik performa.

---

## Arsitektur: 5 Workers + Queue Size 100

```
┌──────────────────────────────────────────────────────────┐
│                    Queue (100 slot)                       │
│  [J1][J2][J3]...[J100]                                   │
└────────┬────────┬────────┬────────┬──────────────────────┘
         │        │        │        │        │
         ▼        ▼        ▼        ▼        ▼
     ┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐
     │Worker 1││Worker 2││Worker 3││Worker 4││Worker 5│
     └────────┘└────────┘└────────┘└────────┘└────────┘
```

---

## Pertanyaan Utama

**"Kalau Worker 1 produce lama, apakah harus selesai dulu sebelum bisa ambil job berikutnya?"**

**YA, HARUS SELESAI DULU.** Berikut cara kerjanya:

---

## 1. Worker Bersifat Blocking/Sinkron

```go
func (wp *WorkerPool) worker(ctx context.Context, id int) {
    for {
        select {
        case job, ok := <-wp.jobChan:
            if !ok {
                return
            }

            // 🔴 BLOCKING DI SINI
            // Worker TIDAK BISA mengambil job berikutnya
            // sampai handler(job) selesai
            if err := wp.handler(ctx, job); err != nil {
                // handle error
            }
        }
    }
}
```

**Implikasi:**
- **Worker 1** ambil `Job A` → proses (misal 10 detik) → selesai → baru bisa ambil `Job B`
- Sementara itu, **Worker 2-5** tetap memproses job mereka masing-masing secara paralel

---

## 2. Flow: 100 Proses dengan 5 Workers

```
Waktu T=0:
Queue: [J1][J2][J3]...[J100]

Worker 1: ambil J1 (proses 10 detik)
Worker 2: ambil J2 (proses 2 detik)  ← selesai duluan
Worker 3: ambil J3 (proses 5 detik)
Worker 4: ambil J4 (proses 3 detik)
Worker 5: ambil J5 (proses 8 detik)

Waktu T=2 detik:
Worker 2 selesai J2 → langsung ambil J6 dari queue
Queue sekarang: [J7][J8]...[J100]

Waktu T=3 detik:
Worker 4 selesai J4 → langsung ambil J7 dari queue
Queue sekarang: [J8][J9]...[J100]

... dan seterusnya
```

---

## 3. Karakteristik Utama

| Aspek | Behavior |
|-------|----------|
| **Konkurensi** | 5 job diproses **secara paralel** (1 per worker) |
| **Blocking** | Worker **tidak bisa** ambil job baru sebelum job sekarang selesai |
| **Queue** | Job menunggu di queue (FIFO), bukan di worker |
| **Non-blocking Submit** | `Submit()` langsung return, tidak menunggu job selesai |

---

## 4. Ilustrasi dengan Produce yang Lama

Asumsikan **Worker 1** butuh **30 detik** untuk produce ke Kafka:

```
T=0:   Worker 1 ambil J1 (produce ke Kafka - 30 detik)
T=5:   Worker 2,3,4,5 selesai job mereka dan ambil J6,J7,J8,J9
T=10:  Worker 2,3,4,5 ambil J10,J11,J12,J13
T=20:  Worker 2,3,4,5 ambil J14,J15,J16,J17
T=25:  Queue hampir kosong, Worker 2-5 idle
T=30:  Worker 1 selesai J1 → baru bisa ambil J18
```

**Dampak:** Worker 1 jadi bottleneck; 4 worker lain bisa nganggur kalau queue kosong.

---

## 5. Solusi Optimasi

### A. Tambah Jumlah Worker

```go
pool := NewWorkerPool(20, 100) // 20 workers, bukan 5
```

**Kelebihan:** Sederhana, lebih banyak paralelisme  
**Kekurangan:** Penggunaan memory/CPU lebih tinggi

---

### B. Async Handler (Produce di Background Goroutine)

```go
handler := func(ctx context.Context, job Job) error {
    // Jangan blocking di sini
    go func() {
        produceToKafka(job) // jalankan di background
    }()
    return nil // langsung return
}
```

**Kelebihan:** Worker tetap tersedia untuk job baru  
**Kekurangan:** ⚠️ Kehilangan kontrol error dan jaminan graceful shutdown

---

### C. Bounded Semaphore untuk Operasi Produce

```go
// Batasi operasi produce yang berjalan bersamaan
sem := make(chan struct{}, 10) // maks 10 produce bersamaan

handler := func(ctx context.Context, job Job) error {
    sem <- struct{}{} // acquire
    go func() {
        defer func() { <-sem }() // release
        produceToKafka(job)
    }()
    return nil
}
```

**Kelebihan:** Kontrol konkurensi, tidak membebani sistem downstream  
**Kekurangan:** Lebih kompleks, butuh penanganan error yang hati-hati

---

## 6. Ringkasan

| Pertanyaan | Jawaban |
|------------|---------|
| Worker harus selesai job sekarang sebelum ambil job baru? | **YA** |
| Apakah worker bekerja independen? | **YA** - setiap worker beroperasi independen |
| Worker lambat bisa blocking worker lain? | **TIDAK** - tapi bisa menyebabkan penumpukan di queue |
| Apakah pemrosesan job sinkron? | **YA** - blocking di dalam setiap worker |
| Apakah submit job sinkron? | **TIDAK** - `Submit()` non-blocking |

---

## Kesimpulan

Desain worker pool ini **aman dan predictable**, tapi bisa jadi bottleneck kalau ada job yang memakan waktu jauh lebih lama daripada job lainnya. Worker **harus menyelesaikan job saat ini** sebelum bisa mengambil job berikutnya. Ini adalah desain untuk keandalan (reliability), tapi bisa dioptimasi menggunakan strategi yang dijelaskan di atas jika diperlukan.
