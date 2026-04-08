// Models: Definisi entity dan DTO
package models

import "time"

// User merepresentasikan entity user
type User struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Order merepresentasikan entity order
type Order struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Product   string    `json:"product" db:"product"`
	Amount    float64   `json:"amount" db:"amount"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Product merepresentasikan entity product
type Product struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Category  string    `json:"category" db:"category"`
	Price     float64   `json:"price" db:"price"`
	Stock     int       `json:"stock" db:"stock"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserWithOrders DTO untuk user beserta orders
type UserWithOrders struct {
	User   User    `json:"user"`
	Orders []Order `json:"orders"`
	Total  float64 `json:"total"`
}

// OrderSummary DTO untuk ringkasan order
type OrderSummary struct {
	OrderID     int       `json:"order_id"`
	UserName    string    `json:"user_name"`
	UserEmail   string    `json:"user_email"`
	Product     string    `json:"product"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Event DTO untuk Kafka messages
type Event struct {
	EventType string                 `json:"event_type"`
	EntityID  int                    `json:"entity_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

// Pagination untuk query pagination
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// PaginatedResult hasil query dengan pagination
type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}
