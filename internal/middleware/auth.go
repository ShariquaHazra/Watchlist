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

func unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(models.Response{
		Success: false,
		Message: message,
	})
}

// db: session table check karne ke liye chahiye — taaki "1 mobile + 1 desktop"
// session-control rule enforce ho sake (naya login hote hi purana token yahin reject ho jayega)
func AuthMiddleware(jwtSecret string, db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Header se token lo
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				unauthorized(w, "token required")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Token verify karo
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				unauthorized(w, "invalid token")
				return
			}

			claims := token.Claims.(jwt.MapClaims)
			userID := int(claims["user_id"].(float64))

			sessionID, sidOK := claims["sid"].(string)
			deviceType, dtOK := claims["device_type"].(string)
			jti, _ := claims["jti"].(string)

			// Purane tokens jinme sid/device_type nahi hai (feature launch se pehle
			// issue hue) — unhe re-login ke liye force karo
			if !sidOK || !dtOK || sessionID == "" || deviceType == "" {
				unauthorized(w, "session expired, please login again")
				return
			}

			// 1. BLACKLIST check — explicit revocation (logout / remote-logout / security event).
			// Ye check pass hua token ke exp se pehle bhi turant reject kar sakta hai.
			if jti != "" {
				var revoked bool
				err = db.QueryRow(
					`SELECT EXISTS (SELECT 1 FROM revoked_tokens WHERE jti = $1)`,
					jti,
				).Scan(&revoked)
				if err != nil {
					unauthorized(w, "internal error")
					return
				}
				if revoked {
					unauthorized(w, "token has been revoked")
					return
				}
			}

			// 2. WHITELIST check — DB mein check karo ki yehi abhi ka ACTIVE session hai is
			// device_type ke liye. Agar kisi aur device (same type) se naya login hua hoga,
			// to yahan mismatch hoga.
			var activeSessionID string
			err = db.QueryRow(
				`SELECT session_id FROM user_sessions WHERE user_id = $1 AND device_type = $2`,
				userID, deviceType,
			).Scan(&activeSessionID)

			if err != nil || activeSessionID != sessionID {
				unauthorized(w, "session expired — logged in from another device")
				return
			}

			// user_id + device_type context mein daalo
			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "device_type", deviceType)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}