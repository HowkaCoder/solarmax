package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"solarmax/internal/authutil"
	"solarmax/internal/config"
	"solarmax/internal/db"
	"solarmax/internal/handlers"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // не критично, если .env отсутствует (например, в Docker)

	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.ConnString())
	if err != nil {
		log.Fatalf("не удалось подключиться к БД: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool, "migrations"); err != nil {
		log.Fatalf("не удалось выполнить миграции: %v", err)
	}
	log.Println("миграции выполнены успешно")

	if err := authutil.SeedDefaultAdmin(ctx, pool); err != nil {
		log.Fatalf("не удалось создать администратора по умолчанию: %v", err)
	}
	log.Println("проверка администратора по умолчанию выполнена (admin/admin, если ещё никого нет)")

	if err := os.MkdirAll(cfg.MediaDir, 0755); err != nil {
		log.Fatalf("не удалось создать директорию для медиа: %v", err)
	}

	router := handlers.NewRouter(pool, cfg)

	log.Printf("SolarMax API запущен на порту %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("сервер остановлен с ошибкой: %v", err)
	}
}
