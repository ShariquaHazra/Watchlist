package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"watchlist-backend/pkg/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

// userAgent: request ka User-Agent header — device-type auto-detect ke liye
// (agar req.DeviceType explicitly bheja gaya ho to wahi priority lega)
func (s *Service) Register(req *models.RegisterRequest, userAgent string) (*models.AuthResponse, error) {

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	// 1. check if user already exists
	_, _, err := s.repo.GetUserByEmail(req.Email)
	if err == nil {
		return nil, errors.New("email already exists")
	}

	if err.Error() != "user not found" {
		return nil, err
	}

	// 2. hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 3. create user
	user := &models.User{
		Name:  req.Name,
		Email: req.Email,
	}

	err = s.repo.CreateUser(user, string(hash))
	if err != nil {
		return nil, err
	}

	// 4. session banao + token generate karo
	return s.issueSessionAndToken(user, req.DeviceType, userAgent)
}

func (s *Service) Login(req *models.LoginRequest, userAgent string) (*models.AuthResponse, error) {

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, passwordHash, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Password verify karo
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Naya login — session banao (isse purana same-device-type session
	// automatically invalid ho jayega, kyunki DB row overwrite hoti hai)
	return s.issueSessionAndToken(user, req.DeviceType, userAgent)
}

// Logout: is device_type ka session hata deta hai + uska jti blacklist mein daal deta
// hai — token ab turant invalid hai, chahe uska "exp" abhi door ho
func (s *Service) Logout(userID int, deviceType string) error {
	return s.repo.DeleteSession(userID, deviceType)
}

// ListSessions: user ke saare active sessions (mobile + desktop) — remote-session view ke liye
func (s *Service) ListSessions(userID int) ([]models.SessionInfo, error) {
	sessions, err := s.repo.GetUserSessions(userID)
	if err != nil {
		return nil, err
	}

	result := make([]models.SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		result = append(result, models.SessionInfo{
			DeviceType: sess.DeviceType,
			DeviceInfo: sess.DeviceInfo,
			CreatedAt:  sess.CreatedAt,
			LastSeenAt: sess.LastSeenAt,
		})
	}
	return result, nil
}

// RevokeSession: kisi bhi device_type (mobile/desktop) ka session force-logout karta hai —
// "logout from other device" jaisa remote-logout feature ke liye. Blacklist mein bhi jti
// daal deta hai taaki wo token turant reject ho, active-session-row delete hone ke bawajood
// bhi agar attacker ke paas old token ho.
func (s *Service) RevokeSession(userID int, deviceType string) error {
	if deviceType != models.DeviceMobile && deviceType != models.DeviceDesktop {
		return errors.New("invalid device_type — must be 'mobile' or 'desktop'")
	}
	return s.repo.DeleteSession(userID, deviceType)
}

// ── Helpers: device detection, session id, token ──────────────────

func (s *Service) issueSessionAndToken(user *models.User, explicitDeviceType, userAgent string) (*models.AuthResponse, error) {
	deviceType := explicitDeviceType
	if deviceType == "" {
		deviceType = detectDeviceType(userAgent)
	}

	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	// Purana session (agar isi device_type ka hai) yahan replace ho jayega
	if err := s.repo.UpsertSession(user.ID, deviceType, sessionID, userAgent); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user.ID, sessionID, deviceType)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{Token: token, User: *user}, nil
}

// Simple User-Agent based heuristic — "mobile" keywords match nahi hue to desktop
func detectDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)
	mobileKeywords := []string{"mobile", "android", "iphone", "ipad", "ipod", "windows phone", "blackberry"}
	for _, kw := range mobileKeywords {
		if strings.Contains(ua, kw) {
			return models.DeviceMobile
		}
	}
	return models.DeviceDesktop
}

// Random session id — crypto/rand se, koi extra dependency ki zaroorat nahi
func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) generateToken(userID int, sessionID, deviceType string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":     userID,
		"sid":         sessionID,
		"device_type": deviceType,
		"exp":         time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}