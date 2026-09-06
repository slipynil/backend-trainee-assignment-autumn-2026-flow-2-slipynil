package dto

type CreateOrderRequest struct {
	ClientID        int         `json:"client_id"`
	DeliveryAddress string      `json:"delivery_address"`
	Items           []OrderItem `json:"items"`
}

type OrderItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}
