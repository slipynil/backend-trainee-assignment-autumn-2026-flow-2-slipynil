package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/slipynil/avito-kitchen/internal/model"
)

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create создаёт заказ: группирует позиции по ресторанам (по restaurant_id
// каждого продукта), создаёт заказ, для каждой группы — подзаказ с subtotal
// и позиции с привязкой к sub_order_id. Всё выполняется в одной транзакции.
// На входе items содержат ProductID и Quantity; PriceAtMoment заполняется
// по текущей цене продукта. Возвращает заказ с созданными подзаказами.
func (r *OrderRepository) Create(ctx context.Context, clientID int, deliveryAddress string, items []model.OrderItem) (model.Order, error) {
	if len(items) == 0 {
		return model.Order{}, fmt.Errorf("order must contain at least one item")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.Order{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // после Commit откат не требуется

	// 1. Запрашиваем restaurant_id и цену для каждого product_id.
	type productInfo struct {
		restaurantID int
		price        float64
	}
	infos := make([]productInfo, len(items))
	for i := range items {
		const priceQuery = `SELECT restaurant_id, price, is_available FROM products WHERE id = $1`
		var available bool
		err := tx.QueryRow(ctx, priceQuery, items[i].ProductID).Scan(&infos[i].restaurantID, &infos[i].price, &available)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return model.Order{}, fmt.Errorf("product %d: %w", items[i].ProductID, model.ErrProductNotFound)
			}
			return model.Order{}, err
		}
		if !available {
			return model.Order{}, fmt.Errorf("product %d: %w", items[i].ProductID, model.ErrProductUnavailable)
		}
		items[i].PriceAtMoment = infos[i].price
	}

	// 2. Группируем позиции по restaurant_id (порядок групп — по первому появлению).
	var groupOrder []int
	groups := make(map[int][]int)
	for i := range items {
		rid := infos[i].restaurantID
		if _, ok := groups[rid]; !ok {
			groupOrder = append(groupOrder, rid)
		}
		groups[rid] = append(groups[rid], i)
	}

	// Считаем subtotal каждой группы и общую стоимость заказа.
	subtotals := make(map[int]float64, len(groups))
	var totalPrice float64
	for rid, idxs := range groups {
		var subtotal float64
		for _, i := range idxs {
			subtotal += infos[i].price * float64(items[i].Quantity)
		}
		subtotals[rid] = subtotal
		totalPrice += subtotal
	}

	// 3. Создаём заказ. Поле Status оставлено для совместимости.
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

	// 4-5. Для каждой группы создаём подзаказ и его позиции.
	const subOrderQuery = `
		INSERT INTO sub_orders (order_id, restaurant_id, status, subtotal)
		VALUES ($1, $2, $3, $4)
		RETURNING id, order_id, restaurant_id, status, subtotal, created_at, updated_at
	`
	const itemQuery = `
		INSERT INTO order_items (sub_order_id, product_id, quantity, price_at_moment)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	createdOrder.SubOrders = make([]model.SubOrder, 0, len(groups))
	for _, rid := range groupOrder {
		var subOrder model.SubOrder
		err = tx.QueryRow(ctx, subOrderQuery, createdOrder.ID, rid, model.SubOrderStatusPending, subtotals[rid]).Scan(
			&subOrder.ID,
			&subOrder.OrderID,
			&subOrder.RestaurantID,
			&subOrder.Status,
			&subOrder.Subtotal,
			&subOrder.CreatedAt,
			&subOrder.UpdatedAt,
		)
		if err != nil {
			return model.Order{}, err
		}

		subOrder.Items = make([]model.OrderItem, 0, len(groups[rid]))
		for _, i := range groups[rid] {
			items[i].SubOrderID = subOrder.ID
			err = tx.QueryRow(ctx, itemQuery, subOrder.ID, items[i].ProductID, items[i].Quantity, items[i].PriceAtMoment).Scan(&items[i].ID)
			if err != nil {
				return model.Order{}, err
			}
			subOrder.Items = append(subOrder.Items, items[i])
		}

		createdOrder.SubOrders = append(createdOrder.SubOrders, subOrder)
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

// GetByIDWithSubOrders возвращает заказ со всеми подзаказами и их позициями.
func (r *OrderRepository) GetByIDWithSubOrders(ctx context.Context, orderID int) (model.Order, error) {
	order, err := r.GetByID(ctx, orderID)
	if err != nil {
		return model.Order{}, err
	}

	subRepo := NewSubOrderRepository(r.db)
	subOrders, err := subRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return model.Order{}, err
	}
	order.SubOrders = subOrders

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
