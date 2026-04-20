# Contoh NanoPony: Arsitektur Berlapis (v0.0.30)

Contoh ini menampilkan arsitektur berlapis (layered architecture) dengan pemisahan file yang jelas untuk setiap tanggung jawab. Arsitektur ini dirancang untuk aplikasi skala menengah hingga besar yang membutuhkan struktur kode yang rapi dan mudah di-maintain.

## 📁 Struktur Folder

```
layered_separation/
├── main.go                  # Entry point aplikasi & orkestrasi
├── mocks.go                 # Implementasi mock untuk demo
├── go.mod                   # Definisi modul Go
├── README.md                # Dokumentasi ini
│
├── config/
│   └── config.go           # Manajemen konfigurasi dari environment
│
├── models/
│   └── models.go           # Definisi Entity dan DTO
│
├── interfaces/
│   └── interfaces.go       # Interface Repository dan Service (Kontrak)
│
├── repository/
│   └── repository.go       # Implementasi Repository (Akses data)
│
└── service/
    └── service.go          # Implementasi Service (Logika bisnis)
```

## 🏗️ Arsitektur Berlapis

```
┌─────────────────────────────────────────────────────────┐
│                      Aplikasi Utama                      │
│                           ↓                              │
│  ┌─────────────────────────────────────────────────┐    │
│  │              Service Layer                       │    │
│  │  (Logika Bisnis & Orkestrasi)                    │    │
│  │  ├── UserService                                 │    │
│  │  ├── OrderService                                │    │
│  │  ├── ProductService                              │    │
│  │  └── EventService                                │    │
│  └──────────────────┬──────────────────────────────┘    │
│                     ↓                                    │
│  ┌─────────────────────────────────────────────────┐    │
│  │           Repository Layer                        │    │
│  │         (Akses Data)                              │    │
│  │  ├── UserRepository                               │    │
│  │  ├── OrderRepository                              │    │
│  │  └── ProductRepository                            │    │
│  └──────────────────┬──────────────────────────────┘    │
│                     ↓                                    │
│  ┌─────────────────────────────────────────────────┐    │
│  │           Infrastructure Layer                    │    │
│  │  ├── Oracle Database                              │    │
│  │  └── Kafka Producer                               │    │
│  └──────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

## 📦 Penjelasan Setiap Layer

### 1. **Config Layer** (`config/config.go`)
Mengelola konfigurasi aplikasi yang dimuat dari variabel environment. Mendukung prefix untuk membedakan antar lingkungan (local/staging/production).

### 2. **Models Layer** (`models/models.go`)
Berisi definisi struct untuk Entity database dan Data Transfer Objects (DTO). Memisahkan representasi database dari representasi API.

### 3. **Interfaces Layer** (`interfaces/interfaces.go`)
Mendefinisikan kontrak antara berbagai layer. Framework NanoPony tidak lagi menyediakan base interface untuk Repository atau Service, sehingga Anda bebas mendefinisikan metode yang dibutuhkan aplikasi Anda.

### 4. **Repository Layer** (`repository/repository.go`)
Implementasi akses data langsung ke Oracle Database menggunakan `*sql.DB`.

```go
type userRepository struct {
    DB *sql.DB
}

func NewUserRepository(db *sql.DB) interfaces.UserRepository {
    return &userRepository{DB: db}
}
```

### 5. **Service Layer** (`service/service.go`)
Tempat utama logika bisnis dijalankan. Jika membutuhkan transaksi database, service akan mengelola `sql.Tx` secara manual atau melalui helper internal aplikasi.

```go
func (s *userService) ActivateUser(ctx context.Context, userID int) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Operasi...
    
    return tx.Commit()
}
```

## 🚀 Cara Menjalankan

```bash
cd examples/layered_separation
go run .
```

## 📋 Best Practices yang Diterapkan

1. **Separation of Concerns**: Setiap layer memiliki tanggung jawab yang spesifik dan terisolasi.
2. **Dependency Injection**: Service mengonsumsi repository melalui interface, memudahkan pengujian unit dengan mock.
3. **Manual Transaction Management**: Memberikan kontrol penuh atas atomisitas operasi bisnis tanpa ketergantungan pada abstraksi framework yang kaku.
4. **Context Propagation**: Semua metode menerima `context.Context` untuk mendukung pembatalan dan timeout.
5. **No Framework Bloat**: Aplikasi mendefinisikan apa yang dibutuhkannya sendiri, tidak dipaksa mengikuti struktur `BaseRepository` atau `BaseService` dari framework.

## 🎓 Kesimpulan

Arsitektur ini sangat direkomendasikan untuk aplikasi enterprise yang membutuhkan kontrol penuh atas alur bisnis dan integrasi database/messaging. Dengan menghapus abstraksi yang berlebihan, kode menjadi lebih mudah dipahami, di-debug, dan di-maintain dalam jangka panjang.
