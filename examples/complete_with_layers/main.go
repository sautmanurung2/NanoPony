// Contoh: Lengkap dengan Interface, Repository, dan Service Layer
// Menampilkan arsitektur 3-tier yang proper untuk aplikasi production
//
// Arsitektur:
//
//	Controller/Handler → Service Layer → Repository Layer → Database
//	                      ↓
//	                Kafka Producer
//
// Jalankan: go run main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sautmanurung2/nanopony"
)

// ============================================================================
// DOMAIN INTERFACES
// Definisi interface untuk domain model
// ============================================================================

// User merepresentasikan entity user
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Order merepresentasikan entity order
type Order struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Product   string    `json:"product"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ============================================================================
// REPOSITORY INTERFACES
// Interface untuk akses data (kontrak)
// ============================================================================

// UserRepository mendefinisikan operasi data user
type UserRepository interface {
	GetByID(id int) (*User, error)
	GetAll() ([]User, error)
	GetActiveUsers() ([]User, error)
	Create(user *User) error
	UpdateStatus(id int, status string) error
}

// OrderRepository mendefinisikan operasi data order
type OrderRepository interface {
	GetByID(id int) (*Order, error)
	GetPendingOrders() ([]Order, error)
	GetOrdersByUserID(userID int) ([]Order, error)
	Create(order *Order) error
	UpdateStatus(id int, status string) error
}

// ============================================================================
// SERVICE INTERFACES
// Interface untuk business logic (kontrak)
// ============================================================================

// UserService mendefinisikan business operations untuk user
type UserService interface {
	GetUserWithOrders(ctx context.Context, userID int) (*User, []Order, error)
	ActivateUser(ctx context.Context, userID int) error
	SendUserEventToKafka(ctx context.Context, userID int, event string) error
	ServiceName() string
}

// OrderService mendefinisikan business operations untuk order
type OrderService interface {
	ProcessPendingOrders(ctx context.Context) error
	CreateOrderAndNotify(ctx context.Context, order *Order) error
	ServiceName() string
}

// ============================================================================
// REPOSITORY IMPLEMENTATIONS
// Implementasi concrete repository
// ============================================================================

// userRepositoryImpl implementasi UserRepository
type userRepositoryImpl struct {
	DB *sql.DB
}

// NewUserRepository membuat instance UserRepository
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepositoryImpl{
		DB: db,
	}
}

func (r *userRepositoryImpl) GetByID(id int) (*User, error) {
	query := "SELECT id, name, email, status, created_at FROM users WHERE id = :1"
	row := r.DB.QueryRow(query, id)

	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *userRepositoryImpl) GetAll() ([]User, error) {
	query := "SELECT id, name, email, status, created_at FROM users"
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Status, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *userRepositoryImpl) GetActiveUsers() ([]User, error) {
	query := "SELECT id, name, email, status, created_at FROM users WHERE status = 'ACTIVE'"
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Status, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *userRepositoryImpl) Create(user *User) error {
	query := `INSERT INTO users (id, name, email, status, created_at) 
			  VALUES (:1, :2, :3, :4, :5)`
	_, err := r.DB.Exec(query, user.ID, user.Name, user.Email, user.Status, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepositoryImpl) UpdateStatus(id int, status string) error {
	query := "UPDATE users SET status = :1 WHERE id = :2"
	_, err := r.DB.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	return nil
}

// orderRepositoryImpl implementasi OrderRepository
type orderRepositoryImpl struct {
	DB *sql.DB
}

// NewOrderRepository membuat instance OrderRepository
func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepositoryImpl{
		DB: db,
	}
}

func (r *orderRepositoryImpl) GetByID(id int) (*Order, error) {
	query := "SELECT id, user_id, product, amount, status, created_at FROM orders WHERE id = :1"
	row := r.DB.QueryRow(query, id)

	var order Order
	err := row.Scan(&order.ID, &order.UserID, &order.Product, &order.Amount, &order.Status, &order.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

func (r *orderRepositoryImpl) GetPendingOrders() ([]Order, error) {
	query := "SELECT id, user_id, product, amount, status, created_at FROM orders WHERE status = 'PENDING'"
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Product, &order.Amount, &order.Status, &order.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (r *orderRepositoryImpl) GetOrdersByUserID(userID int) ([]Order, error) {
	query := "SELECT id, user_id, product, amount, status, created_at FROM orders WHERE user_id = :1"
	rows, err := r.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Product, &order.Amount, &order.Status, &order.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (r *orderRepositoryImpl) Create(order *Order) error {
	query := `INSERT INTO orders (id, user_id, product, amount, status, created_at) 
			  VALUES (:1, :2, :3, :4, :5, :6)`
	_, err := r.DB.Exec(query, order.ID, order.UserID, order.Product, order.Amount, order.Status, order.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

func (r *orderRepositoryImpl) UpdateStatus(id int, status string) error {
	query := "UPDATE orders SET status = :1 WHERE id = :2"
	_, err := r.DB.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// ============================================================================
// SERVICE IMPLEMENTATIONS
// Implementasi concrete service dengan business logic
// ============================================================================

// userServiceImpl implementasi UserService
type userServiceImpl struct {
	serviceName string
	userRepo    UserRepository
	orderRepo   OrderRepository
	producer    *nanopony.KafkaProducer
}

// NewUserService membuat instance UserService
func NewUserService(userRepo UserRepository, orderRepo OrderRepository, producer *nanopony.KafkaProducer) UserService {
	return &userServiceImpl{
		serviceName: "UserService",
		userRepo:    userRepo,
		orderRepo:   orderRepo,
		producer:    producer,
	}
}

func (s *userServiceImpl) ServiceName() string {
	return s.serviceName
}

// GetUserWithOrders mendapatkan user beserta orders-nya (use case)
func (s *userServiceImpl) GetUserWithOrders(ctx context.Context, userID int) (*User, []Order, error) {
	// Get user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get user's orders
	orders, err := s.orderRepo.GetOrdersByUserID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get orders: %w", err)
	}

	return user, orders, nil
}

// ActivateUser mengaktifkan user dan mengirim event ke Kafka (business process)
func (s *userServiceImpl) ActivateUser(ctx context.Context, userID int) error {
	// Logic transaksi manual
	db := s.userRepo.(*userRepositoryImpl).DB

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update status di database
	if err := s.userRepo.UpdateStatus(userID, "ACTIVE"); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Get user data
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send event to Kafka (di luar transaksi)
	event := map[string]interface{}{
		"event_type": "USER_ACTIVATED",
		"user_id":    user.ID,
		"user_name":  user.Name,
		"timestamp":  time.Now().Unix(),
	}

	success, err := s.producer.ProduceWithContext(ctx, "user-events", event)
	if err != nil {
		return fmt.Errorf("failed to send kafka message: %w", err)
	}

	if !success {
		return fmt.Errorf("kafka message not sent successfully")
	}

	log.Printf("[%s] User %d activated successfully", s.ServiceName(), userID)
	return nil
}

// SendUserEventToKafka mengirim user event ke Kafka
func (s *userServiceImpl) SendUserEventToKafka(ctx context.Context, userID int, event string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	eventData := map[string]interface{}{
		"event_type": event,
		"user_id":    user.ID,
		"user_email": user.Email,
		"timestamp":  time.Now().Unix(),
	}

	success, err := s.producer.ProduceWithContext(ctx, "user-events", eventData)
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("failed to send event to kafka")
	}

	log.Printf("[%s] Event '%s' sent to Kafka for user %d", s.ServiceName(), event, userID)
	return nil
}

// orderServiceImpl implementasi OrderService
type orderServiceImpl struct {
	serviceName string
	orderRepo   OrderRepository
	userRepo    UserRepository
	producer    *nanopony.KafkaProducer
}

// NewOrderService membuat instance OrderService
func NewOrderService(orderRepo OrderRepository, userRepo UserRepository, producer *nanopony.KafkaProducer) OrderService {
	return &orderServiceImpl{
		serviceName: "OrderService",
		orderRepo:   orderRepo,
		userRepo:    userRepo,
		producer:    producer,
	}
}

func (s *orderServiceImpl) ServiceName() string {
	return s.serviceName
}

// ProcessPendingOrders memproses semua pending orders (batch processing)
func (s *orderServiceImpl) ProcessPendingOrders(ctx context.Context) error {
	orders, err := s.orderRepo.GetPendingOrders()
	if err != nil {
		return fmt.Errorf("failed to get pending orders: %w", err)
	}

	log.Printf("[%s] Found %d pending orders to process", s.ServiceName(), len(orders))

	for _, order := range orders {
		// Get user info
		user, err := s.userRepo.GetByID(order.UserID)
		if err != nil {
			log.Printf("[%s] Failed to get user %d: %v", s.ServiceName(), order.UserID, err)
			continue
		}

		// Process order (business logic)
		log.Printf("[%s] Processing order %d for user %s (%s)",
			s.ServiceName(), order.ID, user.Name, user.Email)

		// Update order status
		if err := s.orderRepo.UpdateStatus(order.ID, "PROCESSING"); err != nil {
			log.Printf("[%s] Failed to update order %d: %v", s.ServiceName(), order.ID, err)
			continue
		}

		// Send notification to Kafka
		notification := map[string]interface{}{
			"event_type": "ORDER_PROCESSED",
			"order_id":   order.ID,
			"user_id":    order.UserID,
			"product":    order.Product,
			"amount":     order.Amount,
			"timestamp":  time.Now().Unix(),
		}

		if _, err := s.producer.ProduceWithContext(ctx, "order-events", notification); err != nil {
			log.Printf("[%s] Failed to send notification for order %d: %v",
				s.ServiceName(), order.ID, err)
		}

		// Update to completed
		if err := s.orderRepo.UpdateStatus(order.ID, "COMPLETED"); err != nil {
			log.Printf("[%s] Failed to complete order %d: %v", s.ServiceName(), order.ID, err)
		}
	}

	return nil
}

// CreateOrderAndNotify membuat order baru dan mengirim notifikasi
func (s *orderServiceImpl) CreateOrderAndNotify(ctx context.Context, order *Order) error {
	// Validasi manual
	if order.UserID == 0 || order.Product == "" || order.Amount <= 0 {
		return fmt.Errorf("order amount must be positive and user/product must be set")
	}

	order.Status = "PENDING"
	order.CreatedAt = time.Now()

	// Create order in database
	if err := s.orderRepo.Create(order); err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Send notification to Kafka
	notification := map[string]interface{}{
		"event_type": "ORDER_CREATED",
		"order_id":   order.ID,
		"product":    order.Product,
		"amount":     order.Amount,
		"timestamp":  time.Now().Unix(),
	}

	if _, err := s.producer.ProduceWithContext(ctx, "order-events", notification); err != nil {
		return fmt.Errorf("failed to send kafka message: %w", err)
	}

	log.Printf("[%s] Order %d created and notification sent", s.ServiceName(), order.ID)
	return nil
}

// ============================================================================
// DATA FETCHER UNTUK POLLER
// Implementasi DataFetcher untuk polling data dari database
// ============================================================================

// pendingOrderFetcher mengimplementasikan DataFetcher interface
type pendingOrderFetcher struct {
	orderRepo OrderRepository
}

func (f *pendingOrderFetcher) Fetch() ([]interface{}, error) {
	orders, err := f.orderRepo.GetPendingOrders()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending orders: %w", err)
	}

	// Convert to []interface{} untuk worker pool
	result := make([]interface{}, len(orders))
	for i, order := range orders {
		result[i] = order
	}

	return result, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// printUserWithOrders helper untuk menampilkan user dengan orders
func printUserWithOrders(user *User, orders []Order) {
	fmt.Printf("\nUser: %s (%s)\n", user.Name, user.Email)
	fmt.Printf("Status: %s\n", user.Status)
	fmt.Printf("Total Orders: %d\n", len(orders))

	if len(orders) > 0 {
		fmt.Println("\nOrders:")
		for _, order := range orders {
			fmt.Printf("  - Order #%d: %s (Rp %.2f) [%s]\n",
				order.ID, order.Product, order.Amount, order.Status)
		}
	}
	fmt.Println()
}

// ============================================================================
// MAIN APPLICATION
// ============================================================================

func main() {
	// Set environment variables (production: use .env file)
	os.Setenv("GO_ENV", "local")
	os.Setenv("KAFKA-MODELS", "kafka-localhost")
	os.Setenv("KAFKA_BROKERS_STAGING", "localhost:9092")
	os.Setenv("HOST_STAGING", "localhost")
	os.Setenv("PORT_STAGING", "1521")
	os.Setenv("DATABASE_STAGING", "ORCL")
	os.Setenv("USERNAME_STAGING", "user")
	os.Setenv("PASSWORD_STAGING", "password")

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║   NanoPony Example: Interface, Repository & Service     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 1. Inisialisasi konfigurasi
	fmt.Println("[1] Inisialisasi konfigurasi...")
	config := nanopony.NewConfig()

	// 2. Setup database connection (mock untuk demo)
	fmt.Println("[2] Database connection (skipped for demo)")
	fmt.Println()

	// 3. Setup Kafka writer dan producer (mock untuk demo)
	fmt.Println("[3] Inisialisasi Kafka producer...")
	kafkaWriter := nanopony.NewKafkaWriterFromConfig(config)
	producer := nanopony.NewKafkaProducer(kafkaWriter)
	fmt.Println("    ✓ Kafka producer initialized")
	fmt.Println()

	// 4. Membuat repositories (menggunakan nil untuk DB karena demo)
	fmt.Println("[4] Membuat repositories...")
	userRepo := NewUserRepository(nil)
	orderRepo := NewOrderRepository(nil)

	// Untuk demo, kita gunakan mock implementasi agar bisa berjalan tanpa DB
	userRepo = &mockUserRepository{}
	orderRepo = &mockOrderRepository{}

	fmt.Println("    ✓ UserRepository created")
	fmt.Println("    ✓ OrderRepository created")
	fmt.Println()

	// 5. Membuat services dengan dependency injection
	fmt.Println("[5] Membuat services dengan dependency injection...")
	userService := NewUserService(userRepo, orderRepo, producer)
	orderService := NewOrderService(orderRepo, userRepo, producer)
	fmt.Printf("    ✓ %s created\n", userService.ServiceName())
	fmt.Printf("    ✓ %s created\n", orderService.ServiceName())
	fmt.Println()

	// 6. Membuat framework dengan builder pattern
	fmt.Println("[6] Membangun framework...")
	framework := nanopony.NewFramework().
		WithConfig(config).
		WithKafkaWriterFromInstance(kafkaWriter).
		WithProducerFromInstance(producer).
		WithWorkerPool(5, 100).
		WithPoller(nanopony.DefaultPollerConfig(), &pendingOrderFetcher{orderRepo: orderRepo})

	components := framework.Build()
	fmt.Println("    ✓ Framework built successfully")
	fmt.Println()

	// 7. Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 8. Menjalankan framework
	fmt.Println("[7] Menjalankan framework...")
	components.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
		order, ok := job.Data.(Order)
		if !ok {
			return fmt.Errorf("invalid job data type")
		}

		log.Printf("Memproses order #%d: %s (Rp %.2f)",
			order.ID, order.Product, order.Amount)

		// Simulasi pemrosesan
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	fmt.Println("    ✓ Framework started")
	fmt.Println()

	// 9. Demonstrasi penggunaan service layer
	fmt.Println("[8] Demonstrasi operasi service layer...")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Contoh: Get user with orders
	fmt.Println("\n>>> Example 1: Get User with Orders")
	user, orders, err := userService.GetUserWithOrders(ctx, 1)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		printUserWithOrders(user, orders)
	}

	// Contoh: Activate user
	fmt.Println(">>> Example 2: Activate User")
	err = userService.ActivateUser(ctx, 2)
	if err != nil {
		log.Printf("    Error: %v", err)
	}

	// Contoh: Create order
	fmt.Println("\n>>> Example 3: Create Order")
	newOrder := &Order{
		ID:      100,
		UserID:  1,
		Product: "Laptop Gaming",
		Amount:  15000000,
	}
	err = orderService.CreateOrderAndNotify(ctx, newOrder)
	if err != nil {
		log.Printf("    Error: %v", err)
	}

	// Contoh: Invalid order (harus gagal validasi)
	fmt.Println("\n>>> Example 4: Create Invalid Order (Validation Should Fail)")
	invalidOrder := &Order{
		ID:      101,
		UserID:  0,
		Product: "",
		Amount:  -100,
	}
	err = orderService.CreateOrderAndNotify(ctx, invalidOrder)
	if err != nil {
		fmt.Printf("    ✓ Validasi menangkap error: %v\n", err)
	}

	// Contoh: Process pending orders
	fmt.Println("\n>>> Example 5: Process Pending Orders")
	err = orderService.ProcessPendingOrders(ctx)
	if err != nil {
		log.Printf("    Error: %v", err)
	}

	// Contoh: Send event to Kafka
	fmt.Println("\n>>> Example 6: Send Event to Kafka")
	err = userService.SendUserEventToKafka(ctx, 1, "USER_LOGIN")
	if err != nil {
		log.Printf("    Error: %v", err)
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println()

	// 10. Tunggu interrupt
	fmt.Println("[9] Framework berjalan. Tekan Ctrl+C untuk shutdown...")
	fmt.Println()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n[10] Melakukan shutdown...")
	if err := components.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	fmt.Println("[11] Framework dihentikan. Sampai jumpa!")
}

// ============================================================================
// MOCK REPOSITORIES (untuk demo tanpa database)
// Dalam production, gunakan implementasi nyata dengan database
// ============================================================================

type mockUserRepository struct {
	DB *sql.DB
}

func (r *mockUserRepository) GetByID(id int) (*User, error) {
	// Simulasi data dari database
	return &User{
		ID:        id,
		Name:      fmt.Sprintf("User %d", id),
		Email:     fmt.Sprintf("user%d@example.com", id),
		Status:    "INACTIVE",
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}, nil
}

func (r *mockUserRepository) GetAll() ([]User, error) {
	return []User{
		{ID: 1, Name: "John Doe", Email: "john@example.com", Status: "ACTIVE"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Status: "INACTIVE"},
		{ID: 3, Name: "Bob Wilson", Email: "bob@example.com", Status: "ACTIVE"},
	}, nil
}

func (r *mockUserRepository) GetActiveUsers() ([]User, error) {
	users, _ := r.GetAll()
	var active []User
	for _, user := range users {
		if user.Status == "ACTIVE" {
			active = append(active, user)
		}
	}
	return active, nil
}

func (r *mockUserRepository) Create(user *User) error {
	log.Printf("[MockUserRepository] Create user: %s (%s)", user.Name, user.Email)
	return nil
}

func (r *mockUserRepository) UpdateStatus(id int, status string) error {
	log.Printf("[MockUserRepository] Update user %d status to: %s", id, status)
	return nil
}

type mockOrderRepository struct {
	DB *sql.DB
}

func (r *mockOrderRepository) GetByID(id int) (*Order, error) {
	return &Order{
		ID:        id,
		UserID:    1,
		Product:   "Mock Product",
		Amount:    1000,
		Status:    "COMPLETED",
		CreatedAt: time.Now(),
	}, nil
}

func (r *mockOrderRepository) GetPendingOrders() ([]Order, error) {
	return []Order{
		{ID: 1, UserID: 1, Product: "Laptop", Amount: 12000000, Status: "PENDING", CreatedAt: time.Now()},
		{ID: 2, UserID: 1, Product: "Mouse", Amount: 250000, Status: "PENDING", CreatedAt: time.Now()},
	}, nil
}

func (r *mockOrderRepository) GetOrdersByUserID(userID int) ([]Order, error) {
	return []Order{
		{ID: 1, UserID: userID, Product: "Laptop", Amount: 12000000, Status: "PENDING", CreatedAt: time.Now()},
		{ID: 2, UserID: userID, Product: "Mouse", Amount: 250000, Status: "COMPLETED", CreatedAt: time.Now()},
	}, nil
}

func (r *mockOrderRepository) Create(order *Order) error {
	log.Printf("[MockOrderRepository] Create order: %s (Rp %.2f)", order.Product, order.Amount)
	return nil
}

func (r *mockOrderRepository) UpdateStatus(id int, status string) error {
	log.Printf("[MockOrderRepository] Update order %d status to: %s", id, status)
	return nil
}
