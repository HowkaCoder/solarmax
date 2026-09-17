package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"solarmax/internal/models"
	"solarmax/internal/store"
	"solarmax/internal/utils"
)

const maxUploadSize = 25 << 20 // 25 МБ

// POST /api/media/upload  (multipart/form-data: entity_type, entity_id, is_main, files[])
func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "не удалось разобрать форму (файл слишком большой?)")
		return
	}

	entityType := r.FormValue("entity_type")
	isMain := r.FormValue("is_main") == "true"

	if !store.IsKnownEntityType(entityType) {
		utils.WriteError(w, http.StatusBadRequest, "entity_type должен быть одним из: category, subcategory, product, service, object_type, advantage")
		return
	}
	entityID, err := strconv.ParseInt(r.FormValue("entity_id"), 10, 64)
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
	if err := os.MkdirAll(destDir, 0o755); err != nil {
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

		fileName := randomName() + ext
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
		m, err := h.Store.AddMedia(entityType, entityID, publicURL, isMain && i == 0, i)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		saved = append(saved, m)
	}

	utils.WriteJSON(w, http.StatusCreated, saved)
}

// GET /api/media?entity_type=product&entity_id=1
func (h *Handler) ListMedia(w http.ResponseWriter, r *http.Request) {
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
	utils.WriteJSON(w, http.StatusOK, h.Store.ListMedia(entityType, entityID))
}

func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}

	relURL, err := h.Store.DeleteMedia(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if strings.HasPrefix(relURL, "/media/") {
		diskPath := filepath.Join(h.Cfg.MediaDir, filepath.Clean("/"+strings.TrimPrefix(relURL, "/media/")))
		_ = os.Remove(diskPath)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetMainMedia(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	list, err := h.Store.SetMainMedia(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

// removeEntityFiles подчищает папку с картинками удалённой сущности.
func (h *Handler) removeEntityFiles(entityType string, entityID int64) {
	dir := filepath.Join(h.Cfg.MediaDir, entityType, strconv.FormatInt(entityID, 10))
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("не удалось удалить папку %s: %v", dir, err)
	}
}

func isAllowedImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return true
	}
	return false
}

// randomName генерирует случайное имя файла (раньше это делал uuid).
func randomName() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(int64(os.Getpid()), 16)
	}
	return hex.EncodeToString(b)
}
