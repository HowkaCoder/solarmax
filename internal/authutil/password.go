package authutil

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"hash"
	"strconv"
	"strings"
)

// Пароли хэшируются PBKDF2-HMAC-SHA256 на чистой стандартной библиотеке,
// чтобы у проекта не было внешних зависимостей (раньше был bcrypt из x/crypto).
// Формат хэша: pbkdf2_sha256$<итерации>$<соль base64>$<хэш base64>

const (
	pbkdf2Prefix = "pbkdf2_sha256"
	iterations   = 120000
	saltLen      = 16
	keyLen       = 32
)

// HashPassword возвращает строку хэша, готовую к сохранению в хранилище.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	dk := pbkdf2([]byte(password), salt, iterations, keyLen, sha256.New)

	return strings.Join([]string{
		pbkdf2Prefix,
		strconv.Itoa(iterations),
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(dk),
	}, "$"), nil
}

// CheckPassword сверяет пароль с ранее сохранённым хэшем.
// Сравнение постоянное по времени - защита от тайминг-атак.
func CheckPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != pbkdf2Prefix {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}

	got := pbkdf2([]byte(password), salt, iter, len(want), sha256.New)
	return subtle.ConstantTimeCompare(got, want) == 1
}

// pbkdf2 - реализация RFC 2898 (PBKDF2) поверх crypto/hmac.
func pbkdf2(password, salt []byte, iter, keyLength int, h func() hash.Hash) []byte {
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	blocks := (keyLength + hashLen - 1) / hashLen

	var out []byte
	buf := make([]byte, 4)
	u := make([]byte, hashLen)

	for block := 1; block <= blocks; block++ {
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf)
		u = prf.Sum(u[:0])

		t := make([]byte, hashLen)
		copy(t, u)

		for i := 1; i < iter; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLength]
}
