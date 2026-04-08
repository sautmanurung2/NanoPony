// Service implementations
package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/sautmanurung2/nanopony"
	"github.com/sautmanurung2/nanopony/examples/layered_separation/config"
	"github.com/sautmanurung2/nanopony/examples/layered_separation/interfaces"
	"github.com/sautmanurung2/nanopony/examples/layered_separation/models"
)

// ============================================================================
// USER SERVICE IMPLEMENTATION
// ============================================================================

type userService struct {
	*nanopony.BaseService
	userRepo   interfaces.UserRepository
	orderRepo  interfaces.OrderRepository
	eventSvc   interfaces.EventService
	db         *sql.DB
}

// NewUserService membuat instance UserService
func NewUserService(
	userRepo interfaces.UserRepository,
	orderRepo interfaces.OrderRepository,
	eventSvc interfaces.EventService,
	db *sql.DB,
) interfaces.UserService {
	return &userService{
		BaseService: nanopony.NewBaseService("UserService"),
		userRepo:    userRepo,
		orderRepo:   orderRepo,
		eventSvc:    eventSvc,
		db:          db,
	}
}

func (s *userService) Initialize() error {
	log.Printf("[%s] Service initialized", s.ServiceName())
	return nil
}

func (s *userService) Shutdown() error {
	log.Printf("[%s] Service shutdown", s.ServiceName())
	return nil
}

// GetUserWithOrders mendapatkan user beserta orders dan total
func (s *userService) GetUserWithOrders(ctx context.Context, userID int) (*models.UserWithOrders, error) {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get user's orders
	orders, err := s.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	// Calculate total
	var total float64
	for _, order := range orders {
		total += order.Amount
	}

	return &models.UserWithOrders{
		User:   *user,
		Orders: orders,
		Total:  total,
	}, nil
}

// ActivateUser mengaktifkan user dan mengirim event
func (s *userService) ActivateUser(ctx context.Context, userID int) error {
	executor := nanopony.NewTransactionExecutor(s.db)

	err := executor.WithTransaction(func(tx *sql.Tx) error {
		// Update status
		if err := s.userRepo.UpdateStatus(ctx, userID, "ACTIVE"); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}

		// Get user data
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		// Send event
		data := map[string]interface{}{
			"user_name": user.Name,
			"user_email": user.Email,
		}

		return s.eventSvc.PublishUserEvent(ctx, userID, "USER_ACTIVATED", data)
	})

	if err != nil {
		return fmt.Errorf("activate user transaction failed: %w", err)
	}

	log.Printf("[%s] User %d activated successfully", s.ServiceName(), userID)
	return nil
}

// DeactivateUser menonaktifkan user
func (s *userService) DeactivateUser(ctx context.Context, userID int) error {
	executor := nanopony.NewTransactionExecutor(s.db)

	err := executor.WithTransaction(func(tx *sql.Tx) error {
		if err := s.userRepo.UpdateStatus(ctx, userID, "INACTIVE"); err != nil {
			return err
		}

		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return err
		}

		data := map[string]interface{}{
			"user_name": user.Name,
		}

		return s.eventSvc.PublishUserEvent(ctx, userID, "USER_DEACTIVATED", data)
	})

	if err != nil {
		return fmt.Errorf("deactivate user transaction failed: %w", err)
	}

	log.Printf("[%s] User %d deactivated successfully", s.ServiceName(), userID)
	return nil
}

// SendUserEvent mengirim user event ke Kafka
func (s *userService) SendUserEvent(ctx context.Context, userID int, eventType string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	data := map[string]interface{}{
		"user_name": user.Name,
		"user_email": user.Email,
	}

	return s.eventSvc.PublishUserEvent(ctx, userID, eventType, data)
}

// GetUserStats mendapatkan statistik user
func (s *userService) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	activeUsers, err := s.userRepo.GetActiveUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	allUsers, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}

	return map[string]interface{}{
		"total_users":    len(allUsers),
		"active_users":   len(activeUsers),
		"inactive_users": len(allUsers) - len(activeUsers),
	}, nil
}

// ============================================================================
// ORDER SERVICE IMPLEMENTATION
// ============================================================================

type orderService struct {
	*nanopony.BaseService
	orderRepo  interfaces.OrderRepository
	userRepo   interfaces.UserRepository
	productRepo interfaces.ProductRepository
	eventSvc   interfaces.EventService
	db         *sql.DB
	cfg        *config.AppConfig
}

// NewOrderService membuat instance OrderService
func NewOrderService(
	orderRepo interfaces.OrderRepository,
	userRepo interfaces.UserRepository,
	productRepo interfaces.ProductRepository,
	eventSvc interfaces.EventService,
	db *sql.DB,
	cfg *config.AppConfig,
) interfaces.OrderService {
	return &orderService{
		BaseService:   nanopony.NewBaseService("OrderService"),
		orderRepo:     orderRepo,
		userRepo:      userRepo,
		productRepo:   productRepo,
		eventSvc:      eventSvc,
		db:            db,
		cfg:           cfg,
	}
}

