package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/slipynil/avito-kitchen/internal/model"
)

type RestaurantRepository struct {
	db *pgxpool.Pool
}

func NewRestaurantRepository(db *pgxpool.Pool) *RestaurantRepository {
	return &RestaurantRepository{db: db}
}

// restaurantColumns явный список колонок; COALESCE защищает от NULL
// в необязательных текстовых полях (description, phone).
const restaurantColumns = `
	id,
	name,
	COALESCE(description, '') AS description,
	COALESCE(address, '') AS address,
	COALESCE(phone, '') AS phone,
	is_active,
	created_at
`

// GetAll возвращает все активные рестораны.
func (r *RestaurantRepository) GetAll(ctx context.Context) ([]model.Restaurant, error) {
	query := `SELECT ` + restaurantColumns + ` FROM restaurants WHERE is_active = true ORDER BY id`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Restaurant])
}

// GetByID возвращает ресторан по его ID.
func (r *RestaurantRepository) GetByID(ctx context.Context, id int) (model.Restaurant, error) {
	query := `SELECT ` + restaurantColumns + ` FROM restaurants WHERE id = $1`

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return model.Restaurant{}, err
	}
	defer rows.Close()

	restaurant, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Restaurant])
	if err != nil {
		return model.Restaurant{}, err
	}

	return restaurant, nil
}

// GetByAPIKey возвращает ресторан по API ключу.
// Проверяет, что активен и сам ключ, и ресторан (JOIN restaurants).
// restaurant_id вызывающий код берёт из поля ID результата.
func (r *RestaurantRepository) GetByAPIKey(ctx context.Context, apiKey string) (model.Restaurant, error) {
	const query = `
		SELECT
			r.id,
			r.name,
			COALESCE(r.description, '') AS description,
			COALESCE(r.address, '') AS address,
			COALESCE(r.phone, '') AS phone,
			r.is_active,
			r.created_at
		FROM restaurants r
		JOIN restaurant_api_keys k ON k.restaurant_id = r.id
		WHERE k.api_key = $1
		  AND k.is_active = true
		  AND r.is_active = true
	`

	rows, err := r.db.Query(ctx, query, apiKey)
	if err != nil {
		return model.Restaurant{}, err
	}
	defer rows.Close()

	restaurant, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Restaurant])
	if err != nil {
		return model.Restaurant{}, err
	}

	return restaurant, nil
}
