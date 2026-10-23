package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"solarmax/internal/authutil"
	"solarmax/internal/config"
	"solarmax/internal/store"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	st, err := store.New(filepath.Join(dir, "data.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := authutil.HashPassword("admin-pass")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.EnsureAdmin("admin", hash); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{JWTSecret: "test-secret", JWTExpiryHours: 1, MediaDir: filepath.Join(dir, "media")}
	return NewRouter(st, cfg, nil)
}

func do(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func login(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := do(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{"username": "admin", "password": "admin-pass"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	var resp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	return resp.Token
}

func TestLogin(t *testing.T) {
	h := newTestServer(t)
	if login(t, h) == "" {
		t.Fatal("empty token")
	}

	rec := do(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{"username": "admin", "password": "nope"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d", rec.Code)
	}
}

func TestWriteRequiresToken(t *testing.T) {
	h := newTestServer(t)

	rec := do(t, h, http.MethodPost, "/api/categories", "", map[string]string{"name": "Panels"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	rec = do(t, h, http.MethodPost, "/api/categories", login(t, h), map[string]string{"name": "Panels"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 with token, got %d: %s", rec.Code, rec.Body)
	}

	rec = do(t, h, http.MethodGet, "/api/categories", "", nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Panels")) {
		t.Fatalf("public list: %d %s", rec.Code, rec.Body)
	}
}

func TestBackupRequiresToken(t *testing.T) {
	h := newTestServer(t)
	if rec := do(t, h, http.MethodGet, "/api/admin/backup", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestNotFoundIsJSON(t *testing.T) {
	h := newTestServer(t)
	rec := do(t, h, http.MethodGet, "/api/nope", "", nil)
	if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("got %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}
