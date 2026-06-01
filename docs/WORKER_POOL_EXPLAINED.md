# Arsitektur Sharded Worker Pool

## Ringkasan

Dokumen ini menjelaskan arsitektur *sharded worker pool* di framework NanoPony. NanoPony menggunakan sistem *sharding* untuk mengurangi *lock contention* dan mendaur ulang objek `Job` menggunakan `sync.Pool` untuk efisiensi memori.

---

## Arsitektur: Sharded Pool

NanoPony kini membagi beban kerja ke dalam beberapa *shard* (pool) yang independen.

```
                               ┌──→ Shard 0 (Pool 0) ──→ Worker...
Submit ──→ Shard Selector ──┼──→ Shard 1 (Pool 1) ──→ Worker...
                               └──→ Shard N (Pool N) ──→ Worker...
```

Setiap *shard* beroperasi secara independen dengan *channel* (`jobChan`) dan *goroutine worker* sendiri. Saat `Submit()` dipanggil, sistem memilih *shard* tujuan menggunakan mekanisme *round-robin* berbasis *atomic counter*.

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

*   **Reduced Contention**: Dengan *sharding*, proses submit *job* tidak lagi memperebutkan satu mutex *pool* yang sama, sehingga meningkatkan *throughput* secara signifikan di bawah beban tinggi.

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
| **Konkurensi** | *Fan-out* ke beberapa *shard* paralel. |
| **Blocking** | *Worker* sinkron per *job* (menunggu *handler* selesai). |
| **Locking** | *Contention* sangat rendah berkat *sharding*. |
| **Submit** | `Submit()` non-blocking, `SubmitBlocking()` blocking. |

---

## 4. Kesimpulan

Sistem ini menggabungkan keandalan mekanisme *worker pool* dengan skalabilitas yang tinggi melalui *sharding*. Desain ini mempertahankan prinsip *backpressure* untuk menjaga stabilitas aplikasi di bawah beban kerja tinggi.
