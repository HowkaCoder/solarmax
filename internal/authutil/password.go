package authutil

import "golang.org/x/crypto/bcrypt"

// HashPassword хэширует пароль bcrypt-ом перед сохранением в БД.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword сверяет пароль с хэшем из БД.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
