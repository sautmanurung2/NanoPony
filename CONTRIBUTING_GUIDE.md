# 🚀 Panduan Kontribusi & Pengembangan NanoPony

Selamat datang di NanoPony! Framework ini dirancang untuk menjadi "jembatan" performa tinggi antara Oracle Database dan Kafka. Dokumen ini akan membantu kamu memahami struktur kode, pola desain, dan standar teknis yang kami gunakan.

---

## 📑 Daftar Isi
1. [Filosofi Desain](#-filosofi-desain)
2. [Arsitektur Teknis (Deep Dive)](#-arsitektur-teknis-deep-dive)
3. [Standar Coding & Konvensi](#-standar-coding--konvensi)
4. [Langkah-Langkah Pengembangan](#-langkah-langkah-pengembangan)
5. [Testing & Benchmarking](#-testing--benchmarking)
6. [Alur Pull Request](#-alur-pull-request)

---

## 🧠 Filosofi Desain

NanoPony dibangun di atas tiga pilar utama:
1. **Fluent Builder Pattern**: Meminimalkan boilerplate. Setup aplikasi harus terlihat seperti membaca kalimat.
2. **Explicit Lifecycle**: Start dan Shutdown harus terkoordinasi. Tidak boleh ada "goroutine yatim piatu" (*leaked goroutines*) saat aplikasi mati.
3. **Backpressure Aware**: Sistem harus tahu kapan harus berhenti menarik data jika pemrosesan di belakang sedang penuh.

---

## 🏗 Arsitektur Teknis (Deep Dive)

### 1. Framework Builder (`framework.go`)
Ini adalah entry point utama. Framework menggunakan *method chaining*.
* **Kenapa Builder?** Karena komponen seperti Poller butuh `WorkerPool`, dan Producer butuh `KafkaWriter`. Builder memastikan semua dependensi terpasang dengan benar sebelum aplikasi dijalankan.
* **Aturan:** Jika kamu menambah komponen baru, tambahkan method `.WithNamaKomponen()` yang mengembalikan `*Framework`.

### 2. Worker Pool & Job System (`worker.go`, `job.go`)
NanoPony tidak langsung memproses data di satu loop. Data dibungkus menjadi `Job`.
* **Job Struct:** Memiliki ID (*traceability*), Data (*payload*), dan Meta (konteks tambahan).
* **Worker:** Goroutine yang terus menerus mengambil `Job` dari channel.
* **SubmitBlocking:** Ini fitur krusial. Jika antrean (*buffer channel*) penuh, fungsi ini akan "menahan" (*block*) pengirim sampai ada slot kosong. Ini mencegah RAM jebol karena tumpukan data.

### 3. Poller & Semaphore (`poller.go`)
Poller bertugas menarik data secara periodik.
* **Slot/Semaphore:** Kami menggunakan channel sebagai semaphore (`jobSlots`). Jika `JobSlotSize` adalah 1, maka poller berikutnya tidak akan jalan sebelum poller sebelumnya selesai. Ini mencegah duplikasi pemrosesan data yang sama.
* **DataFetcher:** Sebuah interface. Kamu bisa membuat fetcher dari Database, API, atau File, selama ia mengembalikan `[]any`.

### 4. Sistem Logging Terstruktur (`logger.go`)
Logger kami didesain untuk *production monitoring*.
* **Async Logging:** Log tidak langsung ditulis ke file secara *blocking*, melainkan lewat channel agar tidak memperlambat proses utama.
* **Hybrid Output:** Bisa ke Console (untuk dev) dan Elasticsearch (untuk prod) secara bersamaan.

---

## 📏 Standar Coding & Konvensi

### 1. Penamaan (Naming)
* **Interface:** Gunakan akhiran `-er` jika memungkinkan (contoh: `DataFetcher`, `MessageProducer`).
* **Variabel Privat:** Gunakan `camelCase`.
* **Fungsi Ekspor:** Harus memiliki komentar GoDoc yang jelas.

### 2. Error Handling
* **Jangan di-ignore:** Selalu cek `if err != nil`.
* **Wrap Error:** Gunakan `fmt.Errorf("context: %w", err)` untuk memberikan konteks di mana error terjadi.
* **Worker Errors:** Error di dalam worker dikirim ke `pool.Errors()` channel, bukan di-log secara acak di dalam worker.

### 3. Concurrency
* **Mutex:** Selalu gunakan `sync.RWMutex` untuk state yang sering dibaca tapi jarang ditulis.
* **Context:** Gunakan `ctx.Done()` di setiap loop panjang agar goroutine bisa berhenti saat aplikasi dimatikan.

---

## 🛠 Langkah-Langkah Pengembangan

1. **Siapkan Environment:**
   * Copy `.env.example` menjadi `.env`.
   * Pastikan Go versi 1.25+ sudah terinstall.
2. **Membuat Fitur Baru:**
   * Jika menambah fitur di level framework, mulai dari `framework.go`.
   * Jika menambah adaptor database baru, buat file baru (misal: `postgres.go`).
3. **Surgical Update:** Jangan mengubah kode yang tidak berhubungan dengan fitur kamu. Jaga Pull Request (PR) tetap fokus.

---

## 🧪 Testing & Benchmarking

### Menjalankan Unit Test
```bash
# Jalankan semua test dengan detail
go test -v ./...

# Cek apakah ada race condition (sangat penting untuk worker pool)
go test -race ./...
```

### Menjalankan Benchmark
NanoPony sangat peduli pada alokasi memori.
```bash
go test -bench=. -benchmem
```
* **Target:** Alokasi memori harus tetap stabil (tidak naik terus menerus/leak) selama 40+ siklus pemrosesan.

---

## 🤝 Alur Pull Request

1. **Fork & Branch:** Buat branch dengan nama yang deskriptif (misal: `feat/add-postgres-support`).
2. **Commit:** Gunakan pesan commit yang jelas (misal: `feat: implement postgres adapter for framework builder`).
3. **Documentation:** Update file `README.md` atau `DOKUMENTASI.md` jika ada perubahan pada cara penggunaan framework.
4. **Review:** Tag maintainer untuk review. Pastikan semua check (lint & test) berwarna hijau ✅.

---

> **Ingat:** Di NanoPony, kode yang bagus adalah kode yang bisa dibaca seperti cerita dan berjalan secepat kilat. Selamat ngoding! 🐎⚡
