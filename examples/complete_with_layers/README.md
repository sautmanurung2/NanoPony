# Contoh Lengkap: Interface, Repository & Service Layer

Contoh ini menampilkan arsitektur 3-tier yang proper menggunakan NanoPony framework.

## Arsitektur

```
┌─────────────────────────────────────────────────────────┐
│                    Main Application                      │
│                                                          │
│  Controller/Handler → Service Layer → Repository Layer  │
│                        ↓                    ↓            │
│                  Kafka Producer      Oracle Database     │
└─────────────────────────────────────────────────────────┘
```

## Struktur File

```
complete_with_layers/
├── main.go          # Contoh lengkap implementasi
├── go.mod           # Go module definition
└── README.md        # Dokumentasi ini
```

## Komponen yang Ditampilkan

### 1. **Domain Interfaces**

Definisi interface untuk entity:
- `User` - Entity user
- `Order` - Entity order

### 2. **Repository Layer**

Interface dan implementasi untuk akses data:

```go
// Interface
type UserRepository interface {
    nanopony.Repository
    GetByID(id int) (*User, error)
    GetAll() ([]User, error)
    GetActiveUsers() ([]User, error)
    Create(user *User) error
    UpdateStatus(id int, status string) error
}

// Implementasi
type userRepositoryImpl struct {
    *nanopony.BaseRepository
}
```

**Fitur:**
- Embed `nanopony.BaseRepository` untuk akses DB
- CRUD operations
- Custom queries untuk use cases spesifik

### 3. **Service Layer**

Interface dan implementasi untuk business logic:

```go
// Interface
type UserService interface {
    nanopony.Service
    GetUserWithOrders(ctx context.Context, userID int) (*User, []Order, error)
    ActivateUser(ctx context.Context, userID int) error
    SendUserEventToKafka(ctx context.Context, userID int, event string) error
}

// Implementasi
type userServiceImpl struct {
    *nanopony.BaseService
    userRepo  UserRepository
    orderRepo OrderRepository
    producer  *nanopony.KafkaProducer
}
```

**Fitur:**
- Embed `nanopony.BaseService` untuk lifecycle management
- Dependency injection untuk repositories
- Business use cases (multi-step processes)
- Transaction support
- Kafka integration

### 4. **Data Fetcher untuk Poller**

```go
type pendingOrderFetcher struct {
    orderRepo OrderRepository
}

func (f *pendingOrderFetcher) Fetch() ([]interface{}, error) {
    orders, err := f.orderRepo.GetPendingOrders()
    // Convert ke []interface{} untuk worker pool
}
```

### 5. **Framework Integration**

```go
framework := nanopony.NewFramework().
    WithConfig(config).
    WithKafkaWriterFromInstance(kafkaWriter).
    WithProducerFromInstance(producer).
    WithWorkerPool(5, 100).
    WithPoller(config, fetcher).
    AddService(userService).
    AddService(orderService)
```

## Cara Menjalankan

```bash
cd examples/complete_with_layers
go run main.go
```

## Output yang Diharapkan

```
╔══════════════════════════════════════════════════════════╗
║   NanoPony Example: Interface, Repository & Service     ║
╚══════════════════════════════════════════════════════════╝

[1] Initializing configuration...
[2] Database connection (skipped for demo)
[3] Initializing Kafka producer...
    ✓ Kafka producer initialized
[4] Creating repositories...
    ✓ UserRepository created
    ✓ OrderRepository created
[5] Creating services with dependency injection...
    ✓ UserService created
    ✓ OrderService created
[6] Building framework...
    ✓ Framework built successfully
[7] Starting framework...
    ✓ Framework started
[8] Demonstrating service layer operations...
═══════════════════════════════════════════════════════════

>>> Example 1: Get User with Orders
User: User 1 (user1@example.com)
Status: INACTIVE
Total Orders: 2

Orders:
  - Order #1: Laptop (Rp 12000000.00) [PENDING]
  - Order #2: Mouse (Rp 250000.00) [COMPLETED]

>>> Example 2: Activate User
[UserService] User 2 activated successfully

>>> Example 3: Create Order with Pipeline Validation
[MockOrderRepository] Create order: Laptop Gaming (Rp 15000000.00)
[OrderService] Order 100 created and notification sent

>>> Example 4: Create Invalid Order (Validation Should Fail)
    ✓ Validation caught error: order pipeline failed: validation failed: order amount must be positive

>>> Example 5: Process Pending Orders
[OrderService] Found 2 pending orders to process
[OrderService] Processing order 1 for user User 1 (user1@example.com)

>>> Example 6: Send Event to Kafka
[UserService] Event 'USER_LOGIN' sent to Kafka for user 1

═══════════════════════════════════════════════════════════

[9] Framework running. Press Ctrl+C to shutdown...
```

## Contoh Use Cases

### 1. Get User dengan Orders

```go
user, orders, err := userService.GetUserWithOrders(ctx, 1)
```

**Proses:**
1. Service memanggil repository untuk get user
2. Service memanggil repository untuk get orders
3. Service combine data dan return

