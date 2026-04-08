// Mock repositories untuk demo tanpa database
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sautmanurung2/nanopony/examples/layered_separation/models"
)

// ============================================================================
// MOCK USER REPOSITORY
// ============================================================================

type mockUserRepository struct{}

func (r *mockUserRepository) Close() error { return nil }

func (r *mockUserRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	return &models.User{
		ID:        id,
		Name:      fmt.Sprintf("User %d", id),
		Email:     fmt.Sprintf("user%d@example.com", id),
		Status:    "ACTIVE",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}, nil
}

func (r *mockUserRepository) GetAll(ctx context.Context) ([]models.User, error) {
	return []models.User{
		{ID: 1, Name: "John Doe", Email: "john@example.com", Status: "ACTIVE"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Status: "ACTIVE"},
		{ID: 3, Name: "Bob Wilson", Email: "bob@example.com", Status: "INACTIVE"},
	}, nil
}

func (r *mockUserRepository) GetActiveUsers(ctx context.Context) ([]models.User, error) {
	users, _ := r.GetAll(ctx)
	var active []models.User
	for _, user := range users {
		if user.Status == "ACTIVE" {
			active = append(active, user)
		}
	}
	return active, nil
}

func (r *mockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	users, _ := r.GetAll(ctx)
	for _, user := range users {
		if user.Email == email {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	fmt.Printf("   [MockUserRepository] Create user: %s (%s)\n", user.Name, user.Email)
	return nil
}

func (r *mockUserRepository) Update(ctx context.Context, user *models.User) error {
	fmt.Printf("   [MockUserRepository] Update user: %d\n", user.ID)
	return nil
}

func (r *mockUserRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	fmt.Printf("   [MockUserRepository] Update user %d status to: %s\n", id, status)
	return nil
}

func (r *mockUserRepository) Delete(ctx context.Context, id int) error {
	fmt.Printf("   [MockUserRepository] Delete user: %d\n", id)
	return nil
}

// ============================================================================
// MOCK ORDER REPOSITORY
// ============================================================================

type mockOrderRepository struct{}

func (r *mockOrderRepository) Close() error { return nil }

func (r *mockOrderRepository) GetByID(ctx context.Context, id int) (*models.Order, error) {
	return &models.Order{
		ID:        id,
		UserID:    1,
		Product:   "Sample Product",
		Amount:    100000,
		Status:    "PENDING",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (r *mockOrderRepository) GetPendingOrders(ctx context.Context) ([]models.Order, error) {
	return []models.Order{
		{ID: 1, UserID: 1, Product: "Laptop Gaming", Amount: 12000000, Status: "PENDING"},
		{ID: 2, UserID: 2, Product: "Wireless Mouse", Amount: 250000, Status: "PENDING"},
		{ID: 3, UserID: 1, Product: "Mechanical Keyboard", Amount: 1500000, Status: "PENDING"},
	}, nil
}

func (r *mockOrderRepository) GetOrdersByUserID(ctx context.Context, userID int) ([]models.Order, error) {
	return []models.Order{
		{ID: 1, UserID: userID, Product: "Laptop Gaming", Amount: 12000000, Status: "PENDING"},
		{ID: 2, UserID: userID, Product: "Wireless Mouse", Amount: 250000, Status: "COMPLETED"},
		{ID: 3, UserID: userID, Product: "Mechanical Keyboard", Amount: 1500000, Status: "PROCESSING"},
	}, nil
}

func (r *mockOrderRepository) GetOrdersWithPagination(ctx context.Context, page, pageSize int) (*models.PaginatedResult, error) {
	orders, _ := r.GetPendingOrders(ctx)
	return &models.PaginatedResult{
		Data: orders,
		Pagination: models.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    len(orders),
		},
	}, nil
}

func (r *mockOrderRepository) Create(ctx context.Context, order *models.Order) error {
	fmt.Printf("   [MockOrderRepository] Create order: %s (Rp %.2f)\n", order.Product, order.Amount)
	return nil
}

func (r *mockOrderRepository) Update(ctx context.Context, order *models.Order) error {
	fmt.Printf("   [MockOrderRepository] Update order: %d\n", order.ID)
	return nil
}

func (r *mockOrderRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	fmt.Printf("   [MockOrderRepository] Update order %d status to: %s\n", id, status)
	return nil
}

func (r *mockOrderRepository) Delete(ctx context.Context, id int) error {
	fmt.Printf("   [MockOrderRepository] Delete order: %d\n", id)
	return nil
}

func (r *mockOrderRepository) GetOrderSummary(ctx context.Context, orderID int) (*models.OrderSummary, error) {
	return &models.OrderSummary{
		OrderID:   orderID,
		UserName:  "John Doe",
		UserEmail: "john@example.com",
		Product:   "Laptop Gaming",
		Amount:    12000000,
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}, nil
}

// ============================================================================
// MOCK PRODUCT REPOSITORY
// ============================================================================

type mockProductRepository struct{}

func (r *mockProductRepository) Close() error { return nil }

func (r *mockProductRepository) GetByID(ctx context.Context, id int) (*models.Product, error) {
	return &models.Product{
		ID:       id,
		Name:     "Sample Product",
		Category: "Electronics",
		Price:    100000,
		Stock:    50,
	}, nil
}

func (r *mockProductRepository) GetAll(ctx context.Context) ([]models.Product, error) {
	return []models.Product{
		{ID: 1, Name: "Laptop Gaming", Category: "Electronics", Price: 12000000, Stock: 5},
		{ID: 2, Name: "Wireless Mouse", Category: "Electronics", Price: 250000, Stock: 50},
		{ID: 3, Name: "Mechanical Keyboard", Category: "Electronics", Price: 1500000, Stock: 8},
		{ID: 4, Name: "USB Cable", Category: "Accessories", Price: 50000, Stock: 100},
	}, nil
}

func (r *mockProductRepository) GetByCategory(ctx context.Context, category string) ([]models.Product, error) {
	products, _ := r.GetAll(ctx)
	var result []models.Product
	for _, product := range products {
		if product.Category == category {
			result = append(result, product)
		}
	}
	return result, nil
}

func (r *mockProductRepository) Create(ctx context.Context, product *models.Product) error {
	fmt.Printf("   [MockProductRepository] Create product: %s (Rp %.2f)\n", product.Name, product.Price)
	return nil
}

func (r *mockProductRepository) Update(ctx context.Context, product *models.Product) error {
	fmt.Printf("   [MockProductRepository] Update product: %d\n", product.ID)
	return nil
}

func (r *mockProductRepository) UpdateStock(ctx context.Context, id int, stock int) error {
	fmt.Printf("   [MockProductRepository] Update product %d stock to: %d\n", id, stock)
	return nil
}

func (r *mockProductRepository) Delete(ctx context.Context, id int) error {
	fmt.Printf("   [MockProductRepository] Delete product: %d\n", id)
	return nil
}
