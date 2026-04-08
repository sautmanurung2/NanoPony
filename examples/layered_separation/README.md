# NanoPony Example: Layered Architecture dengan Separation of Concerns

Contoh ini menampilkan arsitektur berlapis dengan pemisahan file yang jelas untuk setiap tanggung jawab (responsibility).

## 📁 Struktur Folder

```
layered_separation/
├── main.go                  # Application entry point & orchestration
├── mocks.go                 # Mock implementations untuk demo
├── go.mod                   # Go module definition
├── README.md                # Dokumentasi ini
│
├── config/
│   └── config.go           # Configuration management dari environment
│
├── models/
│   └── models.go           # Entity dan DTO definitions
│
├── interfaces/
│   └── interfaces.go       # Repository dan Service interfaces (contracts)
│
├── repository/
│   └── repository.go       # Repository implementations (data access)
│
└── service/
    └── service.go          # Service implementations (business logic)
```

## 🏗️ Arsitektur Layered

```
┌─────────────────────────────────────────────────────────┐
│                      Main Application                    │
│                           ↓                              │
│  ┌─────────────────────────────────────────────────┐    │
│  │              Service Layer                       │    │
│  │  (Business Logic & Orchestration)                │    │
│  │  ├── UserService                                 │    │
│  │  ├── OrderService                                │    │
│  │  ├── ProductService                              │    │
│  │  └── EventService                                │    │
│  └──────────────────┬──────────────────────────────┘    │
│                     ↓                                    │
│  ┌─────────────────────────────────────────────────┐    │
│  │           Repository Layer                        │    │
│  │         (Data Access)                             │    │
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
│                                                          │
│  ┌─────────────────────────────────────────────────┐    │
│  │              Cross-Cutting Concerns               │    │
│  │  ├── Config (Configuration Management)            │    │
│  │  ├── Models (Entities & DTOs)                     │    │
│  │  └── Interfaces (Contracts)                       │    │
│  └──────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

## 📦 Penjelasan Setiap Layer

### 1. **Config Layer** (`config/config.go`)

**Tujuan**: Centralized configuration management dari environment variables.

**Komponen**:
- `AppConfig` - Main configuration structure
- `DatabaseConfig` - Database connection configuration
- `KafkaConfig` - Kafka brokers dan topics configuration
- `WorkerPoolConfig` - Worker pool configuration
- `PollerConfig` - Poller configuration

**Fitur**:
```go
// Load configuration dari environment
cfg := config.LoadConfig()

// Helper untuk Kafka topic dengan prefix
topic := cfg.KafkaTopic("user-events")  // Returns: "dev.user-events"
```

**Environment Variables**:
```bash
# Application
GO_ENV=local
SERVER_PORT=8080
LOG_LEVEL=info

# Database
DB_HOST=localhost
DB_PORT=1521
DB_NAME=ORCL
DB_USERNAME=user
DB_PASSWORD=password

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC_PREFIX=dev
KAFKA_TOPIC_USER_EVENTS=user-events
KAFKA_TOPIC_ORDER_EVENTS=order-events

# Worker Pool
WORKER_POOL_SIZE=5
WORKER_QUEUE_SIZE=100

