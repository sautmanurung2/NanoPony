# Contributing to NanoPony

Terima kasih telah tertarik untuk berkontribusi di NanoPony! Proyek ini bertujuan untuk menyediakan framework integrasi Kafka-Oracle yang sangat efisien dan mudah digunakan.

## Panduan Pengembangan

### 🛠 Persiapan Environment
1. Pastikan Anda memiliki Go versi terbaru (minimal 1.22+).
2. Clone repository:
   ```bash
   git clone https://github.com/sautmanurung2/NanoPony.git
   cd NanoPony
   ```
3. Install dependensi:
   ```bash
   go mod download
   ```

### 📏 Standar Coding
- **Komentar**: Semua fungsi publik (Exported) **WAJIB** memiliki komentar GoDoc yang menjelaskan fungsi, parameter, dan contoh penggunaan jika perlu.
- **Naming**: Gunakan camelCase untuk variabel lokal dan PascalCase untuk item yang diekspor. Hindari singkatan yang tidak jelas (misal: `conf` daripada `config` di dalam fungsi kecil masih oke, tapi di struct lebih baik nama lengkap).
- **Format**: Selalu jalankan `go fmt ./...` sebelum melakukan commit.

### 🧪 Testing & Benchmark
Kami sangat mementingkan performa. Jika Anda melakukan perubahan pada core logic, pastikan untuk menjalankan benchmark untuk memantau regresi performa.

**Menjalankan Unit Test:**
```bash
go test -v ./...
```

**Menjalankan Benchmark:**
```bash
go test -bench=. -benchmem -v
```

### 📁 Struktur Project
- `framework.go`: Entry point utama menggunakan Builder Pattern.
- `job.go`: Definisi unit kerja (Job Struct) & Handler.
- `worker.go`: Implementasi Worker Pool & manajemen concurrency.
- `poller.go`: Logic untuk pengambilan data periodik & rate limiting.
- `logger.go` & `logger_internal.go`: Sistem logging terstruktur (Public API vs Machinery).
- `database.go` & `kafka.go`: Adaptor untuk database Oracle dan Kafka.

## Cara Mengirimkan Kontribusi
1. Fork repository ini.
2. Buat branch baru untuk fitur atau perbaikan bug Anda (`git checkout -b feature/amazing-feature`).
3. Lakukan commit perubahan Anda dengan pesan yang deskriptif.
4. Push ke branch tersebut (`git push origin feature/amazing-feature`).
5. Buat Pull Request ke branch `main`.

---
Dibuat dengan ❤️ oleh tim NanoPony.
