// Example: Layered Separation - Interface, Repository, Service, Models, Config
// Menampilkan arsitektur yang proper dengan pemisahan file yang jelas
//
// Structure:
//   main.go                      - Application entry point
//   config/
//     config.go                  - Configuration management
//   models/
//     models.go                  - Entity and DTO definitions
//   interfaces/
//     interfaces.go              - Repository and Service interfaces
//   repository/
//     repository.go              - Repository implementations
//   service/
//     service.go                 - Service implementations
//
// Run: go run .
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sautmanurung2/nanopony"
	"github.com/sautmanurung2/nanopony/examples/layered_separation/config"
	"github.com/sautmanurung2/nanopony/examples/layered_separation/interfaces"
	"github.com/sautmanurung2/nanopony/examples/layered_separation/models"
	"github.com/sautmanurung2/nanopony/examples/layered_separation/service"
)

// ============================================================================
// DATA FETCHER untuk Poller
// ============================================================================

type pendingOrderFetcher struct {
	orderRepo interfaces.OrderRepository
}

func (f *pendingOrderFetcher) Fetch() ([]interface{}, error) {
	ctx := context.Background()
	orders, err := f.orderRepo.GetPendingOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending orders: %w", err)
	}

	result := make([]interface{}, len(orders))
	for i, order := range orders {
		result[i] = order
	}

	return result, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func printSeparator(title string) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Printf("  %s\n", title)
	fmt.Println(strings.Repeat("=", 70))
}

func printUserWithOrders(userWithOrders *models.UserWithOrders) {
	fmt.Printf("\n👤 User: %s (%s)\n", userWithOrders.User.Name, userWithOrders.User.Email)
	fmt.Printf("   Status: %s\n", userWithOrders.User.Status)
	fmt.Printf("   Total Orders: %d\n", len(userWithOrders.Orders))
	fmt.Printf("   Total Amount: Rp %.2f\n", userWithOrders.Total)

	if len(userWithOrders.Orders) > 0 {
		fmt.Println("\n   📦 Orders:")
		for _, order := range userWithOrders.Orders {
			fmt.Printf("      - Order #%d: %s (Rp %.2f) [%s]\n",
				order.ID, order.Product, order.Amount, order.Status)
		}
	}
	fmt.Println()
}

