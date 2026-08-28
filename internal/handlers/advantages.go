package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"solarmax/internal/models"
	"solarmax/internal/utils"

	"github.com/jackc/pgx/v5"
)

func scanAdvantage(row rowScanner) (models.Advantage, error) {
	var a models.Advantage
	var translations []byte
	err := row.Scan(&a.ID, &a.Name, &a.Slug, &a.SortOrder, &a.Status, &translations, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, err
	}
	a.Language = map[string]models.AdvantageTranslation{}
	if err := utils.FromJSONB(translations, &a.Language); err != nil {
		return a, err
	}
	return a, nil
}

func (h *Handler) ListAdvantages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	language := r.URL.Query().Get("language")

	query := `SELECT id, name, slug, sort_order, status, translations, created_at, updated_at FROM advantages`
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

	list := []models.Advantage{}
	for rows.Next() {
		a, err := scanAdvantage(rows)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		images, _ := h.getMedia(ctx, "advantage", a.ID)
		a.Images = images
		list = append(list, a)
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) GetAdvantage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	row := h.Pool.QueryRow(ctx, `
		SELECT id, name, slug, sort_order, status, translations, created_at, updated_at
		FROM advantages WHERE id=$1`, id)
	a, err := scanAdvantage(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "преимущество не найдено")
		return
	}
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	images, _ := h.getMedia(ctx, "advantage", a.ID)
	a.Images = images

	utils.WriteJSON(w, http.StatusOK, a)
}

func (h *Handler) CreateAdvantage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in models.AdvantageInput
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
		in.Language = map[string]models.AdvantageTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		INSERT INTO advantages (name, slug, sort_order, status, translations)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		RETURNING id, name, slug, sort_order, status, translations, created_at, updated_at`,
		in.Name, in.Slug, in.SortOrder, in.Status, translationsJSON)

	a, err := scanAdvantage(row)
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "преимущество с таким slug уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, a)
}

func (h *Handler) UpdateAdvantage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.AdvantageInput
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
		in.Language = map[string]models.AdvantageTranslation{}
	}

	translationsJSON, err := utils.ToJSONB(in.Language)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось сериализовать переводы")
		return
	}

	row := h.Pool.QueryRow(ctx, `
		UPDATE advantages
		SET name=$1, slug=$2, sort_order=$3, status=$4, translations=$5::jsonb
		WHERE id=$6
		RETURNING id, name, slug, sort_order, status, translations, created_at, updated_at`,
		in.Name, in.Slug, in.SortOrder, in.Status, translationsJSON, id)

	a, err := scanAdvantage(row)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "преимущество не найдено")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "преимущество с таким slug уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	images, _ := h.getMedia(ctx, "advantage", a.ID)
	a.Images = images

	utils.WriteJSON(w, http.StatusOK, a)
}

// UpdateAdvantageStatus позволяет скрыть/показать карточку без удаления.
func (h *Handler) UpdateAdvantageStatus(w http.ResponseWriter, r *http.Request) {
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
	tag, err := h.Pool.Exec(ctx, `UPDATE advantages SET status=$1 WHERE id=$2`, in.Status, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "преимущество не найдено")
		return
	}
	utils.WriteJSON(w, http.StatusOK, models.StatusInput{Status: in.Status})
}

func (h *Handler) DeleteAdvantage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	tag, err := h.Pool.Exec(ctx, `DELETE FROM advantages WHERE id=$1`, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "преимущество не найдено")
		return
	}
	_ = h.deleteMediaForEntity(ctx, "advantage", id)
	w.WriteHeader(http.StatusNoContent)
}
