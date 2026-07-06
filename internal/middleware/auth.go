package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"watchlist-backend/pkg/models"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtSecret string, db *sql.DB) func(http.Handler) http.Handler {
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

			// ── Session control: sid + device_type nikalo ─────────────
			sessionID, sidOK := claims["sid"].(string)
			deviceType, dtOK := claims["device_type"].(string)

			if !sidOK || !dtOK || sessionID == "" || deviceType == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(models.Response{
					Success: false,
					Message: "session expired, please login again",
				})
				return
			}

			// DB mein check karo ki yehi abhi ka ACTIVE session hai is device_type ke liye
			var activeSessionID string
			err = db.QueryRow(
				`SELECT session_id FROM user_sessions WHERE user_id = $1 AND device_type = $2`,
				userID, deviceType,
			).Scan(&activeSessionID)

			if err != nil || activeSessionID != sessionID {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(models.Response{
					Success: false,
					Message: "session expired — logged in from another device",
				})
				return
			}
			// ────────────────────────────────────────────────────────

			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "device_type", deviceType)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}