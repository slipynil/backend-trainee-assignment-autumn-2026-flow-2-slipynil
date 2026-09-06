package model

import "time"

type User struct {
	ID        int       `json:"id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type Order struct {
	ID              int         `json:"id" db:"id"`
	ClientID        int         `json:"client_id" db:"client_id"`
	Status          OrderStatus `json:"status" db:"status"`
	TotalPrice      float64     `json:"total_price" db:"total_price"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	DeliveryAddress string      `json:"delivery_address" db:"delivery_address"`
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
	ID            int     `json:"id"`
	OrderID       int     `json:"order_id"`
	ProductID     int     `json:"product_id"`
	Quantity      int     `json:"quantity"`
	PriceAtMoment float64 `json:"price_at_moment"`
}
