package handlers

import (
	"net/http"

	"solarmax/internal/models"
	"solarmax/internal/utils"
)

// ==================== КАТЕГОРИИ ====================

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	utils.WriteJSON(w, http.StatusOK, h.Store.ListCategories(q.Get("status"), q.Get("language")))
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	c, err := h.Store.GetCategory(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var in models.CategoryInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	c, err := h.Store.CreateCategory(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.CategoryInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	c, err := h.Store.UpdateCategory(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	if err := h.Store.DeleteCategory(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	h.removeEntityFiles("category", id)
	w.WriteHeader(http.StatusNoContent)
}

// ==================== ПОДКАТЕГОРИИ ====================

func (h *Handler) ListSubcategories(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Store.ListSubcategories(q.Get("status"), q.Get("language"), queryInt(r, "category_id"))
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) GetSubcategory(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	s, err := h.Store.GetSubcategory(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) CreateSubcategory(w http.ResponseWriter, r *http.Request) {
	var in models.SubcategoryInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	s, err := h.Store.CreateSubcategory(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, s)
}

func (h *Handler) UpdateSubcategory(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.SubcategoryInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	s, err := h.Store.UpdateSubcategory(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) DeleteSubcategory(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	if err := h.Store.DeleteSubcategory(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	h.removeEntityFiles("subcategory", id)
	w.WriteHeader(http.StatusNoContent)
}

// ==================== ТОВАРЫ ====================

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Store.ListProducts(q.Get("status"), q.Get("language"), queryInt(r, "subcategory_id"))
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	p, err := h.Store.GetProduct(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var in models.ProductInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	p, err := h.Store.CreateProduct(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var in models.ProductInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	p, err := h.Store.UpdateProduct(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, p)
}

// PATCH /api/products/{id}/status  {"status":"inactive"}
func (h *Handler) UpdateProductStatus(w http.ResponseWriter, r *http.Request) {
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
	if err := h.Store.SetProductStatus(id, in.Status); err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, models.StatusInput{Status: in.Status})
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	if err := h.Store.DeleteProduct(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	h.removeEntityFiles("product", id)
	w.WriteHeader(http.StatusNoContent)
}
