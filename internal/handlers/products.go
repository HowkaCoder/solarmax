package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"solarmax/internal/models"
	"solarmax/internal/utils"

	"github.com/jackc/pgx/v5"
)

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanProduct(row rowScanner) (models.Product, error) {
	var p models.Product
	var translations []byte
	err := row.Scan(&p.ID, &p.SubcategoryID, &p.Name, &p.Price, &p.Slug, &p.Status, &translations, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return p, err
	}
	p.Language = map[string]models.ProductTranslation{}
	if err := utils.FromJSONB(translations, &p.Language); err != nil {
		return p, err
	}
	return p, nil
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	subcategoryID := r.URL.Query().Get("subcategory_id")
	language := r.URL.Query().Get("language")

	query := `SELECT id, subcategory_id, name, price, slug, status, translations, created_at, updated_at FROM products`
	conds := []string{}
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		conds = append(conds, "status=$"+strconv.Itoa(len(args)))
	}
	if subcategoryID != "" {
		args = append(args, subcategoryID)
		conds = append(conds, "subcategory_id=$"+strconv.Itoa(len(args)))
	}
	if language != "" {
		args = append(args, language)
		conds = append(conds, "translations ? $"+strconv.Itoa(len(args)))
	}
	if len(conds) > 0 {
		query += " WHERE " + join(conds, " AND ")
	}
	query += " ORDER BY id DESC"

	rows, err := h.Pool.Query(ctx, query, args...)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	list := []models.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		images, _ := h.getMedia(ctx, "product", p.ID)
		p.Images = images
		list = append(list, p)
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		SELECT id, subcategory_id, name, price, slug, status, translations, created_at, updated_at
		FROM products WHERE id=$1`, id)
	p, err := scanProduct(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "товар не найден")
		return
	}
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	images, _ := h.getMedia(ctx, "product", p.ID)
	p.Images = images

	chars, err := h.loadProductCharacteristics(ctx, p.ID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	p.Characteristics = chars

	utils.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in models.ProductInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Name == "" || in.SubcategoryID == 0 {
		utils.WriteError(w, http.StatusBadRequest, "поля name и subcategory_id обязательны")
		return
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Language == nil {
		in.Language = map[string]models.ProductTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		INSERT INTO products (subcategory_id, name, price, slug, status, translations)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		RETURNING id, subcategory_id, name, price, slug, status, translations, created_at, updated_at`,
		in.SubcategoryID, in.Name, in.Price, in.Slug, in.Status, translationsJSON)

	p, err := scanProduct(row)
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "товар с таким slug уже существует")
			return
		}
		if isFKViolation(err) {
			utils.WriteError(w, http.StatusBadRequest, "указанная подкатегория не существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	var in models.ProductInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Name == "" || in.SubcategoryID == 0 {
		utils.WriteError(w, http.StatusBadRequest, "поля name и subcategory_id обязательны")
		return
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Language == nil {
		in.Language = map[string]models.ProductTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		UPDATE products
		SET subcategory_id=$1, name=$2, price=$3, slug=$4, status=$5, translations=$6::jsonb
		WHERE id=$7
		RETURNING id, subcategory_id, name, price, slug, status, translations, created_at, updated_at`,
		in.SubcategoryID, in.Name, in.Price, in.Slug, in.Status, translationsJSON, id)

	p, err := scanProduct(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "товар не найден")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "товар с таким slug уже существует")
			return
		}
		if isFKViolation(err) {
			utils.WriteError(w, http.StatusBadRequest, "указанная подкатегория не существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	images, _ := h.getMedia(ctx, "product", p.ID)
	p.Images = images
	chars, _ := h.loadProductCharacteristics(ctx, p.ID)
	p.Characteristics = chars

	utils.WriteJSON(w, http.StatusOK, p)
}

// UpdateProductStatus - Правило 3: товар можно скрыть/показать без удаления.
// PATCH /api/products/{id}/status  {"status":"inactive"}
func (h *Handler) UpdateProductStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	var in models.StatusInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Status != "active" && in.Status != "inactive" {
		utils.WriteError(w, http.StatusBadRequest, "status должен быть active или inactive")
		return
	}

	tag, err := h.Pool.Exec(ctx, `UPDATE products SET status=$1 WHERE id=$2`, in.Status, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "товар не найден")
		return
	}

	utils.WriteJSON(w, http.StatusOK, models.StatusInput{Status: in.Status})
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	tag, err := h.Pool.Exec(ctx, `DELETE FROM products WHERE id=$1`, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "товар не найден")
		return
	}

	_ = h.deleteMediaForEntity(ctx, "product", id)

	w.WriteHeader(http.StatusNoContent)
}