# Poller
POLLER_INTERVAL=1s
POLLER_BATCH_SIZE=100
```

---

### 2. **Models Layer** (`models/models.go`)

**Tujuan**: Entity definitions dan Data Transfer Objects (DTOs).

**Entities**:
```go
// Entity User
type User struct {
    ID        int       `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    Email     string    `json:"email" db:"email"`
    Status    string    `json:"status" db:"status"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Entity Order
type Order struct {
    ID        int       `json:"id" db:"id"`
    UserID    int       `json:"user_id" db:"user_id"`
    Product   string    `json:"product" db:"product"`
    Amount    float64   `json:"amount" db:"amount"`
    Status    string    `json:"status" db:"status"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Entity Product
type Product struct {
    ID       int       `json:"id" db:"id"`
    Name     string    `json:"name" db:"name"`
    Category string    `json:"category" db:"category"`
    Price    float64   `json:"price" db:"price"`
    Stock    int       `json:"stock" db:"stock"`
}
```

**DTOs**:
```go
// DTO untuk user dengan orders
type UserWithOrders struct {
    User   User    `json:"user"`
    Orders []Order `json:"orders"`
    Total  float64 `json:"total"`
}

// DTO untuk order summary dengan user info
type OrderSummary struct {
    OrderID     int       `json:"order_id"`
    UserName    string    `json:"user_name"`
    UserEmail   string    `json:"user_email"`
    Product     string    `json:"product"`
    Amount      float64   `json:"amount"`
    Status      string    `json:"status"`
    CreatedAt   time.Time `json:"created_at"`
}

// DTO untuk Kafka events
type Event struct {
    EventType string                 `json:"event_type"`
    EntityID  int                    `json:"entity_id"`
    Data      map[string]interface{} `json:"data"`
    Timestamp int64                  `json:"timestamp"`
}

// DTO untuk pagination
type Pagination struct {
    Page     int `json:"page"`
    PageSize int `json:"page_size"`
    Total    int `json:"total"`
}

type PaginatedResult struct {
    Data       interface{} `json:"data"`
    Pagination Pagination  `json:"pagination"`
}
```

---

### 3. **Interfaces Layer** (`interfaces/interfaces.go`)

**Tujuan**: Mendefinisikan contracts untuk repositories dan services.

**Repository Interfaces**:
```go
// UserRepository - Contract untuk user data access
type UserRepository interface {
    nanopony.Repository
    GetByID(ctx context.Context, id int) (*models.User, error)
    GetAll(ctx context.Context) ([]models.User, error)
    GetActiveUsers(ctx context.Context) ([]models.User, error)
    GetByEmail(ctx context.Context, email string) (*models.User, error)
    Create(ctx context.Context, user *models.User) error
    Update(ctx context.Context, user *models.User) error
    UpdateStatus(ctx context.Context, id int, status string) error
    Delete(ctx context.Context, id int) error
}

// OrderRepository - Contract untuk order data access
type OrderRepository interface {
    nanopony.Repository
    GetByID(ctx context.Context, id int) (*models.Order, error)
    GetPendingOrders(ctx context.Context) ([]models.Order, error)
    GetOrdersByUserID(ctx context.Context, userID int) ([]models.Order, error)
    GetOrdersWithPagination(ctx context.Context, page, pageSize int) (*models.PaginatedResult, error)
    Create(ctx context.Context, order *models.Order) error
    Update(ctx context.Context, order *models.Order) error
    UpdateStatus(ctx context.Context, id int, status string) error
    Delete(ctx context.Context, id int) error
    GetOrderSummary(ctx context.Context, orderID int) (*models.OrderSummary, error)
}

// ProductRepository - Contract untuk product data access
type ProductRepository interface {
    nanopony.Repository
    GetByID(ctx context.Context, id int) (*models.Product, error)
    GetAll(ctx context.Context) ([]models.Product, error)
    GetByCategory(ctx context.Context, category string) ([]models.Product, error)
    Create(ctx context.Context, product *models.Product) error
    Update(ctx context.Context, product *models.Product) error
    UpdateStock(ctx context.Context, id int, stock int) error
    Delete(ctx context.Context, id int) error
}
```

**Service Interfaces**:
```go
// UserService - Business operations untuk user
type UserService interface {
    nanopony.Service
    GetUserWithOrders(ctx context.Context, userID int) (*models.UserWithOrders, error)
    ActivateUser(ctx context.Context, userID int) error
    DeactivateUser(ctx context.Context, userID int) error
    SendUserEvent(ctx context.Context, userID int, eventType string) error
    GetUserStats(ctx context.Context) (map[string]interface{}, error)
}

// OrderService - Business operations untuk order
type OrderService interface {
    nanopony.Service
    ProcessPendingOrders(ctx context.Context) error
    CreateOrderAndNotify(ctx context.Context, order *models.Order) error
    GetOrderWithUser(ctx context.Context, orderID int) (*models.OrderSummary, error)
    CancelOrder(ctx context.Context, orderID int, reason string) error
}

// ProductService - Business operations untuk product
type ProductService interface {
    nanopony.Service
    CreateProduct(ctx context.Context, product *models.Product) error
    UpdateProductStock(ctx context.Context, productID int, stock int) error
    GetLowStockProducts(ctx context.Context) ([]models.Product, error)
}

// EventService - Operations untuk Kafka events
type EventService interface {
    nanopony.Service
    PublishUserEvent(ctx context.Context, userID int, eventType string, data map[string]interface{}) error
    PublishOrderEvent(ctx context.Context, orderID int, eventType string, data map[string]interface{}) error
    PublishNotification(ctx context.Context, title string, message string) error
}
```

---

### 4. **Repository Layer** (`repository/repository.go`)

**Tujuan**: Implementasi concrete untuk data access.

**Implementasi**:
- `userRepository` - Implements `UserRepository`
- `orderRepository` - Implements `OrderRepository`
- `productRepository` - Implements `ProductRepository`

**Fitur**:
- Embed `nanopony.BaseRepository` untuk akses DB
- Context-aware methods
- Proper error handling dengan wrapping
- SQL query dengan named parameters (`:1`, `:2`, dst)

**Contoh Penggunaan**:
```go
// Production code dengan real database
db, err := nanopony.NewOracleFromConfig(nanoponyConfig)
if err != nil {
    log.Fatal(err)
}

userRepo := repository.NewUserRepository(db)
orderRepo := repository.NewOrderRepository(db)
productRepo := repository.NewProductRepository(db)

// Get user
user, err := userRepo.GetByID(ctx, 1)

// Get orders
orders, err := orderRepo.GetOrdersByUserID(ctx, 1)
```

---

### 5. **Service Layer** (`service/service.go`)

**Tujuan**: Business logic dan orchestration.

**Implementasi**:
- `userService` - Implements `UserService`
- `orderService` - Implements `OrderService`
- `productService` - Implements `ProductService`
- `eventService` - Implements `EventService`

**Dependency Injection**:
```go
func NewUserService(
    userRepo interfaces.UserRepository,      // Repository dependency
    orderRepo interfaces.OrderRepository,    // Repository dependency
    eventSvc interfaces.EventService,        // Service dependency
    db *sql.DB,                              // Infrastructure dependency
) interfaces.UserService {
    return &userService{
        BaseService: nanopony.NewBaseService("UserService"),
        userRepo:    userRepo,
        orderRepo:   orderRepo,
        eventSvc:    eventSvc,
        db:          db,
    }
}
```

**Contoh Use Case - Activate User dengan Transaction**:
```go
func (s *userService) ActivateUser(ctx context.Context, userID int) error {
    executor := nanopony.NewTransactionExecutor(s.db)

    return executor.WithTransaction(func(tx *sql.Tx) error {
        // 1. Update status di database
        if err := s.userRepo.UpdateStatus(ctx, userID, "ACTIVE"); err != nil {
            return err
        }

        // 2. Get user data
        user, err := s.userRepo.GetByID(ctx, userID)
        if err != nil {
            return err
        }

        // 3. Send event ke Kafka
        data := map[string]interface{}{
            "user_name": user.Name,
            "user_email": user.Email,
        }

        return s.eventSvc.PublishUserEvent(ctx, userID, "USER_ACTIVATED", data)
    })
}
```

**Contoh Use Case - Create Order dengan Pipeline**:
```go
func (s *orderService) CreateOrderAndNotify(ctx context.Context, order *models.Order) error {
    // Validator
    validator := nanopony.ValidatorFunc(func(data interface{}) error {
        o := data.(*models.Order)
        if o.UserID == 0 {
            return fmt.Errorf("user ID is required")
        }
        if o.Amount <= 0 {
            return fmt.Errorf("order amount must be positive")
        }
        return nil
    })

    // Processor
    processor := nanopony.ProcessorFunc(func(input interface{}) error {
        order := input.(*models.Order)
        
        // Create order
        if err := s.orderRepo.Create(ctx, order); err != nil {
            return err
        }

        // Publish event
        return s.eventSvc.PublishOrderEvent(ctx, order.ID, "ORDER_CREATED", data)
    })

    // Execute pipeline
    pipeline := nanopony.NewPipeline(processor).AddValidator(validator)
    return pipeline.Process(order)
}
```

---

## 🚀 Cara Menjalankan

```bash
cd examples/layered_separation
go run .
```

## 📋 Output yang Diharapkan

```
══════════════════════════════════════════════════════════
  STEP 1: Loading Configuration
══════════════════════════════════════════════════════════
✓ Environment: local
✓ Database: localhost:1521
✓ Kafka Brokers: [localhost:9092]
✓ Worker Pool: 5 workers, queue size 100

══════════════════════════════════════════════════════════
  STEP 2: Initializing Infrastructure
══════════════════════════════════════════════════════════
✓ Database connection initialized (mock)
✓ Kafka producer initialized

══════════════════════════════════════════════════════════
  STEP 3: Creating Repositories
══════════════════════════════════════════════════════════
✓ UserRepository created
✓ OrderRepository created
✓ ProductRepository created

══════════════════════════════════════════════════════════
  STEP 4: Creating Services with Dependency Injection
══════════════════════════════════════════════════════════
✓ EventService created
✓ UserService created
✓ ProductService created
✓ OrderService created

══════════════════════════════════════════════════════════
  STEP 5: Building Framework
══════════════════════════════════════════════════════════
✓ Framework built successfully

══════════════════════════════════════════════════════════
  STEP 6: Starting Framework
══════════════════════════════════════════════════════════
✓ Framework started and running

══════════════════════════════════════════════════════════
  STEP 7: Service Layer Operations
══════════════════════════════════════════════════════════

📝 Example 1: Get User with Orders

👤 User: User 1 (user1@example.com)
   Status: ACTIVE
   Total Orders: 3
   Total Amount: Rp 13750000.00

   📦 Orders:
      - Order #1: Laptop Gaming (Rp 12000000.00) [PENDING]
      - Order #2: Wireless Mouse (Rp 250000.00) [COMPLETED]
      - Order #3: Mechanical Keyboard (Rp 1500000.00) [PROCESSING]

📝 Example 2: Activate User
   [MockUserRepository] Update user 1 status to: ACTIVE
   ✓ User activated successfully

📝 Example 3: Create Order with Pipeline Validation
   [MockOrderRepository] Create order: Laptop Gaming ASUS (Rp 15000000.00)
   ✓ Order created and event published

📝 Example 4: Create Invalid Order (Validation Test)
   ✓ Validation caught error: order pipeline failed: validation failed: user ID is required

... dan 9 examples lainnya
```

## 🎯 Contoh Use Cases

### 1. Get User dengan Orders

```go
userWithOrders, err := userService.GetUserWithOrders(ctx, 1)
// Returns: User + Orders + Total Amount
```

**Flow**:
1. Service calls `userRepo.GetByID()`
2. Service calls `orderRepo.GetOrdersByUserID()`
3. Service calculates total
4. Service returns `UserWithOrders` DTO

### 2. Activate User (Transaction + Event)

```go
err := userService.ActivateUser(ctx, 1)
```

**Flow**:
1. Begin transaction
2. `userRepo.UpdateStatus()` → ACTIVE
3. `userRepo.GetByID()` → get user data
4. `eventSvc.PublishUserEvent()` → send to Kafka
5. Commit transaction (atau rollback jika error)

### 3. Create Order dengan Validation Pipeline

```go
order := &models.Order{
    UserID:  1,
    Product: "Laptop Gaming",
    Amount:  15000000,
}
err := orderService.CreateOrderAndNotify(ctx, order)
```

**Flow**:
1. Pipeline validator: Validate order data
2. Pipeline processor:
   - Set status dan timestamps
   - `orderRepo.Create()` → save to DB
   - `userRepo.GetByID()` → get user
   - `eventSvc.PublishOrderEvent()` → send to Kafka

### 4. Process Pending Orders (Batch)

```go
err := orderService.ProcessPendingOrders(ctx)
```

**Flow**:
1. `orderRepo.GetPendingOrders()` → get all pending orders
2. For each order:
   - `userRepo.GetByID()` → get user info
   - Transaction:
     - Update status → PROCESSING
     - Publish event → Kafka
     - Update status → COMPLETED

### 5. Cancel Order

```go
err := orderService.CancelOrder(ctx, 1, "Customer request")
```

**Flow**:
1. `orderRepo.GetByID()` → get order
2. Validate status (cannot cancel COMPLETED/CANCELLED)
3. Transaction:
   - Update status → CANCELLED
   - Publish event → Kafka

### 6. Product Stock Management

```go
// Update stock
err := productService.UpdateProductStock(ctx, 1, 5)

// Get low stock products
lowStockProducts, err := productService.GetLowStockProducts(ctx)
```

**Flow**:
1. `productRepo.UpdateStock()` → update stock
2. If stock < 10:
   - `eventSvc.PublishOrderEvent()` → LOW_STOCK_ALERT

### 7. Event Publishing

```go
// User event
err := eventSvc.PublishUserEvent(ctx, 1, "USER_LOGIN", map[string]interface{}{
    "ip_address": "192.168.1.1",
})

// Order event
err := eventSvc.PublishOrderEvent(ctx, 100, "ORDER_CREATED", map[string]interface{}{
    "product": "Laptop",
    "amount":  15000000,
})

// Notification
err := eventSvc.PublishNotification(ctx, "System Alert", "Low stock detected")
```

## 🔑 Best Practices yang Ditampilkan

### ✅ 1. Separation of Concerns

Setiap layer punya tanggung jawab yang jelas:
- **Config**: Configuration management
- **Models**: Data structures
- **Interfaces**: Contracts
- **Repository**: Data access
- **Service**: Business logic

### ✅ 2. Dependency Injection

```go
// Semua dependencies di-inject via constructor
userService := service.NewUserService(
    userRepo,    // Repository
    orderRepo,   // Repository
    eventSvc,    // Service
    db,          // Infrastructure
)
```

### ✅ 3. Interface-Based Design

```go
// Program to interface, not implementation
var userRepo interfaces.UserRepository
userRepo = repository.NewUserRepository(db)
```

### ✅ 4. Transaction Management

```go
executor := nanopony.NewTransactionExecutor(db)
err := executor.WithTransaction(func(tx *sql.Tx) error {
    // Multiple operations dalam satu transaction
    s.userRepo.UpdateStatus(ctx, userID, "ACTIVE")
    user, _ := s.userRepo.GetByID(ctx, userID)
    s.eventSvc.PublishUserEvent(ctx, userID, "USER_ACTIVATED", data)
    return nil
})
```

### ✅ 5. Pipeline Pattern

```go
pipeline := nanopony.NewPipeline(processor).
    AddValidator(validator1).
    AddValidator(validator2)

err := pipeline.Process(data)
```

### ✅ 6. Context Propagation

```go
// Semua methods menerima context
func (s *userService) GetUserWithOrders(ctx context.Context, userID int) (*models.UserWithOrders, error)
func (r *userRepository) GetByID(ctx context.Context, id int) (*models.User, error)
```

### ✅ 7. Error Wrapping

```go
if err != nil {
    return fmt.Errorf("failed to get user: %w", err)
}
```

### ✅ 8. DTOs untuk Response

```go
// Jangan return entities langsung
type UserWithOrders struct {
    User   User    `json:"user"`
    Orders []Order `json:"orders"`
    Total  float64 `json:"total"`
}
```

## 🔄 Dependency Graph

```
UserService
  ├── UserRepository (data access)
  ├── OrderRepository (data access)
  ├── EventService (service dependency)
  └── Database (infrastructure)

OrderService
  ├── OrderRepository (data access)
  ├── UserRepository (data access)
  ├── ProductRepository (data access)
  ├── EventService (service dependency)
  ├── Database (infrastructure)
  └── Config (configuration)

ProductService
  ├── ProductRepository (data access)
  ├── EventService (service dependency)
  └── Database (infrastructure)

EventService
  ├── KafkaProducer (infrastructure)
  └── Config (configuration)

Poller
  └── OrderRepository (fetch pending orders)
      └── WorkerPool (submit jobs)
```

## 🎓 Kapan Menggunakan Arsitektur Ini?

### ✅ Gunakan ketika:

- Aplikasi skala medium sampai besar
- Multiple entities dengan complex business logic
- Perlu testing yang mudah (mocking)
- Tim besar dengan multiple developers
- Long-term maintenance

### ❌ Jangan gunakan ketika:

- Aplikasi sangat sederhana (CRUD saja)
- Prototype atau proof of concept
- Deadline sangat ketat
- Tim sangat kecil (1-2 orang)

## 🚀 Production Setup

Untuk production dengan real database dan Kafka:

```go
func main() {
    // 1. Load config
    cfg := config.LoadConfig()

    // 2. Connect to Oracle
    dbConfig := nanopony.DatabaseConfig{
        Host:     cfg.Database.Host,
        Port:     cfg.Database.Port,
        Database: cfg.Database.Database,
        Username: cfg.Database.Username,
        Password: cfg.Database.Password,
    }
    db, err := nanopony.NewOracleConnection(dbConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // 3. Create repositories
    userRepo := repository.NewUserRepository(db)
    orderRepo := repository.NewOrderRepository(db)
    productRepo := repository.NewProductRepository(db)

    // 4. Create Kafka producer
    nanoponyConfig := nanopony.NewConfig()
    kafkaWriter := nanopony.NewKafkaWriterFromConfig(nanoponyConfig)
    producer := nanopony.NewKafkaProducer(kafkaWriter)

    // 5. Create services
    eventSvc := service.NewEventService(producer, cfg)
    userService := service.NewUserService(userRepo, orderRepo, eventSvc, db)
    orderService := service.NewOrderService(orderRepo, userRepo, productRepo, eventSvc, db, cfg)
    productService := service.NewProductService(productRepo, eventSvc, db)

    // 6. Build framework
    framework := nanopony.NewFramework().
        WithConfig(nanoponyConfig).
        WithDatabase().
        WithKafkaWriter().
        WithProducer().
        WithWorkerPool(cfg.WorkerPool.NumWorkers, cfg.WorkerPool.QueueSize).
        WithPoller(nanopony.DefaultPollerConfig(), &pendingOrderFetcher{orderRepo}).
        AddService(userService).
        AddService(orderService).
        AddService(productService).
        AddService(eventSvc)

    components := framework.Build()

    // 7. Start
    ctx := context.Background()
    components.Start(ctx, jobHandler)

    // 8. Wait for shutdown signal
    // ...
}
```

## 📊 Perbandingan dengan Contoh Lainnya

| Fitur | `complete_with_layers` | `layered_separation` |
|-------|------------------------|----------------------|
| File Structure | Single file | Multiple files |
| Separation | Basic | Advanced |
| Config | Inline | Separate package |
| Models | Inline | Separate package |
| Interfaces | Inline | Separate package |
| Repository | Inline | Separate package |
| Service | Inline | Separate package |
| Complexity | Medium | High |
| Use Case | Demo/Tutorial | Production-ready |

## 📝 Kesimpulan

Contoh ini menunjukkan:

✅ **Clean Architecture** - Separation of concerns yang sangat jelas  
✅ **Modular Design** - Setiap package punya tanggung jawab spesifik  
✅ **Interface-Based** - Loose coupling antar komponen  
✅ **Dependency Injection** - Easy untuk testing dan maintenance  
✅ **Transaction Management** - Data consistency  
✅ **Pipeline Pattern** - Validation dan processing  
✅ **Event-Driven** - Kafka integration  
✅ **Context Propagation** - Proper cancellation support  
✅ **Error Handling** - Error wrapping dan proper messages  
✅ **DTOs** - Proper data transfer objects  

Arsitektur ini siap untuk **production** dan mudah untuk di-scale! 🚀
