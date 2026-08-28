package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"solarmax/internal/models"
	"solarmax/internal/utils"

	"github.com/google/uuid"
)

var allowedEntityTypes = map[string]bool{
	"category":    true,
	"subcategory": true,
	"product":     true,
	"service":     true,
	"object_type": true,
	"advantage":   true,
}

const maxUploadSize = 25 << 20

func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "не удалось разобрать форму (файл слишком большой?)")
		return
	}

	entityType := r.FormValue("entity_type")
	entityIDStr := r.FormValue("entity_id")
	isMain := r.FormValue("is_main") == "true"

	if !allowedEntityTypes[entityType] {
		utils.WriteError(w, http.StatusBadRequest, "entity_type должен быть одним из: category, subcategory, product, service, object_type, advantage")
		return
	}
	entityID, err := strconv.ParseInt(entityIDStr, 10, 64)
	if err != nil || entityID <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "некорректный entity_id")
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		utils.WriteError(w, http.StatusBadRequest, "не переданы файлы (поле 'files')")
		return
	}

	destDir := filepath.Join(h.Cfg.MediaDir, entityType, strconv.FormatInt(entityID, 10))
	if err := os.MkdirAll(destDir, 0755); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось создать директорию для файлов")
		return
	}

	saved := []models.Media{}
	for i, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !isAllowedImageExt(ext) {
			utils.WriteError(w, http.StatusBadRequest, fmt.Sprintf("недопустимый тип файла: %s (разрешены jpg, jpeg, png, webp, gif)", fh.Filename))
			return
		}

		src, err := fh.Open()
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "не удалось открыть загруженный файл")
			return
		}

		fileName := uuid.NewString() + ext
		dstPath := filepath.Join(destDir, fileName)
		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			utils.WriteError(w, http.StatusInternalServerError, "не удалось сохранить файл")
			return
		}
		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			utils.WriteError(w, http.StatusInternalServerError, "не удалось записать файл на диск")
			return
		}

		publicURL := fmt.Sprintf("/media/%s/%d/%s", entityType, entityID, fileName)
		main := isMain && i == 0

		var m models.Media
		err = h.Pool.QueryRow(ctx, `
			INSERT INTO media (entity_type, entity_id, url, is_main, sort_order)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, entity_type, entity_id, url, is_main, sort_order, created_at`,
			entityType, entityID, publicURL, main, i).
			Scan(&m.ID, &m.EntityType, &m.EntityID, &m.URL, &m.IsMain, &m.SortOrder, &m.CreatedAt)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if main {
			_, _ = h.Pool.Exec(ctx, `UPDATE media SET is_main=false WHERE entity_type=$1 AND entity_id=$2 AND id<>$3`,
				entityType, entityID, m.ID)
		}

		saved = append(saved, m)
	}

	utils.WriteJSON(w, http.StatusCreated, saved)
}

func isAllowedImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return true
	}
	return false
}

func (h *Handler) ListMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	entityType := r.URL.Query().Get("entity_type")
	entityIDStr := r.URL.Query().Get("entity_id")
	if entityType == "" || entityIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, "нужны параметры entity_type и entity_id")
		return
	}
	entityID, err := strconv.ParseInt(entityIDStr, 10, 64)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный entity_id")
		return
	}
	list, err := h.getMedia(ctx, entityType, entityID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []models.Media{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	var relURL string
	err = h.Pool.QueryRow(ctx, `DELETE FROM media WHERE id=$1 RETURNING url`, id).Scan(&relURL)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "медиафайл не найден")
		return
	}

	if strings.HasPrefix(relURL, "/media/") {
		diskPath := filepath.Join(h.Cfg.MediaDir, strings.TrimPrefix(relURL, "/media/"))
		_ = os.Remove(diskPath)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetMainMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	var entityType string
	var entityID int64
	err = h.Pool.QueryRow(ctx, `SELECT entity_type, entity_id FROM media WHERE id=$1`, id).Scan(&entityType, &entityID)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "медиафайл не найден")
		return
	}

	_, err = h.Pool.Exec(ctx, `UPDATE media SET is_main=false WHERE entity_type=$1 AND entity_id=$2`, entityType, entityID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, err = h.Pool.Exec(ctx, `UPDATE media SET is_main=true WHERE id=$1`, id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	list, _ := h.getMedia(ctx, entityType, entityID)
	utils.WriteJSON(w, http.StatusOK, list)
}
