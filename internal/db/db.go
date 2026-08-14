package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool открывает пул соединений с Postgres и проверяет его пингом.
func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// RunMigrations выполняет SQL-файл миграции. Все выражения в нём написаны
// идемпотентно (IF NOT EXISTS / CREATE OR REPLACE), поэтому файл можно
// безопасно выполнять при каждом старте приложения.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, string(data))
	return err
}
