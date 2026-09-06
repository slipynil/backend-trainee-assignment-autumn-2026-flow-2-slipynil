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

// GetAll возвращает список всех доступных продуктов
func (r *ProductRepository) GetAll(ctx context.Context) ([]model.Product, error) {
	sqlStr := `
	SELECT * FROM products
	`
	rows, err := r.db.Query(ctx, sqlStr)
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
