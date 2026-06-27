# Arsitektur Sharded Worker Pool

## Ringkasan

Dokumen ini menjelaskan arsitektur *sharded worker pool* di framework NanoPony. NanoPony menggunakan sistem *sharding* untuk mengurangi *lock contention* dan mendaur ulang objek `Job` menggunakan `sync.Pool` untuk efisiensi memori.

---

## Arsitektur: Sharded Pool

NanoPony kini menggunakan arsitektur antrean bersama (Shared Queue) dengan Elastic Buffer untuk mencegah pemblokiran Head-of-Line sekaligus mempertahankan nama `ShardedWorkerPool` untuk kompatibilitas.

```
                               ┌──→ Shard 0 (Pool 0) ──→ Worker...
Submit ──→ Shared Elastic Queue ──→ Shared Channel ──→ Workers...
```

Saat `Submit()` dipanggil, pekerjaan dimasukkan ke dalam saluran bersama. Jika antrean utama penuh, pekerjaan dipindahkan ke memori penyangga (Elastic Buffer) dengan mekanisme eksponensial backoff, memastikan keandalan 100% tanpa kehilangan data.

---

## 1. Implementasi Sharding

```go
// Worker Pool yang sharded
type ShardedWorkerPool struct {
    shards          []*workerPoolShard
    // ...
}

// WorkerPoolShard
type workerPoolShard struct {
    jobChan chan *Job
    errChan chan error
    // ...
}
```

*   **Elastic Queue**: Dengan mekanisme Elastic Buffer, lonjakan pekerjaan (spike) ditangani tanpa penghentian instan (deadlock), memberikan ketahanan sistem saat antrean utama penuh.

---

## 2. Bagaimana Worker Memproses Job

Worker tetap bersifat sinkron per *job*.

```go
func (swp *ShardedWorkerPool) worker(shard *workerPoolShard, handler JobHandler, id int) {
    defer shard.wg.Done()
    for {
        select {
        case <-shard.ctx.Done():
            return
        case job, ok := <-shard.jobChan:
            if !ok {
                return
            }

            // 🔴 BLOCKING DI SINI
            // Worker TIDAK BISA mengambil job berikutnya dari shard 
            // sampai handler(job) selesai.
            if err := handler(shard.ctx, job); err != nil {
                // handle error
            }

            // ♻️ OTOMATIS DI-RELEASE
            // Objek job dikembalikan ke pool untuk didaur ulang.
            job.Release()
        }
    }
}
```

**Implikasi:**
- Sebuah *worker* dalam *shard* hanya memproses satu *job* pada satu waktu.
- *Worker* lain di *shard* yang sama, dan *worker* di *shard* berbeda, tetap beroperasi secara paralel.

---

## 3. Karakteristik Utama

| Aspek | Behavior |
|-------|----------|
| **Konkurensi** | *Single shared channel* menghindari masalah *Head-of-Line blocking*. |
| **Blocking** | *Worker* sinkron per *job* (menunggu *handler* selesai). |
| **Locking** | Optimal berkat penggunaan antrean bersama dengan buffer. |
| **Submit** | Menggunakan mekanisme backoff dan *Elastic Buffer* menjamin *queue* tidak membuang pekerjaan. |

---

## 4. Kesimpulan

Sistem ini menggabungkan keandalan mekanisme *worker pool* dengan skalabilitas tinggi melalui saluran tunggal bersama dan *Elastic Buffer*. Desain ini mempertahankan prinsip *backpressure* untuk menjaga stabilitas aplikasi di bawah beban kerja tinggi.