func (s *orderService) Initialize() error {
	log.Printf("[%s] Service initialized", s.ServiceName())
	return nil
}

func (s *orderService) Shutdown() error {
	log.Printf("[%s] Service shutdown", s.ServiceName())
	return nil
}

// ProcessPendingOrders memproses semua pending orders
func (s *orderService) ProcessPendingOrders(ctx context.Context) error {
	orders, err := s.orderRepo.GetPendingOrders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending orders: %w", err)
	}

	log.Printf("[%s] Found %d pending orders to process", s.ServiceName(), len(orders))

	for _, order := range orders {
		if err := s.processSingleOrder(ctx, &order); err != nil {
			log.Printf("[%s] Failed to process order %d: %v", s.ServiceName(), order.ID, err)
			continue
		}
	}

	return nil
}

func (s *orderService) processSingleOrder(ctx context.Context, order *models.Order) error {
	// Get user info
	user, err := s.userRepo.GetByID(ctx, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	log.Printf("[%s] Processing order %d for user %s (%s)",
		s.ServiceName(), order.ID, user.Name, user.Email)

	executor := nanopony.NewTransactionExecutor(s.db)

	return executor.WithTransaction(func(tx *sql.Tx) error {
		// Update to processing
		if err := s.orderRepo.UpdateStatus(ctx, order.ID, "PROCESSING"); err != nil {
			return err
		}

		// Send event
		data := map[string]interface{}{
			"user_name": user.Name,
			"user_email": user.Email,
			"product":    order.Product,
			"amount":     order.Amount,
		}

		if err := s.eventSvc.PublishOrderEvent(ctx, order.ID, "ORDER_PROCESSED", data); err != nil {
			return err
		}

		// Update to completed
		return s.orderRepo.UpdateStatus(ctx, order.ID, "COMPLETED")
	})
}

// CreateOrderAndNotify membuat order baru dan mengirim notifikasi
func (s *orderService) CreateOrderAndNotify(ctx context.Context, order *models.Order) error {
	// Gunakan pipeline untuk validation
	validator := nanopony.ValidatorFunc(func(data interface{}) error {
		o, ok := data.(*models.Order)
		if !ok {
			return fmt.Errorf("invalid order type")
		}
		if o.UserID == 0 {
			return fmt.Errorf("user ID is required")
		}
		if o.Product == "" {
			return fmt.Errorf("product name is required")
		}
		if o.Amount <= 0 {
			return fmt.Errorf("order amount must be positive")
		}
		return nil
	})

	processor := nanopony.ProcessorFunc(func(input interface{}) error {
		order := input.(*models.Order)
		order.Status = "PENDING"
		order.CreatedAt = time.Now()
		order.UpdatedAt = time.Now()

		// Create order
		if err := s.orderRepo.Create(ctx, order); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		// Get user
		user, err := s.userRepo.GetByID(ctx, order.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		// Send event
		eventData := map[string]interface{}{
			"user_name": user.Name,
			"user_email": user.Email,
			"product":    order.Product,
			"amount":     order.Amount,
		}

		return s.eventSvc.PublishOrderEvent(ctx, order.ID, "ORDER_CREATED", eventData)
	})

	pipeline := nanopony.NewPipeline(processor).AddValidator(validator)

	if err := pipeline.Process(order); err != nil {
		return fmt.Errorf("order pipeline failed: %w", err)
	}

	log.Printf("[%s] Order %d created and event published", s.ServiceName(), order.ID)
	return nil
}

// GetOrderWithUser mendapatkan order dengan info user
func (s *orderService) GetOrderWithUser(ctx context.Context, orderID int) (*models.OrderSummary, error) {
	return s.orderRepo.GetOrderSummary(ctx, orderID)
}

// CancelOrder membatalkan order
func (s *orderService) CancelOrder(ctx context.Context, orderID int, reason string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status == "COMPLETED" || order.Status == "CANCELLED" {
		return fmt.Errorf("cannot cancel order with status: %s", order.Status)
	}

	executor := nanopony.NewTransactionExecutor(s.db)

	return executor.WithTransaction(func(tx *sql.Tx) error {
		if err := s.orderRepo.UpdateStatus(ctx, orderID, "CANCELLED"); err != nil {
			return err
		}

		data := map[string]interface{}{
			"reason": reason,
		}

		return s.eventSvc.PublishOrderEvent(ctx, orderID, "ORDER_CANCELLED", data)
	})
}

// ============================================================================
// PRODUCT SERVICE IMPLEMENTATION
// ============================================================================

type productService struct {
	*nanopony.BaseService
	productRepo interfaces.ProductRepository
	eventSvc    interfaces.EventService
	db          *sql.DB
}

// NewProductService membuat instance ProductService
func NewProductService(
	productRepo interfaces.ProductRepository,
	eventSvc interfaces.EventService,
	db *sql.DB,
) interfaces.ProductService {
	return &productService{
		BaseService: nanopony.NewBaseService("ProductService"),
		productRepo: productRepo,
		eventSvc:    eventSvc,
		db:          db,
	}
}

func (s *productService) Initialize() error {
	log.Printf("[%s] Service initialized", s.ServiceName())
	return nil
}

func (s *productService) Shutdown() error {
	log.Printf("[%s] Service shutdown", s.ServiceName())
	return nil
}

// CreateProduct membuat product baru
func (s *productService) CreateProduct(ctx context.Context, product *models.Product) error {
	if product.Name == "" || product.Price <= 0 {
		return fmt.Errorf("invalid product data")
	}

	product.CreatedAt = time.Now()

	if err := s.productRepo.Create(ctx, product); err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	log.Printf("[%s] Product %d created: %s", s.ServiceName(), product.ID, product.Name)
	return nil
}

// UpdateProductStock mengupdate stock product
func (s *productService) UpdateProductStock(ctx context.Context, productID int, stock int) error {
	if stock < 0 {
		return fmt.Errorf("stock cannot be negative")
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	if err := s.productRepo.UpdateStock(ctx, productID, stock); err != nil {
		return err
	}

	// Send notification if stock is low
	if stock < 10 {
		data := map[string]interface{}{
			"product_name": product.Name,
			"current_stock": stock,
		}

		if err := s.eventSvc.PublishOrderEvent(ctx, productID, "LOW_STOCK_ALERT", data); err != nil {
			log.Printf("[%s] Failed to send low stock alert: %v", s.ServiceName(), err)
		}
	}

	return nil
}

// GetLowStockProducts mendapatkan products dengan stock rendah
func (s *productService) GetLowStockProducts(ctx context.Context) ([]models.Product, error) {
	allProducts, err := s.productRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	var lowStock []models.Product
	for _, product := range allProducts {
		if product.Stock < 10 {
			lowStock = append(lowStock, product)
		}
	}

	return lowStock, nil
}

// ============================================================================
// EVENT SERVICE IMPLEMENTATION
// ============================================================================

type eventService struct {
	*nanopony.BaseService
	producer *nanopony.KafkaProducer
	cfg      *config.AppConfig
}

// NewEventService membuat instance EventService
func NewEventService(
	producer *nanopony.KafkaProducer,
	cfg *config.AppConfig,
) interfaces.EventService {
	return &eventService{
		BaseService: nanopony.NewBaseService("EventService"),
		producer:    producer,
		cfg:         cfg,
	}
}

func (s *eventService) Initialize() error {
	log.Printf("[%s] Service initialized", s.ServiceName())
	return nil
}

func (s *eventService) Shutdown() error {
	log.Printf("[%s] Service shutdown", s.ServiceName())
	return nil
}

// PublishUserEvent mengirim user event ke Kafka
func (s *eventService) PublishUserEvent(ctx context.Context, userID int, eventType string, data map[string]interface{}) error {
	event := models.Event{
		EventType: eventType,
		EntityID:  userID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	topic := s.cfg.KafkaTopic(s.cfg.Kafka.UserEventsTopic)
	_, err := s.producer.ProduceWithContext(ctx, topic, event)
	if err != nil {
		return fmt.Errorf("failed to publish user event: %w", err)
	}

	log.Printf("[%s] Published event %s for user %d", s.ServiceName(), eventType, userID)
	return nil
}

// PublishOrderEvent mengirim order event ke Kafka
func (s *eventService) PublishOrderEvent(ctx context.Context, orderID int, eventType string, data map[string]interface{}) error {
	event := models.Event{
		EventType: eventType,
		EntityID:  orderID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	topic := s.cfg.KafkaTopic(s.cfg.Kafka.OrderEventsTopic)
	_, err := s.producer.ProduceWithContext(ctx, topic, event)
	if err != nil {
		return fmt.Errorf("failed to publish order event: %w", err)
	}

	log.Printf("[%s] Published event %s for order %d", s.ServiceName(), eventType, orderID)
	return nil
}

// PublishNotification mengirim notifikasi ke Kafka
func (s *eventService) PublishNotification(ctx context.Context, title string, message string) error {
	notification := map[string]interface{}{
		"title":   title,
		"message": message,
		"time":    time.Now().Unix(),
	}

	topic := s.cfg.KafkaTopic(s.cfg.Kafka.NotificationTopic)
	_, err := s.producer.ProduceWithContext(ctx, topic, notification)
	if err != nil {
		return fmt.Errorf("failed to publish notification: %w", err)
	}

	log.Printf("[%s] Published notification: %s", s.ServiceName(), title)
	return nil
}
