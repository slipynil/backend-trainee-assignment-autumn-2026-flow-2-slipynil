package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/slipynil/avito-kitchen/internal/model"
)

type SubOrderRepository struct {
	db *pgxpool.Pool
}

func NewSubOrderRepository(db *pgxpool.Pool) *SubOrderRepository {
	return &SubOrderRepository{db: db}
}

const subOrderColumns = `
	id,
	order_id,
	restaurant_id,
	status,
	subtotal,
	created_at,
	updated_at
`

const orderItemColumns = `
	id,
	sub_order_id,
	product_id,
	quantity,
	price_at_moment
`

// Create создаёт подзаказ со статусом pending по умолчанию.
func (r *SubOrderRepository) Create(ctx context.Context, orderID, restaurantID int, subtotal float64) (model.SubOrder, error) {
	const query = `
		INSERT INTO sub_orders (order_id, restaurant_id, status, subtotal)
		VALUES ($1, $2, $3, $4)
		RETURNING id, order_id, restaurant_id, status, subtotal, created_at, updated_at
	`

	rows, err := r.db.Query(ctx, query, orderID, restaurantID, model.SubOrderStatusPending, subtotal)
	if err != nil {
		return model.SubOrder{}, err
	}
	defer rows.Close()

	subOrder, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.SubOrder])
	if err != nil {
		return model.SubOrder{}, err
	}

	return subOrder, nil
}

// GetByID возвращает подзаказ по ID вместе с его позициями.
func (r *SubOrderRepository) GetByID(ctx context.Context, id int) (model.SubOrder, error) {
	const query = `SELECT ` + subOrderColumns + ` FROM sub_orders WHERE id = $1`

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return model.SubOrder{}, err
	}
	defer rows.Close()

	subOrder, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.SubOrder])
	if err != nil {
		return model.SubOrder{}, err
	}

	items, err := r.loadItems(ctx, []int{subOrder.ID})
	if err != nil {
		return model.SubOrder{}, err
	}
	subOrder.Items = items[subOrder.ID]

	return subOrder, nil
}

// GetByOrderID возвращает все подзаказы заказа вместе с позициями каждого.
func (r *SubOrderRepository) GetByOrderID(ctx context.Context, orderID int) ([]model.SubOrder, error) {
	const query = `SELECT ` + subOrderColumns + ` FROM sub_orders WHERE order_id = $1 ORDER BY id`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subOrders, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.SubOrder])
	if err != nil {
		return nil, err
	}

	if err := r.attachItems(ctx, subOrders); err != nil {
		return nil, err
	}

	return subOrders, nil
}

// GetByRestaurant возвращает подзаказы ресторана.
// Если status != "", фильтрует по статусу, иначе возвращает все.
func (r *SubOrderRepository) GetByRestaurant(ctx context.Context, restaurantID int, status string) ([]model.SubOrder, error) {
	query := `SELECT ` + subOrderColumns + ` FROM sub_orders WHERE restaurant_id = $1`
	args := []any{restaurantID}

	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC, id DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subOrders, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.SubOrder])
	if err != nil {
		return nil, err
	}

	if err := r.attachItems(ctx, subOrders); err != nil {
		return nil, err
	}

	return subOrders, nil
}

// UpdateStatus обновляет статус подзаказа и время обновления.
func (r *SubOrderRepository) UpdateStatus(ctx context.Context, subOrderID int, status model.SubOrderStatus) (model.SubOrder, error) {
	const query = `
		UPDATE sub_orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, order_id, restaurant_id, status, subtotal, created_at, updated_at
	`

	rows, err := r.db.Query(ctx, query, subOrderID, status)
	if err != nil {
		return model.SubOrder{}, err
	}
	defer rows.Close()

	subOrder, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.SubOrder])
	if err != nil {
		return model.SubOrder{}, err
	}

	items, err := r.loadItems(ctx, []int{subOrder.ID})
	if err != nil {
		return model.SubOrder{}, err
	}
	subOrder.Items = items[subOrder.ID]

	return subOrder, nil
}

// loadItems одним запросом подгружает позиции для указанных подзаказов.
func (r *SubOrderRepository) loadItems(ctx context.Context, subOrderIDs []int) (map[int][]model.OrderItem, error) {
	const query = `SELECT ` + orderItemColumns + ` FROM order_items WHERE sub_order_id = ANY($1) ORDER BY id`

	rows, err := r.db.Query(ctx, query, subOrderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.OrderItem])
	if err != nil {
		return nil, err
	}

	grouped := make(map[int][]model.OrderItem, len(subOrderIDs))
	for _, item := range items {
		grouped[item.SubOrderID] = append(grouped[item.SubOrderID], item)
	}

	return grouped, nil
}

// attachItems подгружает позиции и раскладывает их по подзаказам.
func (r *SubOrderRepository) attachItems(ctx context.Context, subOrders []model.SubOrder) error {
	if len(subOrders) == 0 {
		return nil
	}

	ids := make([]int, len(subOrders))
	for i := range subOrders {
		ids[i] = subOrders[i].ID
	}

	grouped, err := r.loadItems(ctx, ids)
	if err != nil {
		return err
	}

	for i := range subOrders {
		if items, ok := grouped[subOrders[i].ID]; ok {
			subOrders[i].Items = items
		} else {
			subOrders[i].Items = []model.OrderItem{}
		}
	}

	return nil
}
