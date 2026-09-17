package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"solarmax/internal/authutil"
	"solarmax/internal/models"
	"solarmax/internal/utils"
)

type contextKey string

const usernameContextKey contextKey = "username"

// RequireAuth защищает изменяющие эндпоинты.
// Ожидает заголовок: Authorization: Bearer <jwt>
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.WriteError(w, http.StatusUnauthorized, "нужен токен авторизации: заголовок Authorization: Bearer <token>")
			return
		}

		claims, err := authutil.ParseToken(h.Cfg.JWTSecret, strings.TrimPrefix(authHeader, "Bearer "))
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "невалидный или просроченный токен")
			return
		}

		ctx := context.WithValue(r.Context(), usernameContextKey, claims.Username)
		next(w, r.WithContext(ctx))
	}
}

func usernameFromContext(r *http.Request) string {
	v, _ := r.Context().Value(usernameContextKey).(string)
	return v
}

// POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in models.LoginInput
	if err := decodeBody(r, &in); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}
	if in.Username == "" || in.Password == "" {
		utils.WriteError(w, http.StatusBadRequest, "поля username и password обязательны")
		return
	}

	hash, ok := h.Store.PasswordHash(in.Username)
	if !ok || !authutil.CheckPassword(hash, in.Password) {
		utils.WriteError(w, http.StatusUnauthorized, "неверный логин или пароль")
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

// GET /api/auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, map[string]string{"username": usernameFromContext(r)})
}

// POST /api/auth/change-password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	username := usernameFromContext(r)

	var in models.ChangePasswordInput
	if err := decodeBody(r, &in); err != nil {
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

	hash, ok := h.Store.PasswordHash(username)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "пользователь не найден")
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
	if err := h.Store.SetPasswordHash(username, newHash); err != nil {
		writeStoreErr(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "пароль успешно изменён"})
}
