package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/slipynil/avito-kitchen/internal/model"
)

type OrderRepository struct {
	db *pgx.Conn
}

func NewOrderRepository(db *pgx.Conn) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create сохраняет новый заказ в базе данных на основе переданных позиций
func (r *OrderRepository) Create(ctx context.Context, clientID int, deliveryAddress string, items []model.OrderItem) (model.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.Order{}, err
	}
	defer tx.Rollback(ctx)

	// Получаем текущие цены продуктов и вычисляем общую стоимость
	var totalPrice float64
	for i := range items {
		const priceQuery = `SELECT price FROM products WHERE id = $1`
		var price float64
		err := tx.QueryRow(ctx, priceQuery, items[i].ProductID).Scan(&price)
		if err != nil {
			return model.Order{}, err
		}
		items[i].PriceAtMoment = price
		totalPrice += price * float64(items[i].Quantity)
	}

	// Создаем заказ
	const orderQuery = `
		INSERT INTO orders (client_id, status, total_price, delivery_address)
		VALUES ($1, $2, $3, $4)
		RETURNING id, client_id, status, total_price, created_at, delivery_address
	`

	var createdOrder model.Order
	err = tx.QueryRow(ctx, orderQuery, clientID, "pending", totalPrice, deliveryAddress).Scan(
		&createdOrder.ID,
		&createdOrder.ClientID,
		&createdOrder.Status,
		&createdOrder.TotalPrice,
		&createdOrder.CreatedAt,
		&createdOrder.DeliveryAddress,
	)
	if err != nil {
		return model.Order{}, err
	}

	// Вставляем позиции заказа
	const itemQuery = `
		INSERT INTO order_items (order_id, product_id, quantity, price_at_moment)
		VALUES ($1, $2, $3, $4)
	`
	for _, item := range items {
		_, err = tx.Exec(ctx, itemQuery, createdOrder.ID, item.ProductID, item.Quantity, item.PriceAtMoment)
		if err != nil {
			return model.Order{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return model.Order{}, err
	}

	return createdOrder, nil
}

// GetByID возвращает заказ по его ID
func (r *OrderRepository) GetByID(ctx context.Context, id int) (model.Order, error) {
	const query = `
		SELECT id, client_id, status, total_price, created_at, delivery_address
		FROM orders
		WHERE id = $1
	`

	var order model.Order
	err := r.db.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.ClientID,
		&order.Status,
		&order.TotalPrice,
		&order.CreatedAt,
		&order.DeliveryAddress,
	)
	if err != nil {
		return model.Order{}, err
	}

	return order, nil
}

// GetByClientID возвращает все заказы клиента
func (r *OrderRepository) GetByClientID(ctx context.Context, clientID int) ([]model.Order, error) {
	const query = `
		SELECT id, client_id, status, total_price, created_at, delivery_address
		FROM orders
		WHERE client_id = $1
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.Query(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Order])
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// UpdateStatus обновляет статус выбранного заказа по id
func (r *OrderRepository) UpdateStatus(ctx context.Context, id int, status model.OrderStatus) (model.Order, error) {
	const query = `
		UPDATE orders
		SET status = $2
		WHERE id = $1
		RETURNING id, client_id, status, total_price, created_at, delivery_address
	`

	var updatedOrder model.Order
	err := r.db.QueryRow(ctx, query, id, status).Scan(
		&updatedOrder.ID,
		&updatedOrder.ClientID,
		&updatedOrder.Status,
		&updatedOrder.TotalPrice,
		&updatedOrder.CreatedAt,
		&updatedOrder.DeliveryAddress,
	)
	if err != nil {
		return model.Order{}, err
	}

	return updatedOrder, nil
}

// Delete удаляет заказ по id
func (r *OrderRepository) Delete(ctx context.Context, id int) error {
	const query = `DELETE FROM orders WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	return err
}
