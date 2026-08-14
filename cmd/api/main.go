package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"solarmax/internal/config"
	"solarmax/internal/db"
	"solarmax/internal/handlers"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() 

	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.ConnString())
	if err != nil {
		log.Fatalf("не удалось подключиться к бдшке: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool, "migrations/001_init.sql"); err != nil {
		log.Fatalf("не удалось выполнить миграции ебать: %v", err)
	}
	log.Println("миграции выполнены ахуенно")

	if err := os.MkdirAll(cfg.MediaDir, 0755); err != nil {
		log.Fatalf("не удалось создать папку для медиа: %v", err)
	}

	router := handlers.NewRouter(pool, cfg)

	log.Printf("SolarMax API запущен на порту %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
