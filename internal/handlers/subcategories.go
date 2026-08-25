package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"solarmax/internal/models"
	"solarmax/internal/utils"

	"github.com/jackc/pgx/v5"
)

func scanSubcategory(row rowScanner) (models.Subcategory, error) {
	var s models.Subcategory
	var translations []byte
	err := row.Scan(&s.ID, &s.CategoryID, &s.Name, &s.Slug, &s.SortOrder, &s.Status, &translations, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return s, err
	}
	s.Language = map[string]models.SubcategoryTranslation{}
	if err := utils.FromJSONB(translations, &s.Language); err != nil {
		return s, err
	}
	return s, nil
}

func (h *Handler) ListSubcategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	categoryID := r.URL.Query().Get("category_id")
	language := r.URL.Query().Get("language")

	query := `SELECT id, category_id, name, slug, sort_order, status, translations, created_at, updated_at FROM subcategories`
	conds := []string{}
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		conds = append(conds, "status=$"+strconv.Itoa(len(args)))
	}
	if categoryID != "" {
		args = append(args, categoryID)
		conds = append(conds, "category_id=$"+strconv.Itoa(len(args)))
	}
	if language != "" {
		args = append(args, language)
		conds = append(conds, "translations ? $"+strconv.Itoa(len(args)))
	}
	if len(conds) > 0 {
		query += " WHERE " + join(conds, " AND ")
	}
	query += " ORDER BY sort_order ASC, id ASC"

	rows, err := h.Pool.Query(ctx, query, args...)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	list := []models.Subcategory{}
	for rows.Next() {
		s, err := scanSubcategory(rows)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		images, _ := h.getMedia(ctx, "subcategory", s.ID)
		s.Images = images
		list = append(list, s)
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func join(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

func (h *Handler) GetSubcategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		SELECT id, category_id, name, slug, sort_order, status, translations, created_at, updated_at
		FROM subcategories WHERE id=$1`, id)
	s, err := scanSubcategory(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "подкатегория не найдена")
		return
	}
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	images, _ := h.getMedia(ctx, "subcategory", s.ID)
	s.Images = images

	utils.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) CreateSubcategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in models.SubcategoryInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Name == "" || in.CategoryID == 0 {
		utils.WriteError(w, http.StatusBadRequest, "поля name и category_id обязательны")
		return
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Language == nil {
		in.Language = map[string]models.SubcategoryTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		INSERT INTO subcategories (category_id, name, slug, sort_order, status, translations)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		RETURNING id, category_id, name, slug, sort_order, status, translations, created_at, updated_at`,
		in.CategoryID, in.Name, in.Slug, in.SortOrder, in.Status, translationsJSON)

	s, err := scanSubcategory(row)
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "подкатегория с таким slug уже существует")
			return
		}
		if isFKViolation(err) {
			utils.WriteError(w, http.StatusBadRequest, "указанная категория не существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, s)
}

func (h *Handler) UpdateSubcategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	var in models.SubcategoryInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Name == "" || in.CategoryID == 0 {
		utils.WriteError(w, http.StatusBadRequest, "поля name и category_id обязательны")
		return
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Language == nil {
		in.Language = map[string]models.SubcategoryTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		UPDATE subcategories
		SET category_id=$1, name=$2, slug=$3, sort_order=$4, status=$5, translations=$6::jsonb
		WHERE id=$7
		RETURNING id, category_id, name, slug, sort_order, status, translations, created_at, updated_at`,
		in.CategoryID, in.Name, in.Slug, in.SortOrder, in.Status, translationsJSON, id)

	s, err := scanSubcategory(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "подкатегория не найдена")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "подкатегория с таким slug уже существует")
			return
		}
		if isFKViolation(err) {
			utils.WriteError(w, http.StatusBadRequest, "указанная категория не существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	images, _ := h.getMedia(ctx, "subcategory", s.ID)
	s.Images = images

	utils.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) DeleteSubcategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	// Правило 2: нельзя удалить подкатегорию, если есть товары.
	var count int
	if err := h.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE subcategory_id=$1`, id).Scan(&count); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if count > 0 {
		utils.WriteError(w, http.StatusConflict, "нельзя удалить подкатегорию: в ней есть товары")
		return
	}

	tag, err := h.Pool.Exec(ctx, `DELETE FROM subcategories WHERE id=$1`, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "подкатегория не найдена")
		return
	}

	_ = h.deleteMediaForEntity(ctx, "subcategory", id)

	w.WriteHeader(http.StatusNoContent)
}
