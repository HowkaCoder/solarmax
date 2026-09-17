package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"solarmax/internal/config"
	"solarmax/internal/keepalive"
	"solarmax/internal/store"
	"solarmax/internal/utils"
)

type Handler struct {
	Store     *store.Store
	Cfg       *config.Config
	KeepAlive *keepalive.Pinger
}

// ==================== ХЕЛПЕРЫ ====================

// idParam достаёт числовой параметр из пути (/api/categories/{id}).
func idParam(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.PathValue(key), 10, 64)
}

// queryInt читает числовой query-параметр, пустой или кривой даёт 0.
func queryInt(r *http.Request, key string) int64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// decodeBody разбирает JSON-тело запроса.
func decodeBody(r *http.Request, dest interface{}) error {
	return json.NewDecoder(r.Body).Decode(dest)
}

// writeStoreErr переводит ошибку хранилища в HTTP-ответ.
// Прежние коды ответов (400/404/409) сохранены один в один.
func writeStoreErr(w http.ResponseWriter, err error) {
	utils.WriteError(w, store.HTTPStatus(err), err.Error())
}

// ==================== MIDDLEWARE ====================

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Logger пишет в лог метод, путь, код ответа и длительность.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, sw.status, time.Since(started).Round(time.Millisecond))
	})
}

// Recoverer не даёт панике в одном запросе уронить весь сервер.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("паника при обработке %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				utils.WriteError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS разрешает запросы с любого домена (фронтенд живёт отдельно)
// и сам отвечает на preflight-запросы OPTIONS.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "300")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
