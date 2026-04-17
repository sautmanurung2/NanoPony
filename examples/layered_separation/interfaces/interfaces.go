// Interfaces: Repository dan Service interfaces
package interfaces

import (
	"context"

	"github.com/sautmanurung2/nanopony/examples/layered_separation/models"
)

// ============================================================================
// REPOSITORY INTERFACES
// ============================================================================

// UserRepository mendefinisikan operasi data user
type UserRepository interface {
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetAll(ctx context.Context) ([]models.User, error)
	GetActiveUsers(ctx context.Context) ([]models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	UpdateStatus(ctx context.Context, id int, status string) error
	Delete(ctx context.Context, id int) error
}

// OrderRepository mendefinisikan operasi data order
type OrderRepository interface {
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

// ProductRepository mendefinisikan operasi data product
type ProductRepository interface {
	GetByID(ctx context.Context, id int) (*models.Product, error)
	GetAll(ctx context.Context) ([]models.Product, error)
	GetByCategory(ctx context.Context, category string) ([]models.Product, error)
	Create(ctx context.Context, product *models.Product) error
	Update(ctx context.Context, product *models.Product) error
	UpdateStock(ctx context.Context, id int, stock int) error
	Delete(ctx context.Context, id int) error
}

// ============================================================================
// SERVICE INTERFACES
// ============================================================================

// UserService mendefinisikan business operations untuk user
type UserService interface {
	GetUserWithOrders(ctx context.Context, userID int) (*models.UserWithOrders, error)
	ActivateUser(ctx context.Context, userID int) error
	DeactivateUser(ctx context.Context, userID int) error
	SendUserEvent(ctx context.Context, userID int, eventType string) error
	GetUserStats(ctx context.Context) (map[string]interface{}, error)
}

// OrderService mendefinisikan business operations untuk order
type OrderService interface {
	ProcessPendingOrders(ctx context.Context) error
	CreateOrderAndNotify(ctx context.Context, order *models.Order) error
	GetOrderWithUser(ctx context.Context, orderID int) (*models.OrderSummary, error)
	CancelOrder(ctx context.Context, orderID int, reason string) error
}

// ProductService mendefinisikan business operations untuk product
type ProductService interface {
	CreateProduct(ctx context.Context, product *models.Product) error
	UpdateProductStock(ctx context.Context, productID int, stock int) error
	GetLowStockProducts(ctx context.Context) ([]models.Product, error)
}

// EventService mendefinisikan operations untuk event/kafka
type EventService interface {
	PublishUserEvent(ctx context.Context, userID int, eventType string, data map[string]interface{}) error
	PublishOrderEvent(ctx context.Context, orderID int, eventType string, data map[string]interface{}) error
	PublishNotification(ctx context.Context, title string, message string) error
}
