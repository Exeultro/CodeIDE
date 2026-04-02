package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"collab-ide-backend/internal/api/rest"
	"collab-ide-backend/internal/auth"
)

func JWTAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(rest.Fail(401, "missing authorization header"))
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(rest.Fail(401, "invalid authorization format"))
			return
		}
		token := parts[1]
		claims, err := auth.ValidateToken(token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(rest.Fail(401, "invalid token"))
			return
		}
		// Сохраняем claims в контекст запроса (можно использовать request context)
		ctx := context.WithValue(r.Context(), "user", claims)
		next(w, r.WithContext(ctx))
	}
}
