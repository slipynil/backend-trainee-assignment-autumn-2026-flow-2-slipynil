package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/slipynil/avito-kitchen/internal/model"
)

type ProductRepository struct {
	db *pgx.Conn
}

func NewProductRepository(db *pgx.Conn) *ProductRepository {
	return &ProductRepository{db: db}
}

// productColumns явный список колонок; COALESCE защищает от NULL
// в необязательном текстовом поле description.
const productColumns = `
	id,
	restaurant_id,
	name,
	COALESCE(description, '') AS description,
	price,
	is_available
`

// GetAll возвращает список всех продуктов
func (r *ProductRepository) GetAll(ctx context.Context) ([]model.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products ORDER BY id`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Product])
	if err != nil {
		return nil, err
	}

	return products, nil
}

// GetByID возвращает продукт по его ID.
func (r *ProductRepository) GetByID(ctx context.Context, id int) (model.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products WHERE id = $1`

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return model.Product{}, err
	}
	defer rows.Close()

	product, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Product])
	if err != nil {
		return model.Product{}, err
	}

	return product, nil
}

// GetByRestaurant возвращает все продукты ресторана.
func (r *ProductRepository) GetByRestaurant(ctx context.Context, restaurantID int) ([]model.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products WHERE restaurant_id = $1 ORDER BY id`

	rows, err := r.db.Query(ctx, query, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Product])
	if err != nil {
		return nil, err
	}

	return products, nil
}

// Create создаёт новый продукт и возвращает его с присвоенным ID.
func (r *ProductRepository) Create(ctx context.Context, product model.Product) (model.Product, error) {
	const query = `
		INSERT INTO products (restaurant_id, name, description, price, is_available)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, restaurant_id, name, COALESCE(description, '') AS description, price, is_available
	`

	rows, err := r.db.Query(ctx, query,
		product.RestaurantID,
		product.Name,
		product.Description,
		product.Price,
		product.IsAvailable,
	)
	if err != nil {
		return model.Product{}, err
	}
	defer rows.Close()

	created, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Product])
	if err != nil {
		return model.Product{}, err
	}

	return created, nil
}

// Update обновляет продукт (название, описание, цену, доступность)
// и возвращает обновлённую запись. Привязка к ресторану не меняется.
func (r *ProductRepository) Update(ctx context.Context, product model.Product) (model.Product, error) {
	const query = `
		UPDATE products
		SET name = $2, description = $3, price = $4, is_available = $5
		WHERE id = $1
		RETURNING id, restaurant_id, name, COALESCE(description, '') AS description, price, is_available
	`

	rows, err := r.db.Query(ctx, query,
		product.ID,
		product.Name,
		product.Description,
		product.Price,
		product.IsAvailable,
	)
	if err != nil {
		return model.Product{}, err
	}
	defer rows.Close()

	updated, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Product])
	if err != nil {
		return model.Product{}, err
	}

	return updated, nil
}

// Delete удаляет продукт по его ID.
func (r *ProductRepository) Delete(ctx context.Context, id int) error {
	const query = `DELETE FROM products WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	return err
}
