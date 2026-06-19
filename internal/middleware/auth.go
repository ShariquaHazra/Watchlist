package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"watchlist-backend/pkg/models"

	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Header se token lo
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(models.Response{
					Success: false,
					Message: "token required",
				})
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Token verify karo
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(models.Response{
					Success: false,
					Message: "invalid token",
				})
				return
			}

			// user_id context mein daalo
			claims := token.Claims.(jwt.MapClaims)
			userID := int(claims["user_id"].(float64))
			ctx := context.WithValue(r.Context(), "user_id", userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
