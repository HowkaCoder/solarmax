package handlers

import (
	"encoding/json"
	"net/http"

	"solarmax/internal/models"
	"solarmax/internal/utils"

	"github.com/jackc/pgx/v5"
)

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")

	query := `SELECT id, name, description, slug, sort_order, status, created_at, updated_at FROM categories`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE status=$1`
		args = append(args, status)
	}
	query += ` ORDER BY sort_order ASC, id ASC`

	rows, err := h.Pool.Query(ctx, query, args...)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	list := []models.Category{}
	for rows.Next() {
		var c models.Category
		var desc *string
		if err := rows.Scan(&c.ID, &c.Name, &desc, &c.Slug, &c.SortOrder, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if desc != nil {
			c.Description = *desc
		}
		images, err := h.getMedia(ctx, "category", c.ID)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		c.Images = images
		list = append(list, c)
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	var c models.Category
	var desc *string
	err = h.Pool.QueryRow(ctx, `
		SELECT id, name, description, slug, sort_order, status, created_at, updated_at
		FROM categories WHERE id=$1`, id).
		Scan(&c.ID, &c.Name, &desc, &c.Slug, &c.SortOrder, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "категория не найдена")
		return
	}
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if desc != nil {
		c.Description = *desc
	}

	images, err := h.getMedia(ctx, "category", c.ID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	c.Images = images

	utils.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in models.CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "поле name обязательно")
		return
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Status == "" {
		in.Status = "active"
	}

	var c models.Category
	var desc *string
	err := h.Pool.QueryRow(ctx, `
		INSERT INTO categories (name, description, slug, sort_order, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, slug, sort_order, status, created_at, updated_at`,
		in.Name, nullable(in.Description), in.Slug, in.SortOrder, in.Status).
		Scan(&c.ID, &c.Name, &desc, &c.Slug, &c.SortOrder, &c.Status, &c.CreatedAt, &c.UpdatedAt)

	if desc != nil {
		c.Description = *desc
	}

	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "категория с таким slug уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	var in models.CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "поле name обязательно")
		return
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Status == "" {
		in.Status = "active"
	}

	var c models.Category
	var desc *string
	err = h.Pool.QueryRow(ctx, `
		UPDATE categories
		SET name=$1, description=$2, slug=$3, sort_order=$4, status=$5
		WHERE id=$6
		RETURNING id, name, description, slug, sort_order, status, created_at, updated_at`,
		in.Name, nullable(in.Description), in.Slug, in.SortOrder, in.Status, id).
		Scan(&c.ID, &c.Name, &desc, &c.Slug, &c.SortOrder, &c.Status, &c.CreatedAt, &c.UpdatedAt)

	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "категория не найдена")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "категория с таким slug уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if desc != nil {
		c.Description = *desc
	}

	images, _ := h.getMedia(ctx, "category", c.ID)
	c.Images = images

	utils.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	var count int
	if err := h.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM subcategories WHERE category_id=$1`, id).Scan(&count); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if count > 0 {
		utils.WriteError(w, http.StatusConflict, "нельзя удалить категорию: у неё есть подкатегории")
		return
	}

	tag, err := h.Pool.Exec(ctx, `DELETE FROM categories WHERE id=$1`, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "категория не найдена")
		return
	}

	_ = h.deleteMediaForEntity(ctx, "category", id)

	w.WriteHeader(http.StatusNoContent)
}
