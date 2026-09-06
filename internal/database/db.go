package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Подключение к БД
func Connect(ctx context.Context, connString string) (*pgx.Conn, error) {

	if len(connString) == 0 {
		return nil, fmt.Errorf("connString is empty")
	}

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}
	return conn, nil
}
