// Repository implementations
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sautmanurung2/nanopony/examples/layered_separation/interfaces"
	"github.com/sautmanurung2/nanopony/examples/layered_separation/models"
)

// ============================================================================
// USER REPOSITORY IMPLEMENTATION
// ============================================================================

type userRepository struct {
	DB *sql.DB
}

// NewUserRepository membuat instance UserRepository
func NewUserRepository(db *sql.DB) interfaces.UserRepository {
	return &userRepository{
		DB: db,
	}
}

func (r *userRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := "SELECT id, name, email, status, created_at, updated_at FROM users WHERE id = :1"
	row := r.DB.QueryRowContext(ctx, query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetAll(ctx context.Context) ([]models.User, error) {
	query := "SELECT id, name, email, status, created_at, updated_at FROM users"
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	return r.scanUsers(rows)
}

func (r *userRepository) GetActiveUsers(ctx context.Context) ([]models.User, error) {
	query := "SELECT id, name, email, status, created_at, updated_at FROM users WHERE status = 'ACTIVE'"
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active users: %w", err)
	}
	defer rows.Close()

	return r.scanUsers(rows)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := "SELECT id, name, email, status, created_at, updated_at FROM users WHERE email = :1"
	row := r.DB.QueryRowContext(ctx, query, email)

	var user models.User
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (id, name, email, status, created_at, updated_at) 
			  VALUES (:1, :2, :3, :4, :5, :6)`
	_, err := r.DB.ExecContext(ctx, query,
		user.ID, user.Name, user.Email, user.Status, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `UPDATE users SET name = :1, email = :2, status = :3, updated_at = :4 
			  WHERE id = :5`
	_, err := r.DB.ExecContext(ctx, query,
		user.Name, user.Email, user.Status, time.Now(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *userRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := "UPDATE users SET status = :1, updated_at = :2 WHERE id = :3"
	_, err := r.DB.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id int) error {
	query := "DELETE FROM users WHERE id = :1"
	_, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (r *userRepository) scanUsers(rows *sql.Rows) ([]models.User, error) {
	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Status, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// ============================================================================
// ORDER REPOSITORY IMPLEMENTATION
// ============================================================================

type orderRepository struct {
	DB *sql.DB
}

// NewOrderRepository membuat instance OrderRepository
func NewOrderRepository(db *sql.DB) interfaces.OrderRepository {
	return &orderRepository{
		DB: db,
	}
}

func (r *orderRepository) GetByID(ctx context.Context, id int) (*models.Order, error) {
	query := "SELECT id, user_id, product, amount, status, created_at, updated_at FROM orders WHERE id = :1"
	row := r.DB.QueryRowContext(ctx, query, id)

	var order models.Order
	err := row.Scan(&order.ID, &order.UserID, &order.Product, &order.Amount, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

func (r *orderRepository) GetPendingOrders(ctx context.Context) ([]models.Order, error) {
	query := "SELECT id, user_id, product, amount, status, created_at, updated_at FROM orders WHERE status = 'PENDING'"
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

func (r *orderRepository) GetOrdersByUserID(ctx context.Context, userID int) ([]models.Order, error) {
	query := "SELECT id, user_id, product, amount, status, created_at, updated_at FROM orders WHERE user_id = :1"
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

func (r *orderRepository) GetOrdersWithPagination(ctx context.Context, page, pageSize int) (*models.PaginatedResult, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM orders"
	var total int
	if err := r.DB.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count orders: %w", err)
	}

	// Get paginated data
	offset := (page - 1) * pageSize
	query := `SELECT id, user_id, product, amount, status, created_at, updated_at 
			  FROM orders ORDER BY created_at DESC OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY`
	rows, err := r.DB.QueryContext(ctx, query, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	orders, err := r.scanOrders(rows)
	if err != nil {
		return nil, err
	}

	return &models.PaginatedResult{
		Data: orders,
		Pagination: models.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

func (r *orderRepository) Create(ctx context.Context, order *models.Order) error {
	query := `INSERT INTO orders (id, user_id, product, amount, status, created_at, updated_at) 
			  VALUES (:1, :2, :3, :4, :5, :6, :7)`
	_, err := r.DB.ExecContext(ctx, query,
		order.ID, order.UserID, order.Product, order.Amount, order.Status, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

func (r *orderRepository) Update(ctx context.Context, order *models.Order) error {
	query := `UPDATE orders SET product = :1, amount = :2, status = :3, updated_at = :4 
			  WHERE id = :5`
	_, err := r.DB.ExecContext(ctx, query,
		order.Product, order.Amount, order.Status, time.Now(), order.ID)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	return nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := "UPDATE orders SET status = :1, updated_at = :2 WHERE id = :3"
	_, err := r.DB.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

func (r *orderRepository) Delete(ctx context.Context, id int) error {
	query := "DELETE FROM orders WHERE id = :1"
	_, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	return nil
}

func (r *orderRepository) GetOrderSummary(ctx context.Context, orderID int) (*models.OrderSummary, error) {
	query := `SELECT o.id, u.name, u.email, o.product, o.amount, o.status, o.created_at
			  FROM orders o
			  JOIN users u ON o.user_id = u.id
			  WHERE o.id = :1`
	row := r.DB.QueryRowContext(ctx, query, orderID)

	var summary models.OrderSummary
	err := row.Scan(&summary.OrderID, &summary.UserName, &summary.UserEmail,
		&summary.Product, &summary.Amount, &summary.Status, &summary.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get order summary: %w", err)
	}

	return &summary, nil
}

func (r *orderRepository) scanOrders(rows *sql.Rows) ([]models.Order, error) {
	var orders []models.Order
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Product, &order.Amount, &order.Status, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// ============================================================================
// PRODUCT REPOSITORY IMPLEMENTATION
// ============================================================================

type productRepository struct {
	DB *sql.DB
}

// NewProductRepository membuat instance ProductRepository
func NewProductRepository(db *sql.DB) interfaces.ProductRepository {
	return &productRepository{
		DB: db,
	}
}

func (r *productRepository) GetByID(ctx context.Context, id int) (*models.Product, error) {
	query := "SELECT id, name, category, price, stock, created_at FROM products WHERE id = :1"
	row := r.DB.QueryRowContext(ctx, query, id)

	var product models.Product
	err := row.Scan(&product.ID, &product.Name, &product.Category, &product.Price, &product.Stock, &product.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &product, nil
}

func (r *productRepository) GetAll(ctx context.Context) ([]models.Product, error) {
	query := "SELECT id, name, category, price, stock, created_at FROM products"
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	return r.scanProducts(rows)
}

func (r *productRepository) GetByCategory(ctx context.Context, category string) ([]models.Product, error) {
	query := "SELECT id, name, category, price, stock, created_at FROM products WHERE category = :1"
	rows, err := r.DB.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	return r.scanProducts(rows)
}

func (r *productRepository) Create(ctx context.Context, product *models.Product) error {
	query := `INSERT INTO products (id, name, category, price, stock, created_at) 
			  VALUES (:1, :2, :3, :4, :5, :6)`
	_, err := r.DB.ExecContext(ctx, query,
		product.ID, product.Name, product.Category, product.Price, product.Stock, product.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

func (r *productRepository) Update(ctx context.Context, product *models.Product) error {
	query := `UPDATE products SET name = :1, category = :2, price = :3, stock = :4 
			  WHERE id = :5`
	_, err := r.DB.ExecContext(ctx, query,
		product.Name, product.Category, product.Price, product.Stock, product.ID)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	return nil
}

func (r *productRepository) UpdateStock(ctx context.Context, id int, stock int) error {
	query := "UPDATE products SET stock = :1 WHERE id = :2"
	_, err := r.DB.ExecContext(ctx, query, stock, id)
	if err != nil {
		return fmt.Errorf("failed to update product stock: %w", err)
	}

	return nil
}

func (r *productRepository) Delete(ctx context.Context, id int) error {
	query := "DELETE FROM products WHERE id = :1"
	_, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	return nil
}

func (r *productRepository) scanProducts(rows *sql.Rows) ([]models.Product, error) {
	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Category, &product.Price, &product.Stock, &product.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	return products, nil
}
