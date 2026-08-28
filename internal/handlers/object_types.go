package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"solarmax/internal/models"
	"solarmax/internal/utils"

	"github.com/jackc/pgx/v5"
)

func scanObjectType(row rowScanner) (models.ObjectType, error) {
	var o models.ObjectType
	var translations []byte
	err := row.Scan(&o.ID, &o.Name, &o.Slug, &o.SortOrder, &o.Status, &translations, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return o, err
	}
	o.Language = map[string]models.ObjectTypeTranslation{}
	if err := utils.FromJSONB(translations, &o.Language); err != nil {
		return o, err
	}
	return o, nil
}

func (h *Handler) ListObjectTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	language := r.URL.Query().Get("language")

	query := `SELECT id, name, slug, sort_order, status, translations, created_at, updated_at FROM object_types`
	conds := []string{}
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		conds = append(conds, "status=$"+strconv.Itoa(len(args)))
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

	list := []models.ObjectType{}
	for rows.Next() {
		o, err := scanObjectType(rows)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		images, _ := h.getMedia(ctx, "object_type", o.ID)
		o.Images = images
		list = append(list, o)
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) GetObjectType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	row := h.Pool.QueryRow(ctx, `
		SELECT id, name, slug, sort_order, status, translations, created_at, updated_at
		FROM object_types WHERE id=$1`, id)
	o, err := scanObjectType(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "объект не найден")
		return
	}
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	images, _ := h.getMedia(ctx, "object_type", o.ID)
	o.Images = images

	utils.WriteJSON(w, http.StatusOK, o)
}

func (h *Handler) CreateObjectType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in models.ObjectTypeInput
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
	if in.Language == nil {
		in.Language = map[string]models.ObjectTypeTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		INSERT INTO object_types (name, slug, sort_order, status, translations)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		RETURNING id, name, slug, sort_order, status, translations, created_at, updated_at`,
		in.Name, in.Slug, in.SortOrder, in.Status, translationsJSON)

	o, err := scanObjectType(row)
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "объект с таким slug уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, o)
}

func (h *Handler) UpdateObjectType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.ObjectTypeInput
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
	if in.Language == nil {
		in.Language = map[string]models.ObjectTypeTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		UPDATE object_types
		SET name=$1, slug=$2, sort_order=$3, status=$4, translations=$5::jsonb
		WHERE id=$6
		RETURNING id, name, slug, sort_order, status, translations, created_at, updated_at`,
		in.Name, in.Slug, in.SortOrder, in.Status, translationsJSON, id)

	o, err := scanObjectType(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "объект не найден")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "объект с таким slug уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	images, _ := h.getMedia(ctx, "object_type", o.ID)
	o.Images = images

	utils.WriteJSON(w, http.StatusOK, o)
}

// UpdateObjectTypeStatus позволяет скрыть/показать карточку без удаления.
func (h *Handler) UpdateObjectTypeStatus(w http.ResponseWriter, r *http.Request) {
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
	tag, err := h.Pool.Exec(ctx, `UPDATE object_types SET status=$1 WHERE id=$2`, in.Status, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "объект не найден")
		return
	}
	utils.WriteJSON(w, http.StatusOK, models.StatusInput{Status: in.Status})
}

func (h *Handler) DeleteObjectType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	tag, err := h.Pool.Exec(ctx, `DELETE FROM object_types WHERE id=$1`, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "объект не найден")
		return
	}
	_ = h.deleteMediaForEntity(ctx, "object_type", id)
	w.WriteHeader(http.StatusNoContent)
}
