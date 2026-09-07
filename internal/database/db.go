package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect создаёт пул соединений к PostgreSQL.
// Пул потокобезопасен и предназначен для использования из многих горутины.
func Connect(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	if len(connString) == 0 {
		return nil, fmt.Errorf("connString is empty")
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
