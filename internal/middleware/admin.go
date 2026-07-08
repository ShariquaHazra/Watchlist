package middleware

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"watchlist-backend/pkg/models"
)

func forbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(models.Response{
		Success: false,
		Message: message,
	})
}

// AdminMiddleware: AuthMiddleware ke BAAD lagana chahiye (user_id context mein pehle
// se hona chahiye). Har request pe DB se check karta hai ki caller admin hai ya nahi —
// DB-driven check isliye rakha hai (JWT claim mein nahi) taaki agar kisi ka admin access
// hata diya jaye, uska purana token bhi turant block ho jaye, expiry ka wait na karna pade.
func AdminMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value("user_id").(int)
			if !ok {
				forbidden(w, "admin access required")
				return
			}

			var isAdmin bool
			err := db.QueryRow(`SELECT is_admin FROM users WHERE id = $1`, userID).Scan(&isAdmin)
			if err != nil || !isAdmin {
				forbidden(w, "admin access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}