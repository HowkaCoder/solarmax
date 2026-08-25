package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"solarmax/internal/models"
	"solarmax/internal/utils"

	"github.com/jackc/pgx/v5"
)

func scanService(row rowScanner) (models.Service, error) {
	var s models.Service
	var translations []byte
	err := row.Scan(&s.ID, &s.Name, &s.Slug, &s.Status, &translations, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return s, err
	}
	s.Language = map[string]models.ServiceTranslation{}
	if err := utils.FromJSONB(translations, &s.Language); err != nil {
		return s, err
	}
	return s, nil
}

func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	language := r.URL.Query().Get("language")

	query := `SELECT id, name, slug, status, translations, created_at, updated_at FROM services`
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
	query += " ORDER BY id DESC"

	rows, err := h.Pool.Query(ctx, query, args...)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	list := []models.Service{}
	for rows.Next() {
		s, err := scanService(rows)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		images, _ := h.getMedia(ctx, "service", s.ID)
		s.Images = images
		list = append(list, s)
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) GetService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	row := h.Pool.QueryRow(ctx, `
		SELECT id, name, slug, status, translations, created_at, updated_at
		FROM services WHERE id=$1`, id)
	s, err := scanService(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "услуга не найдена")
		return
	}
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	images, _ := h.getMedia(ctx, "service", s.ID)
	s.Images = images

	utils.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) CreateService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in models.ServiceInput
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
		in.Language = map[string]models.ServiceTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		INSERT INTO services (name, slug, status, translations)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id, name, slug, status, translations, created_at, updated_at`,
		in.Name, in.Slug, in.Status, translationsJSON)

	s, err := scanService(row)
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "услуга с таким slug уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, s)
}

func (h *Handler) UpdateService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.ServiceInput
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
		in.Language = map[string]models.ServiceTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		UPDATE services
		SET name=$1, slug=$2, status=$3, translations=$4::jsonb
		WHERE id=$5
		RETURNING id, name, slug, status, translations, created_at, updated_at`,
		in.Name, in.Slug, in.Status, translationsJSON, id)

	s, err := scanService(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "услуга не найдена")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "услуга с таким slug уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	images, _ := h.getMedia(ctx, "service", s.ID)
	s.Images = images

	utils.WriteJSON(w, http.StatusOK, s)
}

// UpdateServiceStatus позволяет скрыть/показать услугу без удаления (по аналогии с товаром).
func (h *Handler) UpdateServiceStatus(w http.ResponseWriter, r *http.Request) {
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

	tag, err := h.Pool.Exec(ctx, `UPDATE services SET status=$1 WHERE id=$2`, in.Status, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "услуга не найдена")
		return
	}
	utils.WriteJSON(w, http.StatusOK, models.StatusInput{Status: in.Status})
}

func (h *Handler) DeleteService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	tag, err := h.Pool.Exec(ctx, `DELETE FROM services WHERE id=$1`, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "услуга не найдена")
		return
	}
	_ = h.deleteMediaForEntity(ctx, "service", id)
	w.WriteHeader(http.StatusNoContent)
}
