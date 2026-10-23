package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"solarmax/internal/authutil"
	"solarmax/internal/config"
	"solarmax/internal/dotenv"
	"solarmax/internal/handlers"
	"solarmax/internal/keepalive"
	"solarmax/internal/store"
)

func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("crypto/rand: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}

func main() {
	dotenv.Load() // не критично, если .env отсутствует

	cfg := config.Load()

	// ---- Хранилище: всё внутри приложения, внешняя БД не нужна ----
	st, err := store.New(cfg.DataFile, cfg.Persist)
	if err != nil {
		log.Fatalf("не удалось инициализировать хранилище: %v", err)
	}
	if cfg.Persist {
		log.Printf("хранилище: данные в памяти + снапшот в %s", cfg.DataFile)
	} else {
		log.Println("хранилище: только в памяти (PERSIST=false), данные исчезнут при перезапуске")
	}

	if cfg.JWTSecret == "" {
		cfg.JWTSecret = randomString(32)
		log.Println("JWT_SECRET не задан: сгенерирован случайный секрет, токены станут недействительны после перезапуска")
	}

	adminPassword := cfg.AdminPassword
	if adminPassword == "" {
		adminPassword = randomString(12)
	}
	hash, err := authutil.HashPassword(adminPassword)
	if err != nil {
		log.Fatalf("не удалось захэшировать пароль администратора: %v", err)
	}
	created, err := st.EnsureAdmin(cfg.AdminUsername, hash)
	if err != nil {
		log.Fatalf("не удалось создать администратора: %v", err)
	}
	if created && cfg.AdminPassword == "" {
		log.Printf("создан администратор %q с паролем %q, смените его через /api/auth/change-password", cfg.AdminUsername, adminPassword)
	} else if created {
		log.Printf("создан администратор %q", cfg.AdminUsername)
	}

	if err := os.MkdirAll(cfg.MediaDir, 0o755); err != nil {
		log.Fatalf("не удалось создать директорию для медиа: %v", err)
	}

	// ---- Фоновые задачи ----
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st.StartAutosave(ctx.Done(), 3*time.Second)

	var pinger *keepalive.Pinger
	if cfg.KeepAliveEnabled {
		pinger = keepalive.New(cfg.KeepAliveURLs, cfg.KeepAliveInterval)
		pinger.Start(ctx)
	} else {
		log.Println("keepalive: выключен (KEEPALIVE_ENABLED=false)")
	}

	// ---- HTTP-сервер ----
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handlers.NewRouter(st, cfg, pinger),
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("SolarMax API запущен на порту %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("сервер остановлен с ошибкой: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("получен сигнал остановки, завершаем работу...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("ошибка при остановке сервера: %v", err)
	}

	// Финальный снапшот, чтобы не потерять последние изменения.
	if err := st.Save(); err != nil {
		log.Printf("не удалось сохранить данные: %v", err)
	}
	log.Println("остановлено")
}
