package authutil

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPassword(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "pbkdf2_sha256$") {
		t.Fatalf("unexpected hash format: %s", hash)
	}
	if !CheckPassword(hash, "s3cret-pass") {
		t.Fatal("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}

	other, _ := HashPassword("s3cret-pass")
	if other == hash {
		t.Fatal("salt is not random")
	}
}

func TestToken(t *testing.T) {
	token, _, err := GenerateToken("secret", "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := ParseToken("secret", token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Username != "admin" {
		t.Fatalf("username = %q", claims.Username)
	}

	if _, err := ParseToken("other", token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("wrong secret: %v", err)
	}

	parts := strings.Split(token, ".")
	forged := parts[0] + "." + b64([]byte(`{"username":"root","exp":9999999999}`)) + "." + parts[2]
	if _, err := ParseToken("secret", forged); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("forged payload: %v", err)
	}
}

func TestTokenExpired(t *testing.T) {
	token, _, err := GenerateToken("secret", "admin", -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken("secret", token); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}
