package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"solarmax/internal/models"
	"solarmax/internal/utils"

	"github.com/jackc/pgx/v5"
)

func (h *Handler) ListCharacteristics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.Pool.Query(ctx, `SELECT id, name, created_at FROM characteristics ORDER BY name ASC`)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	list := []models.Characteristic{}
	for rows.Next() {
		var c models.Characteristic
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		values, err := h.loadCharacteristicValues(ctx, c.ID)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		c.Values = values
		list = append(list, c)
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) loadCharacteristicValues(ctx context.Context, characteristicID int64) ([]models.CharacteristicValue, error) {
	rows, err := h.Pool.Query(ctx, `
		SELECT id, characteristic_id, value, created_at
		FROM characteristic_values WHERE characteristic_id=$1 ORDER BY value ASC`, characteristicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.CharacteristicValue
	for rows.Next() {
		var v models.CharacteristicValue
		if err := rows.Scan(&v.ID, &v.CharacteristicID, &v.Value, &v.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func (h *Handler) CreateCharacteristic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in models.CharacteristicInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "поле name обязательно")
		return
	}

	var c models.Characteristic
	err := h.Pool.QueryRow(ctx, `
		INSERT INTO characteristics (name) VALUES ($1)
		RETURNING id, name, created_at`, in.Name).
		Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "характеристика с таким именем уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) UpdateCharacteristic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.CharacteristicInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "поле name обязательно")
		return
	}

	var c models.Characteristic
	err = h.Pool.QueryRow(ctx, `
		UPDATE characteristics SET name=$1 WHERE id=$2
		RETURNING id, name, created_at`, in.Name, id).
		Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "характеристика не найдена")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "характеристика с таким именем уже существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCharacteristic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	tag, err := h.Pool.Exec(ctx, `DELETE FROM characteristics WHERE id=$1`, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "характеристика не найдена")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListCharacteristicValues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	values, err := h.loadCharacteristicValues(ctx, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, values)
}

func (h *Handler) CreateCharacteristicValue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	charID, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.CharacteristicValueInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Value == "" {
		utils.WriteError(w, http.StatusBadRequest, "поле value обязательно")
		return
	}

	var v models.CharacteristicValue
	err = h.Pool.QueryRow(ctx, `
		INSERT INTO characteristic_values (characteristic_id, value) VALUES ($1, $2)
		RETURNING id, characteristic_id, value, created_at`, charID, in.Value).
		Scan(&v.ID, &v.CharacteristicID, &v.Value, &v.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "такое значение уже существует для этой характеристики")
			return
		}
		if isFKViolation(err) {
			utils.WriteError(w, http.StatusBadRequest, "указанная характеристика не существует")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, v)
}

func (h *Handler) UpdateCharacteristicValue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	valueID, err := idParam(r, "valueID")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.CharacteristicValueInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Value == "" {
		utils.WriteError(w, http.StatusBadRequest, "поле value обязательно")
		return
	}

	var v models.CharacteristicValue
	err = h.Pool.QueryRow(ctx, `
		UPDATE characteristic_values SET value=$1 WHERE id=$2
		RETURNING id, characteristic_id, value, created_at`, in.Value, valueID).
		Scan(&v.ID, &v.CharacteristicID, &v.Value, &v.CreatedAt)
	if err == pgx.ErrNoRows {
		utils.WriteError(w, http.StatusNotFound, "значение не найдено")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			utils.WriteError(w, http.StatusConflict, "такое значение уже существует для этой характеристики")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) DeleteCharacteristicValue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	valueID, err := idParam(r, "valueID")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	tag, err := h.Pool.Exec(ctx, `DELETE FROM characteristic_values WHERE id=$1`, valueID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "значение не найдено")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) loadProductCharacteristics(ctx context.Context, productID int64) ([]models.ProductCharacteristic, error) {
	rows, err := h.Pool.Query(ctx, `
		SELECT pc.id, pc.product_id, pc.characteristic_id, c.name, pc.value_id, cv.value
		FROM product_characteristics pc
		JOIN characteristics c ON c.id = pc.characteristic_id
		JOIN characteristic_values cv ON cv.id = pc.value_id
		WHERE pc.product_id = $1
		ORDER BY c.name ASC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.ProductCharacteristic
	for rows.Next() {
		var pc models.ProductCharacteristic
		if err := rows.Scan(&pc.ID, &pc.ProductID, &pc.CharacteristicID, &pc.CharacteristicName, &pc.ValueID, &pc.Value); err != nil {
			return nil, err
		}
		list = append(list, pc)
	}
	return list, rows.Err()
}

func (h *Handler) GetProductCharacteristics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	list, err := h.loadProductCharacteristics(ctx, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, list)
}
func (h *Handler) SetProductCharacteristic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	productID, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.ProductCharacteristicInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.CharacteristicID == 0 || in.ValueID == 0 {
		utils.WriteError(w, http.StatusBadRequest, "поля characteristic_id и value_id обязательны")
		return
	}

	_, err = h.Pool.Exec(ctx, `
		INSERT INTO product_characteristics (product_id, characteristic_id, value_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (product_id, characteristic_id)
		DO UPDATE SET value_id = EXCLUDED.value_id`,
		productID, in.CharacteristicID, in.ValueID)
	if err != nil {
		if isFKViolation(err) {
			utils.WriteError(w, http.StatusBadRequest, "указанный товар, характеристика или значение не существуют")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	list, err := h.loadProductCharacteristics(ctx, productID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) DeleteProductCharacteristic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	productID, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id товара")
		return
	}
	charID, err := idParam(r, "characteristicID")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id характеристики")
		return
	}

	tag, err := h.Pool.Exec(ctx, `DELETE FROM product_characteristics WHERE product_id=$1 AND characteristic_id=$2`, productID, charID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		utils.WriteError(w, http.StatusNotFound, "связь не найдена")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
