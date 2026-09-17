package handlers

import (
	"net/http"

	"solarmax/internal/config"
	"solarmax/internal/keepalive"
	"solarmax/internal/store"
	"solarmax/internal/utils"
)

// NewRouter собирает все маршруты на стандартном http.ServeMux (Go 1.22+),
// поэтому сторонний роутер (chi) больше не нужен. Пути и коды ответов
// полностью совпадают с прежней версией API.
func NewRouter(st *store.Store, cfg *config.Config, ka *keepalive.Pinger) http.Handler {
	h := &Handler{Store: st, Cfg: cfg, KeepAlive: ka}
	mux := http.NewServeMux()

	// ---- Служебное ----
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/health", h.HealthDetails)
	mux.HandleFunc("GET /api/keepalive", h.KeepAliveStatus)
	mux.HandleFunc("POST /api/keepalive/ping", h.RequireAuth(h.KeepAlivePing))

	// ---- Резервная копия данных ----
	mux.HandleFunc("GET /api/admin/backup", h.RequireAuth(h.Backup))
	mux.HandleFunc("POST /api/admin/restore", h.RequireAuth(h.Restore))

	// ---- Статика: /media/<entity_type>/<id>/<file> ----
	fileServer := http.FileServer(http.Dir(cfg.MediaDir))
	mux.Handle("GET /media/", http.StripPrefix("/media/", fileServer))

	// ---- Авторизация ----
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/auth/me", h.RequireAuth(h.Me))
	mux.HandleFunc("POST /api/auth/change-password", h.RequireAuth(h.ChangePassword))

	// ---- Категории ---- (GET открыт всем, изменения - только с токеном)
	collection(mux, "GET", "/api/categories", h.ListCategories)
	collection(mux, "POST", "/api/categories", h.RequireAuth(h.CreateCategory))
	mux.HandleFunc("GET /api/categories/{id}", h.GetCategory)
	mux.HandleFunc("PUT /api/categories/{id}", h.RequireAuth(h.UpdateCategory))
	mux.HandleFunc("DELETE /api/categories/{id}", h.RequireAuth(h.DeleteCategory))

	// ---- Подкатегории ----
	collection(mux, "GET", "/api/subcategories", h.ListSubcategories)
	collection(mux, "POST", "/api/subcategories", h.RequireAuth(h.CreateSubcategory))
	mux.HandleFunc("GET /api/subcategories/{id}", h.GetSubcategory)
	mux.HandleFunc("PUT /api/subcategories/{id}", h.RequireAuth(h.UpdateSubcategory))
	mux.HandleFunc("DELETE /api/subcategories/{id}", h.RequireAuth(h.DeleteSubcategory))

	// ---- Товары ----
	collection(mux, "GET", "/api/products", h.ListProducts)
	collection(mux, "POST", "/api/products", h.RequireAuth(h.CreateProduct))
	mux.HandleFunc("GET /api/products/{id}", h.GetProduct)
	mux.HandleFunc("PUT /api/products/{id}", h.RequireAuth(h.UpdateProduct))
	mux.HandleFunc("PATCH /api/products/{id}/status", h.RequireAuth(h.UpdateProductStatus))
	mux.HandleFunc("DELETE /api/products/{id}", h.RequireAuth(h.DeleteProduct))
	mux.HandleFunc("GET /api/products/{id}/characteristics", h.GetProductCharacteristics)
	mux.HandleFunc("POST /api/products/{id}/characteristics", h.RequireAuth(h.SetProductCharacteristic))
	mux.HandleFunc("DELETE /api/products/{id}/characteristics/{characteristicID}", h.RequireAuth(h.DeleteProductCharacteristic))

	// ---- Услуги ----
	collection(mux, "GET", "/api/services", h.ListServices)
	collection(mux, "POST", "/api/services", h.RequireAuth(h.CreateService))
	mux.HandleFunc("GET /api/services/{id}", h.GetService)
	mux.HandleFunc("PUT /api/services/{id}", h.RequireAuth(h.UpdateService))
	mux.HandleFunc("PATCH /api/services/{id}/status", h.RequireAuth(h.UpdateServiceStatus))
	mux.HandleFunc("DELETE /api/services/{id}", h.RequireAuth(h.DeleteService))

	// ---- Характеристики ----
	collection(mux, "GET", "/api/characteristics", h.ListCharacteristics)
	collection(mux, "POST", "/api/characteristics", h.RequireAuth(h.CreateCharacteristic))
	mux.HandleFunc("PUT /api/characteristics/{id}", h.RequireAuth(h.UpdateCharacteristic))
	mux.HandleFunc("DELETE /api/characteristics/{id}", h.RequireAuth(h.DeleteCharacteristic))
	mux.HandleFunc("GET /api/characteristics/{id}/values", h.ListCharacteristicValues)
	mux.HandleFunc("POST /api/characteristics/{id}/values", h.RequireAuth(h.CreateCharacteristicValue))
	mux.HandleFunc("PUT /api/characteristics/{id}/values/{valueID}", h.RequireAuth(h.UpdateCharacteristicValue))
	mux.HandleFunc("DELETE /api/characteristics/{id}/values/{valueID}", h.RequireAuth(h.DeleteCharacteristicValue))

	// ---- Объекты ----
	collection(mux, "GET", "/api/object-types", h.ListObjectTypes)
	collection(mux, "POST", "/api/object-types", h.RequireAuth(h.CreateObjectType))
	mux.HandleFunc("GET /api/object-types/{id}", h.GetObjectType)
	mux.HandleFunc("PUT /api/object-types/{id}", h.RequireAuth(h.UpdateObjectType))
	mux.HandleFunc("PATCH /api/object-types/{id}/status", h.RequireAuth(h.UpdateObjectTypeStatus))
	mux.HandleFunc("DELETE /api/object-types/{id}", h.RequireAuth(h.DeleteObjectType))

	// ---- Преимущества ----
	collection(mux, "GET", "/api/advantages", h.ListAdvantages)
	collection(mux, "POST", "/api/advantages", h.RequireAuth(h.CreateAdvantage))
	mux.HandleFunc("GET /api/advantages/{id}", h.GetAdvantage)
	mux.HandleFunc("PUT /api/advantages/{id}", h.RequireAuth(h.UpdateAdvantage))
	mux.HandleFunc("PATCH /api/advantages/{id}/status", h.RequireAuth(h.UpdateAdvantageStatus))
	mux.HandleFunc("DELETE /api/advantages/{id}", h.RequireAuth(h.DeleteAdvantage))

	// ---- Медиа ----
	mux.HandleFunc("POST /api/media/upload", h.RequireAuth(h.UploadMedia))
	collection(mux, "GET", "/api/media", h.ListMedia)
	mux.HandleFunc("DELETE /api/media/{id}", h.RequireAuth(h.DeleteMedia))
	mux.HandleFunc("PATCH /api/media/{id}/main", h.RequireAuth(h.SetMainMedia))

	// Понятный JSON вместо пустой стандартной страницы 404.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteError(w, http.StatusNotFound, "маршрут не найден: "+r.Method+" "+r.URL.Path)
	})

	return Logger(Recoverer(CORS(mux)))
}

// collection вешает обработчик и на /api/xxx, и на /api/xxx/ -
// чтобы фронтенду было всё равно, ставить завершающий слэш или нет.
func collection(mux *http.ServeMux, method, path string, handler http.HandlerFunc) {
	mux.HandleFunc(method+" "+path, handler)
	mux.HandleFunc(method+" "+path+"/{$}", handler)
}
