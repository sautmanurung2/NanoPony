> **Bahasa:** [Bahasa Indonesia](CONTRIBUTING.md) | [English](CONTRIBUTING_EN.md)

# Kontribusi untuk NanoPony

Pertama-tama, terima kasih telah mempertimbangkan untuk berkontribusi di NanoPony! Orang-orang seperti Anda lah yang membuat NanoPony menjadi alat yang luar biasa untuk integrasi Kafka-Oracle dengan efisiensi tinggi.

## Pemberitahuan Hukum & Perjanjian Lisensi Kontributor (CLA)

Dengan berkontribusi pada proyek ini, Anda menyetujui persyaratan dalam [Perjanjian Lisensi Kontributor (CLA)](CLA.md).

**Penting:** Kontribusi Anda akan menjadi hak milik bersama oleh **Saut Manurung** dan **JNE Indonesia**. Dengan mengirimkan Pull Request (PR), Anda mengakui dan menyetujui pengalihan hak kepemilikan ini sebagaimana dijelaskan secara rinci dalam CLA. Setiap Pull Request wajib menyertakan tanda centang pada template PR yang menandakan persetujuan Anda.

## Standar Pengembangan

### Persiapan Lingkungan (Environment Setup)
- **Versi Go**: 1.21+ (direkomendasikan 1.22+)
- **Dependensi**: Dikelola melalui Go modules. Jalankan `go mod download` untuk mempersiapkan.

### Standar Penulisan Kode (Coding Standards)
- **Formatting**: Selalu jalankan `go fmt ./...` sebelum melakukan commit.
- **Dokumentasi**: Semua fungsi, tipe, dan konstanta publik (Exported) **WAJIB** memiliki komentar GoDoc yang menjelaskan tujuan, parameter, dan nilai kembaliannya.
- **Penamaan**: Gunakan `camelCase` untuk variabel lokal dan `PascalCase` untuk item yang diekspor.
- **Gaya (Style)**: Ikuti idiom standar Go. Utamakan komposisi eksplisit daripada pewarisan (inheritance) yang kompleks.

### Pengujian & Performa
Kami memprioritaskan performa dan stabilitas.
- **Unit Test**: Pastikan semua logika tercakup. Jalankan `go test -v ./...`.
- **Benchmark**: Jika Anda mengubah logika inti (Worker Pool, Poller, adaptor Kafka/DB), Anda **WAJIB** menjalankan benchmark untuk memastikan tidak ada penurunan performa (regresi).
  - Jalankan benchmark: `go test -bench=. -benchmem -v ./...`

## Cara Berkontribusi

### Melaporkan Bug
*   Periksa [Issues](https://github.com/sautmanurung/NanoPony/issues) untuk melihat apakah bug tersebut sudah dilaporkan.
*   Jika belum, buka issue baru. Jelaskan masalahnya secara jelas, termasuk detail lingkungan dan langkah-langkah untuk mereproduksi bug tersebut.

### Menyarankan Peningkatan
*   Buka issue baru dan jelaskan fitur yang ingin Anda lihat, masalah yang diselesaikannya, dan potensi dampaknya terhadap performa.

### Pull Request
1.  Fork repository ini.
2.  Buat branch baru (`git checkout -b feature/fitur-baru-saya`).
3.  Terapkan perubahan Anda dan tambahkan pengujian yang sesuai.
4.  Jalankan semua pengujian dan benchmark.
5.  Commit perubahan Anda dengan pesan yang jelas dan deskriptif (`git commit -m 'feat: add support for X'`).
6.  Push ke fork Anda (`git push origin feature/fitur-baru-saya`).
7.  Buat Pull Request baru. Pastikan Anda mengisi template PR secara lengkap, termasuk persetujuan CLA.

## Struktur Proyek
- `framework.go`: Entry point utama menggunakan Builder Pattern.
- `job.go`: Definisi Job inti dan Handler.
- `worker.go`: Implementasi Worker Pool dan manajemen konkurensi.
- `poller.go`: Pengambilan data periodik dan pembatasan laju (rate limiting).
- `logger.go`: Sistem logging terstruktur.
- `database.go` & `kafka.go`: Adaptor untuk Oracle dan Kafka.

## Kode Etik (Code of Conduct)
Harap bersikap hormat, profesional, dan kolaboratif dalam semua interaksi.

---
Dibuat dengan ❤️ oleh tim NanoPony.
