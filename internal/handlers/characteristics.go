package handlers

import (
	"net/http"

	"solarmax/internal/models"
	"solarmax/internal/utils"
)

func (h *Handler) ListCharacteristics(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, h.Store.ListCharacteristics())
}

func (h *Handler) CreateCharacteristic(w http.ResponseWriter, r *http.Request) {
	var in models.CharacteristicInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	c, err := h.Store.CreateCharacteristic(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) UpdateCharacteristic(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.CharacteristicInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	c, err := h.Store.UpdateCharacteristic(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCharacteristic(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	if err := h.Store.DeleteCharacteristic(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListCharacteristicValues(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	utils.WriteJSON(w, http.StatusOK, h.Store.ListCharacteristicValues(id))
}

func (h *Handler) CreateCharacteristicValue(w http.ResponseWriter, r *http.Request) {
	charID, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.CharacteristicValueInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	v, err := h.Store.CreateCharacteristicValue(charID, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, v)
}

func (h *Handler) UpdateCharacteristicValue(w http.ResponseWriter, r *http.Request) {
	valueID, err := idParam(r, "valueID")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.CharacteristicValueInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	v, err := h.Store.UpdateCharacteristicValue(valueID, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) DeleteCharacteristicValue(w http.ResponseWriter, r *http.Request) {
	valueID, err := idParam(r, "valueID")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	if err := h.Store.DeleteCharacteristicValue(valueID); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ==================== ХАРАКТЕРИСТИКИ ТОВАРА ====================

func (h *Handler) GetProductCharacteristics(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	utils.WriteJSON(w, http.StatusOK, h.Store.GetProductCharacteristics(id))
}

func (h *Handler) SetProductCharacteristic(w http.ResponseWriter, r *http.Request) {
	productID, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.ProductCharacteristicInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	list, err := h.Store.SetProductCharacteristic(productID, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) DeleteProductCharacteristic(w http.ResponseWriter, r *http.Request) {
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
	if err := h.Store.DeleteProductCharacteristic(productID, charID); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
