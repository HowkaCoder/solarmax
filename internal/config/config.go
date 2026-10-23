package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config - все настройки приложения. Внешняя БД больше не нужна:
// данные лежат внутри процесса и сохраняются в один JSON-файл.
type Config struct {
	Port string

	// Хранилище
	DataDir  string // папка, где лежат данные и загруженные картинки
	DataFile string // файл снапшота хранилища
	MediaDir string // папка с загруженными файлами
	Persist  bool   // писать ли снапшот на диск (false = чистый in-memory)

	// Авторизация
	JWTSecret      string
	JWTExpiryHours int
	AdminUsername  string // логин администратора, создаётся при первом запуске
	AdminPassword  string // пароль администратора, создаётся при первом запуске

	// Антизасыпание на бесплатном хостинге
	KeepAliveEnabled  bool
	KeepAliveInterval time.Duration
	KeepAliveURLs     []string
}

func Load() *Config {
	dataDir := getEnv("DATA_DIR", "./data")

	cfg := &Config{
		// Render/Railway/Fly сами прокидывают PORT - слушаем именно его.
		Port: getEnv("PORT", "8080"),

		DataDir:  dataDir,
		DataFile: getEnv("DATA_FILE", filepath.Join(dataDir, "solarmax.json")),
		MediaDir: getEnv("MEDIA_DIR", filepath.Join(dataDir, "media")),
		Persist:  getEnvBool("PERSIST", true),

		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 24),
		AdminUsername:  getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:  os.Getenv("ADMIN_PASSWORD"),

		KeepAliveEnabled:  getEnvBool("KEEPALIVE_ENABLED", true),
		KeepAliveInterval: getEnvDuration("KEEPALIVE_INTERVAL", time.Hour),
	}

	cfg.KeepAliveURLs = resolveKeepAliveURLs(cfg.Port)
	return cfg
}

// resolveKeepAliveURLs определяет, куда стучаться раз в час.
// Приоритет: явный KEEPALIVE_URL -> публичный адрес от хостинга -> сам себя.
func resolveKeepAliveURLs(port string) []string {
	if raw := os.Getenv("KEEPALIVE_URL"); raw != "" {
		var out []string
		for _, u := range strings.Split(raw, ",") {
			if u = strings.TrimSpace(u); u != "" {
				out = append(out, u)
			}
		}
		if len(out) > 0 {
			return out
		}
	}

	// Render и Railway сами кладут публичный адрес в окружение.
	for _, key := range []string{"PUBLIC_URL", "RENDER_EXTERNAL_URL", "RAILWAY_PUBLIC_DOMAIN", "FLY_APP_NAME"} {
		v := os.Getenv(key)
		if v == "" {
			continue
		}
		switch key {
		case "RAILWAY_PUBLIC_DOMAIN":
			v = "https://" + v
		case "FLY_APP_NAME":
			v = "https://" + v + ".fly.dev"
		}
		return []string{strings.TrimRight(v, "/") + "/health"}
	}

	// Фоллбэк: пинг самого себя. Процесс точно не считается простаивающим,
	// но внешний трафик это не заменяет (см. README).
	return []string{"http://127.0.0.1:" + port + "/health"}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return def
}

// getEnvDuration понимает и "60m"/"1h30m", и просто число минут ("60").
func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		return d
	}
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return time.Duration(n) * time.Minute
	}
	return def
}
