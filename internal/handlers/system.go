package handlers

import (
	"io"
	"net/http"
	"time"

	"solarmax/internal/utils"
)

var startedAt = time.Now()

// GET /health - используется хостингом и внешними пингерами.
// Тело намеренно оставлено простым ("ok"), как и раньше.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// GET /api/health - подробный статус: аптайм, хранилище, пингер.
func (h *Handler) HealthDetails(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":     "ok",
		"time":       time.Now().UTC(),
		"uptime":     time.Since(startedAt).Round(time.Second).String(),
		"storage":    h.Store.Stats(),
		"persistent": h.Store.Persistent(),
		"data_file":  h.Store.FilePath(),
	}
	if h.KeepAlive != nil {
		resp["keepalive"] = h.KeepAlive.Status()
	}
	utils.WriteJSON(w, http.StatusOK, resp)
}

// GET /api/keepalive - когда был последний фоновый пинг и когда будет следующий.
func (h *Handler) KeepAliveStatus(w http.ResponseWriter, r *http.Request) {
	if h.KeepAlive == nil {
		utils.WriteJSON(w, http.StatusOK, map[string]bool{"enabled": false})
		return
	}
	utils.WriteJSON(w, http.StatusOK, h.KeepAlive.Status())
}

// POST /api/keepalive/ping - ручной запуск пинга (для проверки настроек).
func (h *Handler) KeepAlivePing(w http.ResponseWriter, r *http.Request) {
	if h.KeepAlive == nil {
		utils.WriteError(w, http.StatusBadRequest, "keepalive выключен")
		return
	}
	utils.WriteJSON(w, http.StatusOK, h.KeepAlive.Run(r.Context()))
}

// GET /api/admin/backup - скачать полный дамп данных одним JSON-файлом.
func (h *Handler) Backup(w http.ResponseWriter, r *http.Request) {
	raw, err := h.Store.Snapshot()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось собрать дамп: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="solarmax-backup.json"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

// POST /api/admin/restore - залить дамп обратно (полностью заменяет данные).
func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "не удалось прочитать тело запроса")
		return
	}
	if len(raw) == 0 {
		utils.WriteError(w, http.StatusBadRequest, "пустое тело запроса: ожидается JSON-дамп")
		return
	}
	if err := h.Store.Restore(raw); err != nil {
		writeStoreErr(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "данные восстановлены",
		"storage": h.Store.Stats(),
	})
}
