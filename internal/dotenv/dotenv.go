// Package dotenv - минимальная замена github.com/joho/godotenv.
// Нужен, чтобы проект собирался вообще без внешних зависимостей.
package dotenv

import (
	"bufio"
	"os"
	"strings"
)

// Load читает файл .env (если он есть) и выставляет переменные окружения,
// которые ещё не заданы в системе. Отсутствие файла - не ошибка.
func Load(paths ...string) {
	if len(paths) == 0 {
		paths = []string{".env"}
	}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")

			eq := strings.Index(line, "=")
			if eq < 1 {
				continue
			}
			key := strings.TrimSpace(line[:eq])
			val := strings.TrimSpace(line[eq+1:])

			// Снимаем кавычки, если значение в них обёрнуто.
			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
					val = val[1 : len(val)-1]
				}
			}

			// Системное окружение всегда важнее файла .env.
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, val)
			}
		}
		f.Close()
	}
}
