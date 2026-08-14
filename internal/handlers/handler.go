package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"solarmax/internal/config"
	"solarmax/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	Pool *pgxpool.Pool
	Cfg  *config.Config
}

// isUniqueViolation проверяет, что ошибка postgres - нарушение уникального ограничения (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// isFKViolation проверяет нарушение внешнего ключа (23503) - например,
// при создании подкатегории со ссылкой на несуществующую категорию.
func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}

// nullable превращает пустую строку в nil, чтобы писать NULL в БД вместо "".
func nullable(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func idParam(r *http.Request, key string) (int64, error) {
	v := chi.URLParam(r, key)
	return strconv.ParseInt(v, 10, 64)
}

// getMedia возвращает все изображения сущности, отсортированные так,
// что "главное" фото (is_main) идёт первым.
func (h *Handler) getMedia(ctx context.Context, entityType string, entityID int64) ([]models.Media, error) {
	rows, err := h.Pool.Query(ctx, `
		SELECT id, entity_type, entity_id, url, is_main, sort_order, created_at
		FROM media
		WHERE entity_type=$1 AND entity_id=$2
		ORDER BY is_main DESC, sort_order ASC, id ASC`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Media
	for rows.Next() {
		var m models.Media
		if err := rows.Scan(&m.ID, &m.EntityType, &m.EntityID, &m.URL, &m.IsMain, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// deleteMediaForEntity удаляет все записи медиа для сущности (используется при удалении сущности).
// Сами файлы на диске не удаляются намеренно - подчищать их можно отдельной задачей,
// это исключает риск потерять файл при ошибке в середине транзакции.
func (h *Handler) deleteMediaForEntity(ctx context.Context, entityType string, entityID int64) error {
	_, err := h.Pool.Exec(ctx, `DELETE FROM media WHERE entity_type=$1 AND entity_id=$2`, entityType, entityID)
	return err
}
