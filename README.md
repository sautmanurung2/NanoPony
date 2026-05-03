# 🐎 NanoPony CLI & Framework

NanoPony adalah framework Go untuk integrasi Kafka-Oracle yang dilengkapi dengan alat bantu CLI untuk scaffolding project berbasis Clean Architecture secara instan.

## 🚀 Instalasi CLI

Sekarang Anda dapat menginstal tool CLI NanoPony secara global langsung dari root repository:

```bash
go install github.com/sautmanurung2/nanopony@latest
```

*Pastikan `$GOPATH/bin` sudah ada di dalam `$PATH` sistem Anda.*

## 🛠 Membuat Project Baru

Gunakan command `new-project` untuk membuat boilerplate project dengan struktur folder yang sudah standar:

```bash
nanopony new-project nama-project-anda
```

## 📂 Struktur Folder Scaffold

Command di atas akan menghasilkan struktur project seperti berikut:

```text
nama-project-anda/
├── main.go             # Entry point dengan inisialisasi framework NanoPony
├── go.mod              # Pengaturan module dan dependencies
└── src/                # Folder utama kode sumber
    ├── constant/       # Tempat menyimpan nilai konstanta
    ├── converter/      # Layer untuk mapping/transformasi data (DTO ke Entity, dll)
    ├── infrastructure/ # Inisialisasi driver (DB, Kafka, Redis, dll)
    ├── interfaces/     # Kontrak (interface) untuk Repository dan Service
    ├── models/         # Definisi struct data/entitas
    ├── proto/          # Tempat menyimpan file .proto (Protobuf)
    ├── repository/     # Implementasi akses data (SQL/NoSQL)
    ├── service/        # Layer Business Logic
    └── utils/          # Fungsi pembantu (helper/utilities)
```

## 🏗 Menjalankan Project

Setelah project dibuat, masuk ke folder tersebut dan jalankan:

```bash
cd nama-project-anda
go run main.go
```

## 📖 Library NanoPony

Karena repository ini sekarang berfungsi sebagai CLI Tool di root, library inti NanoPony dipindahkan ke folder `pkg`. Jika Anda ingin menggunakan library ini secara manual di project lain, gunakan import:

```go
import "github.com/sautmanurung2/nanopony/pkg"
```

## 📝 Fitur Utama Framework

- **Oracle Integration**: Koneksi database Oracle yang sudah teroptimasi.
- **Kafka Producer/Consumer**: Support JSON dan Protobuf (Proto) messaging.
- **Worker Pool**: Eksekusi job secara konkuren dengan sistem fan-out.
- **Data Poller**: Pengambilan data berkala (polling) yang terintegrasi dengan worker pool.
- **Structured Logging**: Log yang rapi untuk tracing error dan info proses.

---
Dikembangkan oleh [sautmanurung2](https://github.com/sautmanurung2)