### 2. Activate User dengan Transaction & Kafka

```go
err := userService.ActivateUser(ctx, 2)
```

**Proses:**
1. Begin transaction
2. Update user status di database
3. Get user data
4. Send event ke Kafka
5. Commit transaction (atau rollback jika error)

### 3. Create Order dengan Pipeline

```go
order := &Order{
    ID:      100,
    UserID:  1,
    Product: "Laptop",
    Amount:  15000000,
}
err := orderService.CreateOrderAndNotify(ctx, order)
```

**Proses:**
1. Pipeline validator: Validate order data
2. Pipeline processor:
   - Set status dan timestamp
   - Create order di database
   - Send notification ke Kafka

### 4. Process Pending Orders (Batch)

```go
err := orderService.ProcessPendingOrders(ctx)
```

**Proses:**
1. Get all pending orders dari repository
2. For each order:
   - Get user info
   - Process order (update status)
   - Send notification ke Kafka
   - Update status ke completed

## Best Practices yang Ditampilkan

### ✅ 1. Interface Segregation

```go
// Interface untuk kontrak
type UserService interface {
    nanopony.Service
    GetUserWithOrders(ctx context.Context, userID int) (*User, []Order, error)
    // ...
}

// Implementasi concrete
type userServiceImpl struct {
    *nanopony.BaseService
    userRepo  UserRepository
    orderRepo OrderRepository
    producer  *nanopony.KafkaProducer
}
```

### ✅ 2. Dependency Injection

```go
func NewUserService(
    userRepo UserRepository, 
    orderRepo OrderRepository, 
    producer *nanopony.KafkaProducer
) UserService {
    return &userServiceImpl{
        BaseService: nanopony.NewBaseService("UserService"),
        userRepo:    userRepo,
        orderRepo:   orderRepo,
        producer:    producer,
    }
}
```

### ✅ 3. Transaction Support

```go
executor := nanopony.NewTransactionExecutor(db)
err := executor.WithTransaction(func(tx *sql.Tx) error {
    // Multiple operations dalam transaction
    s.userRepo.UpdateStatus(userID, "ACTIVE")
    user, _ := s.userRepo.GetByID(userID)
    s.producer.ProduceWithContext(ctx, "user-events", event)
    return nil
})
```

### ✅ 4. Pipeline Pattern

```go
pipeline := nanopony.NewPipeline(processor).
    AddValidator(validator)

err := pipeline.Process(order)
```

### ✅ 5. Service Lifecycle

```go
func (s *userServiceImpl) Initialize() error {
    log.Printf("[%s] Service initialized", s.ServiceName())
    return nil
}

func (s *userServiceImpl) Shutdown() error {
    log.Printf("[%s] Service shutdown", s.ServiceName())
    return nil
}
```

### ✅ 6. Repository Pattern

```go
type UserRepository interface {
    nanopony.Repository  // Embed base interface
    GetByID(id int) (*User, error)
    // ...
}
```

## Production Setup

Untuk production dengan database dan Kafka nyata:

```go
func main() {
    // 1. Connect to database
    db, err := nanopony.NewOracleFromConfig(config)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // 2. Create repositories dengan real DB
    userRepo := NewUserRepository(db)
    orderRepo := NewOrderRepository(db)

    // 3. Create Kafka producer
    kafkaWriter := nanopony.NewKafkaWriterFromConfig(config)
    producer := nanopony.NewKafkaProducer(kafkaWriter)

    // 4. Create services
    userService := NewUserService(userRepo, orderRepo, producer)
    orderService := NewOrderService(orderRepo, userRepo, producer)

    // 5. Build framework
    framework := nanopony.NewFramework().
        WithConfig(config).
        WithDatabase().
        WithKafkaWriter().
        WithProducer().
        WithWorkerPool(5, 100).
        WithPoller(config, &pendingOrderFetcher{orderRepo}).
        AddService(userService).
        AddService(orderService)

    components := framework.Build()
    
    // 6. Start dan jalankan
    ctx := context.Background()
    components.Start(ctx, jobHandler)
    
    // ... wait for shutdown
}
```

## Dependency Graph

```
UserService
  ├── UserRepository (data access)
  ├── OrderRepository (data access)
  └── KafkaProducer (event publishing)

OrderService
  ├── OrderRepository (data access)
  ├── UserRepository (data access)
  └── KafkaProducer (event publishing)

Poller
  └── OrderRepository (fetch pending orders)
      └── WorkerPool (submit jobs)
```

## Kesimpulan

Contoh ini menunjukkan:

✅ **Clean Architecture** - Separation of concerns yang jelas  
✅ **Interface-Based Design** - Loose coupling antar layer  
✅ **Dependency Injection** - Easy untuk testing dan maintenance  
✅ **Transaction Management** - Data consistency dengan transaction support  
✅ **Pipeline Processing** - Validation dan processing yang composable  
✅ **Event-Driven** - Kafka integration untuk asynchronous events  
✅ **Service Lifecycle** - Initialize dan shutdown yang proper  
✅ **Framework Integration** - Semua komponen NanoPony bekerja bersama