func printOrderSummary(summary *models.OrderSummary) {
	fmt.Printf("\n📋 Order Summary:\n")
	fmt.Printf("   Order ID: #%d\n", summary.OrderID)
	fmt.Printf("   Customer: %s (%s)\n", summary.UserName, summary.UserEmail)
	fmt.Printf("   Product: %s\n", summary.Product)
	fmt.Printf("   Amount: Rp %.2f\n", summary.Amount)
	fmt.Printf("   Status: %s\n", summary.Status)
	fmt.Printf("   Created: %s\n", summary.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()
}

// ============================================================================
// MAIN APPLICATION
// ============================================================================

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║   NanoPony: Layered Architecture Example                 ║")
	fmt.Println("║   (Interface | Repository | Service | Models | Config)  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	// ========================================================================
	// STEP 1: Load Configuration
	// ========================================================================
	printSeparator("STEP 1: Loading Configuration")

	// Set environment variables for demo (production: use .env file)
	os.Setenv("GO_ENV", "local")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "1521")
	os.Setenv("DB_NAME", "ORCL")
	os.Setenv("DB_USERNAME", "user")
	os.Setenv("DB_PASSWORD", "password")
	os.Setenv("KAFKA_BROKERS", "localhost:9092")
	os.Setenv("KAFKA_TOPIC_USER_EVENTS", "user-events")
	os.Setenv("KAFKA_TOPIC_ORDER_EVENTS", "order-events")
	os.Setenv("KAFKA_TOPIC_NOTIFICATIONS", "notifications")
	os.Setenv("WORKER_POOL_SIZE", "5")
	os.Setenv("WORKER_QUEUE_SIZE", "100")
	os.Setenv("POLLER_INTERVAL", "1s")
	os.Setenv("POLLER_BATCH_SIZE", "100")

	// Load config dari environment
	cfg := config.LoadConfig()
	fmt.Printf("✓ Environment: %s\n", cfg.Env)
	fmt.Printf("✓ Database: %s:%s\n", cfg.Database.Host, cfg.Database.Port)
	fmt.Printf("✓ Kafka Brokers: %v\n", cfg.Kafka.Brokers)
	fmt.Printf("✓ Worker Pool: %d workers, queue size %d\n", cfg.WorkerPool.NumWorkers, cfg.WorkerPool.QueueSize)

	// ========================================================================
	// STEP 2: Initialize Infrastructure (Mock untuk Demo)
	// ========================================================================
	printSeparator("STEP 2: Initializing Infrastructure")

	// Untuk demo, kita gunakan mock database
	// Production code:
	// db, err := nanopony.NewOracleFromConfig(nanoponyConfig)
	// if err != nil { log.Fatal(err) }
	// db := nil // Mock untuk demo
	fmt.Println("✓ Database connection initialized (mock)")

	// Initialize Kafka writer dan producer
	nanoponyConfig := nanopony.NewConfig()
	kafkaWriter := nanopony.NewKafkaWriterFromConfig(nanoponyConfig)
	producer := nanopony.NewKafkaProducer(kafkaWriter)
	fmt.Println("✓ Kafka producer initialized")

	// ========================================================================
	// STEP 3: Create Repositories
	// ========================================================================
	printSeparator("STEP 3: Creating Repositories")

	// Production code:
	// userRepo := repository.NewUserRepository(db)
	// orderRepo := repository.NewOrderRepository(db)
	// productRepo := repository.NewProductRepository(db)

	// Mock repositories untuk demo
	userRepo := &mockUserRepository{}
	orderRepo := &mockOrderRepository{}
	productRepo := &mockProductRepository{}

	fmt.Println("✓ UserRepository created")
	fmt.Println("✓ OrderRepository created")
	fmt.Println("✓ ProductRepository created")

	// ========================================================================
	// STEP 4: Create Services (Dependency Injection)
	// ========================================================================
	printSeparator("STEP 4: Creating Services with Dependency Injection")

	// EventService diperlukan oleh UserService dan OrderService
	eventSvc := service.NewEventService(producer, cfg)
	fmt.Printf("✓ EventService created\n")

	// UserService
	userService := service.NewUserService(userRepo, orderRepo, eventSvc, nil)
	fmt.Printf("✓ UserService created\n")

	// ProductService
	productService := service.NewProductService(productRepo, eventSvc, nil)
	fmt.Printf("✓ ProductService created\n")

	// OrderService (terakhir karena depend ke semua)
	orderService := service.NewOrderService(orderRepo, userRepo, productRepo, eventSvc, nil, cfg)
	fmt.Printf("✓ OrderService created\n")

	// ========================================================================
	// STEP 5: Build Framework
	// ========================================================================
	printSeparator("STEP 5: Building Framework")

	framework := nanopony.NewFramework().
		WithConfig(nanoponyConfig).
		WithKafkaWriterFromInstance(kafkaWriter).
		WithProducerFromInstance(producer).
		WithWorkerPool(cfg.WorkerPool.NumWorkers, cfg.WorkerPool.QueueSize).
		WithPoller(nanopony.DefaultPollerConfig(), &pendingOrderFetcher{orderRepo: orderRepo})

	components := framework.Build()
	fmt.Println("✓ Framework built successfully")

	// ========================================================================
	// STEP 6: Start Framework
	// ========================================================================
	printSeparator("STEP 6: Starting Framework")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Job handler untuk worker pool
	components.Start(ctx, func(ctx context.Context, job nanopony.Job) error {
		order, ok := job.Data.(models.Order)
		if !ok {
			return fmt.Errorf("invalid job data type")
		}

		log.Printf("⚙️  Processing order #%d: %s (Rp %.2f)",
			order.ID, order.Product, order.Amount)

		time.Sleep(100 * time.Millisecond)
		return nil
	})

	fmt.Println("✓ Framework started and running")

	// ========================================================================
	// STEP 7: Demonstrate Service Layer Operations
	// ========================================================================
	printSeparator("STEP 7: Service Layer Operations")

	// Example 1: Get User with Orders
	fmt.Println("\n📝 Example 1: Get User with Orders")
	userWithOrders, err := userService.GetUserWithOrders(ctx, 1)
	if err != nil {
		log.Printf("   ❌ Error: %v", err)
	} else {
		printUserWithOrders(userWithOrders)
	}

	// Example 2: Activate User
	fmt.Println("📝 Example 2: Activate User")
	err = userService.ActivateUser(ctx, 1)
	if err != nil {
		log.Printf("   ⚠️  Error (expected - mock): %v", err)
	} else {
		fmt.Println("   ✓ User activated successfully")
	}

	// Example 3: Create Order with Pipeline
	fmt.Println("\n📝 Example 3: Create Order with Pipeline Validation")
	newOrder := &models.Order{
		ID:      100,
		UserID:  1,
		Product: "Laptop Gaming ASUS",
		Amount:  15000000,
	}
	err = orderService.CreateOrderAndNotify(ctx, newOrder)
	if err != nil {
		log.Printf("   ⚠️  Error (expected - mock): %v", err)
	} else {
		fmt.Println("   ✓ Order created and event published")
	}

	// Example 4: Invalid Order (Validation Should Fail)
	fmt.Println("\n📝 Example 4: Create Invalid Order (Validation Test)")
	invalidOrder := &models.Order{
		ID:      101,
		UserID:  0,
		Product: "",
		Amount:  -100,
	}
	err = orderService.CreateOrderAndNotify(ctx, invalidOrder)
	if err != nil {
		fmt.Printf("   ✓ Validation caught error: %v\n", err)
	}

	// Example 5: Get Order Summary
	fmt.Println("\n📝 Example 5: Get Order Summary")
	summary, err := orderService.GetOrderWithUser(ctx, 1)
	if err != nil {
		log.Printf("   ⚠️  Error: %v", err)
	} else {
		printOrderSummary(summary)
	}

	// Example 6: Process Pending Orders
	fmt.Println("📝 Example 6: Process Pending Orders (Batch)")
	err = orderService.ProcessPendingOrders(ctx)
	if err != nil {
		log.Printf("   ⚠️  Error (expected - mock): %v", err)
	} else {
		fmt.Println("   ✓ Pending orders processed")
	}

	// Example 7: Cancel Order
	fmt.Println("\n📝 Example 7: Cancel Order")
	err = orderService.CancelOrder(ctx, 1, "Customer requested cancellation")
	if err != nil {
		log.Printf("   ⚠️  Error (expected - mock): %v", err)
	} else {
		fmt.Println("   ✓ Order cancelled")
	}

	// Example 8: User Stats
	fmt.Println("\n📝 Example 8: Get User Statistics")
	stats, err := userService.GetUserStats(ctx)
	if err != nil {
		log.Printf("   ⚠️  Error: %v", err)
	} else {
		fmt.Printf("   ✓ Total Users: %v\n", stats["total_users"])
		fmt.Printf("   ✓ Active Users: %v\n", stats["active_users"])
		fmt.Printf("   ✓ Inactive Users: %v\n", stats["inactive_users"])
	}

	// Example 9: Create Product
	fmt.Println("\n📝 Example 9: Create Product")
	newProduct := &models.Product{
		ID:       1,
		Name:     "Mechanical Keyboard",
		Category: "Electronics",
		Price:    1500000,
		Stock:    50,
	}
	err = productService.CreateProduct(ctx, newProduct)
	if err != nil {
		log.Printf("   ⚠️  Error: %v", err)
	} else {
		fmt.Println("   ✓ Product created")
	}

	// Example 10: Low Stock Alert
	fmt.Println("\n📝 Example 10: Update Product Stock (Low Stock Alert)")
	err = productService.UpdateProductStock(ctx, 1, 5)
	if err != nil {
		log.Printf("   ⚠️  Error: %v", err)
	} else {
		fmt.Println("   ✓ Stock updated (low stock alert sent)")
	}

	// Example 11: Get Low Stock Products
	fmt.Println("\n📝 Example 11: Get Low Stock Products")
	lowStockProducts, err := productService.GetLowStockProducts(ctx)
	if err != nil {
		log.Printf("   ⚠️  Error: %v", err)
	} else {
		fmt.Printf("   ✓ Found %d low stock products\n", len(lowStockProducts))
		for _, product := range lowStockProducts {
			fmt.Printf("      - %s (Stock: %d)\n", product.Name, product.Stock)
		}
	}

	// Example 12: Send User Event
	fmt.Println("\n📝 Example 12: Send User Event to Kafka")
	err = userService.SendUserEvent(ctx, 1, "USER_LOGIN")
	if err != nil {
		log.Printf("   ⚠️  Error (expected - mock): %v", err)
	} else {
		fmt.Println("   ✓ User event sent")
	}

	// Example 13: Deactivate User
	fmt.Println("\n📝 Example 13: Deactivate User")
	err = userService.DeactivateUser(ctx, 1)
	if err != nil {
		log.Printf("   ⚠️  Error (expected - mock): %v", err)
	} else {
		fmt.Println("   ✓ User deactivated")
	}

	// ========================================================================
	// STEP 8: Wait for Shutdown
	// ========================================================================
	printSeparator("STEP 8: Framework Running")

	fmt.Println("\n⚡ Framework is running with layered architecture!")
	fmt.Println("   ✓ Config Layer: Centralized configuration")
	fmt.Println("   ✓ Models Layer: Entity and DTO definitions")
	fmt.Println("   ✓ Interface Layer: Repository and Service contracts")
	fmt.Println("   ✓ Repository Layer: Data access implementations")
	fmt.Println("   ✓ Service Layer: Business logic and orchestration")
	fmt.Println("\n💡 Press Ctrl+C to shutdown\n")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// ========================================================================
	// STEP 9: Graceful Shutdown
	// ========================================================================
	printSeparator("STEP 9: Graceful Shutdown")

	fmt.Println("\n🛑 Shutting down...")
	if err := components.Shutdown(ctx); err != nil {
		log.Printf("   ❌ Shutdown error: %v", err)
	}

	fmt.Println("✓ All services shutdown successfully")
	fmt.Println("✓ Framework stopped")
	fmt.Println("\n👋 Goodbye!")
}
