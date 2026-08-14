package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	MediaDir    string
	Port        string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", ""),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "solarmax"),
		DBPassword:  getEnv("DB_PASSWORD", "solarmax"),
		DBName:      getEnv("DB_NAME", "solarmax"),
		DBSSLMode: getEnv("DB_SSLMODE", "disable"),
		MediaDir:  getEnv("MEDIA_DIR", "./media"),
		Port: getEnv("PORT", "8080"),
	}
}

func (c *Config) ConnString() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
