package authutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// JWT (HS256) собран на стандартной библиотеке: формат токена и подпись
// полностью совместимы с прежней реализацией на github.com/golang-jwt/jwt,
// поэтому фронтенду ничего менять не нужно.

type Claims struct {
	Username  string `json:"username"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

var (
	ErrInvalidToken = errors.New("токен невалиден")
	ErrTokenExpired = errors.New("срок действия токена истёк")
)

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// GenerateToken создаёт подписанный токен с заданным временем жизни.
func GenerateToken(secret, username string, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	now := time.Now()
	expiresAt = now.Add(ttl)

	headerJSON, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", time.Time{}, err
	}
	claimsJSON, err := json.Marshal(Claims{
		Username:  username,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  now.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}

	unsigned := b64(headerJSON) + "." + b64(claimsJSON)
	return unsigned + "." + b64(sign(secret, unsigned)), expiresAt, nil
}

// ParseToken проверяет подпись и срок годности, возвращает полезную нагрузку.
func ParseToken(secret, tokenStr string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	var header jwtHeader
	headerJSON, err := unb64(parts[0])
	if err != nil || json.Unmarshal(headerJSON, &header) != nil || header.Alg != "HS256" {
		return nil, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	gotSig, err := unb64(parts[2])
	if err != nil || !hmac.Equal(gotSig, sign(secret, unsigned)) {
		return nil, ErrInvalidToken
	}

	claimsJSON, err := unb64(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Username == "" {
		return nil, ErrInvalidToken
	}
	if claims.ExpiresAt > 0 && time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

func sign(secret, payload string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func b64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func unb64(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
