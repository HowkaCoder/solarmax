package handlers

import (
	"net/http"

	"solarmax/internal/models"
	"solarmax/internal/utils"
)

// ==================== УСЛУГИ ====================

func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	utils.WriteJSON(w, http.StatusOK, h.Store.ListServices(q.Get("status"), q.Get("language")))
}

func (h *Handler) GetService(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	s, err := h.Store.GetService(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) CreateService(w http.ResponseWriter, r *http.Request) {
	var in models.ServiceInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	s, err := h.Store.CreateService(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, s)
}

func (h *Handler) UpdateService(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.ServiceInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	s, err := h.Store.UpdateService(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) UpdateServiceStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.StatusInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if err := h.Store.SetServiceStatus(id, in.Status); err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, models.StatusInput{Status: in.Status})
}

func (h *Handler) DeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	if err := h.Store.DeleteService(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	h.removeEntityFiles("service", id)
	w.WriteHeader(http.StatusNoContent)
}

// ==================== ТИПЫ ОБЪЕКТОВ ====================

func (h *Handler) ListObjectTypes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	utils.WriteJSON(w, http.StatusOK, h.Store.ListObjectTypes(q.Get("status"), q.Get("language")))
}

func (h *Handler) GetObjectType(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	o, err := h.Store.GetObjectType(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, o)
}

func (h *Handler) CreateObjectType(w http.ResponseWriter, r *http.Request) {
	var in models.ObjectTypeInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	o, err := h.Store.CreateObjectType(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, o)
}

func (h *Handler) UpdateObjectType(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.ObjectTypeInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	o, err := h.Store.UpdateObjectType(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, o)
}

func (h *Handler) UpdateObjectTypeStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.StatusInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if err := h.Store.SetObjectTypeStatus(id, in.Status); err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, models.StatusInput{Status: in.Status})
}

func (h *Handler) DeleteObjectType(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	if err := h.Store.DeleteObjectType(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	h.removeEntityFiles("object_type", id)
	w.WriteHeader(http.StatusNoContent)
}

// ==================== ПРЕИМУЩЕСТВА ====================

func (h *Handler) ListAdvantages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	utils.WriteJSON(w, http.StatusOK, h.Store.ListAdvantages(q.Get("status"), q.Get("language")))
}

func (h *Handler) GetAdvantage(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	a, err := h.Store.GetAdvantage(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, a)
}

func (h *Handler) CreateAdvantage(w http.ResponseWriter, r *http.Request) {
	var in models.AdvantageInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	a, err := h.Store.CreateAdvantage(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, a)
}

func (h *Handler) UpdateAdvantage(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.AdvantageInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	a, err := h.Store.UpdateAdvantage(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, a)
}

func (h *Handler) UpdateAdvantageStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.StatusInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if err := h.Store.SetAdvantageStatus(id, in.Status); err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, models.StatusInput{Status: in.Status})
}

func (h *Handler) DeleteAdvantage(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	if err := h.Store.DeleteAdvantage(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	h.removeEntityFiles("advantage", id)
	w.WriteHeader(http.StatusNoContent)
}
