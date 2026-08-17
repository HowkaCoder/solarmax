package authutil

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims - то, что зашито внутри JWT. Хранит только имя пользователя,
// т.к. проект простой и ролей/прав пока нет - есть только один тип аккаунта (админ).
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken создаёт подписанный JWT (HS256) с заданным временем жизни.
func GenerateToken(secret, username string, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	expiresAt = time.Now().Add(ttl)
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = t.SignedString([]byte(secret))
	return token, expiresAt, err
}

// ParseToken проверяет подпись и срок годности токена, возвращает claims.
func ParseToken(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи токена: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("токен невалиден")
	}
	return claims, nil
}
