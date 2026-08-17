package authutil

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedDefaultAdmin создаёт админа admin/admin, если в таблице admins ещё
// вообще никого нет. Если хотя бы один админ уже существует (например,
// пароль уже сменили) - ничего не делает. Безопасно вызывать при каждом
// старте приложения.
func SeedDefaultAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := HashPassword("admin")
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `INSERT INTO admins (username, password_hash) VALUES ($1, $2)`, "admin", hash)
	return err
}
