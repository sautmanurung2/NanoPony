# Contoh Lengkap: Arsitektur Layer (v0.0.30)

Contoh ini menampilkan arsitektur 3-tier yang proper menggunakan framework NanoPony. Arsitektur ini dirancang untuk skalabilitas, kemudahan maintenance, dan kemudahan testing.

## Arsitektur

```
┌─────────────────────────────────────────────────────────┐
│                    Aplikasi Utama                        │
│                                                          │
│  Handler/Main → Service Layer → Repository Layer        │
│                        ↓                    ↓            │
│                  Kafka Producer      Oracle Database     │
└─────────────────────────────────────────────────────────┘
```

## Struktur File

```
complete_with_layers/
├── main.go          # Contoh lengkap implementasi (Handler, Service, Repo)
├── go.mod           # Definisi modul Go
└── README.md        # Dokumentasi ini
```

## Komponen yang Ditampilkan

### 1. **Domain Entities**

Definisi struct untuk objek bisnis:
- `User` - Entity user (ID, Nama, Email, Status)
- `Order` - Entity order (ID, UserID, Produk, Jumlah, Status)

### 2. **Repository Layer**

Layer yang bertanggung jawab untuk akses data langsung ke database.

```go
// Interface (Kontrak)
type UserRepository interface {
    GetByID(id int) (*User, error)
    GetAll() ([]User, error)
    GetActiveUsers() ([]User, error)
    Create(user *User) error
    UpdateStatus(id int, status string) error
}

// Implementasi
type userRepositoryImpl struct {
    DB *sql.DB
}
```

**Fitur:**
- Mengisolasi logika SQL dari logika bisnis.
- Menggunakan `*sql.DB` untuk eksekusi query.
- Memudahkan penggantian sumber data (misal: mock untuk testing).

### 3. **Service Layer**

Layer yang berisi logika bisnis dan mengorkestrasi beberapa repositori atau layanan eksternal (seperti Kafka).

```go
// Interface (Kontrak)
type UserService interface {
    GetUserWithOrders(ctx context.Context, userID int) (*User, []Order, error)
    ActivateUser(ctx context.Context, userID int) error
    SendUserEventToKafka(ctx context.Context, userID int, event string) error
    ServiceName() string
}

// Implementasi
type userServiceImpl struct {
    serviceName string
    userRepo    UserRepository
    orderRepo   OrderRepository
    producer    *nanopony.KafkaProducer
}
```

**Fitur:**
- **Dependency Injection**: Mengonsumsi repositori melalui interface.
- **Orkestrasi**: Menggabungkan data dari beberapa repositori.
- **Integrasi Kafka**: Mengirim event bisnis setelah operasi berhasil.
- **Manajemen Transaksi**: Menangani `Begin`, `Commit`, dan `Rollback` secara manual untuk kontrol penuh.

### 4. **Data Fetcher untuk Poller**

Implementasi interface `nanopony.DataFetcher` untuk mengambil data secara berkala.

```go
type pendingOrderFetcher struct {
    orderRepo OrderRepository
}

func (f *pendingOrderFetcher) Fetch() ([]interface{}, error) {
    orders, err := f.orderRepo.GetPendingOrders()
    // Konversi ke []interface{} agar bisa diterima oleh worker pool
}
```

### 5. **Integrasi Framework**

Menggunakan `nanopony.NewFramework()` untuk merakit semua komponen.

```go
framework := nanopony.NewFramework().
    WithConfig(config).
    WithKafkaWriterFromInstance(kafkaWriter).
    WithProducerFromInstance(producer).
    WithWorkerPool(5, 100).
    WithPoller(nanopony.DefaultPollerConfig(), &pendingOrderFetcher{orderRepo: orderRepo})

components := framework.Build()
```

## Cara Menjalankan

```bash
cd examples/complete_with_layers
go run main.go
```

## Output yang Diharapkan

Aplikasi akan menampilkan langkah-langkah inisialisasi dan mensimulasikan beberapa operasi bisnis:
1. Inisialisasi konfigurasi.
2. Pembuatan repositori dan service.
3. Menjalankan framework (Worker Pool & Poller).
4. Demonstrasi Use Case:
   - Mengambil data User beserta Order-nya.
   - Mengaktifkan User (Simulasi Transaksi).
   - Membuat Order (Simulasi Validasi).
   - Memproses Pending Orders (Simulasi Batch).
   - Mengirim event ke Kafka.

## Best Practices yang Diterapkan

### ✅ 1. Pemisahan Interface (Interface Segregation)
Komponen saling berinteraksi melalui interface, bukan implementasi konkret, sehingga memudahkan pengujian unit (mocking).

### ✅ 2. Injeksi Dependensi (Dependency Injection)
Semua dependensi dimasukkan melalui constructor (`NewUserService`, `NewUserRepository`), membuat kode lebih bersih dan mudah diuji.

### ✅ 3. Manajemen Transaksi Manual
Transaksi database dikelola secara eksplisit di level Service menggunakan `db.BeginTx`, memberikan kontrol penuh atas atomisitas operasi bisnis.

### ✅ 4. Validasi Logika Bisnis
Validasi dilakukan di dalam metode service sebelum data dikirim ke repositori untuk disimpan.

### ✅ 5. Penanganan Shutdown yang Aman (Graceful Shutdown)
Menggunakan `components.Shutdown(ctx)` untuk memastikan poller berhenti dan semua worker menyelesaikan tugasnya sebelum aplikasi benar-benar berhenti.

## Kesimpulan

Contoh ini adalah cetak biru (blueprint) yang direkomendasikan untuk membangun aplikasi menggunakan **NanoPony**. Dengan memisahkan tanggung jawab ke dalam layer-layer yang jelas, aplikasi Anda akan menjadi lebih stabil, mudah diuji, dan siap untuk lingkungan produksi.
