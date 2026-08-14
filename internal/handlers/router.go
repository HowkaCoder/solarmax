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
		r.Route("/categories", func(r chi.Router) {
			r.Get("/", h.ListCategories)
			r.Post("/", h.CreateCategory)
			r.Get("/{id}", h.GetCategory)
			r.Put("/{id}", h.UpdateCategory)
			r.Delete("/{id}", h.DeleteCategory)
		})

		r.Route("/subcategories", func(r chi.Router) {
			r.Get("/", h.ListSubcategories)
			r.Post("/", h.CreateSubcategory)
			r.Get("/{id}", h.GetSubcategory)
			r.Put("/{id}", h.UpdateSubcategory)
			r.Delete("/{id}", h.DeleteSubcategory)
		})

		r.Route("/products", func(r chi.Router) {
			r.Get("/", h.ListProducts)
			r.Post("/", h.CreateProduct)
			r.Get("/{id}", h.GetProduct)
			r.Put("/{id}", h.UpdateProduct)
			r.Patch("/{id}/status", h.UpdateProductStatus)
			r.Delete("/{id}", h.DeleteProduct)

			r.Get("/{id}/characteristics", h.GetProductCharacteristics)
			r.Post("/{id}/characteristics", h.SetProductCharacteristic)
			r.Delete("/{id}/characteristics/{characteristicID}", h.DeleteProductCharacteristic)
		})

		r.Route("/services", func(r chi.Router) {
			r.Get("/", h.ListServices)
			r.Post("/", h.CreateService)
			r.Get("/{id}", h.GetService)
			r.Put("/{id}", h.UpdateService)
			r.Patch("/{id}/status", h.UpdateServiceStatus)
			r.Delete("/{id}", h.DeleteService)
		})

		r.Route("/characteristics", func(r chi.Router) {
			r.Get("/", h.ListCharacteristics)
			r.Post("/", h.CreateCharacteristic)
			r.Put("/{id}", h.UpdateCharacteristic)
			r.Delete("/{id}", h.DeleteCharacteristic)

			r.Get("/{id}/values", h.ListCharacteristicValues)
			r.Post("/{id}/values", h.CreateCharacteristicValue)
			r.Put("/{id}/values/{valueID}", h.UpdateCharacteristicValue)
			r.Delete("/{id}/values/{valueID}", h.DeleteCharacteristicValue)
		})

		r.Route("/media", func(r chi.Router) {
			r.Post("/upload", h.UploadMedia)
			r.Get("/", h.ListMedia)
			r.Delete("/{id}", h.DeleteMedia)
			r.Patch("/{id}/main", h.SetMainMedia)
		})
	})

	return r
}
