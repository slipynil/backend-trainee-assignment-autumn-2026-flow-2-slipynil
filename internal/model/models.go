package model

import "time"

type User struct {
	ID        int       `json:"id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID           int     `json:"id" db:"id"`
	RestaurantID int     `json:"restaurant_id" db:"restaurant_id"`
	Name         string  `json:"name" db:"name"`
	Description  string  `json:"description" db:"description"`
	Price        float64 `json:"price" db:"price"`
	IsAvailable  bool    `json:"is_available" db:"is_available"`
}

type Order struct {
	ID              int         `json:"id" db:"id"`
	ClientID        int         `json:"client_id" db:"client_id"`
	Status          OrderStatus `json:"status" db:"status"`
	TotalPrice      float64     `json:"total_price" db:"total_price"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	DeliveryAddress string      `json:"delivery_address" db:"delivery_address"`
	SubOrders       []SubOrder  `json:"sub_orders,omitempty"`
}

type OrderStatus string

const (
	OrderStatusCreated    OrderStatus = "CREATED"
	OrderStatusPaid       OrderStatus = "PAID"
	OrderStatusDelivering OrderStatus = "DELIVERING"
	OrderStatusCompleted  OrderStatus = "COMPLETED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
)

type OrderItem struct {
	ID            int     `json:"id" db:"id"`
	SubOrderID    int     `json:"sub_order_id" db:"sub_order_id"`
	ProductID     int     `json:"product_id" db:"product_id"`
	Quantity      int     `json:"quantity" db:"quantity"`
	PriceAtMoment float64 `json:"price_at_moment" db:"price_at_moment"`
}

// Restaurant заведение на платформе
type Restaurant struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Address     string    `json:"address" db:"address"`
	Phone       string    `json:"phone" db:"phone"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RestaurantAPIKey API ключ ресторана
type RestaurantAPIKey struct {
	ID           int       `json:"id" db:"id"`
	RestaurantID int       `json:"restaurant_id" db:"restaurant_id"`
	APIKey       string    `json:"api_key" db:"api_key"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// SubOrderStatus статусы подзаказа
type SubOrderStatus string

const (
	SubOrderStatusPending   SubOrderStatus = "pending"
	SubOrderStatusAccepted  SubOrderStatus = "accepted"
	SubOrderStatusPreparing SubOrderStatus = "preparing"
	SubOrderStatusReady     SubOrderStatus = "ready"
	SubOrderStatusCancelled SubOrderStatus = "cancelled"
)

// SubOrder подзаказ для конкретного ресторана
type SubOrder struct {
	ID           int            `json:"id" db:"id"`
	OrderID      int            `json:"order_id" db:"order_id"`
	RestaurantID int            `json:"restaurant_id" db:"restaurant_id"`
	Status       SubOrderStatus `json:"status" db:"status"`
	Subtotal     float64        `json:"subtotal" db:"subtotal"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
	Items        []OrderItem    `json:"items,omitempty"`
}
