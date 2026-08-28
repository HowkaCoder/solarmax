package handlers

import (
	"net/http"

	"solarmax/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool, cfg *config.Config) http.Handler {
	h := &Handler{Pool: pool, Cfg: cfg}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: false,
	}))

	// Статическая раздача загруженных файлов: /media/<entity_type>/<id>/<file>
	fileServer := http.FileServer(http.Dir(cfg.MediaDir))
	r.Handle("/media/*", http.StripPrefix("/media/", fileServer))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Route("/api", func(r chi.Router) {
		// ---- Авторизация ----
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", h.Login) // публичный - единственный способ получить токен
			r.With(h.RequireAuth).Get("/me", h.Me)
			r.With(h.RequireAuth).Post("/change-password", h.ChangePassword)
		})

		// ---- Категории ---- (GET открыт всем, остальное - только с токеном)
		r.Route("/categories", func(r chi.Router) {
			r.Get("/", h.ListCategories)
			r.With(h.RequireAuth).Post("/", h.CreateCategory)
			r.Get("/{id}", h.GetCategory)
			r.With(h.RequireAuth).Put("/{id}", h.UpdateCategory)
			r.With(h.RequireAuth).Delete("/{id}", h.DeleteCategory)
		})

		// ---- Подкатегории ----
		r.Route("/subcategories", func(r chi.Router) {
			r.Get("/", h.ListSubcategories)
			r.With(h.RequireAuth).Post("/", h.CreateSubcategory)
			r.Get("/{id}", h.GetSubcategory)
			r.With(h.RequireAuth).Put("/{id}", h.UpdateSubcategory)
			r.With(h.RequireAuth).Delete("/{id}", h.DeleteSubcategory)
		})

		// ---- Товары ----
		r.Route("/products", func(r chi.Router) {
			r.Get("/", h.ListProducts)
			r.With(h.RequireAuth).Post("/", h.CreateProduct)
			r.Get("/{id}", h.GetProduct)
			r.With(h.RequireAuth).Put("/{id}", h.UpdateProduct)
			r.With(h.RequireAuth).Patch("/{id}/status", h.UpdateProductStatus)
			r.With(h.RequireAuth).Delete("/{id}", h.DeleteProduct)

			r.Get("/{id}/characteristics", h.GetProductCharacteristics)
			r.With(h.RequireAuth).Post("/{id}/characteristics", h.SetProductCharacteristic)
			r.With(h.RequireAuth).Delete("/{id}/characteristics/{characteristicID}", h.DeleteProductCharacteristic)
		})

		// ---- Услуги ----
		r.Route("/services", func(r chi.Router) {
			r.Get("/", h.ListServices)
			r.With(h.RequireAuth).Post("/", h.CreateService)
			r.Get("/{id}", h.GetService)
			r.With(h.RequireAuth).Put("/{id}", h.UpdateService)
			r.With(h.RequireAuth).Patch("/{id}/status", h.UpdateServiceStatus)
			r.With(h.RequireAuth).Delete("/{id}", h.DeleteService)
		})

		// ---- Характеристики ----
		r.Route("/characteristics", func(r chi.Router) {
			r.Get("/", h.ListCharacteristics)
			r.With(h.RequireAuth).Post("/", h.CreateCharacteristic)
			r.With(h.RequireAuth).Put("/{id}", h.UpdateCharacteristic)
			r.With(h.RequireAuth).Delete("/{id}", h.DeleteCharacteristic)

			r.Get("/{id}/values", h.ListCharacteristicValues)
			r.With(h.RequireAuth).Post("/{id}/values", h.CreateCharacteristicValue)
			r.With(h.RequireAuth).Put("/{id}/values/{valueID}", h.UpdateCharacteristicValue)
			r.With(h.RequireAuth).Delete("/{id}/values/{valueID}", h.DeleteCharacteristicValue)
		})

		// ---- Объекты (типы объектов, "Учебные заведения" и т.д.) ----
		r.Route("/object-types", func(r chi.Router) {
			r.Get("/", h.ListObjectTypes)
			r.With(h.RequireAuth).Post("/", h.CreateObjectType)
			r.Get("/{id}", h.GetObjectType)
			r.With(h.RequireAuth).Put("/{id}", h.UpdateObjectType)
			r.With(h.RequireAuth).Patch("/{id}/status", h.UpdateObjectTypeStatus)
			r.With(h.RequireAuth).Delete("/{id}", h.DeleteObjectType)
		})

		// ---- Преимущества ("Что мы предлагаем") ----
		r.Route("/advantages", func(r chi.Router) {
			r.Get("/", h.ListAdvantages)
			r.With(h.RequireAuth).Post("/", h.CreateAdvantage)
			r.Get("/{id}", h.GetAdvantage)
			r.With(h.RequireAuth).Put("/{id}", h.UpdateAdvantage)
			r.With(h.RequireAuth).Patch("/{id}/status", h.UpdateAdvantageStatus)
			r.With(h.RequireAuth).Delete("/{id}", h.DeleteAdvantage)
		})

		// ---- Медиа ----
		r.Route("/media", func(r chi.Router) {
			r.With(h.RequireAuth).Post("/upload", h.UploadMedia)
			r.Get("/", h.ListMedia)
			r.With(h.RequireAuth).Delete("/{id}", h.DeleteMedia)
			r.With(h.RequireAuth).Patch("/{id}/main", h.SetMainMedia)
		})
	})

	return r
}
