package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"solarmax/internal/authutil"
	"solarmax/internal/models"
	"solarmax/internal/utils"

	"github.com/jackc/pgx/v5"
)

type contextKey string

const usernameContextKey contextKey = "username"

// RequireAuth - middleware, защищающий изменяющие эндпоинты (POST/PUT/PATCH/DELETE).
// Ожидает заголовок: Authorization: Bearer <jwt>
func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.WriteError(w, http.StatusUnauthorized, "нужен токен авторизации: заголовок Authorization: Bearer <token>")
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := authutil.ParseToken(h.Cfg.JWTSecret, tokenStr)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "невалидный или просроченный токен")
			return
		}

		ctx := context.WithValue(r.Context(), usernameContextKey, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func usernameFromContext(r *http.Request) string {
	v, _ := r.Context().Value(usernameContextKey).(string)
	return v
}

// POST /api/auth/login - публичный эндпоинт, единственный вход в систему.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in models.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Username == "" || in.Password == "" {
		utils.WriteError(w, http.StatusBadRequest, "поля username и password обязательны")
		return
	}

	var hash string
	err := h.Pool.QueryRow(ctx, `SELECT password_hash FROM admins WHERE username=$1`, in.Username).Scan(&hash)
	if err == pgx.ErrNoRows || (err == nil && !authutil.CheckPassword(hash, in.Password)) {
		utils.WriteError(w, http.StatusUnauthorized, "неверный логин или пароль")
		return
	}
	if err != nil && err != pgx.ErrNoRows {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ttl := time.Duration(h.Cfg.JWTExpiryHours) * time.Hour
	token, expiresAt, err := authutil.GenerateToken(h.Cfg.JWTSecret, in.Username, ttl)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось создать токен")
		return
	}

	utils.WriteJSON(w, http.StatusOK, models.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Username:  in.Username,
	})
}

// GET /api/auth/me - защищённый эндпоинт для проверки токена фронтом.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, map[string]string{"username": usernameFromContext(r)})
}

// POST /api/auth/change-password - защищённый эндпоинт смены пароля текущего админа.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := usernameFromContext(r)

	var in models.ChangePasswordInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.OldPassword == "" || in.NewPassword == "" {
		utils.WriteError(w, http.StatusBadRequest, "поля old_password и new_password обязательны")
		return
	}
	if len(in.NewPassword) < 6 {
		utils.WriteError(w, http.StatusBadRequest, "новый пароль должен быть не короче 6 символов")
		return
	}

	var hash string
	if err := h.Pool.QueryRow(ctx, `SELECT password_hash FROM admins WHERE username=$1`, username).Scan(&hash); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !authutil.CheckPassword(hash, in.OldPassword) {
		utils.WriteError(w, http.StatusUnauthorized, "старый пароль неверен")
		return
	}

	newHash, err := authutil.HashPassword(in.NewPassword)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "не удалось захэшировать пароль")
		return
	}

	if _, err := h.Pool.Exec(ctx, `UPDATE admins SET password_hash=$1 WHERE username=$2`, newHash, username); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "пароль успешно изменён"})
}
