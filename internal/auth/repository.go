package auth

import (
	"database/sql"
	"errors"
	"time"

	"watchlist-backend/pkg/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(user *models.User, passwordHash string) error {
	query := `
		INSERT INTO users (name, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`

	return r.db.QueryRow(query, user.Name, user.Email, passwordHash).Scan(&user.ID)
}

func (r *Repository) GetUserByEmail(email string) (*models.User, string, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	var passwordHash string

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&passwordHash,
		&user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, "", errors.New("user not found")
	}

	if err != nil {
		return nil, "", err
	}

	return user, passwordHash, nil
}

// ── Session Control (1 mobile + 1 desktop per user) ───────────────

// UpsertSession: naya login hote hi is user+device_type ka purana session
// row REPLACE ho jata hai — isi se purana session automatically invalid ho jata hai.
func (r *Repository) UpsertSession(userID int, deviceType, sessionID, deviceInfo string) error {
	query := `
		INSERT INTO user_sessions (user_id, device_type, session_id, jti, device_info, created_at, last_seen_at)
		VALUES ($1, $2, $3, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id, device_type)
		DO UPDATE SET
			session_id   = EXCLUDED.session_id,
			jti          = EXCLUDED.jti,
			device_info  = EXCLUDED.device_info,
			created_at   = NOW(),
			last_seen_at = NOW()
	`
	_, err := r.db.Exec(query, userID, deviceType, sessionID, deviceInfo)
	return err
}

// GetActiveSessionID: is user+device_type ka abhi ka valid session_id deta hai.
func (r *Repository) GetActiveSessionID(userID int, deviceType string) (string, error) {
	var sessionID string
	query := `SELECT session_id FROM user_sessions WHERE user_id = $1 AND device_type = $2`
	err := r.db.QueryRow(query, userID, deviceType).Scan(&sessionID)
	if err == sql.ErrNoRows {
		return "", errors.New("session not found")
	}
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

// DeleteSession: logout ke liye — is device_type ka session row hata deta hai.
func (r *Repository) DeleteSession(userID int, deviceType string) error {
	query := `DELETE FROM user_sessions WHERE user_id = $1 AND device_type = $2`
	_, err := r.db.Exec(query, userID, deviceType)
	return err
}

// GetUserSessions: is user ke saare active sessions (mobile + desktop) list karta hai
func (r *Repository) GetUserSessions(userID int) ([]models.UserSession, error) {
	query := `
		SELECT id, user_id, device_type, session_id, device_info, created_at, last_seen_at
		FROM user_sessions
		WHERE user_id = $1
		ORDER BY device_type
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []models.UserSession{}
	for rows.Next() {
		var s models.UserSession
		var deviceInfo sql.NullString
		if err := rows.Scan(&s.ID, &s.UserID, &s.DeviceType, &s.SessionID, &deviceInfo, &s.CreatedAt, &s.LastSeenAt); err != nil {
			return nil, err
		}
		s.DeviceInfo = deviceInfo.String
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// ── JWT Revocation (blacklist) ─────────────────────────────────────

// RevokeAndDeleteSession: is user+device_type ke session ka jti blacklist mein daal deta
// hai (jab tak token naturally expire na ho jaata) aur whitelist row bhi hata deta hai.
// Logout aur remote-logout, dono isi ko use karte hain.
func (r *Repository) RevokeAndDeleteSession(userID int, deviceType string) error {
	var jti string
	var createdAt time.Time

	err := r.db.QueryRow(
		`SELECT jti, created_at FROM user_sessions WHERE user_id = $1 AND device_type = $2`,
		userID, deviceType,
	).Scan(&jti, &createdAt)

	if err == sql.ErrNoRows {
		return nil // is device_type ka koi active session hi nahi hai, kuch revoke karne ko nahi
	}
	if err != nil {
		return err
	}

	// Token ki TTL (24h, generateToken ke sath match) — us hisaab se expiry approx karo
	expiresAt := createdAt.Add(24 * time.Hour)

	_, err = r.db.Exec(
		`INSERT INTO revoked_tokens (jti, user_id, expires_at) VALUES ($1, $2, $3) ON CONFLICT (jti) DO NOTHING`,
		jti, userID, expiresAt,
	)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		`DELETE FROM user_sessions WHERE user_id = $1 AND device_type = $2`,
		userID, deviceType,
	)
	return err
}

// IsTokenRevoked: middleware isko har request pe call karega
func (r *Repository) IsTokenRevoked(jti string) (bool, error) {
	var revoked bool
	err := r.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM revoked_tokens WHERE jti = $1)`,
		jti,
	).Scan(&revoked)
	return revoked, err
}

// CleanupExpiredRevokedTokens: scheduled job isko periodically call karega taaki
// table chhota/efficient rahe — jin tokens ki natural expiry beet chuki hai, unhe
// blacklist mein rakhne ka koi fayda nahi (wo waise hi ab invalid ho chuke hain)
func (r *Repository) CleanupExpiredRevokedTokens() (int64, error) {
	result, err := r.db.Exec(`DELETE FROM revoked_tokens WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}